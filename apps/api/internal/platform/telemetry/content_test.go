package telemetry

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func capturedAttrs(content bool, action func(context.Context)) map[string]string {
	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	runtime := &Runtime{provider: provider, tracer: provider.Tracer("test"), content: content}
	ctx := WithRuntime(context.Background(), runtime)
	ctx, span := StartRoot(ctx, "theia.ai.turn")
	action(ctx)
	span.End(nil)
	results := map[string]string{}
	for _, captured := range exporter.GetSpans() {
		for _, a := range captured.Attributes {
			results[string(a.Key)] = a.Value.Emit()
		}
	}
	_ = provider.Shutdown(context.Background())
	return results
}

func TestContentNotPresentInMetadataMode(t *testing.T) {
	attrs := capturedAttrs(false, func(ctx context.Context) {
		CaptureRequest(ctx, []byte(`{"messages":[{"role":"user","content":"PRIVATE_PROMPT_42"}]}`), "api-key")
		CaptureResponse(ctx, []byte(`{"content":"PRIVATE_REPLY_88"}`), ContentDisposition{Bytes: 30, Complete: true}, "api-key")
	})
	for key, value := range attrs {
		if strings.Contains(key, "theia.dev.wire") || strings.Contains(value, "PRIVATE_PROMPT_42") || strings.Contains(value, "PRIVATE_REPLY_88") {
			t.Fatal("metadata tracer included private content", key)
		}
	}
}

func TestFullSyntheticWireCaptureAndNestedRedaction(t *testing.T) {
	const secret = "private-provider-key-secret-value"
	request := []byte(`{"messages":[{"role":"user","content":"Synthetic prompt to model"}],"nested":{"authorization":"Bearer abc123","credentials":{"api_key":"hide-this"}},"json_schema":{"type":"object","properties":{"password":{"example":"never"}}}}`)
	sum := sha256.Sum256(request)
	attrs := capturedAttrs(true, func(ctx context.Context) { CaptureRequest(ctx, request, secret) })
	prefix := "theia.dev.wire.request."
	if attrs[prefix+"body"] == "" || !strings.Contains(attrs[prefix+"body"], "Synthetic prompt to model") {
		t.Fatalf("request content missing: %+v", attrs)
	}
	if strings.Contains(attrs[prefix+"body"], "hide-this") || strings.Contains(attrs[prefix+"body"], "Bearer abc123") || strings.Contains(attrs[prefix+"body"], "never") {
		t.Fatal("nested secret leaked")
	}
	if attrs[prefix+"redacted"] != "true" || attrs[prefix+"complete"] != "true" || attrs[prefix+"sha256"] != hex.EncodeToString(sum[:]) {
		t.Fatalf("redaction/completeness/integrity missing: %+v", attrs)
	}
}

func TestCleanWireJSONRemainsByteExact(t *testing.T) {
	request := []byte("{ \"model\": \"synthetic\", \"messages\": [{\"content\":\"exact bytes\"}] }")
	attrs := capturedAttrs(true, func(ctx context.Context) { CaptureRequest(ctx, request, "different-private-token") })
	if attrs["theia.dev.wire.request.body"] != string(request) {
		t.Fatalf("capture changed safe serialized bytes: %q", attrs["theia.dev.wire.request.body"])
	}
}

func TestOversizedAndMalformedWireContentFailClosed(t *testing.T) {
	huge := []byte(`{"text":"` + strings.Repeat("x", MaxWireCaptureBytes) + `"}`)
	malformed := []byte(`{"unclosed":`)
	attrs := capturedAttrs(true, func(ctx context.Context) { CaptureRequest(ctx, huge, "secret") })
	if attrs["theia.dev.wire.request.body"] != "" || attrs["theia.dev.wire.request.omitted"] != "true" || attrs["theia.dev.wire.request.omission_reason"] != "size_limit" {
		t.Fatal("oversized payload was exported as text")
	}
	attrs = capturedAttrs(true, func(ctx context.Context) { CaptureRequest(ctx, malformed, "secret") })
	if attrs["theia.dev.wire.request.body"] != "" || attrs["theia.dev.wire.request.omission_reason"] != "invalid_json" {
		t.Fatal("malformed JSON leaked")
	}
}

func TestResponseSSECompletenessAndSecretRedaction(t *testing.T) {
	raw := []byte("data: {\"choices\":[{\"delta\":{\"content\":\"safe reply\"}}]}\n\ndata: [DONE]\n\n")
	sum := sha256.Sum256(raw)
	attrs := capturedAttrs(true, func(ctx context.Context) {
		CaptureResponse(ctx, raw, ContentDisposition{Bytes: int64(len(raw)), SHA256: hex.EncodeToString(sum[:]), Complete: true, Stream: true}, "private-key")
	})
	if attrs["theia.dev.wire.response.body"] != string(raw) || attrs["theia.dev.wire.response.complete"] != "true" {
		t.Fatalf("SSE capture mismatch: %+v", attrs)
	}
	failure := []byte("data: {\"error\":{\"api_key\":\"hidden\"}}\n")
	attrs = capturedAttrs(true, func(ctx context.Context) {
		CaptureResponse(ctx, failure, ContentDisposition{Bytes: int64(len(failure)), Complete: false, Stream: true}, "private-key")
	})
	if strings.Contains(attrs["theia.dev.wire.response.body"], "hidden") || attrs["theia.dev.wire.response.complete"] != "false" {
		t.Fatal("incomplete SSE leaked a credential or claimed completeness")
	}
}

func TestIngestKeyIsRedactedFromEchoedWireResponse(t *testing.T) {
	const key = "this-is-a-test-ingest-credential-of-42chars"
	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	runtime := &Runtime{provider: provider, tracer: provider.Tracer("test"), content: true, privateIngestKey: key}
	ctx, root := StartRoot(WithRuntime(context.Background(), runtime), "theia.ai.turn")
	raw := []byte(`{"echo":"` + key + `","message":"safe response"}`)
	sum := sha256.Sum256(raw)
	CaptureResponse(ctx, raw, ContentDisposition{Bytes: int64(len(raw)), SHA256: hex.EncodeToString(sum[:]), Complete: true}, "provider-secret")
	root.End(nil)
	for _, span := range exporter.GetSpans() {
		for _, a := range span.Attributes {
			if strings.Contains(a.Value.Emit(), key) {
				t.Fatal("private ingestion credential leaked")
			}
		}
	}
	_ = provider.Shutdown(context.Background())
}

func TestPartialReadMismatchNeverExportsPayload(t *testing.T) {
	attrs := capturedAttrs(true, func(ctx context.Context) {
		CaptureResponse(ctx, []byte(`{"reply":"partial"}`), ContentDisposition{Bytes: 9000, Complete: false}, "")
	})
	if attrs["theia.dev.wire.response.omitted"] != "true" || attrs["theia.dev.wire.response.body"] != "" {
		t.Fatal("partial capture metadata not marked omitted")
	}
}

func TestSyntheticWireCaptureAbovePrevious16KiBLimit(t *testing.T) {
	payload := []byte(`{"input":"` + strings.Repeat("synthetic ", 6000) + `"}`)
	if len(payload) <= 16*1024 || len(payload) >= MaxWireCaptureBytes {
		t.Fatal("bad test fixture size")
	}
	attrs := capturedAttrs(true, func(ctx context.Context) { CaptureRequest(ctx, payload, "other-key") })
	if attrs["theia.dev.wire.request.body"] != string(payload) || attrs["theia.dev.wire.request.omitted"] != "false" {
		t.Fatal("medium-sized synthetic prompt was unnecessarily omitted")
	}
}
