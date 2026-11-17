package retrieval

import (
	"regexp"
	"strings"
)

// Strategy names as they appear in RetrievalMeta.Strategies and the trace.
const (
	StrategyTemporal = "temporal_exact"
	StrategyThread   = "thread"
	StrategyHybrid   = "hybrid"
	StrategyAnaphora = "anaphora"
	StrategyEntity   = "entity"
)

// AllStrategies is the canonical evaluation order. Deterministic precedence
// matters: exact recall evidence must outrank semantic neighbours when the two
// disagree (§4.4 Scenario 2).
var AllStrategies = []string{StrategyTemporal, StrategyThread, StrategyEntity, StrategyAnaphora, StrategyHybrid}

// IsKnownStrategy validates a request-supplied strategy (§4.4 Strategies field).
func IsKnownStrategy(s string) bool {
	for _, known := range AllStrategies {
		if known == s {
			return true
		}
	}
	return false
}

// anaphoraPatterns detect reference-to-the-past phrases. They are language
// specific and tested individually; a false positive only costs one extra
// strategy execution, never an incorrect answer, because retrieval remains
// evidence-bearing and the Brain is told about `ambiguous_reference`.
var anaphoraPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b(same|similar)\s+(feeling|thing|issue|problem|pattern|way)\b`),
	regexp.MustCompile(`(?i)\b(as|like)\s+(before|last time|we discussed|i mentioned)\b`),
	regexp.MustCompile(`(?i)\b(that|this)\s+(thing|time|feeling|situation)\s+(i|we)\s+(told|said|mentioned|discussed|talked)\b`),
	regexp.MustCompile(`(?i)\b(again|once more)\b.*\b(told|mentioned|discussed)\b`),
	regexp.MustCompile(`(?i)\byou\s+(remember|know)\b`),
	regexp.MustCompile(`(?i)\bback\s+then\b`),
	regexp.MustCompile(`(همان|همون)\s*(احساس|حس|موضوع|مشکل|چیز|دفعه)`),
	regexp.MustCompile(`(قبلا|قبل)\s*(گفتم|گفته|ناراحت|حس)`),
	regexp.MustCompile(`(یادت|به\s*یادت)\s*(هست|میاد|می‌آید|نیست)`),
	regexp.MustCompile(`(بار\s*قبل|دفعه\s*قبل|آن\s*موقع)`),
}

// HasAnaphora reports whether the message references earlier conversation.
func HasAnaphora(text string) bool {
	for _, re := range anaphoraPatterns {
		if re.MatchString(text) {
			return true
		}
	}
	return false
}

// StrategyPlan is the deterministic strategy selection result.
type StrategyPlan struct {
	Strategies []string
	Temporal   bool
	Anaphora   bool
	Ambiguous  bool
	RecallVerb bool
}

// Plan selects strategies from cheap signals only (no LLM), per §3.4/§4.4.
//
// Rules (first match wins for the special cases, hybrid always last because it
// is the default recall path):
//   - temporal expression + recall verb, or an explicit temporal window in the
//     request → temporal_exact (plus hybrid as a bounded second source);
//   - an anaphora marker → anaphora (+ thread when a focus thread exists);
//   - a focus thread or a "continues" relation → thread;
//   - always hybrid as the semantic default.
//
// A caller may force strategies; the service still enforces the hard cap.
func Plan(in PlanInput) StrategyPlan {
	var plan StrategyPlan
	forced := make(map[string]bool, len(in.Requested))
	for _, s := range in.Requested {
		if IsKnownStrategy(s) {
			forced[s] = true
		}
	}
	hasWindow := in.Window != nil
	recall := in.RecallVerb || HasRecallVerb(in.Text)

	anaphora := HasAnaphora(in.Text) || in.ExplicitAnaphora
	temporal := hasWindow && (recall || in.ForceTemporal)

	plan.RecallVerb = recall
	plan.Anaphora = anaphora
	plan.Temporal = temporal

	add := func(name string) {
		for _, existing := range plan.Strategies {
			if existing == name {
				return
			}
		}
		plan.Strategies = append(plan.Strategies, name)
	}

	if len(forced) > 0 {
		for _, s := range AllStrategies {
			if forced[s] {
				add(s)
			}
		}
		// A forced anaphora/temporal strategy still needs thread context to be
		// useful; add it rather than silently returning nothing.
		if (forced[StrategyAnaphora] || forced[StrategyTemporal]) && in.FocusThread != "" {
			add(StrategyThread)
		}
	} else {
		if temporal {
			add(StrategyTemporal)
		}
		if anaphora {
			add(StrategyAnaphora)
			if in.FocusThread != "" {
				add(StrategyThread)
			}
			// An anaphoric reference with no thread and no matching tag is by
			// definition ambiguous: the Brain should clarify, not guess (§3.4).
			if in.FocusThread == "" && len(in.Emotions) == 0 {
				plan.Ambiguous = true
			}
		}
		if in.FocusThread != "" && in.RelationContinues {
			add(StrategyThread)
		}
		if len(plan.Strategies) == 0 || !temporal {
			add(StrategyHybrid)
		}
	}

	// Second hop: exactly one extra targeted strategy, never a loop (§4.4).
	if in.Hop > 0 && in.NeedsKind != "" {
		switch in.NeedsKind {
		case "entity":
			add(StrategyEntity)
		case "time_window":
			add(StrategyTemporal)
		case "topic":
			add(StrategyHybrid)
		case "item":
			// Item lookups are direct ID reads, no strategy needed.
		}
	}
	return plan
}

// PlanInput is the cheap-signal input to strategy selection.
type PlanInput struct {
	Requested         []string
	Text              string
	Window            *Window
	RecallVerb        bool
	ForceTemporal     bool
	ExplicitAnaphora  bool
	FocusThread       string
	RelationContinues bool
	Emotions          []string
	Hop               int
	NeedsKind         string
}

// EntityPatterns capture a conservative set of relational/entity mentions.
//
// Scope note (open decision, §7.3 "person alias matching remains [OPEN]"): the
// entity index matches the literal labels the user wrote. Aliasing ("my sister"
// → "Lena") is explicitly *not* implemented, because inventing an alias map is a
// product/clinical decision, not an engineering one.
var entityPatterns = []*regexp.Regexp{
	// "my sister Lena", "my friend Sara"
	regexp.MustCompile(`(?i)\bmy\s+(sister|brother|mother|mom|mum|father|dad|partner|husband|wife|boyfriend|girlfriend|friend|boss|manager|colleague|coworker|roommate|son|daughter|child|therapist|doctor|teacher)\b(?:\s+(?:named\s+|called\s+)?([\p{L}]{2,30}))?`),
	// Persian kinship terms
	regexp.MustCompile(`(خواهرم|خواهرم|برادرم|مادرم|مامانم|بابام|پدرم|همسرم|دوست[م]?|رئیسم|همکارم|همخانه‌ام|پسرم|دخترم)\s*([\p{L}]{2,30})?`),
}

// EntityMentions extracts (role, optional label) pairs. Labels are returned
// without judgment about whether they are real names.
func EntityMentions(text string) [][2]string {
	var out [][2]string
	seen := map[string]struct{}{}
	for _, re := range entityPatterns {
		for _, m := range re.FindAllStringSubmatch(text, -1) {
			role := strings.ToLower(strings.TrimSpace(m[1]))
			label := ""
			if len(m) > 2 {
				label = strings.TrimSpace(m[2])
			}
			key := role + "|" + strings.ToLower(label)
			if _, dup := seen[key]; dup {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, [2]string{role, label})
			if len(out) == 3 {
				return out
			}
		}
	}
	return out
}

// EntityQuery is the query-side entity filter for the entity strategy.
type EntityQuery struct {
	Role  string
	Label string
}

// EntityQueryFrom derives an entity query from the turn text or from a
// second-hop NeedsContext{kind:"entity"}.
func EntityQueryFrom(text, hopQuery string) EntityQuery {
	if hopQuery != "" {
		mentions := EntityMentions(hopQuery)
		if len(mentions) > 0 {
			return EntityQuery{Role: mentions[0][0], Label: mentions[0][1]}
		}
		fields := strings.Fields(hopQuery)
		if len(fields) > 0 {
			return EntityQuery{Label: strings.ToLower(strings.TrimSpace(fields[len(fields)-1]))}
		}
		return EntityQuery{}
	}
	mentions := EntityMentions(text)
	if len(mentions) == 0 {
		return EntityQuery{}
	}
	return EntityQuery{Role: mentions[0][0], Label: mentions[0][1]}
}

// HopNeedsContextKind validates a second-hop NeedsContext kind (§4.4). The set is
// closed so an unknown kind is a contract error rather than a silent no-op.
func HopNeedsContextKind(kind string) bool {
	switch kind {
	case "entity", "topic", "time_window", "item":
		return true
	}
	return false
}
