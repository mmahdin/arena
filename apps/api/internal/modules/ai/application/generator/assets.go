// Package generator implements WP-11. It realizes a gated plan; it never
// selects an objective, invents clinical content, or writes conversation state.
package generator

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/bundle"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/safety"
	"gopkg.in/yaml.v3"
)

var ErrAssets = errors.New("generator: complete reviewed pinned assets required")

type LoadOptions struct {
	AllowSyntheticForTests bool
	Now                    func() time.Time
}

// Assets owns copies of a release-verified WP-03 snapshot. No file-system or
// experimental response implementation is consulted at generation time.
type Assets struct {
	ref                         c.BundleRef
	synthetic                   bool
	locales                     []string
	system, crisis              string
	communication, prohibitions string
	skills                      map[string]bundle.Skill
	protocols                   map[string]bundle.Protocol
	offers                      bundle.ProtocolOfferTexts
	avoids                      []string
	budgets                     bundle.Budgets
	policy                      bundle.SafetyPolicy
	planner                     *safety.Planner
}

func Load(ctx context.Context, source ports.Bundle, opt LoadOptions) (*Assets, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if source == nil {
		return nil, ErrAssets
	}
	named, ok := source.(interface{ ArtifactNames() []string })
	if !ok {
		return nil, ErrAssets
	}
	approved, ok := source.(interface{ SyntheticOnly() bool })
	if !ok || approved.SyntheticOnly() && !opt.AllowSyntheticForTests {
		return nil, ErrAssets
	}
	a := &Assets{ref: source.Reference(), synthetic: approved.SyntheticOnly(), skills: map[string]bundle.Skill{}, protocols: map[string]bundle.Protocol{}}
	var manifest bundle.Manifest
	var boundary bundle.BoundaryPolicy
	var avoids bundle.TagVocabulary
	for _, v := range []struct {
		name  string
		value any
	}{
		{"bundle.yaml", &manifest}, {"config/budgets.yaml", &a.budgets},
		{"policy/boundary_policy.yaml", &boundary}, {"config/vocab/avoid_tags.yaml", &avoids},
		{"policy/safety_policy.yaml", &a.policy},
	} {
		if read(ctx, source, v.name, v.value) != nil {
			return nil, ErrAssets
		}
	}
	if "kb@"+manifest.Version != a.ref.Version || len(manifest.Locales) == 0 || manifest.SyntheticOnly != a.synthetic || boundary.Schema != "boundary_policy.v1" || len(boundary.Modes) == 0 || len(avoids.Values) == 0 {
		return nil, ErrAssets
	}
	a.locales = append([]string(nil), manifest.Locales...)
	a.avoids = avoids.Values
	b := a.budgets
	if b.Schema != "budgets.v1" || b.Tokens.GenIn < 1 || b.Tokens.GenOut < 1 || b.Tokens.GenOut > 8192 || b.Turn.GenTotalMS < 1 || b.Turn.GenFirstTokenMS < 1 || b.Turn.GenFirstTokenMS > b.Turn.GenTotalMS || b.Turn.TotalMS < b.Turn.GenTotalMS || b.Limits.MaxRegenerations < 0 || b.Limits.MaxRegenerations > 1 || b.Limits.MaxSyncLLMCalls < 1 {
		return nil, ErrAssets
	}
	for _, v := range []struct {
		path string
		dst  *string
		cap  int
	}{
		{"prompts/generator.system.tmpl", &a.system, 400},
		{"policy/communication_policy.md", &a.communication, 600},
		{"prompts/generator.crisis.tmpl", &a.crisis, 600},
	} {
		raw, err := source.Artifact(ctx, v.path)
		if err != nil || !utf8.Valid(raw) || strings.TrimSpace(string(raw)) == "" || strings.Contains(string(raw), "SYNTHETIC PLACEHOLDER") || estimate(string(raw)) > v.cap {
			return nil, ErrAssets
		}
		*v.dst = string(raw)
	}
	// These are the existing OV contract's prohibitions, not new clinical
	// policies. Additional forbidden content comes verbatim from the bundle.
	a.prohibitions = "OV-001/002: obey question limits. OV-003/004: no diagnosis or medication advice. OV-005/006: no guarantees or therapist identity. OV-007: obey must_avoid. OV-008/009: preserve epistemic status and provenance. OV-010/011: no internal handles or unapproved contacts. OV-014/015: only planned techniques and required content. OV-016/017: no dependency encouragement or invalidation. OV-020/021: obey crisis requirements. Output plain text, no JSON or markdown headings. Use <<PLAN_UNSAFE>> only as the entire output.\n"
	for _, mode := range boundary.Modes {
		if mode.ID == "" || len(mode.ForbiddenContent) == 0 {
			return nil, ErrAssets
		}
		a.prohibitions += mode.ID + ": " + strings.Join(mode.ForbiddenContent, ", ") + "\n"
	}
	if estimate(a.prohibitions) > 250 {
		return nil, ErrAssets
	}
	names := append([]string(nil), named.ArtifactNames()...)
	sort.Strings(names)
	for _, name := range names {
		if strings.HasPrefix(name, "skills/") && strings.HasSuffix(name, ".yaml") {
			var skill bundle.Skill
			if read(ctx, source, name, &skill) != nil || skill.Schema != "skillcard.v1" || skill.ID == "" || strings.TrimSpace(skill.CardText) == "" {
				return nil, ErrAssets
			}
			if _, exists := a.skills[skill.ID]; exists {
				return nil, ErrAssets
			}
			a.skills[skill.ID] = skill
		}
		if strings.HasPrefix(name, "protocols/") && strings.HasSuffix(name, ".yaml") {
			var p bundle.Protocol
			if read(ctx, source, name, &p) != nil || p.ID == "" {
				return nil, ErrAssets
			}
			if _, exists := a.protocols[p.ID]; exists {
				return nil, ErrAssets
			}
			a.protocols[p.ID] = p
		}
	}
	if len(a.skills) == 0 {
		return nil, ErrAssets
	}
	if len(a.protocols) > 0 && (read(ctx, source, "templates/protocol_offer.yaml", &a.offers) != nil || a.offers.Schema != "protocol_offer.v1") {
		return nil, ErrAssets
	}
	sa, err := safety.Load(ctx, source, safety.LoadOptions{AllowSyntheticForTests: opt.AllowSyntheticForTests})
	if err != nil {
		return nil, ErrAssets
	}
	a.planner, err = safety.NewPlanner(sa, opt.Now)
	if err != nil {
		return nil, ErrAssets
	}
	return a, nil
}

func read(ctx context.Context, source ports.Bundle, name string, dest any) error {
	raw, err := source.Artifact(ctx, name)
	if err != nil || len(raw) == 0 || len(raw) > 256<<10 || !utf8.Valid(raw) {
		return ErrAssets
	}
	d := yaml.NewDecoder(bytes.NewReader(raw))
	d.KnownFields(true)
	if d.Decode(dest) != nil {
		return ErrAssets
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return ErrAssets
	}
	return nil
}
func (a *Assets) Reference() c.BundleRef {
	if a == nil {
		return c.BundleRef{}
	}
	raw, _ := json.Marshal(a.ref)
	var ref c.BundleRef
	_ = json.Unmarshal(raw, &ref)
	return ref
}
func (a *Assets) locale(lang string) (string, bool) {
	for _, l := range a.locales {
		if l == lang {
			return l, true
		}
	}
	base := strings.Split(lang, "-")[0]
	for _, l := range a.locales {
		if l == base {
			return l, true
		}
	}
	return "", false
}
func includes[T comparable](list []T, value T) bool {
	for _, v := range list {
		if v == value {
			return true
		}
	}
	return false
}
