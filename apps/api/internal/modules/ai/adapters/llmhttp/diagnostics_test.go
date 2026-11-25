package llmhttp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/llmhttp"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/llm"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/observability"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"syscall"
	"testing"
	"time"
)

type diagnosticTransport func(*http.Request) (*http.Response, error)

func (f diagnosticTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestProviderCodesAreClosedVocabularyAndBodiesStayPrivate(t *testing.T) {
	for _, tc := range []struct {
		status       int
		code, reason string
	}{
		{401, "invalid_api_key", "authentication"}, {403, "permission_denied", "permission_denied"},
		{400, "unsupported_parameter", "unsupported_parameter"}, {400, "invalid_json_schema", "schema_invalid"},
		{404, "model_not_found", "model_unavailable"}, {429, "insufficient_quota", "quota_exhausted"},
		{429, "rate_limit_exceeded", "rate_limit"}, {500, "PRIVATE_alphanumeric_secret", "server"},
		{400, "PRIVATE_alphanumeric_secret", "invalid_request"},
	} {
		t.Run(tc.reason+tc.code, func(t *testing.T) {
			var b bytes.Buffer
			prior := slog.Default()
			slog.SetDefault(observability.NewLogger(&b, slog.LevelDebug, time.UTC))
			defer slog.SetDefault(prior)
			p, err := llmhttp.New("https://provider.invalid/v1", "PRIVATE-api-key", &http.Client{Transport: diagnosticTransport(func(*http.Request) (*http.Response, error) {
				raw, _ := json.Marshal(map[string]any{"error": map[string]any{"code": tc.code, "message": "PRIVATE prompt and conversation", "param": "PRIVATE_parameter"}})
				return &http.Response{StatusCode: tc.status, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(raw))}, nil
			})})
			if err != nil {
				t.Fatal(err)
			}
			_, err = p.Structured(context.Background(), llm.Model{}, ports.StructuredRequest{Schema: []byte(`{"type":"object"}`)}, llm.ModeJSONObject)
			var failure *llm.Failure
			if !errors.As(err, &failure) || failure.DiagnosticReason() != tc.reason || failure.HTTPStatus != tc.status {
				t.Fatal(err, failure)
			}
			if strings.Contains(b.String(), "PRIVATE") {
				t.Fatal("private metadata escaped", b.String())
			}
			var record map[string]any
			if json.Unmarshal(b.Bytes(), &record) != nil {
				t.Fatal(b.String())
			}
			if record["provider_status"] != float64(tc.status) || record["reason"] != tc.reason {
				t.Fatal(record)
			}
			if strings.HasPrefix(tc.code, "PRIVATE") && record["provider_error_code"] != nil {
				t.Fatal(record)
			}
		})
	}
}
func TestTransportClassificationsKeepCauseWithoutURL(t *testing.T) {
	for _, tc := range []struct {
		cause  error
		reason string
	}{
		{&net.DNSError{Err: "PRIVATE lookup", Name: "PRIVATE-host"}, "dns_error"},
		{syscall.ECONNREFUSED, "connection_refused"}, {context.Canceled, "request_canceled"}, {context.DeadlineExceeded, "request_timeout"},
	} {
		p, _ := llmhttp.New("https://PRIVATE.invalid/v1", "PRIVATE-key", &http.Client{Transport: diagnosticTransport(func(*http.Request) (*http.Response, error) { return nil, tc.cause })})
		ctx := context.Background()
		if tc.cause == context.Canceled {
			var cancel context.CancelFunc
			ctx, cancel = context.WithCancel(ctx)
			cancel()
		}
		if tc.cause == context.DeadlineExceeded {
			var cancel context.CancelFunc
			ctx, cancel = context.WithDeadline(ctx, time.Now().Add(-time.Second))
			defer cancel()
		}
		_, err := p.Structured(ctx, llm.Model{}, ports.StructuredRequest{Schema: []byte(`{}`)}, llm.ModeJSONObject)
		if !errors.Is(err, tc.cause) || observability.ErrorReason(err) != tc.reason {
			t.Fatal(err, observability.ErrorReason(err))
		}
		fields := observability.ErrorFields(err)
		var b bytes.Buffer
		observability.NewLogger(&b, slog.LevelDebug, time.UTC).Warn("probe", fields...)
		if strings.Contains(b.String(), "PRIVATE") {
			t.Fatal(b.String())
		}
	}
}
