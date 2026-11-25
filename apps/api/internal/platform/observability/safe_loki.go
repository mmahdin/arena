package observability

// MON-01B: a second, deliberately lossy log stream for private Loki ingestion.
// Never forward the ordinary slog JSON, Docker container logs, Inspector files,
// or arbitrary attributes. Only these hard-coded events and typed fields leave
// this boundary. This stream is NOT an audit log or a complete event journal.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	safeLogName     = "theia-safe.jsonl"
	safeLogMaxInput = 32 << 10
	safeLogMaxFile  = 4 << 20
	safeLogBackups  = 3
)

var (
	safeID         = regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-[1-8][a-f0-9]{3}-[89ab][a-f0-9]{3}-[a-f0-9]{12}$`)
	safeHex16      = regexp.MustCompile(`^[a-f0-9]{16}$`)
	safeHex32      = regexp.MustCompile(`^[a-f0-9]{32}$`)
	safeCodeShape  = regexp.MustCompile(`^[A-Z][A-Z0-9_]{2,63}$`)
	safeRouteShape = regexp.MustCompile(`^/[a-zA-Z0-9_/:.\-]{1,150}$`)
)

// Explicit event catalog: adding any new log class requires a security review.
var exportedEvents = map[string]bool{
	"http.request": true, "http.panic": true,
	"api.starting": true, "api.listening": true, "api.stopped": true,
	"api.exited": true, "api.server.failed": true, "api.shutdown.failed": true,
	"api.composition.failed": true, "api.database.connect_failed": true,
	"api.metrics.listen_failed": true, "api.metrics.server_failed": true,
	"api.telemetry.invalid": true, "api.telemetry.export_failed": true,
	"ai.turn.started": true, "ai.turn.finished": true, "ai.turn.failed": true,
	"ai.turn.admission_failed": true, "ai.turn.cleanup_failed": true,
	"ai.stage.completed": true, "ai.stage.failed": true, "ai.stage.started": true,
	"ai.stage.degraded": true, "ai.stage.skipped": true,
	"ai.worker.started": true, "ai.worker.failed": true, "ai.worker.job_failed": true,
	"ai.worker.job_completed": true, "ai.worker.retry": true, "ai.worker.recovered": true,
	"ai.telemetry.stage_write_failed": true, "ai.telemetry.heartbeat_write_failed": true,
	"ai.generation.attempt_failed": true, "ai.generation.fallback_used": true,
	"llm.attempt.completed": true, "llm.attempt.failed": true,
	"llm.call.failed": true, "llm.retry.started": true, "llm.chain.exhausted": true,
	"llm.http.rejected":  true,
	"database.connected": true, "database.connection.retry": true,
}

var knownErrorCodes = map[string]bool{
	"DB_UNAVAILABLE": true, "INTERNAL_ERROR": true,
	"AI_CORE_UNAVAILABLE": true, "AI_UNAVAILABLE": true,
	"AUTH_NOT_CONFIGURED": true, "AUTH_UNAVAILABLE": true,
	"AUTH_ORIGIN_MISMATCH": true, "AUTH_CALLBACK_INVALID": true,
	"SESSION_UNAVAILABLE": true, "ADMIN_AUDIT_UNAVAILABLE": true,
	"ADMIN_INSPECTOR_UNAVAILABLE": true, "ADMIN_TRACE_REFS_UNAVAILABLE": true,
	"ADMIN_OVERVIEW_UNAVAILABLE": true, "ADMIN_EVALUATIONS_UNAVAILABLE": true,
	"ADMIN_BUNDLES_UNAVAILABLE": true, "METRICS_LISTENER_UNAVAILABLE": true,
	"OTEL_INITIALIZATION_FAILED": true, "OTLP_UNAVAILABLE": true,
	"IDEMPOTENCY_CONFLICT": true, "VALIDATION_FAILED": true,
	"PROVIDER_UNAVAILABLE": true, "GENERATION_FAILED": true,
}

var knownComponents = map[string]bool{
	"ai_core": true, "database": true, "api": true, "http": true,
	"worker": true, "llm": true, "auth": true,
}

var knownStatuses = map[string]bool{
	"started": true, "success": true, "completed": true,
	"failed": true, "degraded": true, "skipped": true,
	"retry": true, "running": true, "cancelled": true,
	"queued": true, "unavailable": true,
}

var knownStages = map[string]bool{
	"turn": true, "admit": true, "parallel_dependencies": true, "policy_gate": true,
	"retrieval_hop": true, "state_validate": true, "recent": true, "safety": true,
	"retrieval": true, "assembler": true, "route": true, "brain": true,
	"generator": true, "validator": true, "fallback": true, "finalizer": true,
	"recovery": true, "cancel": true, "bundle": true, "state_read": true,
	"protocol_shortlist": true, "brain_dependency": true,
	"memory_schedule": true, "memory_correction": true,
	"protocol_integration": true, "protocol_requirements": true,
	"protocol_safety": true, "risk_delta": true, "protocol_commit": true,
	"generator_integration": true, "brain_integration": true, "crisis": true,
	"proactive": true, "outage_fallback": true, "stream_guard": true,
	"stream_size": true, "validator_bundle": true, "generator_bundle": true,
	"protocol_bundle": true, "reflection_bundle": true, "memory_bundle": true,
	"wakeup_contract": true, "deadline": true, "persistence": true,
}

// sanitizedLogRecord never copies arbitrary JSON objects, nested attributes,
// strings or exception text. Only already-known enum values, numeric measurements
// and syntactically verified opaque correlation IDs survive.
func sanitizedLogRecord(payload []byte) ([]byte, bool) {
	if len(payload) == 0 || len(payload) > safeLogMaxInput || bytes.Count(payload, []byte{'\n'}) > 1 {
		return nil, false
	}
	var incoming map[string]json.RawMessage
	if json.Unmarshal(bytes.TrimSpace(payload), &incoming) != nil {
		return nil, false
	}
	var event, level, timestamp string
	if json.Unmarshal(incoming["msg"], &event) != nil || !exportedEvents[event] {
		return nil, false
	}
	if json.Unmarshal(incoming["level"], &level) != nil {
		return nil, false
	}
	switch level {
	case "DEBUG", "INFO", "WARN", "ERROR":
	default:
		return nil, false
	}
	if json.Unmarshal(incoming["time"], &timestamp) != nil {
		return nil, false
	}
	when, err := time.Parse(time.RFC3339Nano, timestamp)
	if err != nil {
		return nil, false
	}
	record := map[string]any{
		"time": when.UTC().Format(time.RFC3339Nano), "level": level,
		"event": event, "service": "theia-api", "environment": "local",
		"schema": "theia.safe-log.v1",
	}
	for _, field := range []string{"request_id", "turn_ref", "trace_id", "span_id", "component", "stage", "status", "method", "route", "error_code"} {
		raw, ok := incoming[field]
		if !ok {
			continue
		}
		var value string
		if json.Unmarshal(raw, &value) != nil {
			continue
		}
		switch field {
		case "request_id":
			if !safeID.MatchString(value) {
				continue
			}
		case "turn_ref", "span_id":
			if !safeHex16.MatchString(value) {
				continue
			}
		case "trace_id":
			if !safeHex32.MatchString(value) {
				continue
			}
		case "component":
			if !knownComponents[value] {
				continue
			}
		case "stage":
			if !knownStages[value] {
				continue
			}
		case "status":
			if !knownStatuses[value] {
				continue
			}
		case "method":
			switch value {
			case "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS":
			default:
				continue
			}
		case "route":
			if !safeRouteShape.MatchString(value) || strings.Contains(value, "//") ||
				strings.Contains(value, "..") || strings.ContainsAny(value, "@?%") {
				continue
			}
			// Gin template routes have no opaque UUID or numeric user segment.
			segments := strings.Split(value, "/")
			good := true
			for _, segment := range segments {
				if safeID.MatchString(segment) || (len(segment) > 8 && strings.ContainsAny(segment, "0123456789")) {
					good = false
					break
				}
			}
			if !good {
				continue
			}
		case "error_code":
			if !safeCodeShape.MatchString(value) || !knownErrorCodes[value] {
				continue
			}
		}
		record[field] = value
	}
	for _, field := range []string{"duration_ms", "count", "attempt", "retry_index", "max_attempts", "provider_status", "db_error_number", "http_status", "scheduler_failures"} {
		raw, ok := incoming[field]
		if !ok {
			continue
		}
		value, err := strconv.ParseInt(string(raw), 10, 64)
		if err != nil || value < 0 || value > 1000000000 {
			continue
		}
		record[field] = value
	}
	if v, ok := incoming["status"]; ok && len(v) > 0 && v[0] != '"' {
		if n, err := strconv.ParseInt(string(v), 10, 64); err == nil && n >= 100 && n <= 599 {
			record["http_status"] = n
		}
	}
	for _, field := range []string{"failed", "degraded", "fallback_used", "retryable"} {
		raw, ok := incoming[field]
		if !ok {
			continue
		}
		if string(raw) == "true" {
			record[field] = true
		} else if string(raw) == "false" {
			record[field] = false
		}
	}
	result, err := json.Marshal(record)
	if err != nil {
		return nil, false
	}
	return append(result, '\n'), true
}

// SafeLogWriter is an append-only best-effort diagnostic sink. It rotates the
// sanitized file at 4MiB with three bounded backups; no raw input ever reaches
// the disk through this writer. A log write failure cannot affect an AI request.
type SafeLogWriter struct {
	mu              sync.Mutex
	path            string
	file            *os.File
	size            int64
	maxBytes        int64 // test seam; production uses safeLogMaxFile
	failureReported bool
}

func NewSafeLogWriter(path, environment string) (*SafeLogWriter, error) {
	if environment != "local" || !filepath.IsAbs(path) || filepath.Base(path) != safeLogName {
		return nil, errors.New("THEIA_SAFE_LOG_EXPORT requires local environment and absolute dedicated file path")
	}
	dir := filepath.Dir(path)
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0002 != 0 {
		return nil, errors.New("safe-log directory is missing or unsafe")
	}
	if info, err := os.Lstat(path); err == nil && (!info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0) {
		return nil, errors.New("safe-log target must be a regular file")
	} else if err != nil && !os.IsNotExist(err) {
		return nil, errors.New("safe-log target is inaccessible")
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0640)
	if err != nil {
		return nil, errors.New("safe-log export unavailable")
	}
	if err := f.Chmod(0640); err != nil {
		_ = f.Close()
		return nil, errors.New("safe-log permissions invalid")
	}
	fi, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, errors.New("safe-log stat failed")
	}
	return &SafeLogWriter{path: path, file: f, size: fi.Size(), maxBytes: safeLogMaxFile}, nil
}

func (w *SafeLogWriter) rotate() error {
	if err := w.file.Close(); err != nil {
		return err
	}
	for i := safeLogBackups; i > 1; i-- {
		_ = os.Remove(fmt.Sprintf("%s.%d", w.path, i))
		if err := os.Rename(fmt.Sprintf("%s.%d", w.path, i-1), fmt.Sprintf("%s.%d", w.path, i)); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	if err := os.Rename(w.path, w.path+".1"); err != nil && !os.IsNotExist(err) {
		return err
	}
	f, err := os.OpenFile(w.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0640)
	if err != nil {
		return err
	}
	if err := f.Chmod(0640); err != nil {
		_ = f.Close()
		return err
	}
	w.file = f
	w.size = 0
	return nil
}

func (w *SafeLogWriter) Write(p []byte) (int, error) {
	line, ok := sanitizedLogRecord(p)
	if !ok {
		return len(p), nil
	} // refuse unknown events or untrusted fields
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return len(p), nil
	}
	if w.size+int64(len(line)) > w.maxBytes {
		if err := w.rotate(); err != nil {
			w.failed()
			return len(p), nil
		}
	}
	n, err := w.file.Write(line)
	if err != nil || n != len(line) {
		w.failed()
	} else {
		w.size += int64(n)
	}
	// Keep the primary stderr logger and request processing unaffected.
	return len(p), nil
}

func (w *SafeLogWriter) failed() {
	if !w.failureReported {
		w.failureReported = true
		_, _ = io.WriteString(os.Stderr, "api.safe_log_export.failed\n")
	}
}

func (w *SafeLogWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	return err
}
