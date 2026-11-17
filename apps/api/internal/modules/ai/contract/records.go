package contract

import (
	"time"
)

type SessionRecord struct {
	ID                   ID               `json:"id"`
	UserID               ID               `json:"user_id"`
	ConversationID       ID               `json:"conversation_id"`
	StartedAt            time.Time        `json:"started_at"`
	LastActivityAt       time.Time        `json:"last_activity_at"`
	ClosedAt             *time.Time       `json:"closed_at,omitempty"`
	Mode                 SessionMode      `json:"mode"`
	FocusThreadID        *ID              `json:"focus_thread_id,omitempty"`
	TurnCount            int              `json:"turn_count"`
	LastObjective        *Objective       `json:"last_objective,omitempty"`
	ConsecutiveQuestions int              `json:"consecutive_questions"`
	PendingQuestion      *PendingQuestion `json:"pending_question,omitempty"`
	Summary              string           `json:"summary"`
	EmotionTrajectory    []string         `json:"emotion_trajectory"`
	GapSincePrevious     *time.Duration   `json:"gap_since_previous,omitempty"`
	Version              int64            `json:"version"`
}

type ThreadRecord struct {
	ID                 ID                  `json:"id"`
	UserID             ID                  `json:"user_id"`
	Topic              string              `json:"topic"`
	Phase              ThreadPhase         `json:"phase"`
	PrevPhase          ThreadPhase         `json:"prev_phase"`
	PausedReason       *string             `json:"paused_reason,omitempty"`
	Priority           float32             `json:"priority"`
	Slots              map[string]SlotView `json:"slots"`
	Goal               *string             `json:"goal,omitempty"`
	HypothesisIDs      []ID                `json:"hypothesis_ids"`
	RelatedThreadIDs   []ID                `json:"related_thread_ids"`
	FormulationVersion int                 `json:"formulation_version"`
	ProblemTags        []string            `json:"problem_tags"`
	CreatedAt          time.Time           `json:"created_at"`
	LastActiveAt       time.Time           `json:"last_active_at"`
	Version            int64               `json:"version"`
}

type ThreadInit struct {
	Topic       string   `json:"topic"`
	ProblemTags []string `json:"problem_tags"`
	Goal        *string  `json:"goal,omitempty"`
}

type ProtocolInstance struct {
	ID               ID                `json:"id"`
	UserID           ID                `json:"user_id"`
	ThreadID         *ID               `json:"thread_id,omitempty"`
	ProtocolID       string            `json:"protocol_id"`
	ProtocolVersion  string            `json:"protocol_version"`
	Status           ProtocolStatus    `json:"status"`
	StepID           string            `json:"step_id"`
	Inputs           map[string]string `json:"inputs"`
	Attempts         int               `json:"attempts"`
	StuckCount       int               `json:"stuck_count"`
	OfferedAt        *time.Time        `json:"offered_at,omitempty"`
	ConsentAt        *time.Time        `json:"consent_at,omitempty"`
	CompletedAt      *time.Time        `json:"completed_at,omitempty"`
	ConsentMessageID *ID               `json:"consent_message_id,omitempty"`
	Outcome          *string           `json:"outcome,omitempty"`
	UserRating       *int              `json:"user_rating,omitempty"`
	FollowUpDueAt    *time.Time        `json:"follow_up_due_at,omitempty"`
	LastUserSummary  string            `json:"last_user_summary"`
	Version          int64             `json:"version"`
}

type ProtocolInstanceInit struct {
	ProtocolID string    `json:"protocol_id"`
	Version    string    `json:"version"`
	ThreadID   *ID       `json:"thread_id,omitempty"`
	OfferedAt  time.Time `json:"offered_at"`
}

type UserState struct {
	UserID          ID                     `json:"user_id"`
	TZ              string                 `json:"tz"`
	Locale          string                 `json:"locale"`
	Prefs           PrefsView              `json:"prefs"`
	PrefsVersion    int                    `json:"prefs_version"`
	Risk            RiskState              `json:"risk"`
	Engagement      map[string]float64     `json:"engagement"`
	DependencyWatch bool                   `json:"dependency_watch"`
	ProtocolHistory []ProtocolHistoryBrief `json:"protocol_history"`
	ComfortBudget   float32                `json:"comfort_budget"`
	ProactiveOptIn  bool                   `json:"proactive_opt_in"`
	QuietHours      *QuietHours            `json:"quiet_hours,omitempty"`
	MemoryPaused    bool                   `json:"memory_paused"`
	Consent         map[string]bool        `json:"consent"`
	Version         int64                  `json:"version"`
}

type FollowUp struct {
	ID     ID        `json:"id"`
	UserID ID        `json:"user_id"`
	Reason string    `json:"reason"`
	DueAt  time.Time `json:"due_at"`
	Ref    *ID       `json:"ref,omitempty"`
	Status string    `json:"status"`
}

type LLMUsage struct {
	Provider     string        `json:"provider"`
	Model        string        `json:"model"`
	InTokens     int           `json:"in_tokens"`
	OutTokens    int           `json:"out_tokens"`
	CachedTokens int           `json:"cached_tokens"`
	CostMicroUSD int64         `json:"cost_micro_usd"`
	Latency      time.Duration `json:"latency"`
	Attempts     int           `json:"attempts"`
	FinishReason string        `json:"finish_reason"`
}

type QuietHours struct {
	Start string `json:"start"`
	End   string `json:"end"`
}
