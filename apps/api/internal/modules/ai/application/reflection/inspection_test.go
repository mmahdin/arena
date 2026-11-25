package reflection

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/memorycrypto"
	mem "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
)

func TestInspectorAsyncReflectionRetryUsesDurableLinksAndCheckpoint(t *testing.T) {
	a, task, s, result, _ := fixture(t)
	task.TurnID = c.NewID()
	raw, _ := json.Marshal(result)
	payload, _ := json.Marshal(task)
	repo := &repository{task: task, s: s, l: mem.Lease{ID: c.NewID(), UserID: task.UserID, TurnID: task.TurnID, Token: c.NewID(), Kind: "reflection", Payload: payload, Attempts: 1}}
	cl := &client{result: raw}
	model, _ := NewModel(cl, a, repo, policy{})
	cm := &committer{err: errors.New("synthetic transient commit")}
	cipher, _ := memorycrypto.New("test", map[string][]byte{"test": bytes.Repeat([]byte{3}, 32)})
	worker, _ := NewWorker(repo, model, &writer{}, cm, a, policy{}, cipher, "synthetic-worker")
	dir := filepath.Join(t.TempDir(), "private")
	ins, err := trace.NewInspector(trace.InspectorOptions{Directory: dir, Environment: "local", Address: "localhost:8080"})
	if err != nil {
		t.Fatal(err)
	}
	parentCtx, parent := ins.StartTurn(context.Background(), c.TurnRequest{Text: "synthetic parent turn"})
	trace.Associate(parentCtx, task.TurnID)
	parent.End(nil, nil)
	ctx := trace.WithInspector(context.Background(), ins)
	if _, err = worker.RunOnce(ctx); err == nil {
		t.Fatal("fixture should fail first commit")
	}
	repo.l.Attempts = 2
	if _, err = worker.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	files, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	if len(files) != 1 || cl.calls != 1 {
		t.Fatal("retry must reuse model checkpoint", files, cl.calls)
	}
	totalModelSpans, jobAttempts := 0, 0
	for _, file := range files {
		raw, _ := os.ReadFile(file)
		if err = trace.ValidateJSON(raw); err != nil {
			t.Fatal(err)
		}
		var d trace.Document
		_ = json.Unmarshal(raw, &d)
		if d.TurnID != task.TurnID || d.Kind != "turn" || !d.Complete {
			t.Fatal("async identity missing")
		}
		for _, n := range d.Nodes {
			if n.Operation == "application/reflection/model.Reflect" {
				totalModelSpans++
			}
			if n.Operation == "reflection.RunOnce" {
				jobAttempts++
				if !bytes.Contains(n.Input, []byte(repo.l.ID.String())) {
					t.Fatal("reflection job id missing")
				}
			}
		}
	}
	if totalModelSpans != 1 || jobAttempts != 2 {
		t.Fatal("checkpoint retry invented a model operation or lost an attempt", totalModelSpans, jobAttempts)
	}
}
