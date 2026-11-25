package contract

import (
	"time"
)

type Budget struct {
	TotalDeadline    time.Time     `json:"total_deadline"` // hard wall for the whole turn
	BrainTimeout     time.Duration `json:"brain_timeout"`
	GenFirstToken    time.Duration `json:"gen_first_token"`
	RetrievalTimeout time.Duration `json:"retrieval_timeout"`
	SafetyTimeout    time.Duration `json:"safety_timeout"`
	MaxTokensIn      int           `json:"max_tokens_in"`
	MaxTokensOut     int           `json:"max_tokens_out"`
	MaxLLMCalls      int           `json:"max_llm_calls"` // sync calls allowed (default 3: brain, brain-retry/hop, generator)
	MaxCostMicroUSD  int64         `json:"max_cost_micro_usd"`
}

type CallMeta struct {
	TurnID     ID         `json:"turn_id"`
	UserID     ID         `json:"user_id"`
	Bundle     BundleRef  `json:"bundle"`
	Route      Route      `json:"route"`
	TraceFlags TraceFlags `json:"trace_flags"`
}

type BundleRef struct {
	Version    string            `json:"version"`     // "kb@1.4.2"
	PromptSHAs map[string]string `json:"prompt_shas"` // by prompt name
}
