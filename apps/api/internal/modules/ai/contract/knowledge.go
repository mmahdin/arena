package contract

import "time"

// Cards describe resources; only explicitly selected content is disclosed to Unified.
type KnowledgeCard struct {
	ID            string `json:"id" yaml:"id"`
	Category      string `json:"category" yaml:"category"`
	Name          string `json:"name" yaml:"name"`
	Description   string `json:"description" yaml:"description"`
	Purpose       string `json:"purpose" yaml:"purpose"`
	UseWhen       string `json:"use_when" yaml:"use_when"`
	Limitations   string `json:"limitations" yaml:"limitations"`
	Prerequisites string `json:"prerequisites" yaml:"prerequisites"`
	ContentID     string `json:"content_id" yaml:"content_id"`
}
type KnowledgeResource struct {
	ID        string `json:"id"`
	ContentID string `json:"content_id"`
	Content   string `json:"content"`
}
type SourceRevision struct {
	Kind    string `json:"kind"`
	ID      ID     `json:"id"`
	Version string `json:"version"`
}
type FutureHint struct {
	JobID          *ID              `json:"job_id,omitempty"`
	Origin         string           `json:"origin,omitempty"`
	EvidenceIDs    []ID             `json:"evidence_ids,omitempty"`
	Sources        []SourceRevision `json:"source_revisions,omitempty"`
	Topic          string           `json:"topic"`
	ResourceIDs    []string         `json:"resource_ids"`
	RemainingTurns int              `json:"remaining_turns"`
	ExpiresAt      time.Time        `json:"expires_at"`
	ThreadID       *ID              `json:"thread_id,omitempty"`
}
type KnowledgeSelection struct {
	Selected       []string            `json:"selected_resource_ids"`
	TopicContinues bool                `json:"topic_continues"`
	Degraded       bool                `json:"degraded"`
	Reason         string              `json:"reason,omitempty"`
	Resources      []KnowledgeResource `json:"resources"`
	Usage          LLMUsage            `json:"usage"`
}
