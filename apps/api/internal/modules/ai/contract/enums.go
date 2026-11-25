package contract

// ---- Routing & objectives ----
type Route string

const (
	RouteNormal    Route = "normal"
	RouteFast      Route = "fast"
	RouteCrisis    Route = "crisis"
	RouteProactive Route = "proactive"
)

type Objective string

const (
	ObjReflect              Objective = "reflect"
	ObjValidateSupport      Objective = "validate_and_support"
	ObjClarify              Objective = "clarify"
	ObjExplore              Objective = "explore"
	ObjProvideInsight       Objective = "provide_insight"
	ObjProvideGuidance      Objective = "provide_guidance"
	ObjOfferIntervention    Objective = "offer_intervention"
	ObjContinueIntervention Objective = "continue_intervention"
	ObjCloseOrPause         Objective = "close_or_pause_intervention"
	ObjReviewProgress       Objective = "review_progress"
	ObjSummarize            Objective = "summarize"
	ObjRepair               Objective = "repair"
	ObjAnswerDirect         Objective = "answer_direct"
	ObjBoundary             Objective = "boundary"
	ObjSafetySupport        Objective = "safety_support" // router-assigned only
)

// ---- Safety ----
type RiskLevel string // ordered: none < low < elevated < high < imminent
const (
	RiskNone     RiskLevel = "none"
	RiskLow      RiskLevel = "low"
	RiskElevated RiskLevel = "elevated"
	RiskHigh     RiskLevel = "high"
	RiskImminent RiskLevel = "imminent"
)

type RiskCategory string

const (
	RiskSuicidalIdeation RiskCategory = "suicidal_ideation"
	RiskSelfHarm         RiskCategory = "self_harm"
	RiskHarmToOthers     RiskCategory = "harm_to_others"
	RiskAbuseDisclosure  RiskCategory = "abuse_disclosure"
	RiskMedicalEmergency RiskCategory = "medical_emergency"
	RiskSubstanceCrisis  RiskCategory = "substance_crisis"
	RiskOther            RiskCategory = "other"
)

// ---- Epistemics & memory ----
type Epistemic string

const (
	EpReported       Epistemic = "reported"
	EpInferred       Epistemic = "inferred"
	EpSystemObserved Epistemic = "system_observed"
	EpUnknown        Epistemic = "unknown" // only valid for Thread slots, never for stored items
)

type MemoryKind string

const (
	MemFact       MemoryKind = "fact"
	MemPreference MemoryKind = "preference"
	MemEpisode    MemoryKind = "episode"
	MemHypothesis MemoryKind = "hypothesis"
	MemGoal       MemoryKind = "goal"
	MemEvent      MemoryKind = "event"
	MemSafetyEvt  MemoryKind = "safety_event"
)

type MemoryStatus string

const (
	MemActive     MemoryStatus = "active"
	MemContested  MemoryStatus = "contested"
	MemSuperseded MemoryStatus = "superseded"
	MemRejected   MemoryStatus = "rejected"
	MemStale      MemoryStatus = "stale"
	// Hypothesis-only statuses (stored in the same column):
	MemProposed  MemoryStatus = "proposed"
	MemSupported MemoryStatus = "supported"
)

type Sensitivity string

const (
	SensNormal    Sensitivity = "normal"
	SensSensitive Sensitivity = "sensitive"
	SensHighly    Sensitivity = "highly_sensitive"
)

// ConfidenceClass is the low/medium/high vocabulary. Stored memory confidence
// must be derived from evidence counts; model assessments are advisory only.
type ConfidenceClass string

const (
	ConfLow    ConfidenceClass = "low"
	ConfMedium ConfidenceClass = "medium"
	ConfHigh   ConfidenceClass = "high"
)

type MemoryOp string

const (
	OpAdd        MemoryOp = "add"
	OpUpdate     MemoryOp = "update"
	OpConfirm    MemoryOp = "confirm"
	OpContradict MemoryOp = "contradict"
	OpSupersede  MemoryOp = "supersede"
	OpReject     MemoryOp = "reject"
	OpNoop       MemoryOp = "noop"
)

// ---- Threads & protocols ----
type ThreadPhase string

const (
	PhaseExploring     ThreadPhase = "exploring"
	PhaseUnderstanding ThreadPhase = "understanding_sufficient"
	PhaseWorking       ThreadPhase = "working"
	PhaseMaintaining   ThreadPhase = "maintaining"
	PhaseResolved      ThreadPhase = "resolved"
	PhasePaused        ThreadPhase = "paused"
	PhaseDormant       ThreadPhase = "dormant"
)

type ThreadRelation string

const (
	RelContinues ThreadRelation = "continues"
	RelShifts    ThreadRelation = "shifts"
	RelNew       ThreadRelation = "new"
	RelReturns   ThreadRelation = "returns_to_paused"
)

type ProtocolStatus string

const (
	PIOffered   ProtocolStatus = "offered"
	PIActive    ProtocolStatus = "active"
	PIPaused    ProtocolStatus = "paused"
	PICompleted ProtocolStatus = "completed"
	PIStopped   ProtocolStatus = "stopped"
	PIDeclined  ProtocolStatus = "declined"
)

type ProtocolAction string

const (
	PAAdvance  ProtocolAction = "advance"
	PARepeat   ProtocolAction = "repeat_step"
	PAAdapt    ProtocolAction = "adapt"
	PAPause    ProtocolAction = "pause"
	PAStop     ProtocolAction = "stop"
	PAComplete ProtocolAction = "complete"
	PAOffer    ProtocolAction = "offer" // objective offer_intervention
	PAStart    ProtocolAction = "start" // only after consent turn; set by engine, not Brain
	PADecline  ProtocolAction = "decline"
)

type StepOutcome string

const (
	OutSuccess  StepOutcome = "success"
	OutPartial  StepOutcome = "partial"
	OutStuck    StepOutcome = "stuck"
	OutDistress StepOutcome = "distress"
	OutDecline  StepOutcome = "decline"
	OutOffTopic StepOutcome = "off_topic"
)

// ---- Messages / sessions ----
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
)

type SessionMode string

const (
	ModeVenting   SessionMode = "venting"
	ModeExploring SessionMode = "exploring"
	ModeProblem   SessionMode = "problem_solving"
	ModeExercise  SessionMode = "exercise"
	ModeCheckIn   SessionMode = "checking_in"
	ModeSafety    SessionMode = "safety"
	ModeSmallTalk SessionMode = "small_talk"
)

// ---- Generation shape ----
type Tone string    // warm_steady | gentle | calm_direct | light | formal
type Length string  // minimal | short | medium | long
type Emotion string // free-form label from a controlled vocabulary in the bundle (config/emotions.yaml)
