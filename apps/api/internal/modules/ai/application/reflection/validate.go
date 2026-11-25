package reflection

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	mem "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

// Validate is used for new outputs AND recovered encrypted checkpoints. RF-004
// counts independent user evidence; assistant words cannot establish a fact.
func (a *Assets) Validate(ctx context.Context, t Task, s Snapshot, r c.ReflectionResult, p mem.Policy) (result0 c.ReflectionResult, result1 []c.DeltaRejection, result2 error) {
	ctx, inspectionSpan := trace.Start(ctx, "application/reflection/validate.Validate", func() any { return map[string]any{"t": t, "s": s, "r": r, "p": p} })
	defer func() {
		inspectionSpan.Complete(func() any { return map[string]any{"result0": result0, "result1": result1} }, result2, recover())
	}()

	raw, e := json.Marshal(r)
	if e != nil || len(raw) > 96<<10 || a.schemas.Validate(c.ReflectionSchemaVersion, raw) != nil || r.ThreadID != t.ThreadID || !a.Eligible(t, s) || !authorize(ctx, p, t, s) {
		return c.ReflectionResult{}, nil, mem.ErrDenied
	}
	_ = json.Unmarshal(raw, &r)
	rejected := []c.DeltaRejection{}
	reject := func(path, rule string) { rejected = append(rejected, c.DeltaRejection{Path: path, RuleCode: rule}) }
	sources := map[c.ID]mem.Source{}
	for _, src := range s.Sources {
		if src.Message.UserID != t.UserID || !c.ValidID(src.Message.ID) || src.Future {
			return r, nil, mem.ErrDenied
		}
		if src.Hidden && !permitted(ctx, p, s, mem.UseRetainedSource, "message", src.Message.ID) || !permitted(ctx, p, s, mem.UseSource, "message", src.Message.ID) {
			return r, nil, mem.ErrDenied
		}
		sources[src.Message.ID] = src
	}
	lex, ok := a.lexicon(s.User.Locale)
	if !ok {
		return r, nil, mem.ErrAssets
	}
	clean := func(v string) bool {
		if !utf8.ValidString(v) {
			return false
		}
		for _, label := range lex.ClinicalLabels {
			if strings.Contains(strings.ToLower(v), strings.ToLower(label)) {
				return false
			}
		}
		return true
	}
	hedge := func(v string) bool {
		for _, h := range lex.Hedges {
			re := regexp.MustCompile(`(?i)(?:^|[\s\pP])` + regexp.QuoteMeta(h) + `(?:$|[\s\pP])`)
			if re.MatchString(v) {
				return true
			}
		}
		return false
	}
	if r.Formulation != nil {
		framework, e := a.Framework(s.Thread)
		if e != nil {
			return r, nil, mem.ErrDenied
		}
		if r.Formulation.Framework != framework {
			return r, nil, mem.ErrDenied
		}
		f := a.frameworks[framework]
		full := sufficient(a, s)
		for _, field := range f.Fields {
			v := r.Formulation.Fields[field.ID]
			valid := full && v.Status != c.EpUnknown && slices.Contains(field.AllowedStatus, string(v.Status)) && clean(v.Value) && len(strings.Fields(v.Value)) <= 40 && len([]rune(v.Value)) <= 320
			sessions := map[c.ID]bool{}
			refs := map[c.ID]bool{}
			reported := false
			for _, id := range v.Evidence {
				src, exists := sources[id]
				if !exists || src.Message.Role != c.RoleUser || !c.ValidID(src.SessionID) {
					valid = false
					continue
				}
				refs[id] = true
				sessions[src.SessionID] = true
				if strings.Contains(mem.Canonical(src.Message.Content), mem.Canonical(v.Value)) && strings.TrimSpace(v.Value) != "" {
					reported = true
				}
			}
			if v.Status == c.EpReported && !reported {
				valid = false
			}
			if v.Status == c.EpInferred && !hedge(v.Value) {
				valid = false
			}
			if len(refs) < field.MinEvidence || len(sessions) < max(1, field.MinSessions) {
				valid = false
			}
			if !valid {
				v = c.FormulationField{Status: c.EpUnknown, Value: "", Evidence: []c.ID{}, NeedsEvidenceMore: true}
				reject("formulation.fields."+field.ID, "RF-004")
			}
			r.Formulation.Fields[field.ID] = v
		}
		// Rebuild summary from accepted fields. Model prose cannot preserve a claim
		// whose field was just forced unknown, or make an inferred claim certain.
		keys := []string{}
		for k := range r.Formulation.Fields {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		lines := []string{}
		for _, k := range keys {
			v := r.Formulation.Fields[k]
			if v.Status != c.EpUnknown {
				lines = append(lines, fmt.Sprintf("%s (%s): %s", k, v.Status, v.Value))
			}
		}
		r.Formulation.Paragraph = strings.Join(lines, "; ")
		if len(strings.Fields(r.Formulation.Paragraph)) > f.Paragraph.MaxWords {
			r.Formulation.Paragraph = ""
			reject("formulation.paragraph", "RF-004")
		}
	}
	if len(r.HypothesisOps) > 7 || len(r.InformationGaps) > 8 || len(r.ThreadProposals) > 4 {
		return r, nil, mem.ErrDenied
	}
	items := map[c.ID]c.MemoryItem{}
	for _, it := range s.Items {
		items[it.ID] = it
	}
	for _, op := range r.HypothesisOps {
		if op.Kind != "" && op.Kind != c.MemHypothesis {
			return r, nil, mem.ErrDenied
		}
		if op.TargetID != nil {
			it, exists := items[*op.TargetID]
			if !exists || it.Kind != c.MemHypothesis {
				return r, nil, mem.ErrDenied
			}
		}
		for _, id := range op.EvidenceMsg {
			src, ok := sources[id]
			if !ok || src.Message.Role != c.RoleUser {
				return r, nil, mem.ErrDenied
			}
		}
		if !clean(op.Content) {
			return r, nil, mem.ErrDenied
		}
		if op.Op == c.OpAdd && (!hedge(op.Content) || op.Epistemic != c.EpInferred) {
			return r, nil, mem.ErrDenied
		}
		for _, tag := range op.Tags {
			if until := s.Cooldowns[tag]; until.After(time.Now().UTC()) {
				return r, nil, mem.ErrDenied
			}
		}
	}
	if r.Episode != nil {
		ep := r.Episode
		if t.Trigger != "session_closed" || s.Session.ClosedAt == nil || ep.SessionID != s.Session.ID || len(strings.Fields(ep.Summary)) > 90 || !clean(ep.Summary) || len(ep.Topics) > 8 || len(ep.FollowUps) > 5 {
			return r, nil, mem.ErrDenied
		}
		first, ok1 := sources[ep.MsgRange[0]]
		last, ok2 := sources[ep.MsgRange[1]]
		if !ok1 || !ok2 || first.SessionID != s.Session.ID || last.SessionID != s.Session.ID || first.Message.CreatedAt.After(last.Message.CreatedAt) {
			return r, nil, mem.ErrDenied
		}
		// Model-suggested followups remain data only; no timers/consent from prose.
	}
	for _, gap := range r.InformationGaps {
		if !clean(gap) || len(strings.Fields(gap)) > 40 || len([]rune(gap)) > 320 {
			return r, nil, mem.ErrDenied
		}
	}
	return r, rejected, nil
}
