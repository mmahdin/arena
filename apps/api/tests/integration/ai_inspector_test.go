package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/app/composition"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/memorycrypto"
	mem "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/orchestrator"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/llm"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/fake"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
)

// Uses disposable SQL Server databases through the existing local-only harness.
// No external model, live conversation or persistent application database is used.
func TestInspectorSQLFinalizationAndRollback(t *testing.T) {
	db := aiDatabase(t)
	aiMigrate(t, db)
	for _, rollback := range []bool{false, true} {
		name := "commit"
		if rollback {
			name = "rollback"
		}
		t.Run(name, func(t *testing.T) {
			f := aiSetup(t, db)
			f.state.fail = rollback
			core, provider := inspectorRuntime(t, f)
			dir := filepath.Join(t.TempDir(), "private")
			ins, err := trace.NewInspector(trace.InspectorOptions{Directory: dir, Environment: "local", Address: "localhost:8080"})
			if err != nil {
				t.Fatal(err)
			}
			result, runErr := core.ProcessTurn(trace.WithInspector(context.Background(), ins), f.req())
			if (runErr != nil) != rollback {
				t.Fatalf("SQL outcome differs %+v %v", result, runErr)
			}
			wantMessages, wantRecords := 2, 1
			if rollback {
				wantMessages, wantRecords = 1, 0
			}
			if got := aiCount(t, db, `SELECT COUNT(*) FROM dbo.messages WHERE turn_id=@p1`, result.TurnID); got != wantMessages {
				t.Fatal("incorrect message atomicity", got)
			}
			if got := aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_outbox WHERE user_id=@p1`, f.a); got != wantRecords {
				t.Fatal("incorrect outbox atomicity", got)
			}
			if got := aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_turn_traces WHERE turn_id=@p1`, result.TurnID); got != wantRecords {
				t.Fatal("incorrect durable metadata atomicity", got)
			}
			files, _ := filepath.Glob(filepath.Join(dir, "*.json"))
			if len(files) != 1 {
				t.Fatal(files)
			}
			raw, err := os.ReadFile(files[0])
			if err != nil {
				t.Fatal(err)
			}
			if err = trace.ValidateJSON(raw); err != nil {
				t.Fatal(err)
			}
			if target := os.Getenv("THEIA_WRITE_JEV_FIXTURE"); target != "" && !rollback {
				if e := os.WriteFile(target, raw, 0600); e != nil {
					t.Fatal(e)
				}
			}
			var doc trace.Document
			_ = json.Unmarshal(raw, &doc)
			if doc.TurnID != result.TurnID || !doc.ResponseComplete || doc.Complete || doc.BackgroundStatus != "pending" || doc.Totals.ProviderCalls != len(provider.History()) || doc.Totals.ProviderCalls < 2 {
				t.Fatal("SQL trace linkage/usage incorrect", doc.Totals)
			}
			finalizerObserved, outboxObserved := false, false
			for _, node := range doc.Nodes {
				if node.Operation == "finalizer.Complete" {
					finalizerObserved = true
					if (node.Status == "error") != rollback {
						t.Fatal("trace misrepresented commit outcome")
					}
					if !rollback {
						var exported c.TurnResult
						_ = json.Unmarshal(node.Output, &exported)
						if exported.Text != result.Text || exported.AssistantMessageID == nil || *exported.AssistantMessageID != *result.AssistantMessageID {
							t.Fatal("trace differs from persisted result")
						}
					}
				}
				for _, artifact := range node.Artifacts {
					if artifact.Name == "outbox_insert" {
						var record struct {
							ID      c.ID   `json:"id"`
							Payload string `json:"payload_json"`
						}
						if err := json.Unmarshal(artifact.Value, &record); err != nil {
							t.Fatal(err)
						}
						if !c.ValidID(record.ID) || !json.Valid([]byte(record.Payload)) {
							t.Fatal("exact outbox record missing")
						}
						if count := aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_outbox WHERE id=@p1 AND payload_json=@p2`, record.ID, record.Payload); count != wantRecords {
							t.Fatal("outbox trace differs from SQL persistence", count)
						}
					}
					if artifact.Name == "outbox_events" && artifact.Kind == "transaction_pending" {
						outboxObserved = true
					}
				}
			}
			if !finalizerObserved || !outboxObserved {
				t.Fatal("SQL operations omitted", finalizerObserved, outboxObserved)
			}
		})
	}
}

func inspectorRuntime(t *testing.T, f aiFixture) (*orchestrator.Service, *fake.Provider) {
	core, provider := inspectorCore(t, f)
	return core.Turns, provider
}
func inspectorCore(t *testing.T, f aiFixture, capture ...*ports.Retriever) (*composition.AIInternal, *fake.Provider) {
	t.Helper()
	provider := fake.New()
	model := func(name string) llm.Model {
		return llm.Model{Provider: "synthetic", Name: name, NativeJSONSchema: true}
	}
	cfg := llm.Config{Chains: map[ports.ModelTier][]llm.Model{ports.ComponentJEV: {model("synthetic-jev")}, ports.ComponentUnified: {model("synthetic-unified")}, ports.ComponentMemoryWriter: {model("synthetic-memory-writer")}, ports.ComponentReflection: {model("synthetic-reflection")}, ports.ComponentEmbedding: {{Provider: "synthetic", Name: "synthetic-embedding", Dimensions: 3}}}, DefaultTimeout: 2 * time.Second, BreakerWindow: time.Minute, BreakerCooldown: time.Minute, BreakerErrorRate: .5, BreakerMinSamples: 20}
	client, e := llm.New(cfg, map[string]llm.Provider{"synthetic": provider}, llm.Options{})
	if e != nil {
		t.Fatal(e)
	}
	script := &internalScript{Client: client, db: f.db, provider: provider}
	keys, e := memorycrypto.New("test", map[string][]byte{"test": bytes.Repeat([]byte{42}, 32)})
	if e != nil {
		t.Fatal(e)
	}
	core, e := composition.NewAIInternal(context.Background(), composition.AIInternalOptions{DB: f.db, Bundle: internalBundle(t), Client: script, Embeddings: script, UsePolicy: syntheticUsePolicy{true}, MemoryPolicy: wp12Policy{}, Cipher: keys, EmbeddingModel: "synthetic-embedding", Dimension: 3, WorkerID: "inspector-test", Feedback: mem.FeedbackPolicy{MinUnusedRetrievals: 3, ImportanceMultiplier: .9}, Tracer: trace.NewRecorder(), AllowSyntheticForTests: true, TestDecorate: func(d orchestrator.Dependencies) orchestrator.Dependencies {
		d.Finalizer = f.store
		if len(capture) > 0 {
			*capture[0] = d.Retriever
		}
		return d
	}})
	if e != nil {
		t.Fatal(e)
	}
	return core, provider
}

func inspectorDocuments(t *testing.T, dir string) []trace.Document {
	t.Helper()
	files, e := filepath.Glob(filepath.Join(dir, "*.json"))
	if e != nil {
		t.Fatal(e)
	}
	docs := []trace.Document{}
	for _, file := range files {
		raw, e := os.ReadFile(file)
		if e != nil {
			t.Fatal(e)
		}
		if e = trace.ValidateJSON(raw); e != nil {
			t.Fatal(e)
		}
		var doc trace.Document
		if e = json.Unmarshal(raw, &doc); e != nil {
			t.Fatal(e)
		}
		docs = append(docs, doc)
	}
	return docs
}

func TestInspectorOneTurnOneJSONLifecycleSQL(t *testing.T) {
	db := aiDatabase(t)
	aiMigrate(t, db)
	aiMigrate(t, db)
	f := aiSetup(t, db)
	worker := wp12New(t, db)
	core, _ := inspectorCore(t, f)
	dir := filepath.Join(t.TempDir(), "private")
	ins, e := trace.NewInspector(trace.InspectorOptions{Directory: dir, Environment: "local", Address: "localhost:8080"})
	if e != nil {
		t.Fatal(e)
	}
	ctx := trace.WithInspector(context.Background(), ins)
	req := f.req()
	result, e := core.Turns.ProcessTurn(ctx, req)
	if e != nil {
		t.Fatal(e)
	}
	docs := inspectorDocuments(t, dir)
	if len(docs) != 1 || !docs[0].ResponseComplete || docs[0].Complete || docs[0].BackgroundStatus != "pending" {
		t.Fatal("response must finish with jobs still queued", docs)
	}
	if n := aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_jobs WHERE user_id=@p1 AND turn_id IS NULL`, f.a); n != 0 {
		t.Fatal("source trigger lost origin", n)
	}
	wp12Enable(t, worker, f.a)
	// Fail the first actual worker/provider execution. It must remain in the
	// original file while the queue waits for its retry deadline.
	aiExec(t, db, `UPDATE dbo.ai_jobs SET run_after=DATEADD(second,-1,SYSUTCDATETIME()) WHERE user_id=@p1 AND status=N'queued'`, f.a)
	worked, err := false, error(nil)
	for n := 0; n < 12; n++ {
		worked, err = worker.worker.RunOnce(ctx)
		if err != nil || !worked {
			break
		}
	}
	if !worked || err == nil {
		t.Fatal("expected synthetic provider failure", worked, err)
	}
	docs = inspectorDocuments(t, dir)
	if len(docs) != 1 || docs[0].BackgroundStatus != "pending" || docs[0].Complete {
		t.Fatal("retry wait incorrectly completed", docs)
	}
	var retried c.ID
	for _, j := range docs[0].Jobs {
		if j.Attempts == 1 && j.Status == "queued" && j.LastError != "" {
			retried = j.ID
		}
	}
	if !c.ValidID(retried) {
		t.Fatal("durable retry state missing")
	}
	// A second Turn for the same owner must never absorb the delayed first job.
	nextReq := f.req()
	next, e := core.Turns.ProcessTurn(ctx, nextReq)
	if e != nil {
		t.Fatal(e)
	}
	// An HTTP replay also appends to the existing file.
	if _, e = core.Turns.ProcessTurn(ctx, req); e != nil {
		t.Fatal(e)
	}
	worker.embed(128)
	for n := 0; n < 64; n++ {
		aiExec(t, db, `UPDATE dbo.ai_jobs SET run_after=DATEADD(second,-1,SYSUTCDATETIME()) WHERE user_id=@p1 AND status=N'queued'`, f.a)
		worked, e = worker.worker.RunOnce(ctx)
		if e != nil {
			t.Fatal(e)
		}
		if !worked {
			break
		}
	}
	docs = inspectorDocuments(t, dir)
	if len(docs) != 2 {
		t.Fatal("Turn files split by jobs/replay", len(docs))
	}
	for _, doc := range docs {
		if doc.TurnID != result.TurnID && doc.TurnID != next.TurnID {
			t.Fatal("unexpected Turn")
		}
		if !doc.Complete || !doc.ResponseComplete || doc.BackgroundStatus != "complete" {
			t.Fatal("background queue not complete", doc.TurnID, doc.BackgroundStatus, doc.Jobs)
		}
		attempts := map[int]bool{}
		for _, node := range doc.Nodes {
			if node.Operation != "memory.RunOnce" {
				continue
			}
			var input struct {
				Turn    c.ID `json:"turn_id"`
				Job     c.ID `json:"job_id"`
				Attempt int  `json:"attempt"`
			}
			if e = json.Unmarshal(node.Input, &input); e != nil {
				t.Fatal(e)
			}
			if input.Turn != doc.TurnID {
				t.Fatal("cross-Turn worker")
			}
			if input.Job == retried {
				if doc.TurnID != result.TurnID {
					t.Fatal("delayed retry moved Turns")
				}
				attempts[input.Attempt] = true
			}
		}
		if doc.TurnID == result.TurnID {
			if !attempts[1] || !attempts[2] {
				t.Fatal("failed or retried attempt lost", attempts)
			}
			if target := os.Getenv("THEIA_WRITE_LIFECYCLE_FIXTURE"); target != "" {
				raw, _ := json.MarshalIndent(doc, "", "  ")
				if e = os.WriteFile(target, raw, 0600); e != nil {
					t.Fatal(e)
				}
			}
		}
	}
}
