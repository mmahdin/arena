// Package brain implements WP-10. Models propose; this package never writes.
package brain

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"sort"
	"strings"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/bundle"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract/schema"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"gopkg.in/yaml.v3"
)

var ErrAssets = errors.New("WP-10 requires complete pinned Brain assets")

type Assets struct {
	synthetic bool
	ref       c.BundleRef
	schemas   *schema.Set
	skills    map[string]bundle.Skill
	protocols map[string]bundle.Protocol
	vocab     schema.Vocabulary
	gate      bundle.PolicyGate
	budgets   bundle.Budgets
	retrieval bundle.Retrieval
	system    []ports.Block
}

type LoadOptions struct{ AllowSyntheticForTests bool }

// Load consumes a WP-03 validated snapshot, never knowledge files directly.
// A synthetic handle is accepted only for isolated tests. No clinical text is
// supplied by code, and the inert WP-03 placeholder cannot satisfy FB-001.
func Load(ctx context.Context, b ports.Bundle, opt LoadOptions) (*Assets, error) {
	if b == nil {
		return nil, ErrAssets
	}
	named, ok := b.(interface{ ArtifactNames() []string })
	if !ok {
		return nil, ErrAssets
	}
	synthetic, ok := b.(interface{ SyntheticOnly() bool })
	if !ok || synthetic.SyntheticOnly() && !opt.AllowSyntheticForTests {
		return nil, ErrAssets
	}
	a := &Assets{synthetic: synthetic.SyntheticOnly(), ref: b.Reference(), skills: map[string]bundle.Skill{}, protocols: map[string]bundle.Protocol{}}
	if a.ref.Version == "" {
		return nil, ErrAssets
	}
	read := func(p string, v any) error {
		raw, e := b.Artifact(ctx, p)
		if e != nil {
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
	for _, entry := range []struct {
		p string
		v any
	}{
		{"config/policy_gate.yaml", &a.gate}, {"config/budgets.yaml", &a.budgets}, {"config/retrieval.yaml", &a.retrieval},
	} {
		if read(entry.p, entry.v) != nil {
			return nil, ErrAssets
		}
	}
	var problem bundle.ProblemUnderstanding
	if read("schemas/problem_understanding.yaml", &problem) != nil {
		return nil, ErrAssets
	}
	for _, s := range problem.Slots {
		a.vocab.Slots = append(a.vocab.Slots, s.ID)
	}
	for _, v := range []struct {
		name string
		dest *[]string
	}{
		{"emotions", &a.vocab.Emotions}, {"rationale_tags", &a.vocab.RationaleTags}, {"avoid_tags", &a.vocab.AvoidTags}, {"hypothesis_tags", &a.vocab.HypothesisTags},
	} {
		var tags bundle.TagVocabulary
		if read("config/vocab/"+v.name+".yaml", &tags) != nil {
			return nil, ErrAssets
		}
		*v.dest = tags.Values
	}
	names := named.ArtifactNames()
	sort.Strings(names)
	a.vocab.Protocols = []string{}
	for _, p := range names {
		if strings.HasPrefix(p, "skills/") && strings.HasSuffix(p, ".yaml") {
			var s bundle.Skill
			if read(p, &s) != nil || s.ID == "" || s.Descriptor == "" {
				return nil, ErrAssets
			}
			a.skills[s.ID] = s
			a.vocab.Skills = append(a.vocab.Skills, s.ID)
		}
		if strings.HasPrefix(p, "protocols/") && strings.HasSuffix(p, ".yaml") {
			var s bundle.Protocol
			if read(p, &s) != nil {
				return nil, ErrAssets
			}
			a.protocols[s.ID] = s
			a.vocab.Protocols = append(a.vocab.Protocols, s.ID)
		}
	}
	if !contains(a.vocab.RationaleTags, "brain_fallback") {
		return nil, ErrAssets
	}
	if _, ok := a.skills["reflect_emotion"]; !ok {
		return nil, ErrAssets
	}
	for _, tag := range []string{"advice", "claiming_recall", "intervention", "invented_recall"} {
		if !contains(a.vocab.AvoidTags, tag) {
			return nil, ErrAssets
		}
	}
	if a.gate.Schema != "policy_gate.v1" || a.budgets.Schema != "budgets.v1" || a.retrieval.Schema != "retrieval.v1" || a.gate.Risk.BrainMayLower || !a.gate.Risk.BrainMayRaise || a.gate.Questions.MaxPerReply < 0 || a.gate.Questions.MaxPerReply > 1 || a.gate.Questions.MaxConsecutiveQuestionTurns < 0 || a.budgets.Tokens.BrainIn < 1 || a.budgets.Tokens.BrainOut < 1 || a.budgets.Turn.BrainMS < 1 || a.budgets.Turn.RetrievalMS < 1 || a.retrieval.Defaults.HardCapWithHop < 1 {
		return nil, ErrAssets
	}
	var e error
	a.schemas, e = schema.New(a.vocab)
	if e != nil {
		return nil, ErrAssets
	}
	for _, v := range []struct{ name, p string }{{"role_and_output", "prompts/brain.system.tmpl"}, {"decision_policy", "policy/decision_policy.md"}, {"communication_digest", "policy/communication_digest.md"}} {
		raw, e := b.Artifact(ctx, v.p)
		if e != nil || strings.TrimSpace(string(raw)) == "" {
			return nil, ErrAssets
		}
		a.system = append(a.system, ports.Block{Name: v.name, Cache: true, Text: "<" + v.name + ">\n" + string(raw) + "\n</" + v.name + ">"})
	}
	objectives := []c.Objective{c.ObjReflect, c.ObjValidateSupport, c.ObjClarify, c.ObjExplore, c.ObjProvideInsight, c.ObjProvideGuidance, c.ObjOfferIntervention, c.ObjContinueIntervention, c.ObjCloseOrPause, c.ObjReviewProgress, c.ObjSummarize, c.ObjRepair, c.ObjAnswerDirect, c.ObjBoundary}
	a.system = append(a.system, staticBlock("objectives", objectives))
	descriptors := []struct {
		ID         string `json:"id"`
		Descriptor string `json:"descriptor"`
	}{}
	for _, id := range a.vocab.Skills {
		descriptors = append(descriptors, struct {
			ID         string `json:"id"`
			Descriptor string `json:"descriptor"`
		}{id, a.skills[id].Descriptor})
	}
	a.system = append(a.system, staticBlock("skills", descriptors), staticBlock("vocab", struct{ Rationale, Avoid, Hypothesis []string }{a.vocab.RationaleTags, a.vocab.AvoidTags, a.vocab.HypothesisTags}))
	return a, nil
}
func staticBlock(name string, v any) ports.Block {
	raw, _ := json.Marshal(v)
	return ports.Block{Name: name, Cache: true, Text: "<" + name + ">" + string(raw) + "</" + name + ">"}
}
func (a *Assets) Reference() c.BundleRef {
	raw, _ := json.Marshal(a.ref)
	var r c.BundleRef
	_ = json.Unmarshal(raw, &r)
	return r
}
func contains[T comparable](s []T, v T) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// Schemas returns the immutable schema set compiled from this pinned bundle.
func (a *Assets) Schemas() *schema.Set { return a.schemas }
