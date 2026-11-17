package contract

import (
	"time"
)

type WorkingContext struct {
	SchemaVersion string            `json:"schema_version"` // wctx.v1
	Turn          TurnView          `json:"turn"`
	Recent        []MsgView         `json:"recent"` // last N, token-capped, oldest first
	Session       SessionView       `json:"session"`
	FocusThread   *ThreadView       `json:"focus_thread,omitempty"`
	OtherThreads  []ThreadBrief     `json:"other_threads,omitempty"` // max 4
	Protocol      *ProtocolView     `json:"protocol,omitempty"`
	Prefs         PrefsView         `json:"prefs"`
	Risk          RiskView          `json:"risk"`
	OpenLoops     []OpenLoop        `json:"open_loops,omitempty"`
	InfoNeeds     []InfoNeed        `json:"info_needs,omitempty"` // max 2
	Retrieved     []MemoryView      `json:"retrieved,omitempty"`  // max 5 (+1 per extra hop, hard cap 7)
	RetrievalMeta RetrievalMeta     `json:"retrieval_meta"`
	Resumption    *ResumptionView   `json:"resumption,omitempty"`
	Formulation   *FormulationBrief `json:"formulation,omitempty"` // active thread only
	Eligibility   Eligibility       `json:"eligibility"`
	Shortlist     []ProtocolBrief   `json:"shortlist,omitempty"` // 0..5; absent if not eligible
	TokenEstimate int               `json:"token_estimate"`
}

type TurnView struct {
	TurnID         ID            `json:"turn_id"`
	MessageID      ID            `json:"message_id"` // the (coalesced) triggering user message
	Text           string        `json:"text"`
	Lang           string        `json:"lang"`
	LocalTime      string        `json:"local_time"` // "Sat 22:40" in the user's tz
	SinceLastMsg   time.Duration `json:"since_last_msg"`
	Origin         TurnOrigin    `json:"origin"`
	ReplyToPending bool          `json:"reply_to_pending"` // heuristic: answers pending_question
	TemporalRefs   []TemporalRef `json:"temporal_refs,omitempty"`
	WakeUpReason   string        `json:"wake_up_reason,omitempty"`
}

type TemporalRef struct {
	Phrase     string    `json:"phrase"`
	From       time.Time `json:"from"`
	To         time.Time `json:"to"`
	RecallVerb bool      `json:"recall_verb"` // "what did I tell you..."
}

type MsgView struct {
	ID        ID         `json:"id"`
	Role      Role       `json:"role"`
	Text      string     `json:"text"` // truncated per budget; marked with an ellipsis when cut
	Age       string     `json:"age"`  // 2m, 3h, 2d
	Objective *Objective `json:"objective,omitempty"`
}

type SessionView struct {
	SessionID            ID               `json:"session_id"`
	Mode                 SessionMode      `json:"mode"`
	TurnCount            int              `json:"turn_count"`
	LastObjective        *Objective       `json:"last_objective,omitempty"`
	ConsecutiveQuestions int              `json:"consecutive_questions"`
	PendingQuestion      *PendingQuestion `json:"pending_question,omitempty"`
	Summary              string           `json:"summary,omitempty"`            // rolling, <= 120 tokens
	EmotionTrajectory    []string         `json:"emotion_trajectory,omitempty"` // last 5 labels
	Resumed              bool             `json:"resumed"`
}

type PendingQuestion struct {
	Slot      string    `json:"slot,omitempty"`
	Text      string    `json:"text"`
	AskedAt   time.Time `json:"asked_at"`
	MessageID ID        `json:"message_id"`
}

type ThreadView struct {
	ID              ID                     `json:"id"`
	Topic           string                 `json:"topic"`
	Phase           ThreadPhase            `json:"phase"`
	Slots           []SlotView             `json:"slots"`                // all schema slots, unknown included
	Hypotheses      []HypView              `json:"hypotheses,omitempty"` // proposed|supported only, max 3
	Goal            *string                `json:"goal,omitempty"`
	LastActive      time.Time              `json:"last_active"`
	ProtocolHistory []ProtocolHistoryBrief `json:"protocol_history,omitempty"`
}

type SlotView struct {
	Slot     string     `json:"slot"`
	Status   Epistemic  `json:"status"` // reported | inferred | unknown
	Value    string     `json:"value,omitempty"`
	Evidence []ID       `json:"evidence,omitempty"`
	Asked    *time.Time `json:"last_asked,omitempty"`
	Declined bool       `json:"declined"`
}

type HypView struct {
	ID         ID              `json:"id"`
	Text       string          `json:"text"` // tentative wording stored with the item
	Status     MemoryStatus    `json:"status"`
	Confidence ConfidenceClass `json:"confidence"`
	Support    int             `json:"support"`
	Contradict int             `json:"contradict"`
	Voiced     bool            `json:"voiced"` // already surfaced to the user
}

type ThreadBrief struct {
	ID         ID          `json:"id"`
	Topic      string      `json:"topic"`
	Phase      ThreadPhase `json:"phase"`
	LastActive time.Time   `json:"last_active"`
}

type ProtocolView struct {
	InstanceID      ID               `json:"instance_id"`
	ProtocolID      string           `json:"protocol_id"`
	Version         string           `json:"version"`
	Status          ProtocolStatus   `json:"status"`
	StepID          string           `json:"step_id"`
	StepGoal        string           `json:"step_goal"`
	ExpectedInput   string           `json:"expected_input"`
	Attempts        int              `json:"attempts"`
	StuckCount      int              `json:"stuck_count"`
	LastUserSummary string           `json:"last_user_summary,omitempty"`
	AllowedActions  []ProtocolAction `json:"allowed_actions"`
}

type PrefsView struct {
	Length      Length   `json:"length"`
	Tone        Tone     `json:"tone"`
	Language    string   `json:"language"`
	Avoid       []string `json:"avoid,omitempty"` // user "do not" list
	Likes       []string `json:"likes,omitempty"`
	AddressForm string   `json:"address_form,omitempty"` // formal/informal where relevant
}

type RiskView struct {
	Current         RiskLevel `json:"current"` // max(state, this turn's SafetySignal)
	FollowUpActive  bool      `json:"follow_up_active"`
	DependencyWatch bool      `json:"dependency_watch"`
}

type OpenLoop struct {
	ID    ID         `json:"id"`
	Kind  string     `json:"kind"` // unanswered_question|exercise_review|confirm_fact|resume_protocol|user_commitment
	Text  string     `json:"text"`
	DueAt *time.Time `json:"due_at,omitempty"`
	Ref   *ID        `json:"ref,omitempty"`
}

type InfoNeed struct {
	Slot    string  `json:"slot"`
	Purpose string  `json:"purpose"` // short bundle-defined reason
	Score   float32 `json:"score"`
	AskHint string  `json:"ask_hint"` // neutral hint, not a script
}

type MemoryView struct {
	ID          ID           `json:"id"`
	Kind        MemoryKind   `json:"kind"`
	Epistemic   Epistemic    `json:"epistemic"`
	Status      MemoryStatus `json:"status"`
	Text        string       `json:"text"`
	Age         string       `json:"age"`
	ValidAt     *string      `json:"valid_at,omitempty"`
	EvidenceN   int          `json:"evidence_n"`
	Sensitivity Sensitivity  `json:"sensitivity"`
	Verbatim    bool         `json:"verbatim"` // true when a raw message excerpt (exact recall)
	Score       float32      `json:"score"`
}

type RetrievalMeta struct {
	Strategies         []string `json:"strategies"`
	AmbiguousReference bool     `json:"ambiguous_reference"`
	Degraded           bool     `json:"degraded"` // failed/timed out: Brain must not imply recall
	Hops               int      `json:"hops"`
	ExactMatchFound    *bool    `json:"exact_match_found,omitempty"`
}

type ResumptionView struct {
	Gap              time.Duration `json:"gap"`
	Band             string        `json:"band"` // same | short | day | week | long
	RecentRisk       bool          `json:"recent_risk"`
	SuggestedReentry string        `json:"suggested_reentry,omitempty"` // policy hint id, not text
}

type FormulationBrief struct {
	ThreadID  ID       `json:"thread_id"`
	Version   int      `json:"version"`
	Paragraph string   `json:"paragraph"`
	Gaps      []string `json:"gaps"`
}

type Eligibility struct {
	ProtocolsPossible  bool     `json:"protocols_possible"`
	ReasonsBlocked     []string `json:"reasons_blocked,omitempty"` // risk_elevated|recent_decline|phase_too_early|no_action_cue|active_protocol
	MayVoiceHypothesis bool     `json:"may_voice_hypothesis"`
	QuestionAllowed    bool     `json:"question_allowed"` // false if consecutive_questions >= limit
	MemoryAvailable    bool     `json:"memory_available"`
}

type ProtocolBrief struct {
	ProtocolID  string   `json:"protocol_id"`
	Name        string   `json:"name"`
	OneLiner    string   `json:"one_liner"`
	FitTags     []string `json:"fit_tags"`
	EstMinutes  int      `json:"est_minutes"`
	PastOutcome *string  `json:"past_outcome,omitempty"` // helped | not_helpful | declined
}

type ProtocolHistoryBrief struct {
	ProtocolID string    `json:"protocol_id"`
	Outcome    string    `json:"outcome"`
	At         time.Time `json:"at"`
}
