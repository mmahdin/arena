package memory

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/bundle"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract/schema"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"gopkg.in/yaml.v3"
)

type Assets struct {
	ref           c.BundleRef
	synthetic     bool
	config        bundle.MemoryConfig
	thresholds    bundle.Thresholds
	budgets       bundle.Budgets
	info          bundle.UserInformation
	schemas       *schema.Set
	prompt        string
	tags, hypTags []string
	locales       map[string]memoryLexicon
}
type memoryLexicon struct {
	hedges  []*regexp.Regexp
	markers []string
	labels  []string
}
type LoadOptions struct{ AllowSyntheticForTests bool }

func Load(ctx context.Context, b ports.Bundle, opt LoadOptions) (*Assets, error) {
	if b == nil {
		return nil, ErrAssets
	}
	reviewed, ok := b.(interface{ SyntheticOnly() bool })
	if !ok || reviewed.SyntheticOnly() && !opt.AllowSyntheticForTests {
		return nil, ErrAssets
	}
	named, ok := b.(interface{ ArtifactNames() []string })
	if !ok {
		return nil, ErrAssets
	}
	a := &Assets{ref: b.Reference(), synthetic: reviewed.SyntheticOnly(), locales: map[string]memoryLexicon{}}
	var manifest bundle.Manifest
	var problem bundle.ProblemUnderstanding
	for _, entry := range []struct {
		p string
		v any
	}{{"bundle.yaml", &manifest}, {"config/memory.yaml", &a.config}, {"config/thresholds.yaml", &a.thresholds}, {"config/budgets.yaml", &a.budgets}, {"schemas/user_information.yaml", &a.info}, {"schemas/problem_understanding.yaml", &problem}} {
		if readAsset(ctx, b, entry.p, entry.v) != nil {
			return nil, ErrAssets
		}
	}
	if "kb@"+manifest.Version != a.ref.Version || manifest.SyntheticOnly != a.synthetic || manifest.ValidateLanguages() != nil || a.config.Schema != "memory.v1" || !a.config.RequireEvidence || !a.config.PreserveDerivedOnSoftDelete || a.config.AllowHardDelete || len(a.config.AllowedKinds) == 0 || a.info.Schema != "userinfo.v1" || a.thresholds.Schema != "thresholds.v1" {
		return nil, ErrAssets
	}
	t := a.thresholds
	if t.Memory.DedupeCosineGTE <= 0 || t.Memory.DedupeCosineGTE > 1 || t.Memory.StaleAfterDays < 1 || t.Memory.HotSetMaxItems < 1 || t.Hypothesis.SupportedMinSupport < 2 || t.Hypothesis.SupportedMinSessions < 2 || t.Hypothesis.RejectionCooldownDays < 1 || t.Hypothesis.StaleAfterDays < 1 {
		return nil, ErrAssets
	}
	vocab := schema.Vocabulary{Protocols: []string{}}
	for _, slot := range problem.Slots {
		vocab.Slots = append(vocab.Slots, slot.ID)
	}
	for _, v := range []struct {
		name string
		dest *[]string
	}{{"emotions", &vocab.Emotions}, {"rationale_tags", &vocab.RationaleTags}, {"avoid_tags", &vocab.AvoidTags}, {"hypothesis_tags", &vocab.HypothesisTags}, {"memory_tags", &a.tags}} {
		var tags bundle.TagVocabulary
		if readAsset(ctx, b, "config/vocab/"+v.name+".yaml", &tags) != nil || len(tags.Values) == 0 {
			return nil, ErrAssets
		}
		*v.dest = tags.Values
	}
	a.hypTags = append([]string(nil), vocab.HypothesisTags...)
	for _, p := range named.ArtifactNames() {
		if strings.HasPrefix(p, "skills/") && strings.HasSuffix(p, ".yaml") {
			var s bundle.Skill
			if readAsset(ctx, b, p, &s) != nil {
				return nil, ErrAssets
			}
			vocab.Skills = append(vocab.Skills, s.ID)
		}
	}
	var err error
	a.schemas, err = schema.New(vocab)
	if err != nil {
		return nil, ErrAssets
	}
	for _, locale := range manifest.Conversations() {
		var lex bundle.ValidatorLexicon
		if readAsset(ctx, b, "lexicons/validator."+locale+".yaml", &lex) != nil || lex.Schema != "validator_lexicon.v1" || lex.Locale != locale || len(lex.Hedges) == 0 || len(lex.ClinicalLabels) == 0 {
			return nil, ErrAssets
		}
		compiled := memoryLexicon{labels: lex.ClinicalLabels, markers: lex.Hedges}
		for _, h := range lex.Hedges {
			if h == "" {
				return nil, ErrAssets
			}
			re, e := regexp.Compile(`(?i)(?:^|[\s\pP])` + regexp.QuoteMeta(h) + `(?:$|[\s\pP])`)
			if e != nil {
				return nil, ErrAssets
			}
			compiled.hedges = append(compiled.hedges, re)
		}
		a.locales[locale] = compiled
	}
	raw, err := b.Artifact(ctx, "prompts/memory_writer.system.tmpl")
	if err != nil || !utf8.Valid(raw) || strings.TrimSpace(string(raw)) == "" || strings.Contains(string(raw), "SYNTHETIC PLACEHOLDER") || estimate(string(raw)) > 700 {
		return nil, ErrAssets
	}
	a.prompt = string(raw)
	return a, nil
}
func readAsset(ctx context.Context, b ports.Bundle, p string, v any) error {
	raw, e := b.Artifact(ctx, p)
	if e != nil || len(raw) > 256<<10 || !utf8.Valid(raw) {
		return ErrAssets
	}
	d := yaml.NewDecoder(bytes.NewReader(raw))
	d.KnownFields(true)
	if d.Decode(v) != nil {
		return ErrAssets
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return ErrAssets
	}
	return nil
}
func (a *Assets) Reference() c.BundleRef {
	raw, _ := json.Marshal(a.ref)
	var ref c.BundleRef
	_ = json.Unmarshal(raw, &ref)
	return ref
}
func (a *Assets) lex(lang string) (memoryLexicon, bool) {
	l, ok := a.locales[lang]
	if !ok {
		l, ok = a.locales[strings.Split(lang, "-")[0]]
	}
	return l, ok
}
func (a *Assets) validTag(tag string) bool {
	if contains(a.tags, tag) || contains(a.hypTags, tag) {
		return true
	}
	for _, f := range a.info.Fields {
		if f.ID == tag {
			return true
		}
	}
	return false
}
func (a *Assets) attribute(tags []string) string {
	for _, f := range a.info.Fields {
		if contains(tags, f.ID) {
			return f.ID
		}
	}
	return ""
}
func contains[T comparable](xs []T, v T) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}
func uniqueIDs(ids []c.ID) []c.ID {
	out := []c.ID{}
	seen := map[c.ID]bool{}
	for _, id := range ids {
		if !seen[id] {
			out = append(out, id)
			seen[id] = true
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].String() < out[j].String() })
	return out
}
