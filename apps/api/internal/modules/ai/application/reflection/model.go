package reflection

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
	"reflect"
	"sort"
	"strings"
	"time"

	mem "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

type Model struct {
	client ports.Client
	assets *Assets
	repo   Repository
	policy mem.Policy
}

func NewModel(client ports.Client, a *Assets, r Repository, p mem.Policy) (*Model, error) {
	if client == nil || a == nil || r == nil || p == nil {
		return nil, mem.ErrAssets
	}
	return &Model{client, a, r, p}, nil
}
func escaped(s string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}
func block(name string, v any) string {
	raw, _ := json.Marshal(v)
	return "<" + name + ">" + escaped(string(raw)) + "</" + name + ">\n"
}

// Render implements section 11.5; selected messages are already diversity-sampled
// by the owner-scoped reader. IDs appear only as opaque attributes.
func (m *Model) render(in ports.ReflectionInput, s Snapshot) ([]ports.Block, string, error) {
	f, ok := m.assets.frameworks[in.Framework]
	if !ok && in.Framework != "" || len(in.Evidence) > 48 || len(in.Items) > 24 {
		return nil, "", mem.ErrDenied
	}
	lex, ok := m.assets.lexicon(s.User.Locale)
	if !ok {
		return nil, "", mem.ErrAssets
	}
	sys := []ports.Block{{Name: "role_and_task", Cache: true, Text: "<role_and_task>" + m.assets.prompt + "</role_and_task>"}, {Name: "framework", Cache: true, Text: block("framework", f)}, {Name: "hypothesis_rules", Cache: true, Text: block("hypothesis_rules", struct {
		Thresholds        any
		Hedges, Forbidden []string
	}{m.assets.thresholds.Hypothesis, lex.Hedges, lex.ClinicalLabels})}}
	// Do not serialize owner IDs or evidence handles as ordinary prose.
	thread := "<thread id=\"" + in.ThreadID.String() + "\">" + block("topic", s.Thread.Topic) + block("phase", s.Thread.Phase)
	keys := []string{}
	for k := range s.Thread.Slots {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := s.Thread.Slots[k]
		thread += "<slot name=\"" + escaped(k) + "\" epistemic=\"" + string(v.Status) + "\">" + escaped(v.Value)
		for _, id := range v.Evidence {
			thread += "<ref id=\"" + id.String() + "\"/>"
		}
		thread += "</slot>"
	}
	for _, it := range in.Items {
		thread += "<memory id=\"" + it.ID.String() + "\" kind=\"" + string(it.Kind) + "\" epistemic=\"" + string(it.Epistemic) + "\" status=\"" + string(it.Status) + "\" age=\"prior\">" + escaped(it.Content) + block("counts", []int{it.SupportN, it.ContradictN, it.SessionsN}) + "</memory>"
	}
	if in.PrevFormulation != nil {
		thread += "<previous>"
		keys := []string{}
		for k := range in.PrevFormulation.Fields {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			field := in.PrevFormulation.Fields[k]
			thread += "<field name=\"" + escaped(k) + "\" epistemic=\"" + string(field.Status) + "\">" + escaped(field.Value)
			for _, id := range field.Evidence {
				thread += "<ref id=\"" + id.String() + "\"/>"
			}
			thread += "</field>"
		}
		thread += "</previous>"
	}
	thread += "</thread>\n"
	evidence := "<evidence>\n"
	loc, e := time.LoadLocation(s.User.TZ)
	if e != nil {
		return nil, "", mem.ErrDenied
	}
	for _, msg := range in.Evidence {
		if msg.UserID != in.UserID || msg.SessionID == nil {
			return nil, "", mem.ErrDenied
		}
		evidence += "<message id=\"" + msg.ID.String() + "\" session=\"" + msg.SessionID.String() + "\" role=\"" + string(msg.Role) + "\" local_date=\"" + msg.CreatedAt.In(loc).Format("2006-01-02") + "\">" + escaped(msg.Content) + "</message>\n"
	}
	evidence += "</evidence>\n"
	tags := []string{}
	for tag := range s.Cooldowns {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	dynamic := []string{thread, evidence, block("cooldowns", tags), "<task thread_id=\"" + in.ThreadID.String() + "\" session_id=\"" + s.Session.ID.String() + "\">" + block("trigger", in.Trigger) + block("formulation_allowed", sufficient(m.assets, s)) + "</task>"}
	if in.Research != nil {
		extra := block("background_research", in.Research)
		dynamic = append(dynamic, extra)
	}
	return sys, strings.Join(dynamic, ""), nil
}
func (m *Model) Reflect(ctx context.Context, in ports.ReflectionInput) (out c.ReflectionResult, usage c.LLMUsage, err error) {
	ctx, inspectionSpan := trace.Start(ctx, "application/reflection/model.Reflect", func() any { return map[string]any{"in": in} })
	defer func() {
		inspectionSpan.Complete(func() any { return map[string]any{"out": out, "usage": usage} }, err, recover())
	}()

	meta, ok := ctx.Value(metaKey{}).(call)
	if !ok || in.UserID != meta.Task.UserID || in.ThreadID != meta.Task.ThreadID || !reflect.DeepEqual(in.Bundle, m.assets.Reference()) || !authorize(ctx, m.policy, meta.Task, meta.Snapshot) {
		return out, usage, mem.ErrDenied
	}
	if err = m.repo.Verify(ctx, meta.Task, meta.Snapshot); err != nil {
		return out, usage, err
	}
	if len(in.Evidence) != len(meta.Snapshot.Sources) || !reflect.DeepEqual(in.Items, meta.Snapshot.Items) || !reflect.DeepEqual(in.PrevFormulation, meta.Snapshot.Previous) {
		return out, usage, mem.ErrDenied
	}
	for i, msg := range in.Evidence {
		if !reflect.DeepEqual(msg, meta.Snapshot.Sources[i].Message) {
			return out, usage, mem.ErrDenied
		}
	}
	if in.Research != nil {
		verifier, ok := m.repo.(interface {
			VerifyResearch(context.Context, c.ID, c.ID, ports.BackgroundResearch) error
		})
		if !ok {
			return out, usage, mem.ErrDenied
		}
		var trigger c.ID
		for _, source := range meta.Snapshot.Sources {
			if source.Message.TurnID == meta.Task.TurnID && source.Message.Role == c.RoleUser {
				trigger = source.Message.ID
			}
		}
		if err = verifier.VerifyResearch(ctx, in.UserID, trigger, *in.Research); err != nil {
			return out, usage, err
		}
	}
	trace.Capture(ctx, "template", "prompt_template", "prompts/reflection.system.tmpl", m.assets.prompt)
	sys, text, err := m.render(in, meta.Snapshot)
	if trace.Enabled(ctx) {
		trace.Capture(ctx, "rendered_prompt", "rendered_prompt", "reflection.render", map[string]any{"system": sys, "dynamic": text})
	}
	if err != nil {
		return out, usage, err
	}
	raw, _ := m.assets.schemas.JSON(c.ReflectionSchemaVersion)
	n := estimate(text)
	for _, block := range sys {
		n += estimate(block.Text)
	}
	if n > m.assets.budgets.Tokens.ReflectionIn {
		return out, usage, &c.Error{Code: c.ErrBudget, Step: "reflection"}
	}
	maxTokens := 0 // The configured component client supplies the output budget.
	if budget, ok := c.BudgetFromContext(ctx); ok {
		n := estimate(text)
		for _, b := range sys {
			n += estimate(b.Text)
		}
		if budget.MaxTokensIn > 0 && n > budget.MaxTokensIn {
			return out, usage, &c.Error{Code: c.ErrBudget, Step: "reflection"}
		}
		if budget.MaxTokensOut > 0 {
			maxTokens = budget.MaxTokensOut
		}
	}
	resp, err := m.client.Structured(ctx, ports.StructuredRequest{Tier: ports.ComponentReflection, System: sys, Messages: []ports.ChatMessage{{Role: c.RoleUser, Content: text}}, Schema: raw, SchemaName: c.ReflectionSchemaVersion, MaxTokens: maxTokens, Temperature: .3, Meta: c.CallMeta{TurnID: meta.Task.TurnID, UserID: in.UserID, Bundle: in.Bundle}, IdempotencyKey: mem.Hash(meta.Task)})
	if err != nil {
		return out, resp.Usage, err
	}
	out, err = m.assets.schemas.DecodeReflection(resp.JSON)
	if err != nil {
		return c.ReflectionResult{}, resp.Usage, &c.Error{Code: c.ErrValidation, Step: "reflection"}
	}
	if resp.Usage.Model == "" {
		resp.Usage.Model = resp.Model
	}
	return out, resp.Usage, nil
}
