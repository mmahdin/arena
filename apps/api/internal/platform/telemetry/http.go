package telemetry

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
)

// StartHTTP records a physical OpenAI-compatible request as a child of its
// durable LLM attempt. Only fixed, code-owned routes and method are exported;
// provider base URLs, query strings, authorization headers and cookies are not.
func StartHTTP(ctx context.Context, route string) (context.Context, *Span) {
	if !Enabled(ctx) {
		return ctx, nil
	}
	safeRoute := "other"
	switch route {
	case "/chat/completions", "/embeddings":
		safeRoute = route
	}
	ctx, span := Start(ctx, "theia.ai.http.client")
	if span != nil {
		span.span.SetAttributes(attribute.String("http.request.method", "POST"),
			attribute.String("url.path", safeRoute))
	}
	return ctx, span
}

func HTTPStatus(span *Span, status int) {
	if span == nil || status < 100 || status > 599 {
		return
	}
	span.span.SetAttributes(attribute.Int("http.response.status_code", status))
}

func HTTPIncomplete(span *Span) {
	if span == nil {
		return
	}
	span.span.SetAttributes(attribute.Bool("theia.http.response_incomplete", true))
}
