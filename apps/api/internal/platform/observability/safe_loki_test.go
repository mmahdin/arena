package observability

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestSanitizedLogDropsPrivateContentAndKeepsSafeFields(t *testing.T) {
	input := `{"time":"2026-11-23T10:00:00+03:30","level":"ERROR","msg":"http.request","request_id":"aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa","method":"POST","route":"/api/v1/admin/turns/:turnId","status":503,"duration_ms":14,"error_code":"DB_UNAVAILABLE","turn_ref":"0123456789abcdef","prompt":"TOP_SECRET_PASSWORD","response":"private answer","headers":{"authorization":"Bearer TOP_SECRET_PASSWORD"},"source":{"file":"/home/private/path"},"reason":"my private medical statement","raw":"TOP_SECRET_PASSWORD"}`
	safe, ok := sanitizedLogRecord([]byte(input))
	if !ok {
		t.Fatal("safe event was rejected")
	}
	for _, forbidden := range []string{"TOP_SECRET_PASSWORD", "private answer", "authorization", "medical", "/home/private", "prompt", "response", "headers", "reason"} {
		if bytes.Contains(safe, []byte(forbidden)) {
			t.Fatalf("private field %q escaped: %s", forbidden, safe)
		}
	}
	var got map[string]any
	if err := json.Unmarshal(safe, &got); err != nil {
		t.Fatal(err)
	}
	if got["event"] != "http.request" || got["error_code"] != "DB_UNAVAILABLE" || got["http_status"] != float64(503) || got["route"] != "/api/v1/admin/turns/:turnId" {
		t.Fatalf("missing operational metadata: %v", got)
	}
	if got["time"] != "2026-11-23T06:30:00Z" {
		t.Fatalf("expected UTC timestamp, got %v", got["time"])
	}
	if _, ok := got["status"]; ok {
		t.Fatal("numeric http status must not become status label")
	}
}

func TestSanitizedLogRejectsUnsafeMessagesStringsAndTypes(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		allowed bool
	}{
		{"unknown event", `{"time":"2026-11-23T00:00:00Z","level":"INFO","msg":"ai.prompt","prompt":"private"}`, false},
		{"invalid JSON", "not-json private", false},
		{"unexpected severity", `{"time":"2026-11-23T00:00:00Z","level":"TRACE","msg":"http.request"}`, false},
		{"wrong time", `{"time":"private","level":"INFO","msg":"http.request"}`, false},
		{"safe event hostile fields", `{"time":"2026-11-23T00:00:00Z","level":"INFO","msg":"http.request","component":"private","stage":"secret","route":"/api/v1/people/5555555555555","error_code":"SECRET","trace_id":"private","status":"raw-private-content"}`, true},
		{"nested structure", `{"time":"2026-11-23T00:00:00Z","level":"WARN","msg":"ai.stage.failed","status":{"value":"private"},"component":["private"]}`, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := sanitizedLogRecord([]byte(c.input))
			if ok != c.allowed {
				t.Fatalf("accept=%v, wanted=%v", ok, c.allowed)
			}
			if ok && bytes.Contains(got, []byte("private")) {
				t.Fatalf("untrusted content escaped: %s", got)
			}
		})
	}
}

func TestSafeLogWriterSlogPipelineAndRotation(t *testing.T) {
	dir := t.TempDir()
	writer, err := NewSafeLogWriter(filepath.Join(dir, safeLogName), "local")
	if err != nil {
		t.Fatal(err)
	}
	writer.maxBytes = 340
	logger := slog.New(slog.NewJSONHandler(writer, &slog.HandlerOptions{Level: slog.LevelDebug}))
	for n := 0; n < 32; n++ {
		logger.Info("http.request", "status", 503, "method", "GET", "component", "api", "route", "/api/v1/me", "prompt", fmt.Sprintf("PRIVATE%d", n))
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	seen := 0
	for n := 0; n <= safeLogBackups; n++ {
		p := filepath.Join(dir, safeLogName)
		if n > 0 {
			p = fmt.Sprintf("%s.%d", p, n)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			t.Fatal(err)
		}
		if strings.Contains(string(b), "PRIVATE") {
			t.Fatalf("private payload escaped %q", p)
		}
		if !bytes.Contains(b, []byte(`"event":"http.request"`)) {
			t.Fatalf("expected safe records %q: %s", p, b)
		}
		info, err := os.Stat(p)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0640 {
			t.Fatalf("permission of %q: %o", p, info.Mode().Perm())
		}
		seen++
	}
	if seen < 2 {
		t.Fatalf("rotation did not create backups: %d", seen)
	}
}

func TestSafeLogWriterGuardsLocalAndSymlinks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, safeLogName)
	for _, tc := range []struct{ path, environment string }{
		{path, "production"}, {path, "stage"}, {filepath.Join(dir, "unrestricted.jsonl"), "local"}, {"relative/theia-safe.jsonl", "local"}, {filepath.Join(dir, "nonexistent", safeLogName), "local"},
	} {
		if w, err := NewSafeLogWriter(tc.path, tc.environment); err == nil {
			w.Close()
			t.Fatalf("accepted invalid path/environment: %+v", tc)
		}
	}
	if err := os.Symlink(filepath.Join(dir, "secret"), path); err != nil {
		t.Fatal(err)
	}
	if w, err := NewSafeLogWriter(path, "local"); err == nil {
		w.Close()
		t.Fatal("symlink must be rejected")
	}
}

func TestSafeLogWriterConcurrentEvents(t *testing.T) {
	dir := t.TempDir()
	w, err := NewSafeLogWriter(filepath.Join(dir, safeLogName), "local")
	if err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.NewJSONHandler(w, nil))
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for k := 0; k < 20; k++ {
				logger.Info("ai.turn.finished", "status", "success", "duration_ms", 10)
			}
		}()
	}
	wg.Wait()
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, safeLogName))
	if err != nil {
		t.Fatal(err)
	}
	lines := bytes.Split(bytes.TrimSpace(b), []byte{'\n'})
	if len(lines) != 240 {
		t.Fatalf("got %d lines", len(lines))
	}
	for _, line := range lines {
		var item map[string]any
		if json.Unmarshal(line, &item) != nil || item["event"] != "ai.turn.finished" {
			t.Fatalf("bad record: %s", line)
		}
	}
}

func TestSanitizerRejectsOversizedInput(t *testing.T) {
	input := []byte(`{"time":"2026-11-23T00:00:00Z","level":"INFO","msg":"http.request","arbitrary":"` + strings.Repeat("private", 6000) + `"}`)
	if len(input) < safeLogMaxInput {
		t.Fatal("fixture too small")
	}
	if _, ok := sanitizedLogRecord(input); ok {
		t.Fatal("oversized untrusted record accepted")
	}
}

// Avoid stale datetime parsing regressions at DST/offset boundaries.
func TestTimeParsingStable(t *testing.T) {
	b, ok := sanitizedLogRecord([]byte(`{"time":"2026-11-23T11:11:11+01:00","level":"DEBUG","msg":"ai.stage.completed","component":"ai_core","stage":"brain","duration_ms":1}`))
	if !ok {
		t.Fatal("rejected")
	}
	var item map[string]any
	_ = json.Unmarshal(b, &item)
	parsed, err := time.Parse(time.RFC3339Nano, item["time"].(string))
	if err != nil || parsed.Location() != time.UTC {
		t.Fatalf("bad UTC time %v %v", parsed, err)
	}
}
