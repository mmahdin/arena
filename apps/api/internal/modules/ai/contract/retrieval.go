package contract

import (
	"time"
)

type RetrievalRequest struct {
	UserID             ID                  `json:"user_id"`
	Turn               TurnView            `json:"turn"`
	FocusThread        *ID                 `json:"focus_thread,omitempty"`
	Strategies         []RetrievalStrategy `json:"strategies"`          // empty = retriever selects by rules (§8.5)
	Extra              *NeedsContext       `json:"extra,omitempty"`     // set on second hop
	Exclude            []ID                `json:"exclude"`             // ids already in context
	MaxItems           int                 `json:"max_items"`           // default 5
	IncludeSensitivity []Sensitivity       `json:"include_sensitivity"` // default [normal, sensitive]
}

type RetrievalStrategy string

const (
	StratTemporal RetrievalStrategy = "temporal_exact"
	StratThread   RetrievalStrategy = "thread"
	StratHybrid   RetrievalStrategy = "hybrid"
	StratAnaphora RetrievalStrategy = "anaphora"
	StratEntity   RetrievalStrategy = "entity"
)

type NeedsContext struct { // LLM (Brain) -> triggers second hop
	Kind  ContextKind   `json:"kind"`  // entity | topic | time_window | item
	Query string        `json:"query"` // <= 12 words
	Why   ContextReason `json:"why"`   // reference_unresolved | missing_background | verify_fact
}

type RetrievalResult struct {
	Items []MemoryView  `json:"items"`
	Meta  RetrievalMeta `json:"meta"`
	Took  time.Duration `json:"took"`
}
