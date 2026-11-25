package telemetry

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"math"
	"strings"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

type stageTestEvent struct {
	name  string
	attrs map[string]string
}

func observeStageEvents(content bool, privateKey string, action func(context.Context)) []stageTestEvent {
	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter), sdktrace.WithRawSpanLimits(spanLimits(content)))
	runtime := &Runtime{provider: provider, tracer: provider.Tracer("obs04c-test"), content: content, privateIngestKey: privateKey}
	ctx, root := StartRoot(WithRuntime(context.Background(), runtime), "theia.ai.turn")
	action(ctx)
	root.End(nil)
	var events []stageTestEvent
	for _, span := range exporter.GetSpans() {
		for _, event := range span.Events {
			a := map[string]string{}
			for _, attr := range event.Attributes {
				a[string(attr.Key)] = attr.Value.Emit()
			}
			events = append(events, stageTestEvent{name: event.Name, attrs: a})
		}
	}
	_ = provider.Shutdown(context.Background())
	return events
}

func TestStageEventsAreAbsentInMetadataMode(t *testing.T) {
	serialized := false
	events := observeStageEvents(false, "", func(ctx context.Context) {
		StageEvent(ctx, "input", dangerousMarshal{called: &serialized})
		StageArtifact(ctx, "rendered_prompt", "rendered_prompt", dangerousMarshal{called: &serialized})
	})
	if serialized || len(events) != 0 {
		t.Fatalf("metadata mode serialized content: %v / %v", serialized, events)
	}
}

func TestFullStageEventsRedactAndRetainStructuredContent(t *testing.T) {
	const ingest = "private-collector-ingestion-credential-42"
	const provider = "synthetic-provider-credential-42"
	encoded := base64.StdEncoding.EncodeToString([]byte(provider))
	events := observeStageEvents(true, ingest, func(ctx context.Context) {
		StageEvent(ctx, "input", map[string]any{"message": "synthetic question", "api_key": "do-not-export", "nested": map[string]any{"authorization": "Bearer hidden", "echo": ingest}})
		RegisterStageSecret(ctx, provider)
		StageEvent(ctx, "output", map[string]any{"reply": "synthetic answer", "echo": provider, "encoded": encoded})
		StageArtifact(ctx, "rendered_prompt", "rendered_prompt", map[string]any{"blocks": []string{"system prompt", "synthetic message"}})
		StageArtifact(ctx, "chunk", "stream_chunk", "MUST_NOT_BE_EXPORTED")
	})
	if len(events) != 3 {
		t.Fatalf("expected 3 code-owned events, got %+v", events)
	}
	for i, name := range []string{"theia.ai.stage.input", "theia.ai.stage.output", "theia.ai.stage.artifact"} {
		if events[i].name != name {
			t.Fatalf("event %d: %+v", i, events[i])
		}
		if events[i].attrs["theia.dev.stage.omitted"] != "false" {
			t.Fatalf("stage omitted unexpectedly: %+v", events[i])
		}
	}
	raw := events[0].attrs["theia.dev.stage.json"]
	if !strings.Contains(raw, "synthetic question") || strings.Contains(raw, "do-not-export") || strings.Contains(raw, "Bearer hidden") || strings.Contains(raw, ingest) {
		t.Fatalf("missing synthetic content or leaked secret: %s", raw)
	}
	var value map[string]any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		t.Fatal(err)
	}
	if value["api_key"] != "[REDACTED]" {
		t.Fatalf("credential was not redacted: %+v", value)
	}
	output := events[1].attrs["theia.dev.stage.json"]
	if !strings.Contains(output, "synthetic answer") || strings.Contains(output, provider) || strings.Contains(output, encoded) {
		t.Fatal("missing synthetic output or leaked registered credential")
	}
}

func TestStageOversizeAndSerializationFailureAreExplicitlyOmitted(t *testing.T) {
	events := observeStageEvents(true, "", func(ctx context.Context) {
		StageEvent(ctx, "input", map[string]string{"text": strings.Repeat("S", MaxWireCaptureBytes)})
		StageEvent(ctx, "output", map[string]any{"NaN": math.NaN()})
		StageEvent(ctx, "untrusted_event", "no events from user-generated name")
	})
	if len(events) != 2 {
		t.Fatalf("unexpected events: %+v", events)
	}
	if events[0].attrs["theia.dev.stage.omission_reason"] != "size_limit" || events[1].attrs["theia.dev.stage.omission_reason"] != "serialization_failed" {
		t.Fatalf("expected explicit omission: %+v", events)
	}
	for _, e := range events {
		if e.attrs["theia.dev.stage.json"] != "" || e.attrs["theia.dev.stage.omitted"] != "true" {
			t.Fatalf("unsafe stage payload exported: %+v", e)
		}
	}
}

func TestHTTPSpanHasSafeCodeOwnedMetadataOnly(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	runtime := &Runtime{provider: provider, tracer: provider.Tracer("http-test")}
	ctx, root := StartRoot(WithRuntime(context.Background(), runtime), "theia.ai.turn")
	_, httpSpan := StartHTTP(ctx, "/chat/completions")
	HTTPStatus(httpSpan, 429)
	HTTPIncomplete(httpSpan)
	httpSpan.End(nil)
	root.End(nil)
	count := 0
	for _, span := range exporter.GetSpans() {
		if span.Name != "theia.ai.http.client" {
			continue
		}
		count++
		attrs := map[string]string{}
		for _, a := range span.Attributes {
			attrs[string(a.Key)] = a.Value.Emit()
		}
		for name, expected := range map[string]string{"http.request.method": "POST", "url.path": "/chat/completions", "http.response.status_code": "429", "theia.http.response_incomplete": "true"} {
			if attrs[name] != expected {
				t.Fatalf("bad HTTP metadata %s: %q", name, attrs[name])
			}
		}
	}
	if count != 1 {
		t.Fatalf("expected 1 physical HTTP span, got %d", count)
	}
	_ = provider.Shutdown(context.Background())
}
