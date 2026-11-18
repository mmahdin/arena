// Package trace records bounded, metadata-only execution observations for the
// WP-05 skeleton. SQL Server persists the durable TurnTraceSummary as part of
// TurnFinalizer.Complete. Detailed encrypted clinical traces are a later WP.
package trace

import (
	"context"
	"errors"
	"regexp"
	"sync"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

// Observation is the ONLY accepted Record payload; free-form strings, prompts,
// user messages, exceptions and structured model responses are discarded.
type Observation struct {
	Duration time.Duration
	Count    int
	Degraded bool
	RuleID   string
}

type Step struct {
	Name       string
	RuleID     string
	DurationMS int64
	Count      int
	Degraded   bool
	ErrorCode  contract.Code
}
type Snapshot struct {
	TurnID   contract.ID
	Route    contract.Route
	Steps    []Step
	Usage    contract.LLMUsage
	Finished bool
}
type Recorder struct {
	mu    sync.Mutex
	turns map[contract.ID]Snapshot
}

func NewRecorder() *Recorder { return &Recorder{turns: map[contract.ID]Snapshot{}} }

var _ ports.Tracer = (*Recorder)(nil)

func (r *Recorder) Begin(turnID contract.ID, _ contract.CallMeta) ports.TurnTrace {
	r.mu.Lock()
	r.turns[turnID] = Snapshot{TurnID: turnID}
	r.mu.Unlock()
	return &turnTrace{owner: r, id: turnID}
}
func (r *Recorder) Snapshot(id contract.ID) (Snapshot, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.turns[id]
	s.Steps = append([]Step(nil), s.Steps...)
	return s, ok
}

type turnTrace struct {
	owner *Recorder
	id    contract.ID
}

var _ ports.TurnTrace = (*turnTrace)(nil)

func (t *turnTrace) update(fn func(*Snapshot)) {
	t.owner.mu.Lock()
	defer t.owner.mu.Unlock()
	s, ok := t.owner.turns[t.id]
	if !ok || s.Finished {
		return
	}
	fn(&s)
	t.owner.turns[t.id] = s
}
func validStep(name string) bool {
	switch name {
	case "policy_gate", "retrieval_hop", "state_validate", "admit", "recent", "safety", "retrieval", "assembler", "route", "brain", "generator", "validator", "fallback", "finalizer", "recovery", "cancel":
		return true
	default:
		return false
	}
}

var brainRule = regexp.MustCompile(`^(BR-(00[1-9]|01[0-9]|02[0-4])|FB-00[1-6]|SF-00[1-9]|ST-007|PR-004)$`)

var outputRule = regexp.MustCompile(`^OV-(00[1-9]|01[0-7]|02[01]|030|031)$`)

func approvedRule(rule string) bool {
	if outputRule.MatchString(rule) || brainRule.MatchString(rule) {
		return true
	}
	switch rule {
	case "RT-001", "RT-002", "RT-003", "RT-004", "RT-099":
		return true
	default:
		return false
	}
}
func (t *turnTrace) Record(step string, v any) {
	o, ok := v.(Observation)
	if !validStep(step) || !ok {
		return
	}
	if o.Duration < 0 {
		o.Duration = 0
	}
	if o.Count < 0 {
		o.Count = 0
	}
	if !approvedRule(o.RuleID) || (step != "route" && step != "validator" && step != "fallback") {
		o.RuleID = ""
	}
	t.update(func(s *Snapshot) {
		if len(s.Steps) < 40 {
			s.Steps = append(s.Steps, Step{Name: step, RuleID: o.RuleID, DurationMS: o.Duration.Milliseconds(), Count: o.Count, Degraded: o.Degraded})
		}
	})
}
func (t *turnTrace) SetRoute(route contract.Route) {
	if route.Valid() {
		t.update(func(s *Snapshot) { s.Route = route })
	}
}
func (t *turnTrace) AddUsage(u contract.LLMUsage) {
	// Discard provider/model identifiers and free-form finish reasons: metrics
	// use bounded numeric counters only (and never text or user identifiers).
	t.update(func(s *Snapshot) {
		s.Usage.InTokens += u.InTokens
		s.Usage.OutTokens += u.OutTokens
		s.Usage.CachedTokens += u.CachedTokens
		s.Usage.CostMicroUSD += u.CostMicroUSD
		s.Usage.Attempts += u.Attempts
		s.Usage.Latency += u.Latency
	})
}
func (t *turnTrace) Fail(step string, err error) {
	if !validStep(step) {
		return
	}
	code := contract.ErrInternal
	var ce *contract.Error
	if errors.As(err, &ce) && ce.Code.Valid() {
		code = ce.Code
	}
	t.update(func(s *Snapshot) {
		if len(s.Steps) < 40 {
			s.Steps = append(s.Steps, Step{Name: step, ErrorCode: code, Degraded: true})
		}
	})
}
func (t *turnTrace) Finish(context.Context) {
	t.update(func(s *Snapshot) { s.Finished = true })
}
