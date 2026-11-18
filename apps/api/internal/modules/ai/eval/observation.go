package eval

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

// Observation is the explicit, typed boundary for L2-L5 measurements. It may
// contain synthetic test text in memory only; NEVER serialize it into reports.
// Nil component pointers indicate missing evidence, not successful execution.
type Observation struct {
	UserID                    string                `json:"user_id"`
	TurnID                    string                `json:"turn_id"`
	Route                     string                `json:"route"`
	Safety                    *c.SafetySignal       `json:"safety,omitempty"`
	Retrieval                 []Retrieved           `json:"retrieval,omitempty"`
	RetrievalObserved         bool                  `json:"retrieval_observed"`
	RetrievalStatusesObserved bool                  `json:"retrieval_statuses_observed"`
	Brain                     *c.BrainDecision      `json:"brain,omitempty"`
	Generator                 *Generation           `json:"generator,omitempty"`
	Validator                 *c.ValidationReport   `json:"validator,omitempty"`
	State                     *StateObserved        `json:"state,omitempty"`
	Async                     *AsyncObserved        `json:"async,omitempty"`
	Degraded                  *bool                 `json:"degraded,omitempty"`
	Judge                     *JudgeObserved        `json:"judge,omitempty"`
	Longitudinal              *LongitudinalObserved `json:"longitudinal,omitempty"`
	Trace                     *TraceDetails         `json:"trace,omitempty"`
}
type Retrieved struct {
	ID     string `json:"id"`
	UserID string `json:"user_id"`
	Status string `json:"status"`
}
type Generation struct {
	Text     string `json:"text"`
	Aborted  bool   `json:"aborted"`
	Sentinel string `json:"sentinel"`
	Attempts int    `json:"attempts"`
}
type StateObserved struct {
	Session        *SessionObserved `json:"session,omitempty"`
	Thread         *ThreadObserved  `json:"thread,omitempty"`
	Risk           *RiskObserved    `json:"risk,omitempty"`
	Memory         []Retrieved      `json:"memory,omitempty"`
	MemoryObserved bool             `json:"memory_observed"`
}
type SessionObserved struct {
	LastObjective      string `json:"last_objective"`
	PendingQuestionSet bool   `json:"pending_question_set"`
}
type ThreadObserved struct {
	SlotsUpdated []string `json:"slots_updated"`
	Phase        string   `json:"phase"`
}
type RiskObserved struct {
	Level string `json:"level"`
}
type AsyncObserved struct {
	MemoryOps []MemoryOpExpect `json:"memory_ops"`
}

// VerifiedCalibrated is supplied by an authenticated, independent review
// adapter; the fixture loader explicitly strips this claim. Scores in plain
// recordings are diagnostic only and cannot satisfy quality gates.
type JudgeObserved struct {
	Scores             map[string]float64 `json:"scores"`
	CalibrationID      string             `json:"-"`
	VerifiedCalibrated bool               `json:"-"`
}
type LongitudinalObserved struct {
	FactRecall         map[string]bool `json:"fact_recall"`
	CorrectionsApplied map[string]bool `json:"corrections_applied"`
	StaleHypotheses    map[string]bool `json:"stale_hypotheses"`
}
type TraceDetails struct {
	Bundle  string            `json:"bundle"`
	Models  map[string]string `json:"models"`
	Prompts map[string]string `json:"prompts"`
}

// The evaluation files are not trace storage. Evidence must be synthetic and
// explicitly owner-scoped, even when a fixture claims to be clinician-labelled.
type Recorded struct {
	Schema        string        `json:"schema"`
	SyntheticOnly bool          `json:"synthetic_only"`
	ScenarioID    string        `json:"scenario_id"`
	UserID        string        `json:"user_id"`
	Turns         []Observation `json:"turns"`
}

func readJSON(data []byte, v any) error {
	if len(data) > 4<<20 {
		return ErrFixture
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if d.Decode(v) != nil {
		return ErrFixture
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return ErrFixture
	}
	return nil
}
func LoadRecorded(path string) (Recorded, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Recorded{}, ErrFixture
	}
	var r Recorded
	if readJSON(data, &r) != nil || r.Schema != "eval_observations.v1" || !r.SyntheticOnly || !identifier.MatchString(r.ScenarioID) || uuid.Validate(r.UserID) != nil || len(r.Turns) == 0 {
		return Recorded{}, ErrFixture
	}
	for _, o := range r.Turns {
		if o.UserID != r.UserID || uuid.Validate(o.TurnID) != nil {
			return Recorded{}, ErrFixture
		}
	}
	return r, nil
}

// Environment lasts for one run and carries earlier results to subsequent
// turns/sessions. The evaluator never stores or emits this state to disk.
type Environment struct {
	Scenario       Scenario
	UserID         string
	Clock          time.Time
	Past           []Observation
	CurrentSession int
	LastState      *StateObserved
}

func newEnvironment(s Scenario) (*Environment, error) {
	id := s.Setup.User.ID
	if id == "" {
		id = uuid.NewSHA1(uuid.NameSpaceOID, []byte("theia-wp15:"+s.ID)).String()
	}
	if uuid.Validate(id) != nil {
		return nil, ErrFixture
	}
	e := &Environment{Scenario: s, UserID: id}
	if s.Setup.Clock != "" {
		e.Clock, _ = time.Parse(time.RFC3339, s.Setup.Clock)
	}
	return e, nil
}

// Executor is a synthetic isolated adapter. A live-model implementation must
// assemble the same WP-01-12 ports on synthetic state and return their actual
// typed outputs; WP-15 does not alter runtime composition.
type Executor interface {
	Execute(context.Context, *Environment, SequenceTurn) (Observation, error)
}
type ExecuteFunc func(context.Context, *Environment, SequenceTurn) (Observation, error)

func (f ExecuteFunc) Execute(ctx context.Context, e *Environment, t SequenceTurn) (Observation, error) {
	return f(ctx, e, t)
}

// RecordedExecutor is a deterministic fixture driver. Its outputs are NOT
// clinical labels or evidence of live-model quality.
type RecordedExecutor struct {
	mu       sync.Mutex
	recorded Recorded
}

func NewRecordedExecutor(r Recorded) *RecordedExecutor { return &RecordedExecutor{recorded: r} }
func (r *RecordedExecutor) Execute(ctx context.Context, e *Environment, t SequenceTurn) (Observation, error) {
	if ctx.Err() != nil {
		return Observation{}, ctx.Err()
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if e.Scenario.ID != r.recorded.ScenarioID || e.UserID != r.recorded.UserID || t.Index >= len(r.recorded.Turns) {
		return Observation{}, ErrExecution
	}
	return r.recorded.Turns[t.Index], nil
}
