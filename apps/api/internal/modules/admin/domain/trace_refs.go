package domain

import (
	"errors"
	"regexp"
	"time"

	"github.com/google/uuid"
)

const MaxTraceReferencePage = 50
const DefaultTraceReferencePage = 25

var w3cTraceID = regexp.MustCompile(`^[0-9a-f]{32}$`)
var w3cSpanID = regexp.MustCompile(`^[0-9a-f]{16}$`)

// TraceReference intentionally contains only root correlation metadata.
// There is no user ID, payload, local-inspector UUID/path or arbitrary URL.
type TraceReference struct {
	ID         uuid.UUID  `json:"id"`
	Kind       string     `json:"kind"`
	TraceID    string     `json:"traceId"`
	RootSpanID string     `json:"rootSpanId"`
	JobID      *uuid.UUID `json:"jobId"`
	JobAttempt *int       `json:"jobAttempt"`
	RecordedAt time.Time  `json:"recordedAt"`
}

func (r TraceReference) Validate() error {
	if r.ID == uuid.Nil || !w3cTraceID.MatchString(r.TraceID) || !w3cSpanID.MatchString(r.RootSpanID) ||
		r.TraceID == "00000000000000000000000000000000" || r.RootSpanID == "0000000000000000" || r.RecordedAt.IsZero() {
		return ErrInvalidInspector
	}
	switch r.Kind {
	case "turn":
		if r.JobID != nil || r.JobAttempt != nil {
			return ErrInvalidInspector
		}
	case "memory", "reflection":
		if r.JobID == nil || *r.JobID == uuid.Nil || r.JobAttempt == nil || *r.JobAttempt < 1 || *r.JobAttempt > 1000000 {
			return ErrInvalidInspector
		}
	default:
		return ErrInvalidInspector
	}
	return nil
}

type TraceReferencePage struct {
	TurnID     uuid.UUID        `json:"turnId"`
	Items      []TraceReference `json:"items"`
	NextCursor *string          `json:"nextCursor"`
}

type TraceReferenceFilter struct {
	TurnID uuid.UUID
	Limit  int
	Cursor *InspectorCursor
}

func (f TraceReferenceFilter) Validate() error {
	if f.TurnID == uuid.Nil || f.Limit < 1 || f.Limit > MaxTraceReferencePage ||
		f.Cursor != nil && (f.Cursor.ID == uuid.Nil || f.Cursor.At.IsZero() || f.Cursor.At.After(time.Now().UTC().Add(time.Second))) {
		return ErrInvalidInspector
	}
	return nil
}

var ErrTraceRefUnavailable = errors.New("trace references unavailable")
