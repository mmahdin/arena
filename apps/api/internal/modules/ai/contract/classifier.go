package contract

type SafetyClassifierOutput struct {
	Level        RiskLevel           `json:"level"`
	Categories   []RiskCategory      `json:"categories"`
	Confidence   float64             `json:"confidence"`
	ContextFlags []SafetyContextFlag `json:"context_flags,omitempty"`
}

const (
	BrainSchemaVersion          = "brain.v1"
	MemoryWriterSchemaVersion   = "memwriter.v1"
	ReflectionSchemaVersion     = "reflection.v1"
	SafetySchemaVersion         = "safety.v1"
	WorkingContextSchemaVersion = "wctx.v1"
	TurnTraceSchemaVersion      = "turn_trace.v1"
)
