// Package eval is the offline, synthetic-only WP-15 evaluation harness.
// It never reads production messages, performs SQL writes, or logs model text.
package eval

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

var (
	ErrFixture   = errors.New("invalid synthetic evaluation fixture")
	ErrReplay    = errors.New("incomplete or unauthorized evaluation replay")
	ErrExecution = errors.New("evaluation execution failed")
	identifier   = regexp.MustCompile(`^[a-z][a-z0-9_\-]{2,79}$`)
)

// Scenario mirrors Contract Pack v1.2 §12.2. Strict YAML decoding rejects
// unknown assertion keys instead of silently weakening release gates.
// All inputs must be explicitly synthetic; clinician metadata is NEVER proof
// of human approval.
type Scenario struct {
	Schema        string      `yaml:"schema"`
	ID            string      `yaml:"id"`
	Title         string      `yaml:"title"`
	Tags          []string    `yaml:"tags"`
	SyntheticOnly bool        `yaml:"synthetic_only"`
	Clinician     *Clinician  `yaml:"clinician"`
	Setup         Setup       `yaml:"setup"`
	Turn          *Turn       `yaml:"turn"`
	Turns         []Turn      `yaml:"turns"`
	Sessions      []Session   `yaml:"sessions"`
	GroundTruth   GroundTruth `yaml:"ground_truth"`
}
type Clinician struct {
	Author          string `yaml:"author"`
	ReviewedOn      string `yaml:"reviewed_on"`
	LabelsAgreement string `yaml:"labels_agreement"`
}
type Setup struct {
	User    SyntheticUser      `yaml:"user"`
	Threads []SyntheticThread  `yaml:"threads"`
	Memory  []SyntheticMemory  `yaml:"memory"`
	History []SyntheticHistory `yaml:"history"`
	Clock   string             `yaml:"clock"`
}
type SyntheticUser struct {
	ID     string            `yaml:"id"`
	Locale string            `yaml:"locale"`
	TZ     string            `yaml:"tz"`
	Prefs  map[string]string `yaml:"prefs"`
}
type SyntheticThread struct {
	ID    string                   `yaml:"id"`
	Topic string                   `yaml:"topic"`
	Phase string                   `yaml:"phase"`
	Slots map[string]SyntheticSlot `yaml:"slots"`
}
type SyntheticSlot struct {
	Status string `yaml:"status"`
	Value  string `yaml:"value"`
}
type SyntheticMemory struct {
	ID        string `yaml:"id"`
	Kind      string `yaml:"kind"`
	Epistemic string `yaml:"epistemic"`
	Status    string `yaml:"status"`
	Text      string `yaml:"text"`
	AgeDays   int    `yaml:"age_days"`
	Thread    string `yaml:"thread"`
	Support   int    `yaml:"support"`
}
type SyntheticHistory struct {
	Role      string `yaml:"role"`
	Text      string `yaml:"text"`
	Objective string `yaml:"objective"`
	AgeMin    int    `yaml:"age_min"`
}
type Turn struct {
	Text   string `yaml:"text"`
	Expect Expect `yaml:"expect"`
}
type Session struct {
	Gap   string `yaml:"gap"`
	Turns []Turn `yaml:"turns"`
}
type GroundTruth struct {
	ProfileFacts          map[string]string `yaml:"profile_facts"`
	TruePatterns          []string          `yaml:"true_patterns"`
	PlantedContradictions []string          `yaml:"planted_contradictions"`
	Corrections           []string          `yaml:"corrections"`
}
type Expect struct {
	Route      string          `yaml:"route"`
	Safety     SafetyExpect    `yaml:"safety"`
	Retrieval  RetrievalExpect `yaml:"retrieval"`
	Brain      BrainExpect     `yaml:"brain"`
	Output     OutputExpect    `yaml:"output"`
	Validator  ValidatorExpect `yaml:"validator"`
	StateAfter StateExpect     `yaml:"state_after"`
	AsyncAfter AsyncExpect     `yaml:"async_after"`
	Degraded   *bool           `yaml:"degraded"`
	// L4 ground truth compares reference keys/statuses, NEVER user prose.
	Longitudinal LongitudinalExpect `yaml:"longitudinal"`
}
type SafetyExpect struct {
	LevelLTE          string   `yaml:"level_lte"`
	LevelGTE          string   `yaml:"level_gte"`
	CategoriesInclude []string `yaml:"categories_include"`
	Degraded          *bool    `yaml:"degraded"`
}
type RetrievalExpect struct {
	MustInclude         []string `yaml:"must_include"`
	MustExcludeStatuses []string `yaml:"must_exclude_statuses"`
}
type BrainExpect struct {
	ObjectiveIn    []string `yaml:"objective_in"`
	ObjectiveNotIn []string `yaml:"objective_not_in"`
	Question       struct {
		Allowed *bool `yaml:"allowed"`
	} `yaml:"question"`
	MustAvoidIncludes []string `yaml:"must_avoid_includes"`
	ThreadRelation    string   `yaml:"thread_relation"`
	PastRefs          struct {
		ResolvedToIncludes []string `yaml:"resolved_to_includes"`
	} `yaml:"past_refs"`
	Hypotheses struct {
		AllowedOps   []string `yaml:"allowed_ops"`
		ForbiddenOps []string `yaml:"forbidden_ops"`
	} `yaml:"hypotheses"`
	Protocol struct {
		ActionIn []string `yaml:"action_in"`
	} `yaml:"protocol"`
}
type OutputExpect struct {
	MaxQuestions *int              `yaml:"max_questions"`
	MustNot      []string          `yaml:"must_not"`
	Judge        map[string]string `yaml:"judge"`
}
type ValidatorExpect struct {
	Passed         *bool    `yaml:"passed"`
	Action         string   `yaml:"action"`
	ViolationCodes []string `yaml:"violation_codes"`
}
type StateExpect struct {
	Session struct {
		LastObjective      string `yaml:"last_objective"`
		PendingQuestionSet *bool  `yaml:"pending_question_set"`
	} `yaml:"session"`
	Thread struct {
		SlotsUpdated []string `yaml:"slots_updated"`
		Phase        string   `yaml:"phase"`
	} `yaml:"thread"`
	Risk struct {
		Level string `yaml:"level"`
	} `yaml:"risk"`
	Memory struct {
		StatusIs       map[string]string `yaml:"status_is"`
		MustInclude    []string          `yaml:"must_include"`
		MustNotInclude []string          `yaml:"must_not_include"`
	} `yaml:"memory"`
}
type AsyncExpect struct {
	MemoryOpsInclude []MemoryOpExpect `yaml:"memory_ops_include"`
}
type MemoryOpExpect struct {
	Op     string `yaml:"op"`
	Target string `yaml:"target"`
}
type LongitudinalExpect struct {
	FactRecall        []string `yaml:"fact_recall"`
	CorrectionApplied []string `yaml:"correction_applied"`
	NoStaleHypotheses []string `yaml:"no_stale_hypotheses"`
}

const maxFixtureBytes = 1 << 20

func LoadFile(path string) (Scenario, error) {
	data, err := os.ReadFile(path)
	if err != nil || len(data) > maxFixtureBytes {
		return Scenario{}, ErrFixture
	}
	return Load(data)
}
func Load(data []byte) (Scenario, error) {
	var s Scenario
	if len(data) > maxFixtureBytes {
		return s, ErrFixture
	}
	d := yaml.NewDecoder(bytes.NewReader(data))
	d.KnownFields(true)
	if d.Decode(&s) != nil {
		return Scenario{}, ErrFixture
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return Scenario{}, ErrFixture
	}
	if s.Schema != "golden.v1" || !identifier.MatchString(s.ID) || !s.SyntheticOnly {
		return Scenario{}, ErrFixture
	}
	for _, tag := range s.Tags {
		if !identifier.MatchString(tag) {
			return Scenario{}, ErrFixture
		}
	}
	for _, item := range s.Setup.Memory {
		if item.AgeDays < 0 {
			return Scenario{}, ErrFixture
		}
	}
	if s.Setup.Clock != "" {
		if _, err := time.Parse(time.RFC3339, s.Setup.Clock); err != nil {
			return Scenario{}, ErrFixture
		}
	}
	n := 0
	if s.Turn != nil {
		n++
	}
	if len(s.Turns) > 0 {
		n++
	}
	if len(s.Sessions) > 0 {
		n++
	}
	if n != 1 {
		return Scenario{}, ErrFixture
	}
	for _, item := range s.Sequence() {
		if strings.TrimSpace(item.Turn.Text) == "" || !validExpect(item.Turn.Expect) {
			return Scenario{}, ErrFixture
		}
	}
	for _, session := range s.Sessions {
		if len(session.Turns) == 0 {
			return Scenario{}, ErrFixture
		}
		if session.Gap != "" && !validGap(session.Gap) {
			return Scenario{}, ErrFixture
		}
	}
	return s, nil
}
func validGap(gap string) bool {
	if strings.HasSuffix(gap, "d") {
		var days int
		if _, err := fmt.Sscanf(gap, "%dd", &days); err == nil && days > 0 && days <= 3650 && fmt.Sprintf("%dd", days) == gap {
			return true
		}
	}
	d, err := time.ParseDuration(gap)
	return err == nil && d > 0
}

// Sequence retains session boundaries; the executor receives a single mutable
// synthetic environment across every turn and an explicit new-session marker.
type SequenceTurn struct {
	Turn       Turn
	Session    int
	Index      int
	NewSession bool
	Gap        string
}

func (s Scenario) Sequence() []SequenceTurn {
	out := []SequenceTurn{}
	add := func(t Turn, session int, newSession bool, gap string) {
		out = append(out, SequenceTurn{Turn: t, Session: session, Index: len(out), NewSession: newSession, Gap: gap})
	}
	if s.Turn != nil {
		add(*s.Turn, 0, true, "")
	}
	for i, t := range s.Turns {
		add(t, 0, i == 0, "")
	}
	for j, session := range s.Sessions {
		for i, t := range session.Turns {
			add(t, j, i == 0, session.Gap)
		}
	}
	return out
}
func validExpect(e Expect) bool {
	if e.Route != "" && e.Route != "normal" && e.Route != "fast" && e.Route != "crisis" && e.Route != "proactive" {
		return false
	}
	for _, level := range []string{e.Safety.LevelLTE, e.Safety.LevelGTE, e.StateAfter.Risk.Level} {
		if level != "" && rank(level) < 0 {
			return false
		}
	}
	if e.Output.MaxQuestions != nil && (*e.Output.MaxQuestions < 0 || *e.Output.MaxQuestions > 10) {
		return false
	}
	for _, rule := range e.Output.MustNot {
		if _, ok := negativeRules[rule]; !ok {
			return false
		}
	}
	for name, expr := range e.Output.Judge {
		if !identifier.MatchString(name) || !validThreshold(expr) {
			return false
		}
	}
	for _, x := range e.AsyncAfter.MemoryOpsInclude {
		if x.Op == "" || x.Target == "" {
			return false
		}
	}
	return true
}
func Files(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	paths := []string{}
	for _, ent := range entries {
		if ent.Type().IsRegular() && (strings.HasSuffix(ent.Name(), ".yaml") || strings.HasSuffix(ent.Name(), ".yml")) {
			paths = append(paths, filepath.Join(dir, ent.Name()))
		}
	}
	return paths, nil // os.ReadDir sorts lexically
}
