package contract

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// Error deliberately excludes Cause and Step from display/JSON. Cause remains
// available to errors.Is/As for control flow, never as a public error message.
func (e *Error) Error() string { return string(ToPublic(e).Code) }
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}
func (e *Error) Format(s fmt.State, _ rune)   { _, _ = io.WriteString(s, e.Error()) }
func (e *Error) MarshalJSON() ([]byte, error) { return json.Marshal(ToPublic(e)) }

// ToPublic is the only supported conversion across the AI Core boundary.
// Unknown errors and unknown codes fail to a fixed, non-retryable internal error.
func ToPublic(err error) PublicError {
	out := PublicError{Code: ErrInternal, Message: "The request could not be completed."}
	var core *Error
	if !errors.As(err, &core) || core == nil || !core.Code.Valid() {
		return out
	}
	out.Code, out.Retryable = core.Code, core.Retryable
	switch core.Code {
	case ErrQueued:
		out.Message = "The turn is queued."
	case ErrDuplicate:
		out.Message = "The turn has already been submitted."
	case ErrForbidden:
		out.Message = "The requested resource is unavailable."
	case ErrRateLimited:
		out.Message = "Please try again later."
	default:
		out.Message = "The turn could not be completed."
	}
	return out
}

// Diagnostic* expose fixed-format internal metadata, never the private cause
// message or serialized Error. Origin is populated at construction sites.
func (e *Error) DiagnosticCode() string {
	if e == nil || !e.Code.Valid() {
		return ""
	}
	return string(e.Code)
}
func (e *Error) DiagnosticRetryable() bool { return e != nil && e.Retryable }
func (e *Error) DiagnosticOrigin() string {
	if e == nil {
		return ""
	}
	return e.Origin
}

func (e *Error) DiagnosticReason() string {
	if e == nil {
		return ""
	}
	return e.Reason
}
func (e *Error) DiagnosticCheck() string {
	if e == nil {
		return ""
	}
	return e.Check
}

// Stage is the component that executed the failed operation. Step retains the
// precise internal substep and is never serialized into a public response.
func (e *Error) DiagnosticStage() string {
	if e == nil {
		return ""
	}
	switch e.Step {
	case "unified", "unified_context", "unified_input", "unified_schema", "unified_references":
		return "unified"
	case "jev", "jev_selector", "jev_bundle":
		return "jev"
	case "generator_input", "generator_render", "generator_locale", "generator_crisis", "generator", "generator_identity", "generator_sources", "generator_context", "generator_stream", "generator_guard", "generator_format", "generator_bundle":
		return "generator"
	case "brain_render", "brain", "brain_context", "brain_identity", "brain_bundle":
		return "brain"
	case "validator", "fallback", "finalizer", "admit", "recovery", "state_read", "retrieval", "memory_schedule", "assembler":
		return e.Step
	}
	return ""
}
