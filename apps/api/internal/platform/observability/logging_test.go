package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestErrorFieldsNeverLogErrorMessages(t *testing.T) {
	secret := "private SQL text model key patient notes"
	err := errors.Join(errors.New(secret), context.DeadlineExceeded)
	fields := ErrorFields(err)
	var log bytes.Buffer
	logger := NewLogger(&log, slog.LevelDebug, time.UTC)
	logger.Error("failure", fields...)
	if strings.Contains(log.String(), secret) || strings.Contains(log.String(), "patient notes") {
		t.Fatalf("secret entered diagnostic log: %s", log.String())
	}
	if !strings.Contains(log.String(), "error_type") || !strings.Contains(log.String(), "timeout") {
		t.Fatalf("missing safe diagnostics: %s", log.String())
	}
}

func TestLoggerLevelSourceAndTimezone(t *testing.T) {
	var log bytes.Buffer
	tz := time.FixedZone("+0330", 3*3600+30*60)
	logger := NewLogger(&log, slog.LevelInfo, tz)
	logger.Debug("hidden")
	logger.Info("visible")
	var record map[string]any
	if err := json.Unmarshal(log.Bytes(), &record); err != nil {
		t.Fatal(err)
	}
	if record["msg"] != "visible" {
		t.Fatalf("bad record: %v", record)
	}
	if _, ok := record["source"]; !ok {
		t.Fatalf("source missing: %v", record)
	}
	if s, ok := record["time"].(string); !ok || !strings.HasSuffix(s, "+03:30") {
		t.Fatalf("incorrect log time zone: %v", record["time"])
	}
}

func TestOutputPermissionsAndAppend(t *testing.T) {
	path := filepath.Join(t.TempDir(), "api.jsonl")
	for _, text := range []string{"one\n", "two\n"} {
		out, closeOutput, err := Output(path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := out.Write([]byte(text)); err != nil {
			t.Fatal(err)
		}
		if err := closeOutput(); err != nil {
			t.Fatal(err)
		}
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "one\ntwo\n" {
		t.Fatalf("not appended: %q", b)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("file mode = %o", info.Mode().Perm())
	}
}

func TestTurnRefIsEphemeralPseudonym(t *testing.T) {
	id := [16]byte{1, 2, 3, 4}
	a := TurnRef(id)
	if len(a) != 16 || a != TurnRef(id) || a == TurnRef([16]byte{1, 2, 3, 5}) {
		t.Fatalf("unexpected pseudonym behavior: %q", a)
	}
	if TurnRef([16]byte{}) != "" {
		t.Fatal("zero ID must not appear in logs")
	}
}

func TestSourceUsesRepoRelativePath(t *testing.T) {
	src := Source(0)
	if !strings.HasPrefix(src, "apps/api/internal/platform/observability/logging_test.go:") {
		t.Fatalf("unexpected source: %q", src)
	}
}
