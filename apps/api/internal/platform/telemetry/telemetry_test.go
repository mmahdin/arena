package telemetry

import (
	"context"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func TestFailClosedEndpointValidation(t *testing.T) {
	base := Options{Enabled: true, ServiceName: "theia-api", Environment: "local", ServiceVersion: "obs-01a"}
	valid := []string{"http://127.0.0.1:6006/v1/traces", "http://[::1]:4318/v1/traces", "https://tracing.internal.example/v1/traces"}
	for _, url := range valid {
		o := base
		o.Endpoint = url
		if err := Validate(o); err != nil {
			t.Fatalf("valid endpoint rejected %q: %v", url, err)
		}
	}
	invalid := []string{"http://phoenix:6006/v1/traces", "http://0.0.0.0:6006/v1/traces", "https://a.example/v1/traces?token=secret", "https://user:pass@a.example/v1/traces", "https://a.example/not-traces", "file:///v1/traces", "javascript:bad"}
	for _, url := range invalid {
		o := base
		o.Endpoint = url
		if err := Validate(o); err == nil {
			t.Fatalf("invalid endpoint accepted: %q", url)
		}
	}
	o := base
	o.Environment = "prod"
	o.Endpoint = valid[0]
	if Validate(o) == nil {
		t.Fatal("production plaintext OTLP accepted")
	}
	if err := Validate(Options{}); err != nil {
		t.Fatal(err)
	}
}

func TestAllowlistNeverExportsArbitraryText(t *testing.T) {
	fields := sanitize([]Field{
		String("theia.turn_id", "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"),
		String("gen_ai.request.model", "model-1"),
		String("gen_ai.input.messages", "sensitive prompt"),
		String("http.request.body", "sensitive prompt"),
		String("gen_ai.provider.name", "provider-secret: leaked text"),
		String("error.message", "password"),
		Int("gen_ai.usage.output_tokens", 42),
	})
	if len(fields) != 3 {
		t.Fatalf("unexpected metadata attributes: %+v", fields)
	}
	for _, f := range fields {
		if strings.Contains(f.Value.AsString(), "sensitive") {
			t.Fatal("text leaked")
		}
	}
}

func TestMetadataSpansAreParentedAndJobsHaveIndependentRoots(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	defer provider.Shutdown(context.Background())
	runtime := &Runtime{provider: provider, tracer: provider.Tracer("theia.ai")}
	ctx := WithRuntime(context.Background(), runtime)
	turnCtx, root := StartRoot(ctx, "theia.ai.turn")
	attemptCtx, child := Start(turnCtx, "gen_ai.client", String("theia.llm_attempt_id", "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"))
	Annotate(attemptCtx, String("theia.turn_id", "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"))
	child.End(nil)
	root.End(nil)
	_, job := StartRoot(turnCtx, "theia.ai.memory_job", Int("theia.job_attempt", 2))
	job.End(nil)
	spans := exporter.GetSpans()
	if len(spans) != 3 {
		t.Fatalf("span count=%d", len(spans))
	}
	byName := map[string]tracetest.SpanStub{}
	for _, s := range spans {
		byName[s.Name] = s
	}
	rootSpan := byName["theia.ai.turn"]
	childSpan := byName["gen_ai.client"]
	jobSpan := byName["theia.ai.memory_job"]
	if !childSpan.Parent.SpanID().IsValid() || childSpan.Parent.SpanID() != rootSpan.SpanContext.SpanID() || childSpan.SpanContext.TraceID() != rootSpan.SpanContext.TraceID() {
		t.Fatal("provider attempt not in Turn trace")
	}
	if jobSpan.Parent.IsValid() || jobSpan.SpanContext.TraceID() == rootSpan.SpanContext.TraceID() {
		t.Fatal("async job incorrectly linked to completed turn")
	}
}

func TestFullContentRequiresOnlyLocalPrivateReceiverAndOptIn(t *testing.T) {
	base := Options{
		Enabled: true, CaptureContent: true, ServiceName: "theia-api", ServiceVersion: "dev", Environment: "local",
		Endpoint: "http://127.0.0.1:6006/v1/traces", LoopbackListener: true, PrivateIngestKey: strings.Repeat("f", 48),
	}
	if err := Validate(base); err != nil {
		t.Fatal(err)
	}
	scenarios := []struct {
		name   string
		change func(*Options)
	}{
		{"disabled", func(o *Options) { o.Enabled = false }},
		{"not-local", func(o *Options) { o.Environment = "stage" }},
		{"public-http-listener", func(o *Options) { o.LoopbackListener = false }},
		{"remote-https", func(o *Options) { o.Endpoint = "https://trace.example.invalid/v1/traces" }},
		{"localhost-name", func(o *Options) { o.Endpoint = "http://localhost:6006/v1/traces" }},
		{"wrong-local-port", func(o *Options) { o.Endpoint = "http://127.0.0.1:4319/v1/traces" }},
		{"wildcard-collector", func(o *Options) { o.Endpoint = "http://0.0.0.0:4318/v1/traces" }},
		{"untrusted-key", func(o *Options) { o.PrivateIngestKey = "weak" }},
		{"header-injection", func(o *Options) { o.PrivateIngestKey = strings.Repeat("a", 33) + "\n" }},
	}
	for _, tc := range scenarios {
		t.Run(tc.name, func(t *testing.T) {
			o := base
			tc.change(&o)
			if Validate(o) == nil {
				t.Fatal("unsafe full-content configuration accepted")
			}
		})
	}
}

func TestSpanLimitsDoNotTruncatePayloadAndRetainEvents(t *testing.T) {
	limits := spanLimits(true)
	if limits.AttributeValueLengthLimit != -1 || limits.EventCountLimit < 1 || limits.LinkCountLimit < 1 {
		t.Fatalf("content limits discard payload/events: %+v", limits)
	}
	exported := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exported), sdktrace.WithRawSpanLimits(limits))
	runtime := &Runtime{provider: provider, tracer: provider.Tracer("test"), content: true}
	ctx, span := StartRoot(WithRuntime(context.Background(), runtime), "theia.ai.turn")
	input := strings.Repeat("Synthetic sentence ", 2048)
	CaptureRequest(ctx, []byte(`{"input":"`+input+`"}`), "unrelated-provider-key")
	oteltrace.SpanFromContext(ctx).AddEvent("synthetic.event", oteltrace.WithAttributes(attribute.String("theia.event.kind", "test")))
	span.End(nil)
	got := exported.GetSpans()
	if len(got) != 1 || len(got[0].Events) != 1 {
		t.Fatalf("events lost: %+v", got)
	}
	found := false
	for _, a := range got[0].Attributes {
		if string(a.Key) == "theia.dev.wire.request.body" {
			found = strings.Contains(a.Value.AsString(), input)
		}
	}
	if !found {
		t.Fatal("body over old 128-character/16KiB limits missing or truncated")
	}
	if err := provider.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestPrivateIngestionKeyRequiresLocalPhoenixInMetadataMode(t *testing.T) {
	opts := Options{
		Enabled: true, ServiceName: "theia-api", Environment: "local",
		Endpoint: "http://127.0.0.1:6006/v1/traces", PrivateIngestKey: strings.Repeat("k", 48),
	}
	if err := Validate(opts); err != nil {
		t.Fatalf("local Phoenix metadata configuration rejected: %v", err)
	}
	for _, endpoint := range []string{
		"https://collector.example/v1/traces",
		"http://localhost:6006/v1/traces",
		"http://127.0.0.1:4319/v1/traces",
	} {
		invalid := opts
		invalid.Endpoint = endpoint
		if err := Validate(invalid); err == nil {
			t.Fatalf("private ingestion key accepted for %q", endpoint)
		}
	}
}

func TestDualCollectorPrivateIngestionDoesNotWeakenPhoenixRestrictions(t *testing.T) {
	base := Options{Enabled: true, CaptureContent: true, Endpoint: "http://127.0.0.1:4318/v1/traces",
		ServiceName: "theia-api", Environment: "local", LoopbackListener: true,
		PrivateIngestKey: strings.Repeat("a", 48)}
	if err := Validate(base); err != nil {
		t.Fatalf("authenticated local dual collector should be supported: %v", err)
	}
	for _, modify := range []func(*Options){
		func(o *Options) { o.Endpoint = "http://127.0.0.1:4319/v1/traces" },
		func(o *Options) { o.Endpoint = "http://localhost:4318/v1/traces" },
		func(o *Options) { o.Endpoint = "http://otel-collector:4318/v1/traces" },
		func(o *Options) { o.Endpoint = "https://traces.external.test/v1/traces" },
		func(o *Options) { o.PrivateIngestKey = "" },
		func(o *Options) { o.Environment = "stage" },
		func(o *Options) { o.LoopbackListener = false },
	} {
		o := base
		modify(&o)
		if err := Validate(o); err == nil {
			t.Fatalf("unsafe collector config accepted: endpoint=%q environment=%q", o.Endpoint, o.Environment)
		}
	}
	metadata := base
	metadata.CaptureContent = false
	metadata.PrivateIngestKey = ""
	if err := Validate(metadata); err != nil {
		t.Fatalf("metadata-only loopback collector should remain supported: %v", err)
	}
}
