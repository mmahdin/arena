package schema

import (
	"encoding/json"
	"fmt"

	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

func (s *Set) decodeUnifiedSections(data []byte) (out c.UnifiedTurn, err error) {
	if err = s.Validate(c.UnifiedSchemaVersion, data); err != nil {
		return out, err
	}
	var envelope c.UnifiedEnvelope
	if json.Unmarshal(data, &envelope) != nil {
		return out, ErrInvalid
	}
	out = c.UnifiedTurn{SchemaVersion: envelope.SchemaVersion, Reply: envelope.Reply.Text, Conversation: envelope.Reply.Conversation, UsedContext: envelope.Reply.UsedContext, ReplyDependencies: envelope.Reply.DependsOn, Proposals: []c.StateProposal{}}
	raw, _ := s.JSON(c.UnifiedSchemaVersion)
	var doc map[string]any
	_ = json.Unmarshal(raw, &doc)
	validate := func(name string, value []byte) bool {
		defs := doc["$defs"].(map[string]any)
		if _, ok := defs[name]; !ok {
			return false
		}
		inline, _ := json.Marshal(map[string]any{"$schema": "https://json-schema.org/draft/2020-12/schema", "$defs": defs, "$ref": "#/$defs/" + name})
		check, e := CompileInline(inline)
		return e == nil && check(value) == nil
	}
	reject := func(section string, i int) {
		out.Rejected = append(out.Rejected, c.DeltaRejection{Path: fmt.Sprintf("%s.%d", section, i), RuleCode: "OUT-001", Detail: "invalid independent output section"})
	}
	idCounts := map[string]int{}
	kindCounts := map[string]int{}
	for _, raw := range envelope.StateProposals {
		var p c.StateProposal
		if json.Unmarshal(raw, &p) == nil {
			idCounts[p.ID]++
			kindCounts[p.Kind]++
		}
	}
	for _, raw := range envelope.BackgroundRequests {
		var r c.BackgroundRequest
		if json.Unmarshal(raw, &r) == nil {
			idCounts[r.ID]++
		}
	}
	ids := map[string]bool{}
	kinds := map[string]bool{}
	for i, raw := range envelope.StateProposals {
		var p c.StateProposal
		if json.Unmarshal(raw, &p) != nil || !validate("proposal_"+p.Kind, raw) || ids[p.ID] || kinds[p.Kind] || idCounts[p.ID] > 1 || kindCounts[p.Kind] > 1 {
			reject("state_proposals", i)
			continue
		}
		ids[p.ID] = true
		kinds[p.Kind] = true
		out.Proposals = append(out.Proposals, p)
		switch p.Kind {
		case "session_intent":
			out.Session = new(c.SessionIntent)
			_ = json.Unmarshal(p.Value, out.Session)
		case "thread_intent":
			out.Thread = new(c.ThreadIntent)
			_ = json.Unmarshal(p.Value, out.Thread)
		case "protocol_intent":
			out.Protocol = new(c.ProtocolIntent)
			_ = json.Unmarshal(p.Value, out.Protocol)
		case "open_loops":
			_ = json.Unmarshal(p.Value, &out.OpenLoops)
		case "preferences":
			out.Preferences = new(c.PrefsPatch)
			_ = json.Unmarshal(p.Value, out.Preferences)
		case "risk_read":
			out.RiskRead = new(c.BrainRiskRead)
			_ = json.Unmarshal(p.Value, out.RiskRead)
		}
	}
	for i, raw := range envelope.BackgroundRequests {
		var request c.BackgroundRequest
		if !validate("background_request", raw) || json.Unmarshal(raw, &request) != nil || ids[request.ID] || idCounts[request.ID] > 1 {
			reject("background_requests", i)
			continue
		}
		ids[request.ID] = true
		out.BackgroundRequests = append(out.BackgroundRequests, request)
	}
	for i, raw := range envelope.FutureHints {
		var hint c.HintProposal
		if !validate("future_hint", raw) || json.Unmarshal(raw, &hint) != nil {
			reject("future_hints", i)
			continue
		}
		out.FutureHints = append(out.FutureHints, hint)
	}
	return out, nil
}
