package reflection

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"slices"
	"sort"
	"strings"
	"unicode/utf8"

	mem "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/bundle"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract/schema"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"gopkg.in/yaml.v3"
)

type Assets struct {
	ref        c.BundleRef
	synthetic  bool
	prompt     string
	thresholds bundle.Thresholds
	budgets    bundle.Budgets
	frameworks map[string]bundle.Formulation
	lex        map[string]bundle.ValidatorLexicon
	schemas    *schema.Set
}
type LoadOptions struct{ AllowSyntheticForTests bool }

func read(ctx context.Context, b ports.Bundle, name string, v any) error {
	raw, e := b.Artifact(ctx, name)
	if e != nil || len(raw) > 256<<10 || !utf8.Valid(raw) {
		return mem.ErrAssets
	}
	d := yaml.NewDecoder(bytes.NewReader(raw))
	d.KnownFields(true)
	if d.Decode(v) != nil {
		return mem.ErrAssets
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return mem.ErrAssets
	}
	return nil
}
func Load(ctx context.Context, b ports.Bundle, opt LoadOptions) (*Assets, error) {
	reviewed, ok := b.(interface{ SyntheticOnly() bool })
	if !ok || reviewed.SyntheticOnly() && !opt.AllowSyntheticForTests {
		return nil, mem.ErrAssets
	}
	names, ok := b.(interface{ ArtifactNames() []string })
	if !ok {
		return nil, mem.ErrAssets
	}
	a := &Assets{ref: b.Reference(), synthetic: reviewed.SyntheticOnly(), frameworks: map[string]bundle.Formulation{}, lex: map[string]bundle.ValidatorLexicon{}}
	var manifest bundle.Manifest
	var problem bundle.ProblemUnderstanding
	for _, x := range []struct {
		p string
		v any
	}{{"bundle.yaml", &manifest}, {"config/thresholds.yaml", &a.thresholds}, {"config/budgets.yaml", &a.budgets}, {"schemas/problem_understanding.yaml", &problem}} {
		if read(ctx, b, x.p, x.v) != nil {
			return nil, mem.ErrAssets
		}
	}
	if "kb@"+manifest.Version != a.ref.Version || manifest.SyntheticOnly != a.synthetic || len(manifest.Locales) == 0 || a.thresholds.Schema != "thresholds.v1" || a.budgets.Quotas.ReflectionPerUserPerDay < 1 {
		return nil, mem.ErrAssets
	}
	p := a.thresholds.Reflection
	if p.MinEvidenceForFormulation.Sessions < 2 || p.MinEvidenceForFormulation.Situations < 3 || p.Triggers.MinNewItems < 1 || p.Triggers.ThreadTurnsGTE < 1 || p.Triggers.ContradictionEventsGTE < 1 {
		return nil, mem.ErrAssets
	}
	vocab := schema.Vocabulary{Protocols: []string{}, Frameworks: map[string][]string{}}
	for _, s := range problem.Slots {
		vocab.Slots = append(vocab.Slots, s.ID)
	}
	for _, x := range []struct {
		name string
		dest *[]string
	}{{"emotions", &vocab.Emotions}, {"rationale_tags", &vocab.RationaleTags}, {"avoid_tags", &vocab.AvoidTags}, {"hypothesis_tags", &vocab.HypothesisTags}} {
		var v bundle.TagVocabulary
		if read(ctx, b, "config/vocab/"+x.name+".yaml", &v) != nil {
			return nil, mem.ErrAssets
		}
		*x.dest = v.Values
	}
	for _, name := range names.ArtifactNames() {
		if strings.HasPrefix(name, "skills/") && strings.HasSuffix(name, ".yaml") {
			var sk bundle.Skill
			if read(ctx, b, name, &sk) != nil {
				return nil, mem.ErrAssets
			}
			vocab.Skills = append(vocab.Skills, sk.ID)
		}
		if strings.HasPrefix(name, "schemas/formulation_") && strings.HasSuffix(name, ".yaml") {
			var f bundle.Formulation
			if read(ctx, b, name, &f) != nil || f.Schema != "formulation.v1" || f.Framework == "" || len(f.Fields) == 0 || f.Paragraph.MaxWords < 1 || f.Paragraph.MaxWords > 90 || !f.Paragraph.MustMarkInferred {
				return nil, mem.ErrAssets
			}
			if _, exists := a.frameworks[f.Framework]; exists {
				return nil, mem.ErrAssets
			}
			for _, field := range f.Fields {
				if field.ID == "" || field.MinEvidence < 1 || len(field.AllowedStatus) == 0 {
					return nil, mem.ErrAssets
				}
				for _, st := range field.AllowedStatus {
					if st != "reported" && st != "inferred" {
						return nil, mem.ErrAssets
					}
				}
				vocab.Frameworks[f.Framework] = append(vocab.Frameworks[f.Framework], field.ID)
			}
			a.frameworks[f.Framework] = f
		}
	}
	if len(a.frameworks) == 0 {
		return nil, mem.ErrAssets
	}
	for _, lang := range manifest.Locales {
		var l bundle.ValidatorLexicon
		if read(ctx, b, "lexicons/validator."+lang+".yaml", &l) != nil || l.Locale != lang || len(l.Hedges) == 0 || len(l.ClinicalLabels) == 0 {
			return nil, mem.ErrAssets
		}
		a.lex[lang] = l
	}
	var e error
	a.schemas, e = schema.New(vocab)
	if e != nil {
		return nil, mem.ErrAssets
	}
	raw, e := b.Artifact(ctx, "prompts/reflection.system.tmpl")
	if e != nil || len(raw) == 0 || !utf8.Valid(raw) || strings.Contains(string(raw), "SYNTHETIC PLACEHOLDER") || estimate(string(raw)) > 600 {
		return nil, mem.ErrAssets
	}
	a.prompt = string(raw)
	return a, nil
}
func (a *Assets) Reference() c.BundleRef {
	raw, _ := json.Marshal(a.ref)
	var out c.BundleRef
	_ = json.Unmarshal(raw, &out)
	return out
}
func (a *Assets) Framework(t c.ThreadRecord) (string, error) {
	names := []string{}
	for name, f := range a.frameworks {
		for _, tag := range t.ProblemTags {
			if slices.Contains(f.SelectWhen.ProblemTagsAny, tag) {
				names = append(names, name)
				break
			}
		}
	}
	sort.Strings(names)
	if len(names) != 1 {
		return "", mem.ErrAssets
	}
	return names[0], nil
}
func (a *Assets) lexicon(lang string) (bundle.ValidatorLexicon, bool) {
	l, ok := a.lex[lang]
	if !ok {
		l, ok = a.lex[strings.Split(lang, "-")[0]]
	}
	return l, ok
}
func estimate(s string) int { return (utf8.RuneCountInString(s) + 2) / 3 }
