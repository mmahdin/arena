package observability

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"database/sql/driver"
	"errors"
	"net"
	"regexp"
	"strings"
	"syscall"
)

// ErrorFields emits ONLY structural metadata. In particular, neither the
// error message nor an arbitrary third-party error value is ever serialized.
// The optional methods below are implemented only by our internal error types.
// Their values are checked against a closed vocabulary before being logged.
func ErrorFields(err error) []any {
	if err == nil {
		return nil
	}
	fields := []any{"error_type", Type(err)}
	if cause := errors.Unwrap(err); cause != nil {
		fields = append(fields, "cause_type", Type(cause))
		root := cause
		for i := 0; i < 12; i++ {
			next := errors.Unwrap(root)
			if next == nil {
				break
			}
			root = next
		}
		fields = append(fields, "root_cause_type", Type(root))
	}
	reason := ErrorReason(err)
	fields = append(fields, "reason", reason)
	if kind := errorKind(reason); kind != "" {
		fields = append(fields, "error_kind", kind)
	}
	var code interface{ DiagnosticCode() string }
	if errors.As(err, &code) {
		if value := safeCode(code.DiagnosticCode()); value != "" {
			fields = append(fields, "error_code", value)
		}
	}
	// A SQL error must not inherit the unrelated, usually false, API contract
	// Retryable default. Only known SQL numbers receive a retry hint.
	var db interface {
		SQLErrorNumber() int32
		SQLErrorState() uint8
		SQLErrorClass() uint8
	}
	if errors.As(err, &db) && db.SQLErrorNumber() > 0 && db.SQLErrorNumber() <= 999999 {
		number := db.SQLErrorNumber()
		fields = append(fields, "db_error_number", number, "db_error_state", db.SQLErrorState(), "db_error_class", db.SQLErrorClass())
		_, hint, retryable, known := classifySQL(number)
		if hint != "" {
			fields = append(fields, "diagnostic_hint", hint)
		}
		if known {
			fields = append(fields, "retryable", retryable)
		}
	} else {
		value := diagnosticValue(err, func(e error) string {
			if r, ok := e.(interface{ DiagnosticRetryable() bool }); ok {
				if r.DiagnosticRetryable() {
					return "true"
				}
				return "false"
			}
			return ""
		})
		if value != "" {
			fields = append(fields, "retryable", value == "true")
		}
	}
	if value := diagnosticValue(err, func(e error) string {
		if v, ok := e.(interface{ DiagnosticOperation() string }); ok {
			return safeOperation(v.DiagnosticOperation())
		}
		return ""
	}); value != "" {
		fields = append(fields, "operation", value)
	}
	for _, field := range []struct {
		name string
		get  func(error) string
	}{
		{"error_origin", func(e error) string {
			if v, ok := e.(interface{ DiagnosticOrigin() string }); ok {
				return safeOrigin(v.DiagnosticOrigin())
			}
			return ""
		}},
		{"failed_check", func(e error) string {
			if v, ok := e.(interface{ DiagnosticCheck() string }); ok {
				return safeCheck(v.DiagnosticCheck())
			}
			return ""
		}},
	} {
		if value := diagnosticValue(err, field.get); value != "" {
			fields = append(fields, field.name, value)
		}
	}
	if hint := diagnosticHint(reason); hint != "" {
		fields = append(fields, "diagnostic_hint", hint)
	}
	var stage interface{ DiagnosticStage() string }
	if errors.As(err, &stage) {
		if value := safeStage(stage.DiagnosticStage()); value != "" {
			fields = append(fields, "failure_stage", value)
		}
	}
	var status interface{ DiagnosticStatus() int }
	if errors.As(err, &status) {
		if value := status.DiagnosticStatus(); value >= 400 && value <= 599 {
			fields = append(fields, "provider_status", value)
		}
	}
	var providerCode interface{ DiagnosticProviderCode() string }
	if errors.As(err, &providerCode) {
		if code := SafeProviderCode(providerCode.DiagnosticProviderCode()); code != "" {
			fields = append(fields, "provider_error_code", code)
		}
	}
	var timeout interface{ DiagnosticTimeout() string }
	if errors.As(err, &timeout) {
		switch timeout.DiagnosticTimeout() {
		case "first_token", "connection", "request":
			fields = append(fields, "timeout_category", timeout.DiagnosticTimeout())
		}
	}
	return fields
}

// ErrorReason deliberately avoids error strings, network addresses and vendor
// payloads. Unknown errors are unknown rather than being guessed at.
func ErrorReason(err error) string {
	if err == nil {
		return "none"
	}
	// Explicit source timeout labels distinguish first-token/connection/total
	// deadlines. A generic wrapper must not hide a driver or network cause.
	explicit := diagnosticValue(err, func(e error) string {
		if r, ok := e.(interface{ DiagnosticReason() string }); ok && safeReason(r.DiagnosticReason()) {
			switch r.DiagnosticReason() {
			case "dependency_unavailable", "transport", "context_query_failed", "policy_query_failed", "context_state_mismatch", "unknown", "permanent", "unclassified_error":
				return ""
			}
			return r.DiagnosticReason()
		}
		return ""
	})
	if explicit != "" {
		return explicit
	}
	switch {
	case errors.Is(err, context.Canceled):
		return "canceled"
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout"
	}
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return "resource_not_visible"
	case errors.Is(err, driver.ErrBadConn):
		return "db_bad_connection"
	case errors.Is(err, sql.ErrTxDone):
		return "db_transaction_closed"
	case errors.Is(err, syscall.ECONNREFUSED):
		return "connection_refused"
	case errors.Is(err, syscall.ECONNRESET):
		return "connection_reset"
	}
	var dns *net.DNSError
	if errors.As(err, &dns) {
		return "dns_error"
	}
	var cert *tls.CertificateVerificationError
	var unknownCA x509.UnknownAuthorityError
	var invalidCert x509.CertificateInvalidError
	var host x509.HostnameError
	if errors.As(err, &cert) || errors.As(err, &unknownCA) || errors.As(err, &invalidCert) || errors.As(err, &host) {
		return "tls_error"
	}
	var network net.Error
	if errors.As(err, &network) {
		if network.Timeout() {
			return "timeout"
		}
		return "network_error"
	}
	var db interface{ SQLErrorNumber() int32 }
	if errors.As(err, &db) {
		if number := db.SQLErrorNumber(); number > 0 && number <= 999999 {
			reason, _, _, _ := classifySQL(number)
			return reason
		}
	}
	if value := diagnosticValue(err, func(e error) string {
		if r, ok := e.(interface{ DiagnosticReason() string }); ok && safeReason(r.DiagnosticReason()) {
			return r.DiagnosticReason()
		}
		return ""
	}); value != "" {
		return value
	}
	return "unknown"
}

// Classifications and remediation hints are fixed, code-owned labels. In
// particular, no SQL Server Message, SQL statement or server name is emitted.
// The retry flag applies to re-execution of an idempotent DB transaction, not
// automatically to a user-facing request or AI job.
func classifySQL(number int32) (reason, hint string, retryable, known bool) {
	switch number {
	case 207:
		return "db_invalid_column", "verify_schema_migrations", false, true
	case 208:
		return "db_missing_object", "verify_schema_migrations", false, true
	case 2812:
		return "db_missing_procedure", "verify_schema_migrations", false, true
	case 229:
		return "db_permission_denied", "check_database_grants", false, true
	case 547, 515:
		return "db_constraint_violation", "check_constraint_and_write", false, true
	case 2601, 2627:
		return "db_duplicate_key", "check_unique_constraint", false, true
	case 245, 8114:
		return "db_type_conversion", "check_query_types", false, true
	case 1205, 3960:
		return "db_transaction_conflict", "retry_transaction_with_backoff", true, true
	case 1222:
		return "db_lock_timeout", "check_database_locks", true, true
	case 18456, 4060:
		return "db_connection_rejected", "check_database_access", false, true
	default:
		return "db_error", "", false, false
	}
}

// The exact set is intentionally closed; it cannot be populated with request
// input or arbitrary error strings.
func safeOperation(s string) string {
	switch s {
	case "policy", "assess", "assemble", "turn_persistence", "state_write", "retrieve", "verify_context", "verify_generation", "source_use", "consent_read", "database_connect", "database_ping", "database_provision", "turn_admit", "turn_finalize", "turn_read", "turn_fail", "turn_cancel", "message_history", "state_read", "retrieval_query", "memory_access", "ai_jobs.claim_begin", "ai_jobs.reflection_lock", "ai_jobs.expire_exhausted", "ai_jobs.claim_ready", "ai_jobs.claim_commit":
		return s
	}
	return ""
}

func safeReason(s string) bool {
	switch s {
	case "local_quota_exhausted", "usage_invalid", "locale_unavailable", "ownership_mismatch", "task_decryption_failed", "identity_mismatch", "resource_not_visible", "source_role_mismatch", "source_content_mismatch", "source_chronology_invalid", "source_metadata_mismatch", "source_provenance_invalid", "consent_missing", "consent_not_granted", "consent_invalid", "policy_mapping_missing", "policy_denied", "policy_query_failed", "context_query_failed", "context_state_mismatch", "bundle_mismatch", "invalid_request", "permission_denied", "model_unavailable", "unsupported_parameter", "quota_exhausted", "dns_error", "tls_error", "connection_refused", "connection_reset", "connection_timeout", "request_timeout", "first_token_timeout", "request_canceled", "circuit_open", "unclassified_error", "db_bad_connection", "db_transaction_closed", "response_truncated", "stream_incomplete", "stream_guard_rejected", "response_format_invalid", "response_empty", "duplicate_usage", "input_budget_exceeded", "transport", "server", "rate_limit", "authentication", "permanent", "invalid_response", "timeout", "canceled", "network_error", "unknown", "invalid_signal", "unavailable", "dependency_unavailable", "degraded_result", "policy_fallback", "validation_failed", "stream_guard", "schema_invalid", "budget_exceeded", "state_conflict", "provider_unavailable", "retry_exhausted", "rejected", "fallback_unavailable", "generation_failed", "no_provider":
		return true
	}
	return false
}

func safeCode(s string) string {
	switch s {
	case "queued", "duplicate", "budget_exceeded", "safety_degraded", "retrieval_degraded", "brain_invalid", "brain_timeout", "generation_failed", "validation_failed", "state_conflict", "state_commit_failed", "bundle_unavailable", "provider_unavailable", "internal", "rate_limited", "forbidden":
		return s
	}
	return ""
}

var originPattern = regexp.MustCompile(`^apps/api/[A-Za-z0-9_./-]+\.go:[1-9][0-9]{0,6}$`)

func safeOrigin(s string) string {
	// A source code path is never a URL, an error message or a private payload.
	if len(s) > 240 || strings.Contains(s, "..") || !originPattern.MatchString(s) {
		return ""
	}
	return s
}

// DiagnosticFields enriches a stage event with a precise observation point
// even when the original error (e.g. a SQL driver error) has no source marker.
// error_site means observed here; error_origin is only emitted when captured
// at actual construction in one of our own error types.
func DiagnosticFields(err error, observationSite string) []any {
	fields := ErrorFields(err)
	if value := safeOrigin(observationSite); value != "" {
		fields = append(fields, "error_site", value)
	}
	return fields
}

// SafeLabel permits configured ASCII provider/model identifiers only, never
// a dynamic URL, payload, response or arbitrary identifier from an error.
func SafeLabel(s string) string {
	if len(s) == 0 || len(s) > 80 {
		return ""
	}
	for _, c := range s {
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_' || c == '.' || c == ':' {
			continue
		}
		return ""
	}
	return s
}

func safeCheck(s string) string {
	switch s {
	case "generator_input", "generator_render", "generator_locale", "generator_crisis", "brain_render", "hypothesis_policy", "offer_policy", "voice_policy", "memory_owner", "brain_hop", "safety_classifier", "safety_locale", "retrieval_owner", "job_validation", "job_decryption", "call_identity", "bundle_reference", "context_state", "context_handles", "turn_identity", "turn_admission", "current_message", "recent_message", "message_role", "message_content", "message_chronology", "recent_source_use", "retrieved_source_use", "memory_source_use", "memory_metadata", "memory_evidence", "permitted_current", "permitted_recent", "verbatim_provenance", "source_visibility", "consent_mapping", "consent_state", "source_kind":
		return s
	}
	return ""
}
func safeStage(s string) string {
	switch s {
	case "generator", "brain", "validator", "fallback", "finalizer", "admit", "recovery", "state_read", "retrieval", "memory_schedule", "assembler":
		return s
	}
	return ""
}
func diagnosticHint(reason string) string {
	switch reason {
	case "local_quota_exhausted":
		return "check_local_llm_quota"
	case "consent_missing", "consent_not_granted", "policy_mapping_missing":
		return "check_source_use_consent_and_mapping"
	case "resource_not_visible":
		return "check_owner_scoped_visibility_and_lifecycle"
	case "source_role_mismatch", "source_content_mismatch", "source_chronology_invalid", "source_metadata_mismatch", "source_provenance_invalid", "context_state_mismatch":
		return "check_context_assembly_and_source_provenance"
	case "authentication", "permission_denied":
		return "check_provider_credentials_and_access"
	case "model_unavailable", "unsupported_parameter", "invalid_request":
		return "check_provider_model_and_request_configuration"
	case "quota_exhausted", "rate_limit":
		return "check_provider_quota_and_limits"
	case "dns_error", "tls_error", "connection_refused", "connection_reset", "connection_timeout":
		return "check_dependency_connectivity"
	case "timeout", "request_timeout", "first_token_timeout":
		return "check_stage_budget_and_provider_latency"
	case "invalid_response", "schema_invalid", "stream_incomplete":
		return "check_provider_response_contract"
	}
	return ""
}

// Only known machine codes are accepted, even when a vendor returns an
// alphanumeric field containing private data. Never sanitize arbitrary bodies
// into printable labels.
func SafeProviderCode(s string) string {
	switch s {
	case "invalid_api_key", "insufficient_quota", "model_not_found", "unsupported_parameter", "invalid_parameter", "invalid_json_schema", "context_length_exceeded", "rate_limit_exceeded", "permission_denied":
		return s
	}
	return ""
}

func errorKind(reason string) string {
	switch reason {
	case "timeout", "request_timeout", "connection_timeout", "first_token_timeout":
		return "timeout"
	case "canceled", "request_canceled":
		return "canceled"
	case "identity_mismatch", "ownership_mismatch", "resource_not_visible", "source_role_mismatch", "source_content_mismatch", "source_chronology_invalid", "source_metadata_mismatch", "source_provenance_invalid", "consent_missing", "consent_not_granted", "consent_invalid", "policy_mapping_missing", "policy_denied", "context_state_mismatch":
		return "authorization"
	case "authentication", "permission_denied", "invalid_request", "model_unavailable", "unsupported_parameter", "quota_exhausted", "rate_limit", "server", "rejected":
		return "provider"
	case "dns_error", "tls_error", "connection_refused", "connection_reset", "transport", "network_error":
		return "network"
	case "schema_invalid", "invalid_response", "stream_incomplete", "response_truncated", "stream_guard_rejected", "response_format_invalid":
		return "validation"
	}
	if strings.HasPrefix(reason, "db_") {
		return "database"
	}
	return ""
}

// UniqueFields keeps records valid for parsers that reject duplicate JSON keys.
// More precise fields appended by a source override boundary defaults.
func UniqueFields(fields []any) []any {
	out := make([]any, 0, len(fields))
	positions := map[string]int{}
	for i := 0; i+1 < len(fields); i += 2 {
		key, ok := fields[i].(string)
		if !ok {
			continue
		}
		if at, exists := positions[key]; exists {
			out[at+1] = fields[i+1]
		} else {
			positions[key] = len(out)
			out = append(out, key, fields[i+1])
		}
	}
	return out
}
