package contract

const (
	ToneWarmSteady Tone = "warm_steady"
	ToneGentle     Tone = "gentle"
	ToneCalmDirect Tone = "calm_direct"
	ToneLight      Tone = "light"
	ToneFormal     Tone = "formal"
)

const (
	LengthMinimal Length = "minimal"
	LengthShort   Length = "short"
	LengthMedium  Length = "medium"
	LengthLong    Length = "long"
)

type TurnStatus string

const (
	TurnQueued    TurnStatus = "queued"
	TurnRunning   TurnStatus = "running"
	TurnCompleted TurnStatus = "completed"
	TurnFailed    TurnStatus = "failed"
	TurnCancelled TurnStatus = "cancelled"
)

type Intent string

const (
	IntentShareDistress    Intent = "share_distress"
	IntentSeekAdvice       Intent = "seek_advice"
	IntentAnswerQuestion   Intent = "answer_question"
	IntentReportExercise   Intent = "report_exercise"
	IntentAskRecall        Intent = "ask_recall"
	IntentAskInfo          Intent = "ask_info"
	IntentSmallTalk        Intent = "small_talk"
	IntentDisagreeCorrect  Intent = "disagree_correct"
	IntentRequestTechnique Intent = "request_technique"
	IntentDiscloseEvent    Intent = "disclose_event"
	IntentOther            Intent = "other"
)

type MessageType string

const (
	MessageDisclosure MessageType = "disclosure"
	MessageQuestion   MessageType = "question"
	MessageAnswer     MessageType = "answer"
	MessageReport     MessageType = "report"
	MessageRequest    MessageType = "request"
	MessageMetaType   MessageType = "meta"
	MessageGreeting   MessageType = "greeting"
)

type HypothesisOp string

const (
	HypSupport      HypothesisOp = "support"
	HypContradict   HypothesisOp = "contradict"
	HypPropose      HypothesisOp = "propose"
	HypUserEndorsed HypothesisOp = "user_endorsed"
	HypUserRejected HypothesisOp = "user_rejected"
)

type ContextKind string

const (
	ContextEntity     ContextKind = "entity"
	ContextTopic      ContextKind = "topic"
	ContextTimeWindow ContextKind = "time_window"
	ContextItem       ContextKind = "item"
)

type ContextReason string

const (
	ContextReferenceUnresolved ContextReason = "reference_unresolved"
	ContextMissingBackground   ContextReason = "missing_background"
	ContextVerifyFact          ContextReason = "verify_fact"
)

type SafetyContextFlag string

const (
	ContextQuoted       SafetyContextFlag = "quoted"
	ContextFictional    SafetyContextFlag = "fictional"
	ContextHypothetical SafetyContextFlag = "hypothetical"
	ContextThirdPerson  SafetyContextFlag = "third_person"
	ContextPastEvent    SafetyContextFlag = "past_event"
)
