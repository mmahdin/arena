package background

import (
	"context"
	mem "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract/schema"
	"testing"
	"time"
)

type testPolicy struct{}

func (testPolicy) Allow(context.Context, mem.Permission) (bool, error) { return true, nil }

type testBundle struct{}

func (testBundle) Reference() c.BundleRef                           { return c.BundleRef{Version: "kb@0.1.0"} }
func (testBundle) Artifact(context.Context, string) ([]byte, error) { return nil, nil }
func setup(t *testing.T) (*Service, mem.Lease, Prepared, Output) {
	t.Helper()
	user, id := c.NewID(), c.NewID()
	now := time.Now().UTC()
	message := c.Message{ID: id, UserID: user, Role: c.RoleUser, Content: "I prefer fictional green tea", CreatedAt: now}
	p := Prepared{Input: Input{UserID: user, Request: Request{Kind: "memory_intelligence", Scope: Scope{Kind: "user", ThreadIDs: []c.ID{}}}, Messages: []c.Message{message}, Memories: []mem.Record{}}, Snapshot: mem.Snapshot{User: c.UserState{UserID: user, Locale: "en"}, Sources: map[c.ID]mem.Source{id: {Message: message, SessionID: c.NewID()}}, Suppressed: map[string]bool{}, Cooldowns: map[string]time.Time{}}}
	o := Output{Schema: OutputSchema, Operations: []Operation{{ID: "fact", Action: "add", Kind: c.MemPreference, Content: "Prefers fictional green tea", Evidence: []Evidence{{ID: id, Role: "supports"}}}}, Formulations: []Formulation{}, KnowledgeIDs: []string{}, InformationGaps: []string{}, Hints: []Hint{}}
	return &Service{Policy: testPolicy{}, Bundle: testBundle{}}, mem.Lease{ID: c.NewID(), UserID: user}, p, o
}
func TestRawEvidenceAndResponsibilityBoundaries(t *testing.T) {
	for _, change := range []string{"assistant", "absent", "hypothesis", "no_support"} {
		t.Run(change, func(t *testing.T) {
			s, l, p, o := setup(t)
			switch change {
			case "assistant":
				p.Input.Messages[0].Role = c.RoleAssistant
			case "absent":
				o.Operations[0].Evidence[0].ID = c.NewID()
			case "hypothesis":
				o.Operations[0].Kind = c.MemHypothesis
				o.Operations[0].Uncertainty = "tentative"
			case "no_support":
				o.Operations[0].Evidence[0].Role = "context"
			}
			plan, _, e := s.resolve(context.Background(), l, p, o, c.LLMUsage{Model: "synthetic"})
			if e != nil || len(plan.Changes) != 0 || len(plan.Report.Rejected) != 1 {
				t.Fatal(plan, e)
			}
		})
	}
}
func TestHypothesesNeverPromoteByRepetition(t *testing.T) {
	s, l, p, o := setup(t)
	p.Input.Request.Kind = "reflection"
	op := &o.Operations[0]
	op.Kind = c.MemHypothesis
	op.Content = "May prefer predictability in fictional settings"
	op.Uncertainty = "tentative and incomplete"
	op.Alternatives = []string{"May reflect a temporary situation"}
	plan, _, e := s.resolve(context.Background(), l, p, o, c.LLMUsage{Model: "synthetic"})
	if e != nil || len(plan.Changes) != 1 {
		t.Fatal(plan, e)
	}
	record := plan.Changes[0].After
	for i := 0; i < 5; i++ {
		p.Input.Memories = []mem.Record{record}
		op.Action = "confirm"
		op.TargetID = &record.Item.ID
		plan, _, e = s.resolve(context.Background(), l, p, o, c.LLMUsage{Model: "synthetic"})
		if e != nil || len(plan.Changes) != 1 {
			t.Fatal(plan, e)
		}
		record = plan.Changes[0].After
	}
	if record.Item.Epistemic != c.EpInferred || record.Item.Confidence != c.ConfLow || record.Item.SupportN != 1 || record.Item.Status != c.MemProposed {
		t.Fatal("system repetition promoted a hypothesis", record)
	}
}
func TestContradictionAndSupersessionPreserveHistory(t *testing.T) {
	for _, action := range []string{"contradict", "supersede", "revise", "invalidate"} {
		t.Run(action, func(t *testing.T) {
			s, l, p, o := setup(t)
			first, _, _ := s.resolve(context.Background(), l, p, o, c.LLMUsage{Model: "synthetic"})
			old := first.Changes[0].After
			p.Input.Memories = []mem.Record{old}
			o.Operations[0].Action = action
			o.Operations[0].TargetID = &old.Item.ID
			o.Operations[0].Content = "Now prefers fictional black tea"
			o.Operations[0].Evidence[0].Role = "contradicts"
			if action == "supersede" || action == "revise" {
				o.Operations[0].Evidence[0].Role = "supports"
			}
			plan, _, e := s.resolve(context.Background(), l, p, o, c.LLMUsage{Model: "synthetic"})
			if e != nil || len(plan.Report.Rejected) > 0 || len(plan.Changes) == 0 {
				t.Fatal(plan, e)
			}
			if action == "supersede" {
				if len(plan.Changes) != 2 || plan.Changes[1].After.Item.Content != old.Item.Content || plan.Changes[1].After.Item.Status != c.MemSuperseded || plan.Changes[0].After.Item.Supersedes == nil {
					t.Fatal(plan)
				}
			} else if plan.Changes[0].Before == nil || plan.Changes[0].Before.Item.Content != old.Item.Content {
				t.Fatal("original revision lost", plan)
			}
		})
	}
}
func TestUnknownTargetsAndDerivedEvidenceCannotEstablishConclusions(t *testing.T) {
	s, l, p, o := setup(t)
	id := c.NewID()
	o.Operations[0].Action = "confirm"
	o.Operations[0].TargetID = &id
	o.Operations[0].Evidence = []Evidence{{ID: id, Role: "supports"}}
	plan, _, e := s.resolve(context.Background(), l, p, o, c.LLMUsage{Model: "synthetic"})
	if e != nil || len(plan.Report.Rejected) != 1 || len(plan.Changes) > 0 {
		t.Fatal(plan, e)
	}
}
func TestScopeAndConfigBounds(t *testing.T) {
	if !DefaultConfig().Valid() || !(Scope{Kind: "user", ThreadIDs: []c.ID{}}).Valid() || (Scope{Kind: "threads"}).Valid() || (Scope{Kind: "user", ThreadIDs: []c.ID{c.NewID()}}).Valid() {
		t.Fatal("scope/config validation")
	}
	cfg := DefaultConfig()
	cfg.BatchSize = 100
	if cfg.Valid() {
		t.Fatal("unbounded batch accepted")
	}
}
func TestAuthoritativeOutputSchemaIsStrict(t *testing.T) {
	check, e := schema.CompileInline(schema.BackgroundOutputSchema())
	if e != nil {
		t.Fatal(e)
	}
	if check([]byte(`{"schema_version":"background_intelligence.v2","operations":[],"formulations":[],"knowledge_ids":[],"information_gaps":[],"future_hints":[]}`)) != nil {
		t.Fatal("valid empty review rejected")
	}
	if check([]byte(`{"schema_version":"background_intelligence.v2","operations":[],"formulations":[],"knowledge_ids":[],"information_gaps":[],"future_hints":[],"background_requests":[]}`)) == nil {
		t.Fatal("uncontrolled request chain accepted")
	}
}

func TestConsolidationKeepsEveryHistoricalTarget(t *testing.T) {
	s, l, p, o := setup(t)
	one, _, e := s.resolve(context.Background(), l, p, o, c.LLMUsage{Model: "synthetic"})
	if e != nil {
		t.Fatal(e)
	}
	first := one.Changes[0].After
	second := Clone(first)
	second.Item.ID = c.NewID()
	second.Item.Content = "Another fictional preference report"
	p.Input.Memories = []mem.Record{first, second}
	o.Operations[0].Action = "consolidate"
	o.Operations[0].TargetID = &first.Item.ID
	o.Operations[0].MergeIDs = []c.ID{second.Item.ID}
	o.Operations[0].Content = "Prefers the fictional examples in a shared setting"
	plan, _, e := s.resolve(context.Background(), l, p, o, c.LLMUsage{Model: "synthetic"})
	if e != nil || len(plan.Changes) != 3 || len(plan.Report.Rejected) > 0 {
		t.Fatal(plan, e)
	}
	for _, old := range plan.Changes[1:] {
		if old.Before == nil || old.After.Item.Content != old.Before.Item.Content || old.After.Item.Status != c.MemSuperseded || old.After.Item.SupersededBy == nil || *old.After.Item.SupersededBy != plan.Changes[0].After.Item.ID {
			t.Fatal("consolidation silently overwrote history", old)
		}
	}
}
func TestRoutineBudgetNeverDropsPendingPrimaryMessages(t *testing.T) {
	s, _, p, _ := setup(t)
	s.Config = DefaultConfig()
	s.Config.InputTokens = 1
	p.Input.Request.MessageIDs = []c.ID{p.Input.Messages[0].ID}
	if e := s.fit(&p, nil); e == nil || len(p.Input.Messages) != 1 {
		t.Fatal("budget exhaustion silently dropped coverage")
	}
}

func TestDuplicateClaimsInOneBatchCombinePrimaryEvidence(t *testing.T) {
	s, l, p, o := setup(t)
	other := p.Input.Messages[0]
	other.ID = c.NewID()
	p.Input.Messages = append(p.Input.Messages, other)
	p.Snapshot.Sources[other.ID] = mem.Source{Message: other, SessionID: c.NewID()}
	another := Clone(o.Operations[0])
	another.ID = "same_fact_second_message"
	another.Evidence[0].ID = other.ID
	o.Operations = append(o.Operations, another)
	plan, _, e := s.resolve(context.Background(), l, p, o, c.LLMUsage{Model: "synthetic"})
	if e != nil || len(plan.Changes) != 1 || len(plan.Changes[0].After.Item.Evidence) != 2 || plan.Changes[0].Before != nil {
		t.Fatal(plan, e)
	}
}

func TestIndependentIdenticalEventsPreservePrimaryLineage(t *testing.T) {
	s, l, p, o := setup(t)
	o.Operations[0].Kind = c.MemEvent
	other := p.Input.Messages[0]
	other.ID = c.NewID()
	p.Input.Messages = append(p.Input.Messages, other)
	p.Snapshot.Sources[other.ID] = mem.Source{Message: other, SessionID: c.NewID()}
	another := Clone(o.Operations[0])
	another.ID = "independent_event"
	another.Evidence[0].ID = other.ID
	o.Operations = append(o.Operations, another)
	plan, _, e := s.resolve(context.Background(), l, p, o, c.LLMUsage{Model: "synthetic"})
	if e != nil || len(plan.Changes) != 2 || plan.Changes[0].After.Item.ID == plan.Changes[1].After.Item.ID {
		t.Fatal("independent event accounts collapsed by wording", plan, e)
	}
}
