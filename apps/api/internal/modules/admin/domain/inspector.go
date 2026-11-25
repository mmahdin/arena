package domain

import (
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

var ErrInvalidInspector = errors.New("invalid admin inspector filter")
var ErrTurnNotFound = errors.New("admin turn not found")

const (
	DefaultInspectorLimit    = 25
	MaxInspectorLimit        = 100
	MaxInspectorWindow       = 31 * 24 * time.Hour
	MaxInspectorCursor       = 180
	InspectorStageLimit      = 50
	InspectorAttemptLimit    = 50
	InspectorRelatedJobLimit = 50
	InspectorWorkerLimit     = 100
)

type InspectorCursor struct {
	At time.Time
	ID uuid.UUID
}

func (c InspectorCursor) Encode() string {
	return base64.RawURLEncoding.EncodeToString([]byte(c.At.UTC().Format(time.RFC3339Nano) + "|" + c.ID.String()))
}

func DecodeInspectorCursor(value string) (*InspectorCursor, error) {
	if len(value) == 0 || len(value) > MaxInspectorCursor {
		return nil, ErrInvalidInspector
	}
	b, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil || len(b) > 120 {
		return nil, ErrInvalidInspector
	}
	parts := strings.Split(string(b), "|")
	if len(parts) != 2 {
		return nil, ErrInvalidInspector
	}
	at, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil || at.IsZero() {
		return nil, ErrInvalidInspector
	}
	id, err := uuid.Parse(parts[1])
	if err != nil || id == uuid.Nil || parts[1] != id.String() {
		return nil, ErrInvalidInspector
	}
	return &InspectorCursor{At: at.UTC(), ID: id}, nil
}

type InspectorFilter struct {
	From, To time.Time
	Limit    int
	Status   string
	Kind     string
	Cursor   *InspectorCursor
}

func NewInspectorFilter(now time.Time) InspectorFilter {
	end := now.UTC()
	return InspectorFilter{From: end.Add(-24 * time.Hour), To: end, Limit: DefaultInspectorLimit}
}
func (f InspectorFilter) Validate() error {
	if f.From.IsZero() || f.To.IsZero() || !f.To.After(f.From) || f.To.Sub(f.From) > MaxInspectorWindow ||
		f.Limit < 1 || f.Limit > MaxInspectorLimit ||
		f.Cursor != nil && (f.Cursor.ID == uuid.Nil || f.Cursor.At.IsZero() || !f.Cursor.At.Before(f.To) || f.Cursor.At.Before(f.From)) {
		return ErrInvalidInspector
	}
	if f.Status != "" && !ValidInspectorStatus(f.Status) {
		return ErrInvalidInspector
	}
	if f.Kind != "" && !ValidJobKind(f.Kind) {
		return ErrInvalidInspector
	}
	return nil
}
func ValidInspectorStatus(s string) bool {
	switch s {
	case "queued", "running", "completed", "failed", "cancelled":
		return true
	}
	return false
}
func ValidJobKind(s string) bool {
	switch s {
	case "memory_write", "memory_embed", "memory_feedback", "memory_maintain", "reflection":
		return true
	}
	return false
}

// SafeCode rejects arbitrary DB text, even for historical rows created before
// telemetry validation. Empty/malformed codes cannot reach an admin browser.
func SafeCode(s string) *string {
	if len(s) == 0 || len(s) > 80 {
		return nil
	}
	for _, r := range s {
		if r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_' || r == '-' || r == '.' || r == ':' {
			continue
		}
		return nil
	}
	return &s
}

// TurnRecord is a metadata projection; it deliberately omits user identity,
// messages, request/result JSON, input hashes, and trace encryption payload.
type TurnRecord struct {
	ID             uuid.UUID  `json:"id"`
	Status         string     `json:"status"`
	AttemptCount   int        `json:"attemptCount"`
	ErrorCode      *string    `json:"errorCode"`
	CreatedAt      time.Time  `json:"createdAt"`
	CompletedAt    *time.Time `json:"completedAt"`
	LeaseExpiresAt *time.Time `json:"leaseExpiresAt"`
}
type TurnPage struct {
	Items      []TurnRecord `json:"items"`
	NextCursor *string      `json:"nextCursor"`
}

type StageObservation struct {
	ID           uuid.UUID `json:"id"`
	Stage        string    `json:"stage"`
	Status       string    `json:"status"`
	DurationMs   *int64    `json:"durationMs"`
	Attempt      *int      `json:"attempt"`
	FallbackUsed bool      `json:"fallbackUsed"`
	RuleID       *string   `json:"ruleId"`
	RecordedAt   time.Time `json:"recordedAt"`
}
type LLMAttempt struct {
	ID                uuid.UUID  `json:"id"`
	Kind              string     `json:"kind"`
	Tier              string     `json:"tier"`
	Provider          string     `json:"provider"`
	Model             string     `json:"model"`
	Status            string     `json:"status"`
	FailureKind       *string    `json:"failureKind"`
	StartedAt         time.Time  `json:"startedAt"`
	FinishedAt        *time.Time `json:"finishedAt"`
	DurationMs        *int64     `json:"durationMs"`
	InputTokens       *int64     `json:"inputTokens"`
	OutputTokens      *int64     `json:"outputTokens"`
	CachedTokens      *int64     `json:"cachedTokens"`
	CostStatus        string     `json:"costStatus"`
	EstimatedMicroUsd *int64     `json:"estimatedMicroUsd"`
}
type JobRecord struct {
	ID             uuid.UUID  `json:"id"`
	TurnID         *uuid.UUID `json:"turnId"`
	Kind           string     `json:"kind"`
	Status         string     `json:"status"`
	Attempts       int        `json:"attempts"`
	MaxAttempts    int        `json:"maxAttempts"`
	LastErrorCode  *string    `json:"lastErrorCode"`
	RunAfter       time.Time  `json:"runAfter"`
	CreatedAt      time.Time  `json:"createdAt"`
	LeaseExpiresAt *time.Time `json:"leaseExpiresAt"`
}
type JobPage struct {
	Items      []JobRecord `json:"items"`
	NextCursor *string     `json:"nextCursor"`
}

// A bounded, best-effort observation of a turn, not a claim of full trace replay.
type TurnDetail struct {
	Turn              TurnRecord         `json:"turn"`
	Stages            []StageObservation `json:"stages"`
	LLMAttempts       []LLMAttempt       `json:"llmAttempts"`
	Jobs              []JobRecord        `json:"jobs"`
	StagesTruncated   bool               `json:"stagesTruncated"`
	AttemptsTruncated bool               `json:"attemptsTruncated"`
	JobsTruncated     bool               `json:"jobsTruncated"`
	Warnings          []string           `json:"warnings"`
}

type WorkerInstance struct {
	InstanceID uuid.UUID `json:"instanceId"`
	Kind       string    `json:"kind"`
	Status     string    `json:"status"` // running, stopped, failed, stale
	LastSeenAt time.Time `json:"lastSeenAt"`
	AgeSeconds int64     `json:"ageSeconds"`
}
type WorkerPage struct {
	AsOf      time.Time        `json:"asOf"`
	Instances []WorkerInstance `json:"instances"`
	Truncated bool             `json:"truncated"`
	Warnings  []string         `json:"warnings"`
}

// SafeInspectorLabel is the read-side guard for labels that predate durable
// telemetry validation or were written directly to SQL. Invalid metadata is
// replaced with a stable sentinel, never reflected verbatim to Admin.
func SafeInspectorLabel(value string) string {
	if SafeCode(value) == nil {
		return "unclassified"
	}
	return value
}

func SafeTurnStatus(value string) string {
	if ValidInspectorStatus(value) {
		return value
	}
	return "unknown"
}

func SafeStageStatus(value string) string {
	switch value {
	case "success", "failed", "degraded", "skipped":
		return value
	}
	return "unknown"
}

func SafeAttemptKind(value string) string {
	switch value {
	case "structured", "stream", "embed":
		return value
	}
	return "unknown"
}

func SafeAttemptStatus(value string) string {
	switch value {
	case "started", "succeeded", "failed", "incomplete":
		return value
	}
	return "unknown"
}

func SafeCostStatus(value string) string {
	switch value {
	case "estimated", "unpriced", "unknown":
		return value
	}
	return "unknown"
}

func SafeWorkerKind(value string) string {
	switch value {
	case "memory", "reflection", "scheduler":
		return value
	}
	return "unclassified"
}
