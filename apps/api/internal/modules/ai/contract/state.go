package contract

import (
	"time"
)

// StateDelta is the ONLY proposed AI state change. Built by the Orchestrator
// from validated Brain output; applied by the new TurnFinalizer transaction.
type StateDelta struct {
	SchemaVersion    string         `json:"schema_version,omitempty"`
	TurnID           ID             `json:"turn_id"`
	UserID           ID             `json:"user_id"`
	Session          SessionPatch   `json:"session"`
	Thread           *ThreadPatch   `json:"thread,omitempty"` // focus thread update or creation
	ThreadOps        []ThreadOp     `json:"thread_ops"`       // pause/resume/link other threads
	Protocol         *ProtocolPatch `json:"protocol,omitempty"`
	User             UserStatePatch `json:"user"`
	OpenLoops        []OpenLoopOp   `json:"open_loops"`
	FollowUps        []FollowUpOp   `json:"follow_ups"`
	Events           []DomainEvent  `json:"events"`            // for the outbox
	ExpectedVersions Versions       `json:"expected_versions"` // optimistic concurrency
}

// Versions are optimistic versions in AI-owned tables.
type Versions struct {
	Session  int64 `json:"session"`
	Thread   int64 `json:"thread"`
	Protocol int64 `json:"protocol"`
	User     int64 `json:"user"`
}

type SessionPatch struct {
	ClearFocus           bool             `json:"clear_focus,omitempty"`
	Mode                 *SessionMode     `json:"mode,omitempty"`
	FocusThread          *ID              `json:"focus_thread,omitempty"`
	LastObjective        Objective        `json:"last_objective"`
	PendingQuestion      *PendingQuestion `json:"pending_question,omitempty"`
	ClearPending         bool             `json:"clear_pending"`
	ConsecutiveQuestions int              `json:"consecutive_questions"`
	SummaryReplace       *string          `json:"summary_replace,omitempty"`
	SummaryDelta         *string          `json:"summary_delta,omitempty"`
	TurnCountInc         int              `json:"turn_count_inc"`
	EmotionLabels        []string         `json:"emotion_labels"`
}

type ThreadPatch struct {
	Goal              *string            `json:"goal,omitempty"`
	ThreadID          ID                 `json:"thread_id"`
	Create            *ThreadInit        `json:"create,omitempty"`
	Slots             []SlotUpdate       `json:"slots"`
	Phase             *ThreadPhase       `json:"phase,omitempty"`
	Touch             bool               `json:"touch"`
	HypothesisSignals []HypothesisSignal `json:"hypothesis_signals"` // recorded as events; applied by the Memory Committer (async)
	RelatedThreadIDs  []ID               `json:"related_thread_ids"`
}

type ThreadOp struct {
	ThreadID ID     `json:"thread_id"`
	Op       string `json:"op"` // pause | resume | dormant | resolve
	Reason   string `json:"reason"`
}

type UserStatePatch struct {
	Risk               *RiskTransition       `json:"risk,omitempty"`
	EngagementDelta    map[string]float64    `json:"engagement_delta"`
	DependencyWatch    *bool                 `json:"dependency_watch,omitempty"`
	ProtocolHistoryAdd *ProtocolHistoryBrief `json:"protocol_history_add,omitempty"`
	PrefsPatch         *PrefsPatch           `json:"prefs_patch,omitempty"`
}

type PrefsPatch struct {
	Length      *Length  `json:"length,omitempty"`
	Tone        *Tone    `json:"tone,omitempty"`
	AvoidAdd    []string `json:"avoid_add"`
	AvoidRemove []string `json:"avoid_remove"`
	AddressForm *string  `json:"address_form,omitempty"`
}

type OpenLoopOp struct {
	Op   string   `json:"op"` // add|close|expire
	Loop OpenLoop `json:"loop"`
}
type FollowUpOp struct {
	Op         string    `json:"op"` // schedule|cancel
	FollowUpID ID        `json:"follow_up_id"`
	DueAt      time.Time `json:"due_at"`
	Reason     string    `json:"reason"`
	Ref        *ID       `json:"ref,omitempty"`
}
type DomainEvent struct {
	Type    EventType `json:"type"`
	Payload any       `json:"payload"`
}

type CommitResult struct {
	Applied     bool             `json:"applied"`
	Rejected    []DeltaRejection `json:"rejected"` // parts dropped by FSM/invariant checks
	NewVersions Versions         `json:"new_versions"`
	Conflict    bool             `json:"conflict"` // version mismatch; caller reloads and retries once
}

type DeltaRejection struct {
	Path     string `json:"path"`
	RuleCode string `json:"rule_code"` // ST-xxx, see §10.2
	Detail   string `json:"detail"`
}
