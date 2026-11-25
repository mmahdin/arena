// Package observability provides structured, privacy-safe diagnostic logging.
// Never send request bodies, chat text, prompts, credentials, raw SQL errors,
// URLs or untrusted panic values to the logger.
package observability

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

func NewLogger(out io.Writer, level slog.Level, tz *time.Location) *slog.Logger {
	if tz == nil {
		tz = time.UTC
	}
	return slog.New(slog.NewJSONHandler(out, &slog.HandlerOptions{
		Level: level, AddSource: true,
		ReplaceAttr: func(groups []string, attr slog.Attr) slog.Attr {
			if len(groups) == 0 && attr.Key == slog.TimeKey {
				return slog.Time(slog.TimeKey, attr.Value.Time().In(tz))
			}
			if len(groups) == 0 && attr.Key == slog.SourceKey {
				if source, ok := attr.Value.Any().(*slog.Source); ok && source != nil {
					copy := *source
					copy.File = sourceFile(copy.File)
					return slog.Any(slog.SourceKey, &copy)
				}
			}
			return attr
		},
	}))
}

// Output duplicates logs to stderr and an optional operator-managed file.
// Use logrotate or the container's logging driver for rotation/retention.
// It does not create a public HTTP endpoint or persist conversation content.
func Output(path string) (io.Writer, func() error, error) {
	if strings.TrimSpace(path) == "" {
		return os.Stderr, func() error { return nil }, nil
	}
	f, err := os.OpenFile(filepath.Clean(path), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		return nil, nil, err
	}
	if err := f.Chmod(0600); err != nil {
		_ = f.Close()
		return nil, nil, err
	}
	return io.MultiWriter(os.Stderr, f), f.Close, nil
}

func Type(value any) string {
	if value == nil {
		return "nil"
	}
	t := reflect.TypeOf(value)
	if t == nil {
		return "nil"
	}
	return t.String()
}

// Source returns only a source-relative file location, not local machine paths.
func Source(skip int) string {
	_, file, line, ok := runtime.Caller(skip + 1)
	if !ok {
		return "unknown"
	}
	return sourceFile(file) + ":" + strconv.Itoa(line)
}

func sourceFile(file string) string {
	file = filepath.ToSlash(file)
	if at := strings.LastIndex(file, "/apps/api/"); at >= 0 {
		return file[at+1:]
	}
	return filepath.Base(file)
}

// Do not store raw turn IDs or user IDs in operational logs. An ephemeral,
// per-process pseudonym lets HTTP request logs correlate with AI stage logs.
var turnSalt struct {
	once  sync.Once
	value [32]byte
}

func TurnRef(id [16]byte) string {
	if id == ([16]byte{}) {
		return ""
	}
	turnSalt.once.Do(func() {
		if _, err := rand.Read(turnSalt.value[:]); err != nil {
			panic("cryptographic randomness unavailable")
		}
	})
	h := sha256.New()
	_, _ = h.Write(turnSalt.value[:])
	_, _ = h.Write(id[:])
	return hex.EncodeToString(h.Sum(nil)[:8])
}
