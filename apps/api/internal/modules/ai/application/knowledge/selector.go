// Package knowledge owns the pinned card catalog and the single JEV selection call.
// It never searches personal memory, changes state or generates a user reply.
package knowledge

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract/schema"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
	"gopkg.in/yaml.v3"
)

var ErrCatalog = errors.New("knowledge catalog invalid")

type Input struct {
	UserID  c.ID             `json:"user_id"`
	Context c.WorkingContext `json:"working_context"`
	Bundle  c.BundleRef      `json:"bundle"`
}
type Options struct {
	Verifier interface {
		VerifyContext(context.Context, c.ID, c.ID, c.WorkingContext) error
	}
	Timeout                          time.Duration
	MaxInputTokens, MaxContentTokens int
}
type Selector struct {
	client    ports.Client
	bundle    ports.Bundle
	cards     []c.KnowledgeCard
	resources map[string]c.KnowledgeResource
	system    string
	opt       Options
}

const selectionSchema = `{"type":"object","additionalProperties":false,"required":["schema_version","selected_resource_ids","topic_continues"],"properties":{"schema_version":{"const":"jev_selection.v1"},"selected_resource_ids":{"type":"array","maxItems":4,"uniqueItems":true,"items":{"type":"string","maxLength":100}},"topic_continues":{"type":"boolean"}}}`

func Load(ctx context.Context, b ports.Bundle) ([]c.KnowledgeCard, map[string]c.KnowledgeResource, error) {
	if b == nil {
		return nil, nil, ErrCatalog
	}
	raw, err := b.Artifact(ctx, "cards.yaml")
	if err != nil {
		return nil, nil, ErrCatalog
	}
	var catalog struct {
		Schema string            `yaml:"schema"`
		Cards  []c.KnowledgeCard `yaml:"cards"`
	}
	d := yaml.NewDecoder(bytes.NewReader(raw))
	d.KnownFields(true)
	if d.Decode(&catalog) != nil || d.Decode(new(any)) != io.EOF || catalog.Schema != "knowledge_cards.v1" || len(catalog.Cards) == 0 || len(catalog.Cards) > 128 {
		return nil, nil, ErrCatalog
	}
	resources := map[string]c.KnowledgeResource{}
	paths := map[string]bool{}
	for _, card := range catalog.Cards {
		if card.ID == "" || len(card.ID) > 100 || resources[card.ID].ID != "" || paths[card.ContentID] || !fs.ValidPath(card.ContentID) || strings.HasPrefix(card.ContentID, "policy/") || strings.HasPrefix(card.ContentID, "prompts/") || strings.HasPrefix(card.ContentID, "resources/crisis") {
			return nil, nil, ErrCatalog
		}
		if card.Category != "skill" && card.Category != "framework" && card.Category != "protocol" && card.Category != "exercise" && card.Category != "other" {
			return nil, nil, ErrCatalog
		}
		for _, v := range []string{card.Name, card.Description, card.Purpose, card.UseWhen, card.Limitations, card.Prerequisites} {
			if strings.TrimSpace(v) == "" || len(v) > 1200 {
				return nil, nil, ErrCatalog
			}
		}
		content, e := b.Artifact(ctx, card.ContentID)
		if e != nil || len(content) == 0 || len(content) > 32000 {
			return nil, nil, ErrCatalog
		}
		resources[card.ID] = c.KnowledgeResource{ID: card.ID, ContentID: card.ContentID, Content: string(content)}
		paths[card.ContentID] = true
	}
	// Every pre-existing psychological skill, exercise/protocol and framework
	// remains reachable. Cards cannot silently replace or discard knowledge.
	if source, ok := b.(interface{ ArtifactNames() []string }); ok {
		for _, path := range source.ArtifactNames() {
			if (strings.HasPrefix(path, "skills/") || strings.HasPrefix(path, "protocols/") || strings.HasPrefix(path, "schemas/formulation_")) && strings.HasSuffix(path, ".yaml") && !paths[path] {
				return nil, nil, ErrCatalog
			}
		}
	}
	sort.Slice(catalog.Cards, func(i, j int) bool { return catalog.Cards[i].ID < catalog.Cards[j].ID })
	return catalog.Cards, resources, nil
}
func New(ctx context.Context, client ports.Client, b ports.Bundle, opt Options) (*Selector, error) {
	if client == nil {
		return nil, ErrCatalog
	}
	cards, resources, err := Load(ctx, b)
	if err != nil {
		return nil, err
	}
	raw, err := b.Artifact(ctx, "prompts/jev.system.tmpl")
	if err != nil || len(raw) == 0 {
		return nil, ErrCatalog
	}
	if opt.Timeout <= 0 || opt.MaxInputTokens < 1 || opt.MaxContentTokens < 1 {
		return nil, ErrCatalog
	}
	return &Selector{client, b, cards, resources, string(raw), opt}, nil
}
func (s *Selector) Reference() c.BundleRef     { return s.bundle.Reference() }
func (s *Selector) Catalog() []c.KnowledgeCard { return append([]c.KnowledgeCard(nil), s.cards...) }
func (s *Selector) Select(ctx context.Context, in Input) (out c.KnowledgeSelection, err error) {
	ctx, span := trace.Start(ctx, "jev.Select", func() any { return map[string]any{"input": in, "cards": s.cards} })
	defer func() { span.Complete(func() any { return out }, err, recover()) }()
	if !c.ValidID(in.UserID) || !reflect.DeepEqual(in.Bundle, s.Reference()) {
		return out, ErrCatalog
	}
	if filter, ok := s.opt.Verifier.(interface {
		FilterHints(context.Context, c.ID, []c.FutureHint) []c.FutureHint
	}); ok {
		in.Context.FutureHints = filter.FilterHints(ctx, in.UserID, in.Context.FutureHints)
	}
	if s.opt.Verifier != nil {
		if err = s.opt.Verifier.VerifyContext(ctx, in.UserID, in.Context.Turn.TurnID, in.Context); err != nil {
			return out, err
		}
	} else if source, ok := s.bundle.(interface{ SyntheticOnly() bool }); !ok || !source.SyntheticOnly() {
		return out, ErrCatalog
	}
	stateContext := in.Context
	stateContext.FutureHints = nil // Carry-over is supplied once, alongside context.
	data, _ := json.Marshal(map[string]any{
		"selection_guidance": s.system,
		"working_context":    stateContext,
		"carry_over":         in.Context.FutureHints,
		"knowledge_cards":    s.cards,
	})
	out.Resources = []c.KnowledgeResource{}
	out.Selected = []string{}
	degrade := func(reason string) (c.KnowledgeSelection, error) {
		out.Degraded = true
		out.Reason = reason
		// Recommendations survive a transient JEV outage only for their bounded
		// lifetime and matching thread. Core Policies never depend on this path.
		for _, hint := range in.Context.FutureHints {
			out.Selected = append(out.Selected, hint.ResourceIDs...)
		}
		out.Selected, out.Resources = s.resolve(out.Selected, in.Context.Protocol)
		return out, nil
	}
	decision := &ports.SystemOneRequest{State: string(data), Questions: map[string]ports.SystemOneQuestion{}}
	decision.Questions["topic_continues"] = ports.SystemOneQuestion{Type: "noul", Instructions: "Does the current user turn continue the preceding conversation topic? Answer no if the topic changed or there is no preceding topic. Use selection_guidance."}
	for _, card := range s.cards {
		decision.Questions["resource:"+card.ID] = ports.SystemOneQuestion{Type: "noul", Instructions: "Would resource " + strconv.Quote(card.ID) + " in knowledge_cards be useful for the current turn, given its purpose, prerequisites and limitations? Use selection_guidance."}
	}
	wireInput, _ := json.Marshal(decision)
	if (len(wireInput)+3)/4 > s.opt.MaxInputTokens {
		return degrade("jev_input_budget")
	}
	work, cancel := context.WithTimeout(ctx, s.opt.Timeout)
	defer cancel()
	if trace.Enabled(ctx) {
		trace.Capture(ctx, "rendered_prompt", "rendered_prompt", "jev.systemone.v1", decision)
		rendered := trace.ArtifactRef(ctx, "rendered_prompt")
		trace.Capture(ctx, "selection_instructions", "prompt_template", "prompts/jev.system.tmpl", s.system)
		instructions := trace.ArtifactRef(ctx, "selection_instructions")
		cardSources := map[string]trace.Reference{}
		for i, card := range s.cards {
			cardSources["resource:"+card.ID] = trace.Ref(ctx, "/input/cards/"+strconv.Itoa(i))
		}
		state := trace.Compose("decision_state", "state", trace.Ref(ctx, rendered.Path+"/state"), decision.State, func(string, map[string]string) []trace.Reference {
			return []trace.Reference{instructions, trace.Ref(ctx, "/input/input/working_context"), trace.Ref(ctx, "/input/cards")}
		})
		state.Source = "shared selection guidance, Working Context, carry-over and Knowledge Cards"
		parts := []trace.PromptPart{state}
		for _, name := range sortedQuestionNames(decision.Questions) {
			q := decision.Questions[name]
			key := strings.ReplaceAll(strings.ReplaceAll(name, "~", "~0"), "/", "~1")
			part := trace.Compose(name, "instructions", trace.Ref(ctx, rendered.Path+"/questions/"+key+"/instructions"), q.Instructions, func(string, map[string]string) []trace.Reference {
				if card, ok := cardSources[name]; ok {
					return []trace.Reference{card}
				}
				return nil // The continuity question is fixed code-owned text.
			})
			part.Source = "JEV question instructions (code)"
			parts = append(parts, part)
		}
		trace.Capture(ctx, "prompt_composition", "prompt_composition", "jev.systemone.v1", parts)
	}
	response, e := s.client.Structured(work, ports.StructuredRequest{Tier: ports.ComponentJEV, SystemOne: decision, Schema: []byte(selectionSchema), SchemaName: "jev_selection.v1", Timeout: s.opt.Timeout, Meta: c.CallMeta{UserID: in.UserID, TurnID: in.Context.Turn.TurnID, Bundle: in.Bundle}, IdempotencyKey: in.Context.Turn.TurnID.String() + ":jev.systemone.v1"})
	out.Usage = response.Usage
	if ctx.Err() != nil {
		return out, ctx.Err()
	}
	if e != nil {
		return degrade("jev_unavailable")
	}
	validate, e := schema.CompileInline([]byte(selectionSchema))
	if e != nil || validate(response.JSON) != nil {
		return degrade("jev_invalid_output")
	}
	var result struct {
		Schema    string   `json:"schema_version"`
		IDs       []string `json:"selected_resource_ids"`
		Continues bool     `json:"topic_continues"`
	}
	if json.Unmarshal(response.JSON, &result) != nil {
		return degrade("jev_invalid_output")
	}
	for _, id := range result.IDs {
		if _, ok := s.resources[id]; !ok {
			return degrade("jev_unknown_resource")
		}
	}
	out.TopicContinues = result.Continues
	out.Selected, out.Resources = s.resolve(result.IDs, in.Context.Protocol)
	trace.Capture(ctx, "selected_knowledge", "knowledge_selection", "jev", out)
	return out, nil
}
func (s *Selector) resolve(ids []string, active *c.ProtocolView) ([]string, []c.KnowledgeResource) {
	// Operational graph content for an already active exercise must remain
	// available even when selection degrades. This never starts an exercise.
	if active != nil {
		for _, card := range s.cards {
			if card.ContentID == "protocols/"+active.ProtocolID+".yaml" {
				ids = append([]string{card.ID}, ids...)
			}
		}
	}
	selected := []string{}
	resources := []c.KnowledgeResource{}
	seen := map[string]bool{}
	tokens := 0
	for _, id := range ids {
		r, ok := s.resources[id]
		if !ok || seen[id] || len(selected) >= 4 {
			continue
		}
		cost := (len(r.Content) + 3) / 4
		if tokens+cost > s.opt.MaxContentTokens {
			continue
		}
		seen[id] = true
		tokens += cost
		selected = append(selected, id)
		resources = append(resources, r)
	}
	return selected, resources
}

// Decay uses a short bounded lifetime, not a manually tuned relevance score.
func Decay(hints []c.FutureHint, thread *c.ID, now time.Time) []c.FutureHint {
	out := []c.FutureHint{}
	for _, hint := range hints {
		if hint.RemainingTurns <= 0 || !hint.ExpiresAt.After(now) || hint.ThreadID != nil && (thread == nil || *hint.ThreadID != *thread) {
			continue
		}
		hint.RemainingTurns--
		out = append(out, hint)
		if len(out) >= 6 {
			break
		}
	}
	return out
}

func sortedQuestionNames(questions map[string]ports.SystemOneQuestion) []string {
	names := make([]string, 0, len(questions))
	for name := range questions {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
