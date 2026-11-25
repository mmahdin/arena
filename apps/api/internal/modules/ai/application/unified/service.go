// Package unified owns the single synchronous conversational inference. It has
// no database writer and never invokes classifiers, query models or generators.
package unified

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/knowledge"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/bundle"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract/schema"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
	"gopkg.in/yaml.v3"
)

var ErrInvalid = errors.New("unified turn invalid")

type Verifier interface {
	VerifyContext(context.Context, c.ID, c.ID, c.WorkingContext) error
}
type Options struct {
	AllowSyntheticForTests                            bool
	Timeout                                           time.Duration
	MaxInputTokens, MaxOutputTokens, MaxContentTokens int
}
type Service struct {
	client    ports.Client
	bundle    ports.Bundle
	schemas   *schema.Set
	system    []ports.Block
	verify    Verifier
	resources map[string]c.KnowledgeResource
	opt       Options
}

func New(ctx context.Context, client ports.Client, b ports.Bundle, v Verifier, opt Options) (*Service, error) {
	if client == nil || b == nil || v == nil && !opt.AllowSyntheticForTests {
		return nil, ErrInvalid
	}
	source, ok := b.(interface {
		SyntheticOnly() bool
		ArtifactNames() []string
	})
	if !ok || source.SyntheticOnly() && !opt.AllowSyntheticForTests {
		return nil, ErrInvalid
	}
	s := &Service{client: client, bundle: b, verify: v, opt: opt}
	if s.opt.Timeout <= 0 || s.opt.MaxInputTokens < 1 || s.opt.MaxOutputTokens < 1 || s.opt.MaxContentTokens < 1 {
		return nil, ErrInvalid
	}
	read := func(path string, out any) error {
		raw, e := b.Artifact(ctx, path)
		if e != nil {
			return e
		}
		return yaml.Unmarshal(raw, out)
	}
	vocab := schema.Vocabulary{Protocols: []string{}}
	for _, field := range []struct {
		name string
		to   *[]string
	}{{"emotions", &vocab.Emotions}, {"rationale_tags", &vocab.RationaleTags}, {"avoid_tags", &vocab.AvoidTags}, {"hypothesis_tags", &vocab.HypothesisTags}} {
		var tags bundle.TagVocabulary
		if read("config/vocab/"+field.name+".yaml", &tags) != nil {
			return nil, ErrInvalid
		}
		*field.to = tags.Values
	}
	var problem bundle.ProblemUnderstanding
	if read("schemas/problem_understanding.yaml", &problem) != nil {
		return nil, ErrInvalid
	}
	for _, slot := range problem.Slots {
		vocab.Slots = append(vocab.Slots, slot.ID)
	}
	names := source.ArtifactNames()
	sort.Strings(names)
	for _, name := range names {
		if strings.HasPrefix(name, "skills/") && strings.HasSuffix(name, ".yaml") {
			var skill bundle.Skill
			if read(name, &skill) != nil {
				return nil, ErrInvalid
			}
			vocab.Skills = append(vocab.Skills, skill.ID)

		}
		if strings.HasPrefix(name, "protocols/") && strings.HasSuffix(name, ".yaml") {
			var p bundle.Protocol
			if read(name, &p) != nil {
				return nil, ErrInvalid
			}
			vocab.Protocols = append(vocab.Protocols, p.ID)
		}
	}
	var err error
	_, s.resources, err = knowledge.Load(ctx, b)
	if err != nil {
		return nil, err
	}
	s.schemas, err = schema.New(vocab)
	if err != nil {
		return nil, err
	}
	for _, name := range []string{"prompts/unified.system.tmpl", "policy/communication_policy.md", "policy/boundary_policy.yaml", "policy/safety_policy.yaml", "resources/crisis_resources.yaml"} {
		raw, e := b.Artifact(ctx, name)
		if e != nil || strings.TrimSpace(string(raw)) == "" {
			return nil, ErrInvalid
		}
		s.system = append(s.system, ports.Block{Name: name, Text: string(raw), Cache: true})
	}
	return s, nil
}
func (s *Service) Reference() c.BundleRef { return s.bundle.Reference() }
func (s *Service) Infer(ctx context.Context, in ports.UnifiedInput) (out ports.UnifiedOutput, err error) {
	ctx, span := trace.Start(ctx, "unified.Infer", func() any { return in })
	defer func() { span.Complete(func() any { return out }, err, recover()) }()
	if !c.ValidID(in.UserID) || !c.ValidID(in.Context.Turn.TurnID) || !reflect.DeepEqual(in.Bundle, s.Reference()) {
		return out, &c.Error{Code: c.ErrBundle, Step: "unified"}
	}
	if filter, ok := s.verify.(interface {
		FilterHints(context.Context, c.ID, []c.FutureHint) []c.FutureHint
	}); ok {
		in.Context.FutureHints = filter.FilterHints(ctx, in.UserID, in.Context.FutureHints)
	}
	if s.verify != nil {
		if err = s.verify.VerifyContext(ctx, in.UserID, in.Context.Turn.TurnID, in.Context); err != nil {
			return out, err
		}
	}
	system := append([]ports.Block{}, s.system...)
	seenResources := map[string]bool{}
	if len(in.Knowledge) > 4 {
		return out, ErrInvalid
	}
	contentTokens := 0
	for _, resource := range in.Knowledge {
		actual, ok := s.resources[resource.ID]
		if !ok || seenResources[resource.ID] || actual != resource {
			return out, ErrInvalid
		}
		seenResources[resource.ID] = true
		contentTokens += (len(resource.Content) + 3) / 4
		system = append(system, ports.Block{Name: resource.ContentID, Text: resource.Content, Cache: true})
	}
	if contentTokens > s.opt.MaxContentTokens {
		return out, &c.Error{Code: c.ErrBudget, Step: "selected_knowledge"}
	}
	raw, err := json.Marshal(in.Context)
	if err != nil {
		return out, ErrInvalid
	}
	resolved, _ := s.schemas.JSON(c.UnifiedSchemaVersion)
	tokens := (len(raw) + len(resolved) + 3) / 4
	for _, b := range system {
		tokens += (len(b.Text) + 3) / 4
	}
	if tokens > s.opt.MaxInputTokens {
		return out, &c.Error{Code: c.ErrBudget, Step: "unified_input"}
	}
	request := ports.StructuredRequest{Tier: ports.ComponentUnified, System: system, Messages: []ports.ChatMessage{{Role: c.RoleUser, Content: string(raw)}}, Schema: resolved, SchemaName: c.UnifiedSchemaVersion, MaxTokens: s.opt.MaxOutputTokens, Timeout: s.opt.Timeout, Meta: c.CallMeta{UserID: in.UserID, TurnID: in.Context.Turn.TurnID, Bundle: in.Bundle}, IdempotencyKey: in.Context.Turn.TurnID.String() + ":unified.v1"}
	trace.Capture(ctx, "rendered_prompt", "rendered_prompt", "unified", map[string]any{"system": system, "context": string(raw)})
	if trace.Enabled(ctx) {
		rendered := trace.ArtifactRef(ctx, "rendered_prompt")
		contextRef := trace.Ref(ctx, rendered.Path+"/context")
		parts := []trace.PromptPart{trace.Compose("working_context", "user", contextRef, string(raw), func(string, map[string]string) []trace.Reference {
			return []trace.Reference{trace.Ref(ctx, "/input/Context")}
		})}
		for i, block := range system {
			ref := trace.Ref(ctx, rendered.Path+"/system/"+strconv.Itoa(i)+"/Text")
			var source []trace.Reference
			if i >= len(s.system) {
				source = []trace.Reference{trace.Ref(ctx, "/input/knowledge/"+strconv.Itoa(i-len(s.system))+"/content")}
			}
			part := trace.Compose(block.Name, "system", ref, block.Text, func(string, map[string]string) []trace.Reference { return source })
			part.Source = block.Name
			parts = append(parts, part)
		}
		trace.Capture(ctx, "prompt_composition", "prompt_composition", "unified", parts)
		trace.Bind(ctx, trace.Ref(ctx, "/input/Context"), contextRef, "transform")
	}
	response, err := s.client.Structured(ctx, request)
	trace.Bind(ctx, trace.Ref(ctx, trace.ArtifactRef(ctx, "rendered_prompt").Path+"/context"), trace.ChildRef(ctx, "llm/client.Structured", "/input/req/Messages/0/Content"), "copy")
	out.Usage = response.Usage
	if err != nil {
		return out, err
	}
	out.Turn, err = s.schemas.DecodeUnified(response.JSON)
	if err != nil {
		return out, &c.Error{Code: c.ErrValidation, Step: "unified_schema", Cause: err}
	}
	if err = ValidateReply(out.Turn, in.Context); err != nil {
		return ports.UnifiedOutput{Usage: out.Usage}, &c.Error{Code: c.ErrValidation, Step: "unified_references", Cause: err}
	}
	return out, nil
}

// ValidateReferences is structural/provenance validation, never a conversational
// safety classifier. Orchestration validates optional proposals independently
// and enforces reply dependencies before atomic finalization.
func ValidateReferences(u c.UnifiedTurn, w c.WorkingContext) error {
	if u.SchemaVersion != c.UnifiedSchemaVersion || strings.TrimSpace(u.Reply) == "" || len(utf16.Encode([]rune(u.Reply))) > 20000 || !u.Conversation.Objective.Valid() || !u.Conversation.Intent.Valid() {
		return ErrInvalid
	}
	threads := map[c.ID]bool{}
	known := map[c.ID]bool{w.Turn.MessageID: true, w.Session.SessionID: true}
	evidence := map[c.ID]bool{w.Turn.MessageID: true}
	for _, m := range w.Recent {
		known[m.ID] = true
		if m.Role == c.RoleUser {
			evidence[m.ID] = true
		}
	}
	if w.Session.Summary != "" && c.ValidID(w.Session.SessionID) {
		known[w.Session.SessionID] = true
	}
	for _, m := range append(append([]c.MemoryView{}, w.Persistent.Items...), w.Retrieved...) {
		known[m.ID] = true
		if m.Source == "thread" {
			threads[m.ID] = true
		}
		if m.Source == "message" && m.Epistemic == c.EpReported {
			evidence[m.ID] = true
		}
	}
	if w.FocusThread != nil {
		threads[w.FocusThread.ID] = true
		known[w.FocusThread.ID] = true
		for _, h := range w.FocusThread.Hypotheses {
			known[h.ID] = true
		}
	}
	for _, t := range w.ActiveThreads {
		threads[t.ID] = true
		known[t.ID] = true
		for _, h := range t.Hypotheses {
			known[h.ID] = true
		}
	}
	for _, t := range w.OtherThreads {
		threads[t.ID] = true
		known[t.ID] = true
	}
	for _, f := range w.Formulations {
		if c.ValidID(f.ID) {
			known[f.ID] = true
		}
	}
	if w.Formulation != nil && c.ValidID(w.Formulation.ID) {
		known[w.Formulation.ID] = true
	}
	if w.Protocol != nil {
		known[w.Protocol.InstanceID] = true
	}
	for _, loop := range w.OpenLoops {
		known[loop.ID] = true
	}
	for _, id := range u.UsedContext {
		if !known[id] {
			return ErrInvalid
		}
	}
	if t := u.Thread; t != nil {
		if !t.Relation.Valid() {
			return ErrInvalid
		}
		if t.Relation == c.RelNew {
			if t.Topic == nil || len(strings.Fields(*t.Topic)) == 0 || len(strings.Fields(*t.Topic)) > 8 || t.Target != nil {
				return ErrInvalid
			}
		} else if t.Target != nil && !threads[*t.Target] {
			return ErrInvalid
		}
		if t.Relation != c.RelNew && t.Topic != nil {
			return ErrInvalid
		}
		if t.Relation == c.RelContinues && t.Target == nil && w.FocusThread == nil && (len(t.Slots) > 0 || len(t.Hypotheses) > 0 || t.Goal != nil || t.Phase != nil) {
			return ErrInvalid
		}
		if (t.Relation == c.RelShifts || t.Relation == c.RelReturns) && t.Target == nil {
			return ErrInvalid
		}
		for _, slot := range t.Slots {
			if len(slot.Evidence) == 0 {
				return ErrInvalid
			}
			for _, id := range slot.Evidence {
				if !evidence[id] {
					return ErrInvalid
				}
			}
		}
		for _, h := range t.Hypotheses {
			if h.ID != nil && !known[*h.ID] {
				return ErrInvalid
			}
			for _, id := range h.Evidence {
				if !evidence[id] {
					return ErrInvalid
				}
			}
		}
		for _, id := range t.Related {
			if !threads[id] {
				return ErrInvalid
			}
		}
		for _, op := range t.Operations {
			if !threads[op.ThreadID] {
				return ErrInvalid
			}
		}
	}
	for _, loop := range u.OpenLoops {
		if loop.Op != "add" && (loop.ID == nil || !known[*loop.ID]) {
			return ErrInvalid
		}
		if loop.Ref != nil && !known[*loop.Ref] {
			return ErrInvalid
		}
	}
	return nil
}

func ValidateReply(u c.UnifiedTurn, w c.WorkingContext) error {
	v := u
	v.Session = nil
	v.Thread = nil
	v.Protocol = nil
	v.OpenLoops = nil
	v.Preferences = nil
	v.RiskRead = nil
	v.MemorySignals = nil
	return ValidateReferences(v, w)
}
