package llmhttp

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/llm"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
)

// NewSystemOne reuses the existing bounded, credential-safe HTTP transport and
// Inspector capture, but never calls chat completions, embeddings or streaming.
func NewSystemOne(baseURL, apiKey string, client *http.Client) (*Provider, error) {
	p, err := New(baseURL, apiKey, client)
	if err == nil {
		p.systemOne = true
	}
	return p, err
}

func (p *Provider) systemOneSelection(ctx context.Context, model llm.Model, req ports.StructuredRequest) (llm.StructuredResult, error) {
	var out llm.StructuredResult
	input := req.SystemOne
	if req.Tier != ports.ComponentJEV || req.SchemaName != "jev_selection.v1" || model.Provider != "OpenCode" || model.Name != "jev-1.13-free" || model.BillingSource != "free" ||
		model.InputMicroUSDPerM != 0 || model.OutputMicroUSDPerM != 0 || model.CachedMicroUSDPerM != 0 || input == nil || strings.TrimSpace(input.State) == "" || len(input.Questions) < 2 || len(input.Questions) > 129 {
		return out, requestFailure()
	}
	if _, ok := input.Questions["topic_continues"]; !ok {
		return out, requestFailure()
	}
	for name, q := range input.Questions {
		if q.Type != "noul" || q.Instructions == "" || name != "topic_continues" && (!strings.HasPrefix(name, "resource:") || len(strings.TrimPrefix(name, "resource:")) == 0) {
			return out, requestFailure()
		}
	}
	payload := struct {
		Model string `json:"model"`
		*ports.SystemOneRequest
	}{model.Name, input}
	response, err := p.send(ctx, "/systemone", payload)
	if err != nil {
		return out, err
	}
	var decoded struct {
		Model   string                     `json:"model"`
		Answers map[string]json.RawMessage `json:"answers"`
		Usage   *struct {
			Input  *int `json:"input_tokens"`
			Output *int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := readJSON(response, &decoded); err != nil {
		return out, err
	}
	trace.Capture(ctx, "systemone_response", "decision_response", "JEV System One", decoded)
	if decoded.Usage == nil || decoded.Usage.Input == nil || decoded.Usage.Output == nil || *decoded.Usage.Input < 0 || *decoded.Usage.Output < 0 {
		return out, providerFailure(llm.FailureInvalid)
	}
	out.Usage = contract.LLMUsage{InTokens: *decoded.Usage.Input, OutTokens: *decoded.Usage.Output, FinishReason: "stop"}
	// Invalid typed answers deliberately reach the existing schema validator.
	// That boundary charges reported usage before rejecting, without repairing or
	// repeating JEV's single call. Never invent a false/empty successful decision.
	out.JSON = []byte("null")
	valid := decoded.Model != "" && len(decoded.Answers) == len(input.Questions)
	type candidate struct {
		id string
		p  float64
	}
	selected := []candidate{}
	continues := false
	for name := range input.Questions {
		raw, ok := decoded.Answers[name]
		var a struct {
			Type string   `json:"type"`
			Noul *float64 `json:"noul"`
		}
		if !ok || json.Unmarshal(raw, &a) != nil || a.Type != "noul" || a.Noul == nil || *a.Noul < 0 || *a.Noul > 1 {
			valid = false
			continue
		}
		if name == "topic_continues" {
			continues = *a.Noul > 0.5
		} else if *a.Noul > 0.5 {
			selected = append(selected, candidate{strings.TrimPrefix(name, "resource:"), *a.Noul})
		}
	}
	trace.Capture(ctx, "decision_validation", "diagnostic", "JEV System One", map[string]any{"valid": valid, "billing_source": model.BillingSource, "reported_model": decoded.Model})
	if !valid {
		return out, nil
	}
	// Noul is the probability of yes. Keep only affirmative cards, highest first;
	// an exact tie at 0.5 is uncertainty. ID order makes equal scores reproducible.
	sort.Slice(selected, func(i, j int) bool {
		if selected[i].p == selected[j].p {
			return selected[i].id < selected[j].id
		}
		return selected[i].p > selected[j].p
	})
	ids := []string{}
	for _, card := range selected {
		if len(ids) == 4 {
			break
		}
		ids = append(ids, card.id)
	}
	out.JSON, _ = json.Marshal(struct {
		Schema    string   `json:"schema_version"`
		IDs       []string `json:"selected_resource_ids"`
		Continues bool     `json:"topic_continues"`
	}{"jev_selection.v1", ids, continues})
	trace.Capture(ctx, "normalized_selection", "knowledge_selection", "JEV System One", json.RawMessage(out.JSON))
	return out, nil
}
