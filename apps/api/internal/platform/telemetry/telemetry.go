// Package telemetry owns opt-in OpenTelemetry tracing. Metadata-only is the
// default. Full wire capture is an explicit local-loopback development option
// for synthetic fixtures, never public/shared production traffic. Incoming
// browser traceparent is not trusted.
package telemetry

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// Options are independently validated here, even for direct users that bypass
// config.Load. PrivateIngestKey is a server-held local ingestion key for
// either authenticated Phoenix (:6006) or the authenticated dual Collector
// (:4318). Never attach it to spans, browser responses or logs.
type Options struct {
	Enabled          bool
	Endpoint         string
	ServiceName      string
	ServiceVersion   string
	Environment      string
	CaptureContent   bool
	LoopbackListener bool
	PrivateIngestKey string
}

type Runtime struct {
	provider         *sdktrace.TracerProvider
	tracer           oteltrace.Tracer
	content          bool
	privateIngestKey string
}

var label = regexp.MustCompile(`^[a-zA-Z0-9_.:-]{1,80}$`)
var uuid = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-8][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)

// Validate rejects URLs with credentials, query strings or unknown paths.
// Plaintext HTTP is permitted only to a loopback local collector.
func Validate(o Options) error {
	if !o.Enabled {
		if o.CaptureContent {
			return errors.New("content capture requires THEIA_OTEL_ENABLED")
		}
		return nil
	}
	u, err := url.Parse(o.Endpoint)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "/v1/traces" || u.Opaque != "" {
		return errors.New("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT must be an explicit /v1/traces URL without credentials")
	}
	if u.Scheme != "https" {
		host := u.Hostname()
		ip := net.ParseIP(host)
		if u.Scheme != "http" || o.Environment != "local" || !(strings.EqualFold(host, "localhost") || ip != nil && ip.IsLoopback()) {
			return errors.New("OTLP traces require HTTPS, except local loopback HTTP")
		}
	}
	if !label.MatchString(o.ServiceName) || !label.MatchString(o.Environment) || (o.ServiceVersion != "" && !label.MatchString(o.ServiceVersion)) {
		return errors.New("OTel service/resource labels must be fixed safe identifiers")
	}
	if o.CaptureContent && (o.Environment != "local" || !o.LoopbackListener) {
		return errors.New("full-content tracing requires APP_ENV=local and a loopback HTTP_ADDR")
	}
	if o.CaptureContent || o.PrivateIngestKey != "" {
		ip := net.ParseIP(u.Hostname())
		if o.Environment != "local" || u.Scheme != "http" || ip == nil || !ip.IsLoopback() || (u.Port() != "6006" && u.Port() != "4318") {
			return errors.New("private OTel ingestion requires numeric loopback HTTP on Phoenix :6006 or dual Collector :4318")
		}
		if len(o.PrivateIngestKey) < 32 || len(o.PrivateIngestKey) > 4096 ||
			!regexp.MustCompile(`^[A-Za-z0-9._~-]+$`).MatchString(o.PrivateIngestKey) {
			return errors.New("valid private Phoenix ingestion key required")
		}
	}
	return nil
}

func New(ctx context.Context, o Options) (*Runtime, error) {
	if err := Validate(o); err != nil {
		return nil, err
	}
	if !o.Enabled {
		return nil, nil
	}
	exportOptions := []otlptracehttp.Option{
		otlptracehttp.WithEndpointURL(o.Endpoint), otlptracehttp.WithTimeout(4 * time.Second),
		otlptracehttp.WithRetry(otlptracehttp.RetryConfig{Enabled: true, InitialInterval: time.Second, MaxInterval: 2 * time.Second, MaxElapsedTime: 5 * time.Second}),
	}
	if o.PrivateIngestKey != "" {
		// Phoenix and the dual Collector require authenticated OTLP ingestion.
		client := &http.Client{
			Transport: &http.Transport{},
			Timeout:   4 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
		exportOptions = append(exportOptions,
			otlptracehttp.WithHTTPClient(client),
			otlptracehttp.WithHeaders(map[string]string{"authorization": "Bearer " + o.PrivateIngestKey}),
		)
	}
	exporter, err := otlptracehttp.New(ctx, exportOptions...)
	if err != nil {
		return nil, errors.New("OTLP exporter initialization failed")
	}
	res, err := resource.New(ctx, resource.WithAttributes(
		attribute.String("service.name", o.ServiceName),
		attribute.String("service.version", o.ServiceVersion),
		attribute.String("deployment.environment.name", o.Environment)))
	if err != nil {
		_ = exporter.Shutdown(ctx)
		return nil, errors.New("OTLP resource initialization failed")
	}
	limits := spanLimits(o.CaptureContent)
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res), sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.AlwaysSample())),
		sdktrace.WithRawSpanLimits(limits),
		sdktrace.WithBatcher(exporter, sdktrace.WithMaxQueueSize(512), sdktrace.WithMaxExportBatchSize(64), sdktrace.WithBatchTimeout(time.Second)))
	return &Runtime{provider: provider, tracer: provider.Tracer("theia.ai", oteltrace.WithInstrumentationVersion("obs-01b")), content: o.CaptureContent, privateIngestKey: o.PrivateIngestKey}, nil
}

// Attribute values are never silently truncated by the SDK: capture itself
// explicitly omits unsafe/oversized bodies with a reason. The SDK still has
// finite attribute/event/link counts and the exporter retains a bounded queue.
func spanLimits(content bool) sdktrace.SpanLimits {
	limits := sdktrace.NewSpanLimits()
	limits.AttributeCountLimit = 32
	if content {
		limits.AttributeCountLimit = 128
	}
	limits.AttributeValueLengthLimit = -1
	limits.EventCountLimit = 128
	limits.AttributePerEventCountLimit = 16
	limits.LinkCountLimit = 16
	return limits
}

func (r *Runtime) Shutdown(ctx context.Context) error {
	if r == nil || r.provider == nil {
		return nil
	}
	return r.provider.Shutdown(ctx)
}

type runtimeKey struct{}

// Enabled is intentionally allocation-free in the disabled hot path.
// ContentEnabled is true only when a validated private content exporter was
// bootstrapped. Do not put payload fields in the general Set/Annotate allowlist.
func ContentEnabled(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	r, _ := ctx.Value(runtimeKey{}).(*Runtime)
	return r != nil && r.content
}

func Enabled(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	r, _ := ctx.Value(runtimeKey{}).(*Runtime)
	return r != nil
}

func WithRuntime(ctx context.Context, r *Runtime) context.Context {
	if r == nil || ctx == nil {
		return ctx
	}
	return context.WithValue(ctx, runtimeKey{}, r)
}

type Field struct {
	Key     string
	Str     string
	Number  int64
	Integer bool
}

func String(key, value string) Field { return Field{Key: key, Str: value} }
func Int(key string, n int64) Field  { return Field{Key: key, Number: n, Integer: true} }

func allowed(f Field) (attribute.KeyValue, bool) {
	if f.Integer {
		if f.Number < 0 {
			return attribute.KeyValue{}, false
		}
		switch f.Key {
		case "theia.job_attempt", "gen_ai.usage.input_tokens", "gen_ai.usage.output_tokens", "theia.llm.cost_micro_usd":
			return attribute.Int64(f.Key, f.Number), true
		}
		return attribute.KeyValue{}, false
	}
	switch f.Key {
	case "theia.turn_id", "theia.job_id", "theia.llm_attempt_id":
		if uuid.MatchString(f.Str) {
			return attribute.String(f.Key, f.Str), true
		}
	case "theia.job_kind":
		if f.Str == "memory" || f.Str == "reflection" {
			return attribute.String(f.Key, f.Str), true
		}
	case "theia.llm.kind":
		if f.Str == "structured" || f.Str == "embed" || f.Str == "stream" {
			return attribute.String(f.Key, f.Str), true
		}
	case "gen_ai.operation.name":
		if f.Str == "chat" || f.Str == "embeddings" {
			return attribute.String(f.Key, f.Str), true
		}
	case "gen_ai.provider.name", "gen_ai.request.model", "theia.llm.tier":
		if label.MatchString(f.Str) {
			return attribute.String(f.Key, f.Str), true
		}
	case "theia.llm.status":
		if f.Str == "succeeded" || f.Str == "failed" || f.Str == "incomplete" {
			return attribute.String(f.Key, f.Str), true
		}
	case "theia.llm.cost_status":
		if f.Str == "unknown" || f.Str == "unpriced" || f.Str == "estimated" {
			return attribute.String(f.Key, f.Str), true
		}
	case "theia.stage.operation":
		// These values come only from hardcoded Go callsites; no payload labels.
		if label.MatchString(f.Str) {
			return attribute.String(f.Key, f.Str), true
		}
	}
	return attribute.KeyValue{}, false
}

func sanitize(fields []Field) []attribute.KeyValue {
	out := make([]attribute.KeyValue, 0, len(fields))
	for _, f := range fields {
		if a, ok := allowed(f); ok {
			out = append(out, a)
		}
	}
	return out
}

type Span struct {
	span         oteltrace.Span
	semanticKind string
}

func (s *Span) End(err error) {
	if s == nil {
		return
	}
	if err != nil {
		s.span.SetStatus(codes.Error, "operation_failed")
	} // never RecordError(err)
	s.span.End()
}
func (s *Span) Set(fields ...Field) {
	if s == nil {
		return
	}
	if attrs := sanitize(fields); len(attrs) > 0 {
		s.span.SetAttributes(attrs...)
		// Export standard OpenInference usage only on the physical LLM attempt.
		if s.semanticKind == "LLM" && s.span.IsRecording() {
			for _, attr := range attrs {
				switch string(attr.Key) {
				case "gen_ai.usage.input_tokens":
					s.span.SetAttributes(attribute.Int64("llm.token_count.prompt", attr.Value.AsInt64()))
				case "gen_ai.usage.output_tokens":
					s.span.SetAttributes(attribute.Int64("llm.token_count.completion", attr.Value.AsInt64()))
				case "theia.llm.cost_micro_usd":
					s.span.SetAttributes(attribute.Float64("llm.cost.total", float64(attr.Value.AsInt64())/1_000_000))
				}
			}
		}
	}
}

func Start(ctx context.Context, name string, fields ...Field) (context.Context, *Span) {
	if ctx == nil {
		return nil, nil
	}
	r, _ := ctx.Value(runtimeKey{}).(*Runtime)
	if r == nil {
		return ctx, nil
	}
	attrs := sanitize(fields)
	attrs = append(attrs, semanticStart(name, attrs)...)
	ctx, s := r.tracer.Start(ctx, name, oteltrace.WithAttributes(attrs...))
	if name == "gen_ai.client" {
		ctx = context.WithValue(ctx, semanticAttemptKey{}, s)
	}
	return ctx, &Span{span: s, semanticKind: semanticKind(name, attrs)}
}

type rootSpanKey struct{}

func StartRoot(ctx context.Context, name string, fields ...Field) (context.Context, *Span) {
	if ctx == nil {
		return nil, nil
	}
	r, _ := ctx.Value(runtimeKey{}).(*Runtime)
	if r == nil {
		return ctx, nil
	}
	attrs := sanitize(fields)
	attrs = append(attrs, semanticStart(name, attrs)...)
	ctx, s := r.tracer.Start(ctx, name, oteltrace.WithNewRoot(), oteltrace.WithAttributes(attrs...))
	if r.content {
		ctx = context.WithValue(ctx, stageSecretsKey{}, &stageSecrets{})
	}
	return context.WithValue(ctx, rootSpanKey{}, s.SpanContext()), &Span{span: s, semanticKind: semanticKind(name, attrs)}
}
func Annotate(ctx context.Context, fields ...Field) {
	if ctx == nil {
		return
	}
	r, _ := ctx.Value(runtimeKey{}).(*Runtime)
	if r == nil {
		return
	}
	if attrs := sanitize(fields); len(attrs) > 0 {
		oteltrace.SpanFromContext(ctx).SetAttributes(attrs...)
	}
}

// ActiveRootIDs returns the actual code-created root span (not its current
// stage child). The private context key is assigned only by StartRoot, never
// from incoming traceparent/baggage or the local Inspector's file UUID.
func ActiveRootIDs(ctx context.Context) (string, string, bool) {
	if !Enabled(ctx) {
		return "", "", false
	}
	sc, ok := ctx.Value(rootSpanKey{}).(oteltrace.SpanContext)
	if !ok || !sc.IsValid() {
		return "", "", false
	}
	return sc.TraceID().String(), sc.SpanID().String(), true
}
