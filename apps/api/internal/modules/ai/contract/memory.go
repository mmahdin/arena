package contract

import (
	"time"
)

type MemoryItem struct {
	ID              ID              `json:"id"`
	UserID          ID              `json:"user_id"`
	Kind            MemoryKind      `json:"kind"`
	Content         string          `json:"content"`
	Lang            string          `json:"lang"`
	Epistemic       Epistemic       `json:"epistemic"`
	Status          MemoryStatus    `json:"status"`
	Confidence      ConfidenceClass `json:"confidence"`
	Evidence        []EvidenceRef   `json:"evidence"`
	SupportN        int             `json:"support_n"`
	ContradictN     int             `json:"contradict_n"`
	SessionsN       int             `json:"sessions_n"`
	ThreadIDs       []ID            `json:"thread_ids"`
	Entities        []EntityRef     `json:"entities"`
	Tags            []string        `json:"tags"`
	Sensitivity     Sensitivity     `json:"sensitivity"`
	Importance      float32         `json:"importance"`
	ValidFrom       *time.Time      `json:"valid_from,omitempty"`
	ValidTo         *time.Time      `json:"valid_to,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	LastConfirmedAt *time.Time      `json:"last_confirmed_at,omitempty"`
	LastUsedAt      *time.Time      `json:"last_used_at,omitempty"`
	Supersedes      *ID             `json:"supersedes,omitempty"`
	SupersededBy    *ID             `json:"superseded_by,omitempty"`
	ClusterID       *ID             `json:"cluster_id,omitempty"`
	Source          ProvenanceSrc   `json:"source"`
}

type EvidenceRef struct {
	Kind string    `json:"kind"` // message | item | event
	Ref  ID        `json:"ref"`
	Role string    `json:"role"` // supports | contradicts | context
	At   time.Time `json:"at"`
}

type EntityRef struct {
	Role  string  `json:"role"` // sister|boss|friend|partner|...
	Label *string `json:"label,omitempty"`
}

type ProvenanceSrc struct {
	PromptVersion string `json:"prompt_version"`
	Model         string `json:"model"`
	Origin        string `json:"origin"` // memory_writer|reflection|user_edit|onboarding|brain_slot
}

// MemoryOpProposal is the Memory Writer / Reflection output element (schema memwriter.v1).
type MemoryOpProposal struct { // LLM
	Op          MemoryOp    `json:"op"`
	TargetID    *ID         `json:"target_id,omitempty"` // required for update|confirm|contradict|supersede|reject
	Kind        MemoryKind  `json:"kind,omitempty"`      // required for add
	Content     string      `json:"content,omitempty"`   // <= 40 words; hypotheses use hedged wording
	Epistemic   Epistemic   `json:"epistemic,omitempty"`
	EvidenceMsg []ID        `json:"evidence_msg_ids"` // required, non-empty
	Entities    []EntityRef `json:"entities,omitempty"`
	Tags        []string    `json:"tags,omitempty"`
	Sensitivity Sensitivity `json:"sensitivity,omitempty"`
	Importance  *float32    `json:"importance,omitempty"` // 0..1
	ValidFrom   *string     `json:"valid_from,omitempty"` // ISO date: when it was true, not when said
	ReasonTag   string      `json:"reason_tag"`
}

type MemoryWriterOutput struct {
	SchemaVersion string             `json:"schema_version"` // memwriter.v1
	Ops           []MemoryOpProposal `json:"ops"`
}

type ReflectionResult struct { // LLM, schema reflection.v1
	SchemaVersion   string             `json:"schema_version"`
	ThreadID        ID                 `json:"thread_id"`
	Formulation     *FormulationDoc    `json:"formulation,omitempty"`
	HypothesisOps   []MemoryOpProposal `json:"hypothesis_ops"`
	Episode         *EpisodeDoc        `json:"episode,omitempty"`
	ThreadProposals []ThreadProposal   `json:"thread_proposals"`
	InformationGaps []string           `json:"information_gaps"`
}

type FormulationDoc struct {
	Framework string                      `json:"framework"` // cbt_conceptualization|five_p_lite|functional_analysis
	Fields    map[string]FormulationField `json:"fields"`    // keys from the framework schema
	Paragraph string                      `json:"paragraph"` // <= 90 words, used in WorkingContext
}

type FormulationField struct {
	Value             string    `json:"value"`
	Status            Epistemic `json:"status"` // reported | inferred | unknown
	Evidence          []ID      `json:"evidence"`
	NeedsEvidenceMore bool      `json:"needs_more_evidence"`
}

type EpisodeDoc struct {
	SessionID ID        `json:"session_id"`
	Summary   string    `json:"summary"` // <= 90 words
	Topics    []string  `json:"topics"`
	Emotions  []Emotion `json:"emotions"`
	Outcome   string    `json:"outcome"` // explored|exercise_done|decision_made|unresolved
	MsgRange  [2]ID     `json:"msg_range"`
	FollowUps []string  `json:"follow_ups"`
}

type ThreadProposal struct {
	ThreadID ID           `json:"thread_id"`
	Phase    *ThreadPhase `json:"phase,omitempty"`
	Op       string       `json:"op"` // merge | split | resolve | dormant
	Reason   string       `json:"reason"`
}
