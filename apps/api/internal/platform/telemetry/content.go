package telemetry

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"strings"
	"unicode/utf8"

	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// The sole wire-size bound shared with the HTTP response reader. Unlike the old
// 16 KiB cap this handles substantial prompts/responses; larger payloads remain
// explicitly omitted (never silently truncated). Keep finite to avoid unbounded
// allocations from an untrusted provider response.
const MaxWireCaptureBytes = 1024 * 1024

var (
	bearerToken  = regexp.MustCompile(`(?i)\bBearer\s+[^\s"'{},;]+`)
	tokenPattern = regexp.MustCompile(`\bsk-(?:proj-)?[A-Za-z0-9_-]{12,}\b`)
	inlineSecret = regexp.MustCompile(`(?i)\b(api[_-]?key|access[_-]?token|refresh[_-]?token|password|authorization|client[_-]?secret)\s*[:=]\s*([^\s"'{};,]+)`)
)

// ContentDisposition represents bytes *actually* read from a provider stream.
// The SHA-256 describes the original consumed bytes, not the redacted display.
// Complete represents a clean JSON-body EOF or validated SSE [DONE]; a closed
// or truncated stream without [DONE] must never be labelled complete.
type ContentDisposition struct {
	Bytes    int64
	SHA256   string
	Complete bool
	Stream   bool
}

func secretField(key string) bool {
	key = strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(key, "_", ""), "-", ""))
	for _, k := range []string{"authorization", "apikey", "accesstoken", "refreshtoken", "password", "secret", "cookie", "credential", "privatekey", "sessiontoken", "idtoken", "accesskey", "authheader"} {
		if strings.Contains(key, k) {
			return true
		}
	}
	return false
}

func redactText(value, providerSecret string) (string, bool) {
	before := value
	if providerSecret != "" {
		value = strings.ReplaceAll(value, providerSecret, "[REDACTED]")
		// Go serializes byte fields such as structured LLM responses as base64.
		if decoded, err := base64.StdEncoding.DecodeString(value); err == nil && bytes.Contains(decoded, []byte(providerSecret)) {
			value = base64.StdEncoding.EncodeToString(bytes.ReplaceAll(decoded, []byte(providerSecret), []byte("[REDACTED]")))
		}
	}
	value = bearerToken.ReplaceAllString(value, "Bearer [REDACTED]")
	value = tokenPattern.ReplaceAllString(value, "[REDACTED]")
	value = inlineSecret.ReplaceAllString(value, "$1=[REDACTED]")
	return value, value != before
}

func redactJSONValue(value any, providerSecret string) (any, bool) {
	changed := false
	switch v := value.(type) {
	case map[string]any:
		for key, item := range v {
			if secretField(key) {
				v[key] = "[REDACTED]"
				changed = true
			} else {
				clean, redacted := redactJSONValue(item, providerSecret)
				v[key] = clean
				changed = changed || redacted
			}
		}
		return v, changed
	case []any:
		for i, item := range v {
			clean, redacted := redactJSONValue(item, providerSecret)
			v[i] = clean
			changed = changed || redacted
		}
		return v, changed
	case string:
		return redactText(v, providerSecret)
	default:
		return v, false
	}
}

func redactJSON(raw []byte, providerSecret string) (string, bool, bool) {
	var obj any
	if !utf8.Valid(raw) || json.Unmarshal(raw, &obj) != nil {
		return "", false, false
	}
	clean, redacted := redactJSONValue(obj, providerSecret)
	if !redacted {
		return string(raw), false, true
	}
	encoded, err := json.Marshal(clean)
	if err != nil {
		return "", false, false
	}
	return string(encoded), true, true
}

func redactSSE(raw []byte, providerSecret string) (string, bool, bool) {
	if !utf8.Valid(raw) {
		return "", false, false
	}
	var output strings.Builder
	changed := false
	// Each data: line is JSON or [DONE]. Never permit unparseable arbitrary
	// vendor payloads to escape to the private collector unsanitized.
	for _, line := range bytes.SplitAfter(raw, []byte("\n")) {
		if len(line) == 0 {
			continue
		}
		content := strings.TrimSuffix(string(line), "\n")
		content = strings.TrimSuffix(content, "\r")
		ending := string(line[len(content):])
		if strings.HasPrefix(content, "data:") {
			payload := strings.TrimSpace(strings.TrimPrefix(content, "data:"))
			if payload != "[DONE]" {
				sanitized, redacted, ok := redactJSON([]byte(payload), providerSecret)
				if !ok {
					return "", false, false
				}
				if redacted {
					changed = true
					content = "data: " + sanitized
				}
			}
		} else if strings.TrimSpace(content) != "" {
			// Non-data SSE frames (including comments) are not used by this
			// adapter; do not export arbitrary strings from unknown fields.
			return "", false, false
		}
		output.WriteString(content)
		output.WriteString(ending)
	}
	return output.String(), changed, true
}

// CaptureRequest records exactly the post-JSON-serialization body sent to the
// provider (or its redacted counterpart). It is never run in metadata mode.
func CaptureRequest(ctx context.Context, body []byte, providerSecret string) {
	if !ContentEnabled(ctx) {
		return
	}
	sum := sha256.Sum256(body)
	recordWire(ctx, "request", body, ContentDisposition{Bytes: int64(len(body)), SHA256: hex.EncodeToString(sum[:]), Complete: true}, providerSecret)
}

// CaptureResponse records the wire bytes already consumed by application code.
// The body reader supplies the total and digest independently of the bounded
// retained bytes. Capture never reads or drains the network itself.
func CaptureResponse(ctx context.Context, body []byte, disposition ContentDisposition, providerSecret string) {
	if !ContentEnabled(ctx) {
		return
	}
	recordWire(ctx, "response", body, disposition, providerSecret)
}

func recordWire(ctx context.Context, direction string, raw []byte, d ContentDisposition, secret string) {
	if !ContentEnabled(ctx) || (direction != "request" && direction != "response") {
		return
	}
	span := oteltrace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}
	// The ingestion credential must never appear even if a synthetic fixture
	// accidentally echoes the value. Keep the integrity hash of ORIGINAL bytes.
	privateRedacted := false
	if runtime, ok := ctx.Value(runtimeKey{}).(*Runtime); ok && runtime.privateIngestKey != "" {
		clean := bytes.ReplaceAll(raw, []byte(runtime.privateIngestKey), []byte("[REDACTED]"))
		privateRedacted = !bytes.Equal(raw, clean)
		raw = clean
	}
	prefix := "theia.dev.wire." + direction + "."
	attrs := []attribute.KeyValue{
		attribute.Int64(prefix+"bytes", d.Bytes),
		attribute.String(prefix+"sha256", d.SHA256),
		attribute.Bool(prefix+"complete", d.Complete),
		attribute.Bool(prefix+"stream", d.Stream),
	}
	omitted, redacted, sanitized := "", false, ""
	if d.Bytes < 0 || d.Bytes > MaxWireCaptureBytes || (!privateRedacted && int64(len(raw)) != d.Bytes) {
		omitted = "size_limit"
	} else if d.Stream {
		var ok bool
		sanitized, redacted, ok = redactSSE(raw, secret)
		if !ok {
			omitted = "invalid_sse"
		}
	} else {
		var ok bool
		sanitized, redacted, ok = redactJSON(raw, secret)
		if !ok {
			omitted = "invalid_json"
		}
	}
	redacted = redacted || privateRedacted
	if omitted == "" && len(sanitized) > MaxWireCaptureBytes {
		omitted = "redacted_size_limit"
	}
	if omitted != "" {
		attrs = append(attrs, attribute.Bool(prefix+"omitted", true), attribute.String(prefix+"omission_reason", omitted))
	} else {
		attrs = append(attrs,
			attribute.Bool(prefix+"omitted", false), attribute.Bool(prefix+"redacted", redacted),
			attribute.String(prefix+"body", sanitized))
	}
	span.SetAttributes(attrs...)
	if omitted == "" {
		semanticWire(ctx, direction, sanitized, d.Stream, d.Complete)
	}
}
