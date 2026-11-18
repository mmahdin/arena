package brain

import (
	"encoding/json"
	"encoding/xml"
	"strings"
	"unicode/utf8"

	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

func escaped(s string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}
func data(name string, v any) string {
	raw, _ := json.Marshal(v)
	return "<" + name + ">" + escaped(string(raw)) + "</" + name + ">\n"
}

// Uses the same conservative character approximation as WP-08. This is
// deliberately provider-independent; deployment must calibrate with a tokenizer.
func estimate(s string) int { return (utf8.RuneCountInString(s) + 2) / 3 }

func (a *Assets) render(in ports.BrainInput) ([]ports.Block, string, c.WorkingContext, error) {
	raw, e := json.Marshal(in.WCtx)
	if e != nil || !utf8.Valid(raw) {
		return nil, "", c.WorkingContext{}, ErrAssets
	}
	var wc c.WorkingContext
	_ = json.Unmarshal(raw, &wc)
	blocks := append([]ports.Block(nil), a.system...)
	render := func() string {
		out := renderState(wc, in.Safety)
		out += data("retrieval_meta", wc.RetrievalMeta)
		for _, m := range wc.Retrieved {
			out += "<memory id=\"" + m.ID.String() + "\" kind=\"" + escaped(string(m.Kind)) + "\" epistemic=\"" + escaped(string(m.Epistemic)) + "\" status=\"" + escaped(string(m.Status)) + "\" age=\"" + escaped(m.Age) + "\" verbatim=\"" + boolText(m.Verbatim) + "\">" + escaped(m.Text) + "</memory>\n"
		}
		out += data("shortlist", wc.Shortlist)
		out += "<recent>\n"
		for _, m := range wc.Recent {
			out += "<message id=\"" + m.ID.String() + "\" role=\"" + escaped(string(m.Role)) + "\" age=\"" + escaped(m.Age) + "\">" + escaped(m.Text) + "</message>\n"
		}
		out += "</recent>\n"
		out += "<user_message id=\"" + wc.Turn.MessageID.String() + "\" lang=\"" + escaped(wc.Turn.Lang) + "\" local_time=\"" + escaped(wc.Turn.LocalTime) + "\" origin=\"" + escaped(string(wc.Turn.Origin)) + "\">" + escaped(wc.Turn.Text) + data("temporal_refs", wc.Turn.TemporalRefs) + "</user_message>"
		return out
	}
	size := 0
	for _, b := range blocks {
		size += estimate(b.Text)
	}
	// JSON keys/UUIDs are accounted for as well as text. Never trim instructions
	// or the current message: an impossible budget takes FB-001 before inference.
	for {
		dynamic := render()
		if size+estimate(dynamic) <= a.budgets.Tokens.BrainIn {
			return blocks, dynamic, wc, nil
		}
		if len(wc.Recent) > 0 {
			wc.Recent = wc.Recent[1:]
			continue
		}
		if len(wc.Retrieved) > 0 {
			min := 0
			for i := range wc.Retrieved {
				if wc.Retrieved[i].Score < wc.Retrieved[min].Score {
					min = i
				}
			}
			wc.Retrieved = append(wc.Retrieved[:min], wc.Retrieved[min+1:]...)
			continue
		}
		if wc.Session.Summary != "" {
			wc.Session.Summary = ""
			continue
		}
		return nil, "", wc, &c.Error{Code: c.ErrBudget, Step: "brain_render", Degrades: true}
	}
}
func boolText(v bool) string {
	if v {
		return "true"
	}
	return "false"
}
