package contract

import (
	"encoding/json"
	"time"
)

// Adapted from authenticated Gin input; caller never supplies an authoritative UserID.
type TurnRequest struct {
	UserID         ID          `json:"user_id"`
	ConversationID ID          `json:"conversation_id"`
	ClientTurnID   ID          `json:"client_turn_id"` // stable client idempotency key
	Text           string      `json:"text"`
	ClientTS       *time.Time  `json:"client_ts,omitempty"`
	Locale         string      `json:"locale"` // input language hint; response comes from saved Prefs.Language
	TZ             string      `json:"tz"`
	Channel        string      `json:"channel"` // app_chat; future voice/push
	Origin         TurnOrigin  `json:"origin"`
	ReplyToID      *ID         `json:"reply_to_id,omitempty"`
	WakeUp         *WakeUpInfo `json:"wake_up,omitempty"`
}
type TurnOrigin string

const (
	OriginUser   TurnOrigin = "user"
	OriginWakeUp TurnOrigin = "wakeup"
)

type WakeUpInfo struct {
	Reason     string `json:"reason"`
	FollowUpID ID     `json:"follow_up_id"`
}

// Raw durable message, read and written through Conversation storage ports.
type Message struct {
	ID        ID          `json:"id"`
	UserID    ID          `json:"user_id"`
	ConvID    ID          `json:"conv_id"`
	TurnID    ID          `json:"turn_id"`
	SessionID *ID         `json:"session_id,omitempty"` // link in AI-owned state, not duplicated text
	Role      Role        `json:"role"`
	Content   string      `json:"content"`
	Lang      string      `json:"lang,omitempty"`
	CreatedAt time.Time   `json:"created_at"`
	Meta      MessageMeta `json:"meta"`
}
type MessageMeta struct {
	Objective     *Objective `json:"objective,omitempty"`
	QuestionText  *string    `json:"question_text,omitempty"`
	ProtocolStep  *string    `json:"protocol_step,omitempty"`
	Route         Route      `json:"route,omitempty"`
	BundleVersion string     `json:"bundle_version,omitempty"`
}

// Admission outcome: user input persisted durably before model generation.
// The port holds a durable, expiring ownership lease, not a SQL transaction.
type Admission struct {
	// Trusted ownership travels with the fenced admission, including on replay.
	UserID         ID        `json:"user_id"`
	ConversationID ID        `json:"conversation_id"`
	TurnID         ID        `json:"turn_id"`
	UserMessageID  ID        `json:"user_message_id"`
	SessionID      ID        `json:"session_id"`
	DuplicateOf    *ID       `json:"duplicate_of,omitempty"`
	Queued         bool      `json:"queued"`
	LeaseToken     ID        `json:"lease_token"`
	Recent         []MsgView `json:"recent"`
	Versions       Versions  `json:"versions"`
}

// The final validated AI result. Never contains an unvalidated draft.
type PreparedTurn struct {
	Decisions     *TurnDecisions     `json:"decisions,omitempty"`
	TurnID        ID                 `json:"turn_id"`
	UserMessageID ID                 `json:"user_message_id"`
	Text          string             `json:"text"`
	Route         Route              `json:"route"`
	Objective     Objective          `json:"objective"`
	Degraded      bool               `json:"degraded"`
	State         StateDelta         `json:"state"`
	Jobs          []AsyncJobProposal `json:"jobs,omitempty"`
	Trace         TurnTraceSummary   `json:"trace"`
	UI            *ProtocolUI        `json:"ui,omitempty"`
	Safety        *SafetyBanner      `json:"safety,omitempty"`
}
type TurnDecisions struct {
	Carry             []FutureHint     `json:"carry"`
	SelectedKnowledge []string         `json:"selected_knowledge"`
	Accepted          []string         `json:"accepted"`
	Rejected          []DeltaRejection `json:"rejected"`
	JEVDegraded       bool             `json:"jev_degraded"`
}
type AsyncJobProposal struct {
	Kind      string          `json:"kind"`
	Payload   json.RawMessage `json:"payload"`
	DedupeKey string          `json:"dedupe_key"`
}
type TurnTraceSummary struct {
	TurnID        ID        `json:"turn_id"`
	BundleVersion string    `json:"bundle_version"`
	Route         Route     `json:"route"`
	Objective     Objective `json:"objective"`
}

type TurnResult struct {
	TurnID             ID            `json:"turn_id"`
	UserMessageID      ID            `json:"user_message_id"`
	AssistantMessageID *ID           `json:"assistant_message_id,omitempty"`
	Text               string        `json:"text,omitempty"`
	Status             TurnStatus    `json:"status"` // queued|running|completed|failed|cancelled
	Route              Route         `json:"route,omitempty"`
	Degraded           bool          `json:"degraded"`
	UI                 *ProtocolUI   `json:"ui,omitempty"`
	Safety             *SafetyBanner `json:"safety,omitempty"`
}
type ProtocolUI struct {
	ProtocolInstanceID ID             `json:"protocol_instance_id"`
	StepID             string         `json:"step_id"`
	Widget             string         `json:"widget"`
	Params             map[string]any `json:"params,omitempty"`
}
type SafetyBanner struct {
	Level     RiskLevel        `json:"level"`
	Resources []CrisisResource `json:"resources"`
}
