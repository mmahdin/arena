// Package background implements bounded Memory Intelligence and longitudinal
// Reflection on the existing queues. It never participates in synchronous inference.
package background

import (
	"context"
	"errors"
	mem "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"time"
)

var ErrPaused = errors.New("background intelligence paused")

const TaskSchema = "intelligence_job.v2"
const OutputSchema = "background_intelligence.v2"

type Config struct {
	BatchSize    int `yaml:"batch_size" json:"batch_size"`
	MinMessages  int `yaml:"min_messages" json:"min_messages"`
	FlushSeconds int `yaml:"flush_seconds" json:"flush_seconds"`
	SweepSeconds int `yaml:"sweep_seconds" json:"sweep_seconds"`
	MaxUsers     int `yaml:"max_users" json:"max_users"`
	InputTokens  int `yaml:"input_tokens" json:"input_tokens"`
	OutputTokens int `yaml:"output_tokens" json:"output_tokens"`
}

func DefaultConfig() Config { return Config{16, 4, 300, 30, 32, 16000, 3000} }
func (v Config) Valid() bool {
	return v.BatchSize > 0 && v.MinMessages > 0 && v.MinMessages <= v.BatchSize && v.FlushSeconds > 0 && v.SweepSeconds > 0 && v.MaxUsers > 0 && v.InputTokens > 0 && v.OutputTokens > 0
}

type Scope struct {
	Kind      string `json:"kind"`
	ThreadIDs []c.ID `json:"thread_ids"`
}

func (s Scope) Valid() bool {
	if s.Kind == "user" {
		return len(s.ThreadIDs) == 0
	}
	if s.Kind != "threads" || len(s.ThreadIDs) < 1 || len(s.ThreadIDs) > 4 {
		return false
	}
	seen := map[c.ID]bool{}
	for _, id := range s.ThreadIDs {
		if !c.ValidID(id) || seen[id] {
			return false
		}
		seen[id] = true
	}
	return true
}

type Task struct {
	Schema      string      `json:"schema"`
	Bundle      c.BundleRef `json:"bundle"`
	OperationID c.ID        `json:"operation_id"`
	Data        []byte      `json:"data_enc"`
}
type Request struct {
	Origin     string   `json:"origin"`
	Kind       string   `json:"kind"`
	Scope      Scope    `json:"scope"`
	Objective  string   `json:"objective"`
	Topic      string   `json:"topic"`
	Queries    []string `json:"queries"`
	MessageIDs []c.ID   `json:"message_ids"`
	Trigger    c.ID     `json:"trigger_message_id"`
}
type Batch struct {
	UserID     c.ID
	TurnID     c.ID
	Trigger    c.ID
	MessageIDs []c.ID
	Versions   []string
}
type Thread struct {
	ID                 c.ID   `json:"id"`
	Version            int64  `json:"version"`
	FormulationVersion int    `json:"formulation_version"`
	Topic              string `json:"topic"`
}
type Input struct {
	Schema    string                `json:"schema_version"`
	UserID    c.ID                  `json:"user_id"`
	Request   Request               `json:"request"`
	Messages  []c.Message           `json:"raw_evidence"`
	Memories  []mem.Record          `json:"structured_memory"`
	Records   []c.MemoryView        `json:"derived_and_operational_context"`
	Threads   []Thread              `json:"threads"`
	Cards     []c.KnowledgeCard     `json:"knowledge_cards"`
	Knowledge []c.KnowledgeResource `json:"knowledge"`
	Retrieval []c.RetrievalMeta     `json:"retrieval"`
}
type Prepared struct {
	Input    Input          `json:"input"`
	Snapshot mem.Snapshot   `json:"snapshot"`
	Sources  []c.MemoryView `json:"source_revisions"`
}
type Evidence struct {
	ID   c.ID   `json:"id"`
	Role string `json:"role"`
}
type Operation struct {
	Status       c.MemoryStatus `json:"status,omitempty"`
	ID           string         `json:"id"`
	Action       string         `json:"action"`
	TargetID     *c.ID          `json:"target_id,omitempty"`
	MergeIDs     []c.ID         `json:"merge_ids,omitempty"`
	Kind         c.MemoryKind   `json:"kind,omitempty"`
	Content      string         `json:"content,omitempty"`
	Evidence     []Evidence     `json:"evidence"`
	ContextIDs   []c.ID         `json:"context_ids,omitempty"`
	Entities     []c.EntityRef  `json:"entities,omitempty"`
	Tags         []string       `json:"tags,omitempty"`
	Sensitivity  c.Sensitivity  `json:"sensitivity,omitempty"`
	ValidFrom    *time.Time     `json:"valid_from,omitempty"`
	ValidTo      *time.Time     `json:"valid_to,omitempty"`
	Uncertainty  string         `json:"uncertainty,omitempty"`
	Alternatives []string       `json:"alternatives,omitempty"`
}
type Hint struct {
	MemoryIDs   []c.ID   `json:"memory_ids,omitempty"`
	Topic       string   `json:"topic"`
	ResourceIDs []string `json:"resource_ids"`
	EvidenceIDs []c.ID   `json:"evidence_ids"`
	TTLTurns    int      `json:"ttl_turns"`
}
type Formulation struct {
	ThreadID *c.ID            `json:"thread_id,omitempty"`
	Doc      c.FormulationDoc `json:"document"`
}
type Output struct {
	Schema          string        `json:"schema_version"`
	Operations      []Operation   `json:"operations"`
	Formulations    []Formulation `json:"formulations"`
	KnowledgeIDs    []string      `json:"knowledge_ids"`
	InformationGaps []string      `json:"information_gaps"`
	Hints           []Hint        `json:"future_hints"`
}
type Checkpoint struct {
	Request  Request    `json:"request"`
	Prepared Prepared   `json:"prepared"`
	Output   Output     `json:"output"`
	Usage    c.LLMUsage `json:"usage"`
}
type Repository interface {
	Candidates(context.Context, Config) ([]Batch, error)
	Reserve(context.Context, Batch, c.AsyncJobProposal) error
	Prepare(context.Context, mem.Lease, Request, []c.MemoryView) (Prepared, error)
	Verify(context.Context, mem.Lease, Prepared) error
	Commit(context.Context, mem.Lease, Prepared, Output, c.LLMUsage, mem.Plan, []Hint) error
}
type FrameworkField struct {
	ID       string        `yaml:"id"`
	Allowed  []c.Epistemic `yaml:"allowed_status"`
	Evidence int           `yaml:"min_evidence"`
	Sessions int           `yaml:"min_sessions"`
}
type Framework struct {
	Name      string           `yaml:"framework"`
	Fields    []FrameworkField `yaml:"fields"`
	Paragraph struct {
		MaxWords int `yaml:"max_words"`
	} `yaml:"paragraph"`
}
type Service struct {
	Frameworks map[string]Framework
	Repo       Repository
	Queue      mem.Queue
	Cipher     mem.Cipher
	Client     ports.Client
	Bundle     ports.Bundle
	Research   ports.ResearchPreparer
	Policy     mem.Policy
	Config     Config
	Cards      []c.KnowledgeCard
	Knowledge  []c.KnowledgeResource
	prompt     string
}
