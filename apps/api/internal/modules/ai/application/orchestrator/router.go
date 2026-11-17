package orchestrator

import (
	"strings"
	"unicode/utf8"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

// Router is deliberately deterministic. Bundle-defined policy predicates,
// intervention eligibility, and clinician-owned rules are NOT interpreted by
// this walking skeleton; these are WP-03/WP-10/WP-13 responsibilities.
type Router struct{}

var _ ports.Router = Router{}

func riskRank(level contract.RiskLevel) int {
	switch level {
	case contract.RiskNone:
		return 0
	case contract.RiskLow:
		return 1
	case contract.RiskElevated:
		return 2
	case contract.RiskHigh:
		return 3
	case contract.RiskImminent:
		return 4
	default:
		return 2 // unknown is never treated as risk-free
	}
}

func (Router) Select(in ports.RouteInput) ports.RouteDecision {
	risk := riskRank(in.Safety.Level)
	if v := riskRank(in.WCtx.Risk.Current); v > risk {
		risk = v
	}
	if risk >= 3 {
		return ports.RouteDecision{Route: contract.RouteCrisis, Reason: "RT-001", RiskConstraints: true, ForceBufferedOutput: true}
	}
	if in.WCtx.Risk.FollowUpActive {
		return ports.RouteDecision{Route: contract.RouteNormal, Reason: "RT-002", RiskConstraints: true, ForceBufferedOutput: true}
	}
	if in.Origin == contract.OriginWakeUp {
		return ports.RouteDecision{Route: contract.RouteProactive, Reason: "RT-003", RiskConstraints: risk >= 2, ForceBufferedOutput: risk >= 2}
	}
	if risk <= 1 && in.WCtx.Session.PendingQuestion == nil && in.WCtx.Protocol == nil && trivial(in.WCtx.Turn.Text, in.WCtx.Turn.Lang, in.Config) {
		return ports.RouteDecision{Route: contract.RouteFast, Reason: "RT-004"}
	}
	return ports.RouteDecision{Route: contract.RouteNormal, Reason: "RT-099", RiskConstraints: risk >= 2, ForceBufferedOutput: risk >= 2}
}

func trivial(raw, lang string, config ports.RouterConfig) bool {
	limit := config.TrivialMaxChars
	if limit == 0 {
		limit = 24
	}
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw == "" || utf8.RuneCountInString(raw) > limit {
		return false
	}
	// Only explicitly supplied syntactic patterns are classified as trivial.
	// We do not guess which clinical disclosures are "small talk".
	for _, candidate := range config.TrivialPatterns[lang] {
		if raw == strings.ToLower(strings.TrimSpace(candidate)) && candidate != "" {
			return true
		}
	}
	return false
}
