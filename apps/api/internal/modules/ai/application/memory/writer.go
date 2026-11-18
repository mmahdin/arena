package memory

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"reflect"
	"strings"
	"time"
	"unicode/utf8"

	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

type WriterVerifier interface {
	VerifyWriter(context.Context, Meta, ports.MemWriterInput) error
}
type Writer struct {
	client  ports.Client
	assets  *Assets
	verify  WriterVerifier
	metrics Metrics
}

func NewWriter(client ports.Client, a *Assets, v WriterVerifier, metrics Metrics) (*Writer, error) {
	if client == nil || a == nil || v == nil && !a.synthetic {
		return nil, ErrAssets
	}
	return &Writer{client, a, v, metrics}, nil
}

var _ ports.MemoryWriter = (*Writer)(nil)

func estimate(s string) int { return (utf8.RuneCountInString(s) + 2) / 3 }
func escaped(s string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}
func data(name string, v any) string {
	var b strings.Builder
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
	return "<" + name + ">" + escaped(strings.TrimSuffix(b.String(), "\n")) + "</" + name + ">\n"
}

func (w *Writer) render(in ports.MemWriterInput, tags []string) ([]ports.Block, string, error) {
	if len(in.Messages) == 0 || len(in.Messages) > 12 || len(in.Neighbors) > 5 {
		return nil, "", ErrDenied
	}
	lang := in.Messages[0].Lang
	if lang == "" {
		return nil, "", ErrAssets
	}
	if _, ok := w.assets.lex(lang); !ok {
		return nil, "", ErrAssets
	}
	lex, _ := w.assets.lex(lang)
	kinds := []string{}
	for _, kind := range w.assets.config.AllowedKinds {
		if kind != string(c.MemSafetyEvt) {
			kinds = append(kinds, kind)
		}
	}
	// Collection of sensitive information is never prompted proactively. The
	// approved Writer may propose volunteered content, subject to commit gates.
	fields := []any{}
	for _, f := range w.assets.info.Fields {
		if f.Sensitivity == "normal" {
			fields = append(fields, struct{ ID, Purpose, Update string }{f.ID, f.Purpose, f.UpdateRule})
		}
	}
	system := []ports.Block{{Name: "role_and_ops", Cache: true, Text: "<role_and_ops>" + w.assets.prompt + "</role_and_ops>"}, {Name: "schema_ref", Cache: true, Text: data("schema_ref", struct {
		Schema                                          string
		Kinds, Epistemic, Sensitivity, HypothesisHedges []string
		EntityRoles                                     string
	}{c.MemoryWriterSchemaVersion, kinds, []string{"reported", "inferred"}, []string{"normal", "sensitive", "highly_sensitive"}, lex.markers, "source-grounded role, at most 30 characters"})}, {Name: "information_schema", Cache: true, Text: data("information_schema", fields)}}
	parts := []string{"<turn>\n"}
	seen := map[c.ID]bool{}
	for _, m := range in.Messages {
		if m.UserID != in.UserID || !c.ValidID(m.ID) || seen[m.ID] || !utf8.ValidString(m.Content) || (m.Role != c.RoleUser && m.Role != c.RoleAssistant) {
			return nil, "", ErrDenied
		}
		seen[m.ID] = true
		parts[0] += "<message id=\"" + m.ID.String() + "\" role=\"" + string(m.Role) + "\" lang=\"" + escaped(m.Lang) + "\">" + escaped(m.Content) + "</message>\n"
	}
	parts[0] += "</turn>\n"
	// UUIDs in analysis/claims are handles in attributes, never JSON values.
	analysis := in.Analysis
	analysis.PastRefs = nil
	analysis.TargetThreadID = nil
	analysis.CorrectionTarget = nil
	parts = append(parts, data("analysis", struct {
		Analysis   c.TurnAnalysis
		MemoryTags []string
	}{analysis, tags}))
	neighbors := "<neighbors>\n"
	for _, n := range in.Neighbors {
		if n.UserID != in.UserID || !c.ValidID(n.ID) || n.Sensitivity == c.SensHighly {
			return nil, "", ErrDenied
		}
		neighbors += "<memory id=\"" + n.ID.String() + "\" kind=\"" + string(n.Kind) + "\" epistemic=\"" + string(n.Epistemic) + "\" status=\"" + string(n.Status) + "\" age=\"prior\">" + escaped(n.Content) + "</memory>\n"
	}
	parts = append(parts, neighbors+"</neighbors>\n")
	claims := "<claims>\n"
	for _, claim := range in.Claims {
		attrs := ""
		if claim.Source != nil {
			if !seenItem(in.Neighbors, *claim.Source) {
				return nil, "", ErrDenied
			}
			attrs = " source=\"" + claim.Source.String() + "\""
		}
		claims += "<claim" + attrs + ">" + data("text", claim.Text) + data("hedged", claim.Hedged) + data("epistemic", claim.Epistemic) + "</claim>\n"
	}
	parts = append(parts, claims+"</claims>\n")
	for i, cap := range []int{800, 250, 500, 150} {
		if estimate(parts[i]) > cap {
			return nil, "", &c.Error{Code: c.ErrBudget, Step: "memory_render"}
		}
	}
	for i, cap := range []int{700, 300, 300} {
		if estimate(system[i].Text) > cap {
			return nil, "", ErrAssets
		}
	}
	return system, strings.Join(parts, ""), nil
}
func (w *Writer) Propose(ctx context.Context, in ports.MemWriterInput) (out c.MemoryWriterOutput, usage c.LLMUsage, err error) {
	started := time.Now()
	defer func() {
		if w.metrics != nil {
			w.metrics.Observe("memory_writer", time.Since(started), usage, nil)
		}
	}()
	m, err := Auth(ctx, in.UserID)
	if err != nil {
		return out, usage, err
	}
	if !reflect.DeepEqual(m.Bundle, w.assets.Reference()) || !reflect.DeepEqual(in.Bundle, m.Bundle) {
		return out, usage, &c.Error{Code: c.ErrBundle, Step: "memory_writer"}
	}
	if w.verify != nil {
		if err = w.verify.VerifyWriter(ctx, m, in); err != nil {
			return out, usage, ErrDenied
		}
	}
	signalOps, e := signalProposals(m.Signals)
	if e != nil {
		return out, usage, e
	}
	tags := append([]string(nil), m.MemoryTags...)
	for _, signal := range m.Signals {
		if signal.Op == c.HypPropose && signal.Tag != nil && contains(w.assets.hypTags, *signal.Tag) {
			tags = append(tags, *signal.Tag)
		}
	}
	system, text, err := w.render(in, tags)
	if err != nil {
		return out, usage, err
	}
	input := estimate(text)
	for _, b := range system {
		input += estimate(b.Text)
	}
	output := 500
	if budget, ok := c.BudgetFromContext(ctx); ok {
		if budget.MaxTokensIn > 0 && input > budget.MaxTokensIn {
			return out, usage, &c.Error{Code: c.ErrBudget, Step: "memory_writer"}
		}
		if budget.MaxTokensOut > 0 && budget.MaxTokensOut < output {
			output = budget.MaxTokensOut
		}
	}
	if input > 3000 {
		return out, usage, &c.Error{Code: c.ErrBudget, Step: "memory_writer"}
	}
	schema, _ := w.assets.schemas.JSON(c.MemoryWriterSchemaVersion)
	// Reserve bounded operation capacity for deterministic support/contradiction
	// proposals. This narrows the existing schema; it never broadens its contract.
	if len(signalOps) > 0 {
		var document map[string]any
		if json.Unmarshal(schema, &document) != nil {
			return out, usage, ErrAssets
		}
		document["properties"].(map[string]any)["ops"].(map[string]any)["maxItems"] = 8 - len(signalOps)
		schema, _ = json.Marshal(document)
	}
	resp, e := w.client.Structured(ctx, ports.StructuredRequest{Tier: ports.TierSmall, System: system, Messages: []ports.ChatMessage{{Role: c.RoleUser, Content: text}}, Schema: schema, SchemaName: c.MemoryWriterSchemaVersion, MaxTokens: output, Temperature: .1, Timeout: 15 * time.Second, IdempotencyKey: m.OperationID.String(), Meta: c.CallMeta{UserID: m.UserID, TurnID: m.TurnID, Bundle: m.Bundle}})
	usage = resp.Usage
	if e != nil {
		return out, usage, &c.Error{Code: c.ErrProvider, Step: "memory_writer", Retryable: true}
	}
	out, e = w.assets.schemas.DecodeMemoryWriter(resp.JSON)
	if e != nil || len(out.Ops) > 8-len(signalOps) {
		return c.MemoryWriterOutput{}, usage, &c.Error{Code: c.ErrValidation, Step: "memory_writer"}
	}
	// Model handles may reference only the input actually disclosed to it.
	for _, op := range out.Ops {
		for _, id := range op.EvidenceMsg {
			if !seenMessage(in.Messages, id) {
				return c.MemoryWriterOutput{}, usage, ErrDenied
			}
		}
		if op.TargetID != nil && !seenItem(in.Neighbors, *op.TargetID) {
			return c.MemoryWriterOutput{}, usage, ErrDenied
		}
	}
	return out, usage, nil
}
func seenMessage(xs []c.Message, id c.ID) bool {
	for _, x := range xs {
		if x.ID == id {
			return true
		}
	}
	return false
}
func seenItem(xs []c.MemoryItem, id c.ID) bool {
	for _, x := range xs {
		if x.ID == id {
			return true
		}
	}
	return false
}
