// Package contract holds the typed, provider-neutral in-process contracts of the
// AI Core as specified by Theia AI Core Contract Pack v1.2 (§3, §4).
//
// This file is the WP-09 (Retrieval) subset: it contains exactly the types the
// Retrieval Service consumes and produces plus the storage records its SQL Server
// adapter returns. WP-01 owns the complete contract package; every type below is
// copied verbatim (names, JSON tags, enum values) from §4.3/§4.4 so the WP-01
// superset can absorb it without renaming or wire-format changes.
//
// The additional storage record types (MemoryRecord, MessageRecord,
// EmbeddingRef) are retrieval read-model types, not LLM/wire contracts; they
// mirror the columns of §7.2 `ai_memory_items`, `ai_memory_evidence`,
// `dbo.messages` and `ai_embeddings`.
package contract

import (
	"time"

	"github.com/google/uuid"
)

// ID is the existing backend UUID contract (§3.1). No prefixes, no ULIDs.
type ID = uuid.UUID

func NewID() ID { return uuid.New() }

// ValidID reports whether id is usable as a persisted identifier.
func ValidID(id ID) bool { return id != uuid.Nil }

// Epistemic is the epistemic status of a stored claim (§3.2).
// EpUnknown is only valid for Thread slots, never for a stored item.
type Epistemic string

const (
	EpReported       Epistemic = "reported"
	EpInferred       Epistemic = "inferred"
	EpSystemObserved Epistemic = "system_observed"
	EpUnknown        Epistemic = "unknown"
)

// MemoryKind enumerates the physical memory item kinds (§3.2).
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

// MemoryStatus enumerates item lifecycle statuses (§3.2, §7.5).
type MemoryStatus string

const (
	MemActive     MemoryStatus = "active"
	MemContested  MemoryStatus = "contested"
	MemSuperseded MemoryStatus = "superseded"
	MemRejected   MemoryStatus = "rejected"
	MemStale      MemoryStatus = "stale"
	MemProposed   MemoryStatus = "proposed"
	MemSupported  MemoryStatus = "supported"
)

// Sensitivity is the privacy classification of a stored item (§3.2).
type Sensitivity string

const (
	SensNormal    Sensitivity = "normal"
	SensSensitive Sensitivity = "sensitive"
	SensHighly    Sensitivity = "highly_sensitive"
)

// Role is a message author role (§3.2).
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
)

// TurnOrigin distinguishes a user turn from a scheduler wake-up (§4.1).
type TurnOrigin string

const (
	OriginUser   TurnOrigin = "user"
	OriginWakeUp TurnOrigin = "wakeup"
)

// RetrievalStrategy identifies one retrieval method (§4.4).
type RetrievalStrategy string

const (
	StratTemporal RetrievalStrategy = "temporal_exact"
	StratThread   RetrievalStrategy = "thread"
	StratHybrid   RetrievalStrategy = "hybrid"
	StratAnaphora RetrievalStrategy = "anaphora"
	StratEntity   RetrievalStrategy = "entity"
)

// RefKindMemory / RefKindMessage are the ai_embeddings.ref_kind values (§7.2).
const (
	RefKindMemory  = "memory"
	RefKindMessage = "message"
)

// TurnView is the retrieval-relevant projection of the current turn (§4.3).
type TurnView struct {
	TurnID    ID `json:"turn_id"`
	MessageID ID `json:"message_id"`
	// ConversationID is not part of the §4.3 TurnView wire shape; it is carried
	// locally so retrieval can scope the novelty window to one conversation
	// without a second request type.
	ConversationID ID            `json:"-"`
	Text           string        `json:"text"`
	Lang           string        `json:"lang"`
	LocalTime      string        `json:"local_time"`
	SinceLastMsg   time.Duration `json:"since_last_msg"`
	Origin         TurnOrigin    `json:"origin"`
	ReplyToPending bool          `json:"reply_to_pending"`
	TemporalRefs   []TemporalRef `json:"temporal_refs,omitempty"`
	WakeUpReason   string        `json:"wake_up_reason,omitempty"`
}

// TemporalRef is a time window resolved from the message by the caller
// (Gateway/Assembler) or, defensively, by the retrieval temporal parser (§4.3).
type TemporalRef struct {
	Phrase     string    `json:"phrase"`
	From       time.Time `json:"from"`
	To         time.Time `json:"to"`
	RecallVerb bool      `json:"recall_verb"`
}

// MsgView is a recent/history message projection (§4.3).
type MsgView struct {
	ID        ID      `json:"id"`
	Role      Role    `json:"role"`
	Text      string  `json:"text"`
	Age       string  `json:"age"`
	Objective *string `json:"objective,omitempty"`
}

// RetrievalRequest is the consumer-owned retrieval input (§4.4).
type RetrievalRequest struct {
	UserID             ID
	Turn               TurnView
	FocusThread        *ID
	Strategies         []RetrievalStrategy // empty = retriever selects by rules (§8.5)
	Extra              *NeedsContext       // set on second hop only
	Exclude            []ID                // ids already in context
	MaxItems           int                 // default 5
	IncludeSensitivity []Sensitivity       // default [normal, sensitive]
}

// NeedsContext is the Brain's bounded request for one extra retrieval hop (§4.4).
type NeedsContext struct {
	Kind  string `json:"kind"`  // entity | topic | time_window | item
	Query string `json:"query"` // <= 12 words
	Why   string `json:"why"`   // reference_unresolved | missing_background | verify_fact
}

// MemoryView is a single retrieved context item (§4.3).
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

// RetrievalMeta reports how retrieval behaved so the Brain cannot imply recall
// that did not happen (§4.3).
type RetrievalMeta struct {
	Strategies         []string `json:"strategies"`
	AmbiguousReference bool     `json:"ambiguous_reference"`
	Degraded           bool     `json:"degraded"`
	Hops               int      `json:"hops"`
	ExactMatchFound    *bool    `json:"exact_match_found,omitempty"`
}

// RetrievalResult is the complete retrieval output (§4.4).
type RetrievalResult struct {
	Items []MemoryView  `json:"items"`
	Meta  RetrievalMeta `json:"meta"`
	Took  time.Duration `json:"took"`
}

// MemoryRecord is the retrieval read model of dbo.ai_memory_items.
// It is never sent to an LLM; the service projects it into a MemoryView.
type MemoryRecord struct {
	ID        ID
	UserID    ID
	Kind      MemoryKind
	Epistemic Epistemic
	Status    MemoryStatus
	Content   string
	Lang      string

	Sensitivity Sensitivity
	Importance  float32

	// EvidenceN is the number of ai_memory_evidence rows (provenance invariant,
	// §2.5 #3). Items with zero evidence are never retrieved.
	EvidenceN int

	ThreadIDs []ID
	Entities  []string
	Tags      []string

	ClusterID *ID

	ValidFrom *time.Time
	ValidTo   *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time

	// DeletedAt implements the soft-delete policy: a row is only ever flagged,
	// never removed. Retrieval excludes flagged rows.
	DeletedAt *time.Time
}

// MessageRecord is the retrieval read model of the owner-scoped dbo.messages
// join used by exact/temporal recall.
type MessageRecord struct {
	ID             ID
	UserID         ID
	ConversationID ID
	TurnID         ID
	Role           Role
	Content        string
	CreatedAt      time.Time
	DeletedAt      *time.Time
}

// EmbeddingRef identifies one stored vector owner row.
type EmbeddingRef struct {
	Kind string // memory | message
	ID   ID
}

// EmbeddingKey is the map key returned by the embedding store port.
type EmbeddingKey struct {
	Kind string
	ID   ID
}

// Vector is a provider-neutral embedding.
//
// ModelID and Dim travel with the values because §7.3 requires that dimension
// and model identity are always validated before cosine ranking: vectors from
// different models are never compared.
type Vector struct {
	ModelID string
	Dim     int
	Values  []float32
}
