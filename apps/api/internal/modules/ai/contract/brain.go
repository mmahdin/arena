package contract

type BrainDecision struct {
	SchemaVersion string              `json:"schema_version"` // brain.v1
	TurnAnalysis  TurnAnalysis        `json:"turn_analysis"`
	Understanding UnderstandingUpdate `json:"understanding_update"`
	Needs         Needs               `json:"needs"`
	Decision      Decision            `json:"decision"`
	Plan          ResponsePlan        `json:"response_plan"`
	RiskRead      *BrainRiskRead      `json:"risk_read,omitempty"` // may only raise
}

type TurnAnalysis struct {
	Intent           Intent          `json:"intent"`       // share_distress|seek_advice|answer_question|report_exercise|ask_recall|ask_info|small_talk|disagree_correct|request_technique|disclose_event|other
	MessageType      MessageType     `json:"message_type"` // disclosure|question|answer|report|request|meta|greeting
	ThreadRelation   ThreadRelation  `json:"thread_relation"`
	TargetThreadID   *ID             `json:"target_thread_id,omitempty"` // returns_to_paused / shift to existing
	NewThreadTopic   *string         `json:"new_thread_topic,omitempty"` // <= 8 words, for new
	Emotions         []EmotionRead   `json:"emotions"`
	PastRefs         []PastRef       `json:"past_references"`
	AnsweredPending  bool            `json:"answered_pending_question"`
	Ambiguities      []Ambiguity     `json:"ambiguities"`
	ActionCue        bool            `json:"action_cue"`                  // user signals readiness for suggestions/techniques
	UserReadiness    ConfidenceClass `json:"user_readiness"`              // low | medium | high
	CorrectionTarget *ID             `json:"correction_target,omitempty"` // memory item being disputed (repair)
}

type EmotionRead struct {
	Label     Emotion         `json:"label"`
	Intensity ConfidenceClass `json:"intensity"` // low | medium | high
}

type PastRef struct {
	Phrase     string `json:"phrase"`
	ResolvedTo []ID   `json:"resolved_to"` // ids present in the Working Context only
	Ambiguous  bool   `json:"ambiguous"`
}

type Ambiguity struct {
	Kind string `json:"kind"` // reference|emotion|situation|goal|timeframe|person
	Note string `json:"note"` // <= 12 words
}

type UnderstandingUpdate struct {
	ThreadID            *ID                `json:"thread_id,omitempty"`
	Slots               []SlotUpdate       `json:"slots"`
	Hypotheses          []HypothesisSignal `json:"hypotheses"`
	Unknowns            []string           `json:"unknowns"` // <= 3 short phrases
	SessionMode         *SessionMode       `json:"session_mode,omitempty"`
	SessionSummaryDelta *string            `json:"session_summary_delta,omitempty"` // <= 40 words; merged by code
	ThreadPhaseProposal *ThreadPhase       `json:"thread_phase_proposal,omitempty"`
}

type SlotUpdate struct {
	Slot     string    `json:"slot"`     // must exist in the thread schema
	Value    string    `json:"value"`    // <= 30 words, user language
	Status   Epistemic `json:"status"`   // reported | inferred (never unknown)
	Evidence []ID      `json:"evidence"` // message ids in context; required, non-empty
	Declined bool      `json:"declined,omitempty"`
}

type HypothesisSignal struct {
	Op       HypothesisOp `json:"op"`            // support|contradict|propose|user_endorsed|user_rejected
	ID       *ID          `json:"id,omitempty"`  // required for all but propose
	Tag      *string      `json:"tag,omitempty"` // required for propose; config/hypothesis_tags.yaml
	Evidence []ID         `json:"evidence"`
}

type Needs struct {
	Context      *NeedsContext `json:"needs_context,omitempty"`
	Formulation  bool          `json:"needs_formulation"`
	MemoryWorthy bool          `json:"memory_worthy"`
	MemoryTags   []string      `json:"memory_tags,omitempty"` // recurrence|goal|preference|event|hypothesis_support|correction|feedback|protocol_outcome
}

type Decision struct {
	Primary      Objective       `json:"primary_objective"`
	Rationale    []string        `json:"rationale_tags"`          // config/rationale_tags.yaml, <= 5
	Alternatives []Objective     `json:"alternatives_considered"` // <= 2
	Confidence   ConfidenceClass `json:"confidence"`              // low | medium | high
}

type ResponsePlan struct {
	Skills            SkillSelection     `json:"skills"`
	Question          QuestionPlan       `json:"question"`
	Tone              Tone               `json:"tone"`
	Length            Length             `json:"length"`
	PermittedContext  []ID               `json:"permitted_context_ids"`
	MustAvoid         []string           `json:"must_avoid"` // config/avoid_tags.yaml
	Protocol          *ProtocolDirective `json:"protocol,omitempty"`
	VoiceHypothesisID *ID                `json:"voice_hypothesis_id,omitempty"` // only with provide_insight
}

type SkillSelection struct {
	Primary   string   `json:"primary"`
	Secondary []string `json:"secondary"` // <= 2
}

type QuestionPlan struct {
	Allowed  bool   `json:"allowed"`
	Purpose  string `json:"purpose,omitempty"` // close_gap|invite_detail|check_readiness|confirm|choose|none
	Slot     string `json:"slot,omitempty"`
	MaxCount int    `json:"max_count"` // 0 or 1
}

type ProtocolDirective struct {
	Action         ProtocolAction    `json:"action"`
	ProtocolID     *string           `json:"protocol_id,omitempty"`     // required for offer; must be in shortlist
	Outcome        *StepOutcome      `json:"step_outcome,omitempty"`    // required when a protocol is active
	InputsCaptured map[string]string `json:"inputs_captured,omitempty"` // keys in step.expected_inputs; <= 30 words each
	Note           *string           `json:"note_tag,omitempty"`        // user_declined|too_distressed|needs_simpler|topic_shift|completed_all
}

type BrainRiskRead struct {
	Level      RiskLevel      `json:"level"` // honored only if above current risk
	Categories []RiskCategory `json:"categories"`
	ReasonTag  string         `json:"reason_tag"`
}
