package trace_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
)

func newInspector(t *testing.T) (*trace.Inspector, string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "private")
	i, err := trace.NewInspector(trace.InspectorOptions{Directory: dir, Environment: "local", Address: "127.0.0.1:8080"})
	if err != nil {
		t.Fatal(err)
	}
	return i, dir
}
func readDocuments(t *testing.T, dir string) []trace.Document {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	var docs []trace.Document
	for _, file := range files {
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if err = trace.ValidateJSON(raw); err != nil {
			t.Fatalf("%s: %v", file, err)
		}
		var d trace.Document
		if err = json.Unmarshal(raw, &d); err != nil {
			t.Fatal(err)
		}
		docs = append(docs, d)
		info, _ := os.Stat(file)
		if info.Mode().Perm() != 0600 {
			t.Fatal("trace file must be 0600")
		}
	}
	return docs
}
func TestInspectorSnapshotsHierarchyProvenanceAndUsage(t *testing.T) {
	i, dir := newInspector(t)
	i.RegisterSecret("fixture-api-secret")
	ctx, root := i.StartTurn(context.Background(), c.TurnRequest{Text: "full synthetic user text", UserID: c.NewID()})
	id := c.NewID()
	trace.Associate(ctx, id)
	trace.Capture(ctx, "source", "memory", "memory_item:synthetic", map[string]any{"text": "remembered synthetic fact"})
	childCtx, child := trace.Start(ctx, "brain", func() any { return map[string]string{"memory": "remembered synthetic fact"} })
	trace.Capture(childCtx, "bytes", "raw_bytes", "provider", []byte("encoded fixture-api-secret"))
	trace.Capture(childCtx, "rendered", "rendered_prompt", "prompts/brain.system.tmpl", "full system fixture-api-secret")
	pctx, provider := trace.Start(childCtx, "provider.Structured", func() any {
		return map[string]any{"request": "full final provider request", "lease_token": "fence-secret"}
	})
	trace.EstimateText(pctx, "history", "full synthetic history")
	cost := int64(22)
	trace.Usage(pctx, c.LLMUsage{Provider: "fixture", Model: "small", InTokens: 12, OutTokens: 3, CachedTokens: 2}, &cost)
	provider.End(func() any { return "raw provider response" }, nil)
	child.End(func() any { return "gated structured decision" }, nil)
	root.End(func() any { return "visible final response" }, nil)
	docs := readDocuments(t, dir)
	if len(docs) != 1 {
		t.Fatal(docs)
	}
	d := docs[0]
	if !d.Complete || d.TurnID != id || len(d.Nodes) != 3 || d.Totals.ProviderCalls != 1 || d.Totals.InputTokens != 12 || d.Totals.OutputTokens != 3 || d.Totals.EstimatedCostMicroUSD != 22 {
		t.Fatalf("invalid document: %+v", d)
	}
	matched := false
	for _, e := range d.Edges {
		if e.Kind == "data" && e.Evidence == "exact_json_value_match" {
			matched = true
		}
	}
	if !matched {
		t.Fatal("missing value provenance")
	}
	raw, _ := json.Marshal(d)
	for _, secret := range []string{"fixture-api-secret", "fence-secret"} {
		if strings.Contains(string(raw), secret) {
			t.Fatal("authentication secret captured")
		}
	}
	for _, node := range d.Nodes {
		for _, artifact := range node.Artifacts {
			if artifact.Name == "bytes" {
				var value []byte
				if err := json.Unmarshal(artifact.Value, &value); err != nil {
					t.Fatal(err)
				}
				if bytes.Contains(value, []byte("fixture-api-secret")) {
					t.Fatal("credential survived base64 encoding")
				}
			}
			var compact bytes.Buffer
			if err := json.Compact(&compact, artifact.Value); err != nil {
				t.Fatal(err)
			}
			sum := sha256.Sum256(compact.Bytes())
			if artifact.SHA256 != fmt.Sprintf("%x", sum) {
				t.Fatal("artifact digest does not match exported content")
			}
		}
	}
	if !strings.Contains(string(raw), "full synthetic user text") || !strings.Contains(string(raw), "remembered synthetic fact") {
		t.Fatal("local content was omitted")
	}
}
func TestInspectorDisabledHasLazyNoOp(t *testing.T) {
	if trace.Enabled(nil) {
		t.Fatal("nil context enabled")
	}
	_, nilSpan := trace.Start(nil, "invalid_input", nil)
	nilSpan.End(nil, nil)
	var i *trace.Inspector
	ctx, root := i.StartTurn(context.Background(), c.TurnRequest{})
	ctx, span := trace.Start(ctx, "never", func() any { t.Fatal("disabled input evaluated"); return nil })
	span.End(func() any { t.Fatal("disabled output evaluated"); return nil }, errors.New("ignored"))
	root.Finish(nil, nil)
	if trace.Enabled(ctx) || span != nil || root != nil {
		t.Fatal("disabled inspector active")
	}
	allocations := testing.AllocsPerRun(100, func() {
		_, s := trace.Start(ctx, "stage", func() any { return map[string]any{"unused": 42} })
		s.End(nil, nil)
	})
	if allocations != 0 {
		t.Fatalf("disabled span allocated: %v", allocations)
	}
	candidateValue := struct {
		Text   string
		Vector [128]float32
	}{Text: c.NewID().String()}
	captureAllocations := testing.AllocsPerRun(100, func() {
		candidateValue.Vector[0]++
		trace.Capture(ctx, "candidate", "memory", "repository", candidateValue)
	})
	if captureAllocations != 0 {
		t.Fatalf("disabled artifact capture allocated: %v", captureAllocations)
	}

}
func TestInspectorPartialFailureAndAsyncLinks(t *testing.T) {
	i, dir := newInspector(t)
	turn, job := c.NewID(), c.NewID()
	ctx, root := i.StartTurn(context.Background(), c.TurnRequest{Text: "synthetic pending turn"})
	trace.Associate(ctx, turn)
	_, pending := trace.Start(ctx, "unfinished", nil)
	docs := readDocuments(t, dir)
	if len(docs) != 1 || docs[0].Complete || docs[0].Status != "running" {
		t.Fatal(docs)
	}
	pending.End(nil, context.DeadlineExceeded)
	root.End(nil, context.Canceled)
	jctx, jspan := trace.StartJob(trace.WithInspector(context.Background(), i), turn, job, "memory", 2)
	if jspan == nil {
		t.Fatal("related job was not attached")
	}
	trace.Capture(jctx, "proposal", "memory_changes", "memory_writer", map[string]any{"before": "old synthetic fact", "after": "new synthetic fact"})
	jspan.End(nil, nil)
	docs = readDocuments(t, dir)
	if len(docs) != 1 {
		t.Fatal("each message must have exactly one file", docs)
	}
	d := docs[0]
	if d.TurnID != turn || d.Kind != "turn" || d.Status != "cancelled" || !d.Complete || len(d.Nodes) != 3 || d.Nodes[2].ParentID != d.Nodes[0].ID || d.Nodes[2].Operation != "memory.RunOnce" {
		t.Fatalf("async trace was not appended to the parent: %+v", d)
	}
	var jobInput struct {
		TurnID  c.ID `json:"turn_id"`
		JobID   c.ID `json:"job_id"`
		Attempt int  `json:"attempt"`
	}
	if err := json.Unmarshal(d.Nodes[2].Input, &jobInput); err != nil || jobInput.TurnID != turn || jobInput.JobID != job || jobInput.Attempt != 2 || len(d.Nodes[2].Artifacts) != 1 {
		t.Fatalf("durable job identity or evidence lost: %+v %v", jobInput, err)
	}
	// Explicitly opted-in background jobs remain inspectable without a turn file.
	_, orphan := trace.StartJob(trace.WithInspector(context.Background(), i), c.NewID(), c.NewID(), "reflection", 1)
	if orphan == nil {
		t.Fatal("unrecorded background job lost its trace")
	}
	orphan.End(nil, nil)
	if len(readDocuments(t, dir)) != 2 {
		t.Fatal("standalone job document missing")
	}
}

func TestInspectorWorkerOverlapsTurnWithoutLosingInFlightChildren(t *testing.T) {
	i, dir := newInspector(t)
	turn := c.NewID()
	ctx, root := i.StartTurn(context.Background(), c.TurnRequest{Text: "synthetic concurrent turn"})
	trace.Associate(ctx, turn)
	jctx, job := trace.StartJob(trace.WithInspector(context.Background(), i), turn, c.NewID(), "memory", 1)
	if job == nil {
		t.Fatal("worker not attached before turn completion")
	}
	_, child := trace.Start(jctx, "provider.Embed", func() any { return "synthetic embedding" })
	root.End(func() any { return "visible response" }, nil)
	docs := readDocuments(t, dir)
	if len(docs) != 1 || docs[0].Complete {
		t.Fatal("turn completion hid an in-flight asynchronous job", docs)
	}
	child.End(func() any { return "synthetic vector" }, nil)
	job.End(nil, nil)
	docs = readDocuments(t, dir)
	if len(docs) != 1 || !docs[0].Complete || docs[0].Status != "success" || len(docs[0].Nodes) != 3 {
		t.Fatal("asynchronous work was not collected into a complete turn", docs)
	}
}

func TestInspectorLateJobSurvivesRestartAndConcurrentAttempts(t *testing.T) {
	i, dir := newInspector(t)
	turn := c.NewID()
	ctx, root := i.StartTurn(context.Background(), c.TurnRequest{Text: "hello from a synthetic turn"})
	trace.Associate(ctx, turn)
	root.End(func() any { return "final reply" }, nil)
	// A fresh Inspector has no in-memory map: reload the original checkpoint.
	i2, err := trace.NewInspector(trace.InspectorOptions{Directory: dir, Environment: "local", Address: "localhost:8080"})
	if err != nil {
		t.Fatal(err)
	}
	const jobs = 8
	var wg sync.WaitGroup
	for n := 0; n < jobs; n++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			jctx, jspan := trace.StartJob(trace.WithInspector(context.Background(), i2), turn, c.NewID(), "reflection", n+1)
			if jspan == nil {
				t.Error("lost async turn checkpoint")
				return
			}
			childCtx, child := trace.Start(jctx, "provider.Structured", func() any { return map[string]any{"attempt": n + 1} })
			trace.Usage(childCtx, c.LLMUsage{Provider: "synthetic", Model: "test", InTokens: 11, OutTokens: 3}, nil)
			child.End(func() any { return "synthetic result" }, nil)
			jspan.End(nil, nil)
		}(n)
	}
	wg.Wait()
	docs := readDocuments(t, dir)
	if len(docs) != 1 || !docs[0].Complete || docs[0].Totals.ProviderCalls != jobs || docs[0].Totals.InputTokens != jobs*11 || len(docs[0].Nodes) != 1+jobs*2 {
		t.Fatalf("lost concurrent/restarted worker events: %+v", docs)
	}
	for _, n := range docs[0].Nodes[1:] {
		if n.Operation == "reflection.RunOnce" && n.ParentID != docs[0].Nodes[0].ID {
			t.Fatalf("job not nested under turn: %+v", n)
		}
	}
}

func TestInspectorConcurrentTurnsAndChildren(t *testing.T) {
	i, dir := newInspector(t)
	var wg sync.WaitGroup
	for n := 0; n < 12; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id := c.NewID()
			ctx, root := i.StartTurn(context.Background(), c.TurnRequest{UserID: id, Text: id.String()})
			var children sync.WaitGroup
			for k := 0; k < 3; k++ {
				children.Add(1)
				go func() {
					defer children.Done()
					_, s := trace.Start(ctx, "parallel", func() any { return id })
					s.End(func() any { return id }, nil)
				}()
			}
			children.Wait()
			root.End(func() any { return id }, nil)
		}()
	}
	wg.Wait()
	docs := readDocuments(t, dir)
	if len(docs) != 12 {
		t.Fatal(len(docs))
	}
	seen := map[string]bool{}
	for _, d := range docs {
		if seen[d.TraceID] || len(d.Nodes) != 4 || !d.Complete {
			t.Fatal(d)
		}
		seen[d.TraceID] = true
		var req struct {
			Request c.TurnRequest `json:"request"`
		}
		_ = json.Unmarshal(d.Nodes[0].Input, &req)
		for _, node := range d.Nodes[1:] {
			if string(node.Output) != "\""+req.Request.UserID.String()+"\"" {
				t.Fatal("cross-turn payload")
			}
		}
	}
}
func TestInspectorWriteFailureDoesNotFailTurnAndCanRecover(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "private")
	var failures atomic.Int32
	i, err := trace.NewInspector(trace.InspectorOptions{Directory: dir, Environment: "local", Address: "[::1]:8080", OnError: func(err error) {
		if err != trace.ErrInspectorWrite {
			t.Error("unsafe diagnostic")
		}
		failures.Add(1)
	}})
	if err != nil {
		t.Fatal(err)
	}
	ctx, root := i.StartTurn(context.Background(), c.TurnRequest{})
	if err = os.Rename(dir, dir+"-away"); err != nil {
		t.Fatal(err)
	}
	_, child := trace.Start(ctx, "stage", nil)
	child.End(nil, nil)
	if err = os.Rename(dir+"-away", dir); err != nil {
		t.Fatal(err)
	}
	root.End(nil, nil)
	docs := readDocuments(t, dir)
	if failures.Load() < 1 || len(docs) != 1 || docs[0].Status != "success" || len(docs[0].Diagnostics) != 1 {
		t.Fatal(docs, failures.Load())
	}
}
func TestInspectorRejectsSharedEnvironmentsAndUnsafeDirectories(t *testing.T) {
	for _, address := range []string{":8080", "0.0.0.0:8080", "example.com:8080"} {
		if _, err := trace.NewInspector(trace.InspectorOptions{Environment: "local", Address: address, Directory: t.TempDir()}); err == nil {
			t.Fatal(address)
		}
	}
	for _, env := range []string{"production", "staging", "stage", "unknown", ""} {
		if _, err := trace.NewInspector(trace.InspectorOptions{Environment: env, Address: "localhost:8080", Directory: t.TempDir()}); err == nil {
			t.Fatal(env)
		}
	}
	dir := t.TempDir()
	if err := os.Chmod(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := trace.NewInspector(trace.InspectorOptions{Environment: "local", Address: "localhost:8080", Directory: dir}); err == nil {
		t.Fatal("shared directory accepted")
	}
}
func TestInspectorPanicKeepsFailureAndRepanics(t *testing.T) {
	i, dir := newInspector(t)
	func() {
		defer func() {
			if recover() != "synthetic panic" {
				t.Error("panic was swallowed")
			}
		}()
		_, root := i.StartTurn(context.Background(), c.TurnRequest{})
		defer func() { root.Complete(nil, nil, recover()) }()
		panic("synthetic panic")
	}()
	docs := readDocuments(t, dir)
	if len(docs) != 1 || docs[0].Status != "error" {
		t.Fatal(docs)
	}
}

func TestInspectorInvalidOrDuplicateUsageDoesNotInventTotals(t *testing.T) {
	i, dir := newInspector(t)
	ctx, root := i.StartTurn(context.Background(), c.TurnRequest{})
	pctx, p := trace.Start(ctx, "provider.Stream", nil)
	trace.Usage(pctx, c.LLMUsage{InTokens: -1, OutTokens: 2}, nil)
	trace.Usage(pctx, c.LLMUsage{InTokens: 9, OutTokens: 2}, nil)
	trace.Usage(pctx, c.LLMUsage{InTokens: 9, OutTokens: 2}, nil)
	p.End(nil, nil)
	root.End(nil, nil)
	docs := readDocuments(t, dir)
	if docs[0].Totals.InputTokens != 9 || docs[0].Totals.ReportedCalls != 1 {
		t.Fatal(docs[0].Totals)
	}
}

func TestInspectorProvenanceDoesNotRoundLargeVersions(t *testing.T) {
	i, dir := newInspector(t)
	ctx, root := i.StartTurn(context.Background(), c.TurnRequest{})
	trace.Capture(ctx, "prior", "state", "database", map[string]any{"version": int64(9007199254740992), "text": "synthetic_state"})
	_, child := trace.Start(ctx, "different_version", func() any { return map[string]any{"version": int64(9007199254740993), "text": "different_state"} })
	child.End(nil, nil)
	root.End(nil, nil)
	docs := readDocuments(t, dir)
	for _, edge := range docs[0].Edges {
		if edge.Kind == "data" {
			t.Fatal("different JSON values were rounded into a provenance match")
		}
	}
}

func TestInspectorUserLevelJobHasNoFabricatedTurn(t *testing.T) {
	i, dir := newInspector(t)
	_, span := trace.StartJob(trace.WithInspector(context.Background(), i), c.ID{}, c.NewID(), "reflection", 1)
	if span == nil {
		t.Fatal("user-level job was not traced")
	}
	span.End(nil, nil)
	docs := readDocuments(t, dir)
	if len(docs) != 1 || docs[0].Kind != "job" || c.ValidID(docs[0].TurnID) || !c.ValidID(docs[0].JobID) {
		t.Fatal(docs)
	}
	raw, _ := json.Marshal(docs[0])
	var fields map[string]any
	_ = json.Unmarshal(raw, &fields)
	if _, exists := fields["turn_id"]; exists {
		t.Fatal("fabricated turn in export")
	}
}

func TestInspectorReplaysAndConcurrentJobsShareOneCheckpoint(t *testing.T) {
	i, dir := newInspector(t)
	req := c.TurnRequest{UserID: c.NewID(), ClientTurnID: c.NewID(), Text: "synthetic retry"}
	turn, job := c.NewID(), c.NewID()
	ctx, root := i.StartTurn(context.Background(), req)
	trace.Associate(ctx, turn)
	root.End(func() any { return "first reply" }, nil)
	// A delayed retry of the same durable job retains both attempt nodes.
	jctx, attempt := trace.StartJob(trace.WithInspector(context.Background(), i), turn, job, "memory", 1)
	trace.Skip(jctx, "memory_embed.skipped", "obsolete_source_revision")
	attempt.End(nil, context.DeadlineExceeded)
	// Reopening through the request key while a worker owns the Turn must use
	// the SAME in-memory inspection (otherwise the two writers lose updates).
	_, attempt = trace.StartJob(trace.WithInspector(context.Background(), i), turn, job, "memory", 2)
	replay, replaySpan := i.StartTurn(context.Background(), req)
	trace.Associate(replay, turn)
	_, child := trace.Start(replay, "replay.lookup", nil)
	child.End(func() any { return "stored response" }, nil)
	replaySpan.End(func() any { return "stored response" }, nil)
	if d := readDocuments(t, dir); len(d) != 1 || !d[0].ResponseComplete || d[0].Complete {
		t.Fatal("response and background completion conflated", d)
	}
	attempt.End(nil, nil)
	docs := readDocuments(t, dir)
	if len(docs) != 1 || !docs[0].Complete || len(docs[0].Nodes) != 6 {
		t.Fatal("replay/retry events lost", docs)
	}
	// Restarted HTTP replay also stays in that file.
	restarted, err := trace.NewInspector(trace.InspectorOptions{Directory: dir, Environment: "local", Address: "localhost:8080"})
	if err != nil {
		t.Fatal(err)
	}
	_, last := restarted.StartTurn(context.Background(), req)
	last.End(nil, nil)
	if d := readDocuments(t, dir); len(d) != 1 || len(d[0].Nodes) != 7 {
		t.Fatal("restart split the Turn", d)
	}
}

func TestInspectorMissingTurnCheckpointDoesNotSplitConcurrentJobs(t *testing.T) {
	i, dir := newInspector(t)
	turn := c.NewID()
	var wg sync.WaitGroup
	for n := 0; n < 12; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, s := trace.StartJob(trace.WithInspector(context.Background(), i), turn, c.NewID(), "memory", 1)
			s.End(nil, nil)
		}()
	}
	wg.Wait()
	docs := readDocuments(t, dir)
	if len(docs) != 1 || docs[0].Kind != "turn" || docs[0].TurnID != turn || docs[0].ResponseComplete || len(docs[0].Nodes) != 13 {
		t.Fatal("correlated jobs created independent documents", docs)
	}
}

func TestInspectorConcurrentRequestReplaysHaveOneRoot(t *testing.T) {
	i, dir := newInspector(t)
	req := c.TurnRequest{UserID: c.NewID(), ClientTurnID: c.NewID()}
	var wg sync.WaitGroup
	for n := 0; n < 12; n++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, s := i.StartTurn(context.Background(), req); s.End(nil, nil) }()
	}
	wg.Wait()
	if docs := readDocuments(t, dir); len(docs) != 1 || len(docs[0].Nodes) != 12 {
		t.Fatal("concurrent replay split the request", docs)
	}
}

func TestInspectorTurnlessJobCannotAttachToUnadmittedRequest(t *testing.T) {
	i, dir := newInspector(t)
	_, request := i.StartTurn(context.Background(), c.TurnRequest{Text: "synthetic rejected request"})
	request.End(nil, errors.New("rejected before admission"))
	_, job := trace.StartJob(trace.WithInspector(context.Background(), i), c.ID{}, c.NewID(), "reflection", 1)
	job.End(nil, nil)
	docs := readDocuments(t, dir)
	if len(docs) != 2 {
		t.Fatal("independent work attached to an unadmitted request", docs)
	}
	for _, doc := range docs {
		if len(doc.Nodes) != 1 {
			t.Fatal("unrelated operations merged", doc)
		}
	}
}

func TestInspectorRequestKeyIsOwnerScoped(t *testing.T) {
	i, dir := newInspector(t)
	client := c.NewID()
	for n := 0; n < 2; n++ {
		ctx, root := i.StartTurn(context.Background(), c.TurnRequest{UserID: c.NewID(), ClientTurnID: client})
		trace.Associate(ctx, c.NewID())
		root.End(nil, nil)
	}
	if docs := readDocuments(t, dir); len(docs) != 2 || docs[0].TurnID == docs[1].TurnID {
		t.Fatal("cross-owner idempotency key collision", docs)
	}
}
