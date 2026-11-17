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
