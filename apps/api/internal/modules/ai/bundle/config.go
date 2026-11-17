package bundle

import (
	"math"
	"strings"
)

func validateConfig(files map[string][]byte, vocab map[string]map[string]bool, r *Report) {
	var router Router
	if parse(files, "config/router.yaml", "router.v1", &router, r) {
		if len(router.Rules) == 0 {
			r.fail("config/router.yaml", "router_rules", "ordered rules required")
		}
		seen := map[string]bool{}
		for i, rule := range router.Rules {
			if rule.ID == "" || seen[rule.ID] {
				r.fail("config/router.yaml", "router_id", "empty/duplicate rule id")
			}
			seen[rule.ID] = true
			if !oneOf(rule.Route, "normal", "fast", "crisis", "proactive") {
				r.fail("config/router.yaml", "router_route", "invalid route")
			}
			if i < len(router.Rules)-1 && len(rule.When) == 0 {
				r.fail("config/router.yaml", "router_order", "only final rule can be default")
			}
			for _, pred := range rule.When {
				if len(pred) != 1 {
					r.fail("config/router.yaml", "router_predicate", "each when must name one predicate")
				}
				for k, v := range pred {
					if !oneOf(k, "safety_level_gte", "safety_level_lte", "origin_is", "risk_follow_up_active", "trivial_message", "pending_question", "protocol_active") {
						r.fail("config/router.yaml", "router_predicate", "unknown named predicate")
					}
					if !scalar(v) {
						r.fail("config/router.yaml", "router_predicate", "predicate value must be scalar")
					}
				}
			}
			for key, v := range rule.Set {
				if !oneOf(key, "risk_constraints", "force_buffered_output", "shortlist") {
					r.fail("config/router.yaml", "router_set", "unknown router override")
				}
				if _, ok := v.(bool); !ok {
					r.fail("config/router.yaml", "router_set", "override must be bool")
				}
			}
		}
		if len(router.Rules) > 0 {
			last := router.Rules[len(router.Rules)-1]
			if last.ID != "RT-099" || len(last.When) > 0 || last.Route != "normal" {
				r.fail("config/router.yaml", "router_default", "last rule must be RT-099 default normal")
			}
		}
		if router.ShortlistRules.MaxItems < 0 || router.ShortlistRules.MaxItems > 5 {
			r.fail("config/router.yaml", "shortlist", "maximum five shortlist items")
		}
		for loc, p := range router.TrivialPatterns {
			if !localeRE.MatchString(loc) || p.MaxChars < 0 {
				r.fail("config/router.yaml", "trivial_patterns", "invalid locale/max_chars")
			}
		}
	}
	var gate PolicyGate
	if parse(files, "config/policy_gate.yaml", "policy_gate.v1", &gate, r) {
		if gate.Questions.MaxPerReply > 1 || gate.Questions.MaxPerReply < 0 || gate.Questions.MaxConsecutiveQuestionTurns < 0 || gate.Risk.BrainMayLower || !gate.Risk.BrainMayRaise || gate.Protocols.MaxOffersPerSession < 0 {
			r.fail("config/policy_gate.yaml", "policy_safety", "invalid question/offer limits or risk downgrade")
		}
		if !oneOf(gate.Hypotheses.VoiceRequiresStatus, "proposed", "supported") || gate.Hypotheses.VoiceMinGapHours < 0 {
			r.fail("config/policy_gate.yaml", "hypothesis_rules", "invalid voice-hypothesis gate")
		}
		if !oneOf(gate.Repairs.OnInvalidObjective, "reflect", "clarify") || gate.Repairs.OnMissingEvidence != "downgrade_to_inferred_or_drop" || gate.Repairs.OnUnknownSkill != "default_skill_for_objective" {
			r.fail("config/policy_gate.yaml", "policy_repairs", "unknown repair policy")
		}
	}
	var retrieval Retrieval
	if parse(files, "config/retrieval.yaml", "retrieval.v1", &retrieval, r) {
		weights := retrieval.Weights
		total := weights.Similarity + weights.Recency + weights.Importance + weights.ThreadMatch
		if math.Abs(total-1) > 0.0001 || weights.Similarity < 0 || weights.Recency < 0 || weights.Importance < 0 || weights.ThreadMatch < 0 {
			r.fail("config/retrieval.yaml", "retrieval_weights", "weights must be nonnegative and sum to 1")
		}
		if retrieval.Defaults.MaxItems < 1 || retrieval.Defaults.HardCapWithHop < retrieval.Defaults.MaxItems || retrieval.Defaults.MinScore < 0 || retrieval.Defaults.MinScore > 1 || retrieval.Timeouts.TotalMS < 1 || retrieval.Timeouts.PerStrategyMS > retrieval.Timeouts.TotalMS || retrieval.Recency.HalfLifeDays < 1 || retrieval.Temporal.MaxMessages < 1 {
			r.fail("config/retrieval.yaml", "retrieval_ranges", "invalid retrieval limits or timeouts")
		}
	}
	var budgets Budgets
	if parse(files, "config/budgets.yaml", "budgets.v1", &budgets, r) {
		t := budgets.Turn
		if t.TotalMS < 1 || t.SafetyMS < 1 || t.RetrievalMS < 1 || t.BrainMS < 1 || t.GenFirstTokenMS < 1 || t.GenTotalMS < 1 || t.GenFirstTokenMS > t.GenTotalMS || t.SafetyMS+t.RetrievalMS+t.BrainMS+t.GenTotalMS > t.TotalMS {
			r.fail("config/budgets.yaml", "turn_budget", "invalid budget or impossible critical-path deadline")
		}
		if budgets.Limits.MaxSyncLLMCalls < 1 || budgets.Limits.MaxRegenerations < 0 || budgets.Limits.MaxBrainHops < 0 || budgets.Tokens.BrainIn < 1 || budgets.Tokens.BrainOut < 1 || budgets.Tokens.GenIn < 1 || budgets.Tokens.GenOut < 1 {
			r.fail("config/budgets.yaml", "budget_limits", "missing/invalid token or call budget")
		}
	}
	var thresholds Thresholds
	if parse(files, "config/thresholds.yaml", "thresholds.v1", &thresholds, r) {
		if thresholds.Hypothesis.SupportedMinSupport < 1 || thresholds.Hypothesis.SupportedMinSessions < 1 || thresholds.Memory.DedupeCosineGTE < 0 || thresholds.Memory.DedupeCosineGTE > 1 || thresholds.Safety.L1MinConfidenceForHigh < 0 || thresholds.Safety.L1MinConfidenceForHigh > 1 || thresholds.Memory.HotSetMaxItems < 1 || thresholds.Thread.MaxActive < 1 {
			r.fail("config/thresholds.yaml", "threshold_range", "invalid threshold range")
		}
		if !thresholds.Memory.ContradictionRequiresUserConfirmation || !thresholds.Safety.Deescalate.RequireUserStatement {
			r.fail("config/thresholds.yaml", "threshold_safety", "confirmation/deescalation gates cannot be disabled")
		}
		for kind, value := range thresholds.Validator {
			if !oneOf(kind, "directive_advice", "certainty_about_inference", "dependency_encouraging", "invalidating") || math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) || value <= 0 || value > 1 {
				r.fail("config/thresholds.yaml", "validator_threshold", "invalid WP-07 classifier threshold")
			}
		}
		// Staleness thresholds are NOT retention or deletion instructions.
	}
	var memory MemoryConfig
	if parse(files, "config/memory.yaml", "memory.v1", &memory, r) {
		if memory.AllowHardDelete || !memory.PreserveDerivedOnSoftDelete || !memory.RequireEvidence {
			r.fail("config/memory.yaml", "soft_delete", "hard deletion/derived purge or ungrounded memory is disallowed")
		}
		if len(memory.AllowedKinds) == 0 {
			r.fail("config/memory.yaml", "memory_kind", "at least one memory kind required")
		}
		for _, kind := range memory.AllowedKinds {
			if !oneOf(kind, "fact", "preference", "episode", "hypothesis", "goal", "event", "safety_event") {
				r.fail("config/memory.yaml", "memory_kind", "unrecognized memory kind")
			}
		}
	}
	var models Models
	if parse(files, "config/models.yaml", "models.v1", &models, r) {
		for _, tier := range []string{"small", "strong", "classifier", "embedding"} {
			cfg, ok := models.Tiers[tier]
			if !ok || len(cfg.Chain) == 0 {
				r.fail("config/models.yaml", "model_tier", "required model tier has no provider chain")
				continue
			}
			for _, entry := range cfg.Chain {
				if entry.Provider == "" || entry.Model == "" {
					r.fail("config/models.yaml", "model_chain", "empty provider/model")
				}
			}
			for task, temp := range cfg.Temperature {
				if task == "" || temp < 0 || temp > 2 {
					r.fail("config/models.yaml", "model_temperature", "invalid task/temperature")
				}
			}
			if tier == "embedding" && cfg.Dim < 1 {
				r.fail("config/models.yaml", "embedding_dim", "embedding dimension required")
			}
		}
		for tier := range models.Tiers {
			if !oneOf(tier, "small", "strong", "classifier", "embedding") {
				r.fail("config/models.yaml", "model_tier", "unknown model tier")
			}
		}
		if models.Breaker.ErrorRateOpen <= 0 || models.Breaker.ErrorRateOpen > 1 || models.Breaker.WindowS < 1 || models.Breaker.HalfOpenAfterS < 1 {
			r.fail("config/models.yaml", "breaker", "invalid circuit breaker configuration")
		}
	}
	var sampling Sampling
	if parse(files, "config/sampling.yaml", "sampling.v1", &sampling, r) {
		if sampling.ReviewQueueEvent == "" {
			r.fail("config/sampling.yaml", "sampling", "review_queue_event required")
		}
		for route, p := range sampling.Random {
			if !oneOf(route, "normal", "fast", "proactive") || p < 0 || p > 1 {
				r.fail("config/sampling.yaml", "sampling_probability", "invalid sampling route/probability")
			}
		}
	}
	for _, p := range filesWithPrefix(files, "evals/golden/") {
		var g Golden
		if !parse(files, p, "golden.v1", &g, r) {
			continue
		}
		if !validID(g.ID) || g.ID != basenameID(p) || g.Title == "" {
			r.fail(p, "golden_id", "invalid golden scenario envelope")
		}
		if !g.SyntheticOnly {
			r.warn(p, "golden_unreviewed", "fixture is not explicitly synthetic; WP-15 must review its assertions")
		}
		for _, tag := range g.Tags {
			if strings.HasPrefix(tag, "objective_") {
				if !oneOf(strings.TrimPrefix(tag, "objective_"), "reflect", "validate_and_support", "clarify", "explore", "provide_insight", "provide_guidance", "offer_intervention", "continue_intervention", "close_or_pause_intervention", "review_progress", "summarize", "repair", "answer_direct", "boundary", "safety_support") {
					r.fail(p, "golden_objective", "unknown objective in golden tag")
				}
			}
		}
	}
	_ = vocab
}
func scalar(v any) bool {
	switch v.(type) {
	case bool, string, int, int64, float64:
		return true
	}
	return false
}
