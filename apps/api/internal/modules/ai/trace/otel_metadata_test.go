package trace_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/telemetry"
)

// Exercise the actual OTLP/HTTP exporter without a live Phoenix dependency.
// The unique user/response markers must never appear in exported protobuf.
func TestOTelMetadataOnlyAndLazyInputWithoutLocalInspector(t *testing.T) {
	var mu sync.Mutex
	var raw []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/traces" {
			t.Errorf("unexpected export path: %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		raw = append(raw, body...)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/x-protobuf")
		w.WriteHeader(200)
	}))
	defer srv.Close()
	runtime, err := telemetry.New(context.Background(), telemetry.Options{
		Enabled: true, Endpoint: srv.URL + "/v1/traces", Environment: "local", ServiceName: "theia-api", ServiceVersion: "obs-01a",
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := telemetry.WithRuntime(context.Background(), runtime)
	ctx, root := (*trace.Inspector)(nil).StartTurn(ctx, contract.TurnRequest{Text: "PRIVATE_USER_MARKER"})
	inputCalled := false
	_, stage := trace.Start(ctx, "gateway.Admit", func() any { inputCalled = true; return "PRIVATE_SYSTEM_MARKER" })
	stage.End(func() any { return "PRIVATE_RESPONSE_MARKER" }, nil)
	trace.Associate(ctx, contract.NewID())
	root.End(func() any { return "PRIVATE_OUTPUT_MARKER" }, nil)
	ctx2, job := trace.StartJob(ctx, contract.NewID(), contract.NewID(), "memory", 2)
	_, workerStage := trace.Start(ctx2, "application/memory/writer.Propose", func() any { t.Fatal("worker input evaluated"); return "PRIVATE_TOOL_MARKER" })
	workerStage.End(nil, nil)
	job.End(nil, nil)
	shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := runtime.Shutdown(shutdown); err != nil {
		t.Fatal(err)
	}
	if inputCalled {
		t.Fatal("OTel only must not invoke local Inspector input closures")
	}
	mu.Lock()
	got := append([]byte(nil), raw...)
	mu.Unlock()
	for _, marker := range [][]byte{[]byte("PRIVATE_USER_MARKER"), []byte("PRIVATE_SYSTEM_MARKER"), []byte("PRIVATE_RESPONSE_MARKER"), []byte("PRIVATE_OUTPUT_MARKER"), []byte("PRIVATE_TOOL_MARKER")} {
		if bytes.Contains(got, marker) {
			t.Fatalf("private marker found in OTLP: %s", marker)
		}
	}
	for _, name := range [][]byte{[]byte("theia.ai.turn"), []byte("theia.ai.gateway.Admit"), []byte("theia.ai.memory_job"), []byte("theia.ai.application.memory.writer.Propose")} {
		if !bytes.Contains(got, name) {
			t.Fatalf("span name missing from export: %s", name)
		}
	}
}
