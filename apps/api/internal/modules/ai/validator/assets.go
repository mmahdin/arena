// Package validator implements WP-07's conservative, deterministic output
// gate. Clinical wording, locale matchers and classifier thresholds are loaded
// from a release-verified Knowledge Bundle, not invented by the Go service.
package validator

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/bundle"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"gopkg.in/yaml.v3"
)

var (
	ErrAssets         = errors.New("validator: reviewed assets unavailable or invalid")
	ErrCoverage       = errors.New("validator: locale coverage unavailable")
	ErrClassification = errors.New("validator: classifier unavailable or invalid")
)

// Classifier is optional. When configured, any classifier failure is a hard
// validation error; the caller may try a reviewed fallback but must validate it.
// All keys must be present, finite and in [0,1].
type Classifier interface {
	Scores(ctx context.Context, text string, recent []contract.MsgView, lang string) (map[string]float32, error)
}

type Options struct {
	Classifier Classifier
	// Synthetic fixtures may only be used in isolated tests; NEVER in
	// production composition or with real mental-health data.
	AllowSyntheticForTests bool
	ClassifierTimeout      time.Duration
}

// Reuse WP-03's release-compatible lexicon schema.
type lexicon = bundle.ValidatorLexicon

type rules struct {
	hedges         []string
	questions      []*regexp.Regexp
	clinicalLabels []string
	patterns       map[string][]*regexp.Regexp
}

type Service struct {
	ref               contract.BundleRef
	locales           map[string]rules
	contacts          map[string]struct{}
	thresholds        map[string]float32
	classifier        Classifier
	classifierTimeout time.Duration
	synthetic         bool
}

var _ ports.Validator = (*Service)(nil)

var supportedPatternKeys = map[string]string{
	"diagnosis": "OV-003", "medication": "OV-004", "guarantee": "OV-005",
	"therapist_identity": "OV-006", "directive_advice": "OV-007",
	"internal_leak": "OV-010", "technique": "OV-015",
	"dependency": "OV-016", "invalidating": "OV-017", "crisis_forbidden": "OV-021",
}
var scoreToRule = map[string]string{
	"directive_advice": "OV-007", "certainty_about_inference": "OV-008",
	"dependency_encouraging": "OV-016", "invalidating": "OV-017",
}

// New refuses unknown or unreviewed bundles. A release snapshot must include
// lexicons/validator.<locale>.yaml for each declared locale and approved
// resources/crisis_resources.yaml. This intentionally does not make synthetic
// or incomplete knowledge production-ready.
func New(ctx context.Context, source ports.Bundle, opt Options) (*Service, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if source == nil {
		return nil, ErrAssets
	}
	reviewed, ok := source.(interface{ SyntheticOnly() bool })
	if !ok || (reviewed.SyntheticOnly() && !opt.AllowSyntheticForTests) {
		return nil, ErrAssets
	}
	ref := source.Reference()
	if !strings.HasPrefix(ref.Version, "kb@") {
		return nil, ErrAssets
	}
	var manifest bundle.Manifest
	if err := read(ctx, source, "bundle.yaml", &manifest); err != nil || manifest.Version == "" || len(manifest.Locales) == 0 || "kb@"+manifest.Version != ref.Version {
		return nil, ErrAssets
	}
	if manifest.SyntheticOnly && !opt.AllowSyntheticForTests {
		return nil, ErrAssets
	}
	svc := &Service{ref: ref, locales: map[string]rules{}, contacts: map[string]struct{}{}, thresholds: map[string]float32{}, classifier: opt.Classifier, synthetic: manifest.SyntheticOnly}
	for _, locale := range manifest.Locales {
		if locale == "" || svc.locales[locale].patterns != nil {
			return nil, ErrAssets
		}
		var v lexicon
		if err := read(ctx, source, "lexicons/validator."+locale+".yaml", &v); err != nil || v.Schema != "validator_lexicon.v1" || v.Locale != locale {
			return nil, ErrAssets
		}
		compiled, err := compileRules(v)
		if err != nil {
			return nil, err
		}
		svc.locales[locale] = compiled
	}
	var resources bundle.CrisisResources
	if err := read(ctx, source, "resources/crisis_resources.yaml", &resources); err != nil || resources.Schema != "crisis_resources.v1" || resources.FallbackLocale == "" {
		return nil, ErrAssets
	}
	for _, list := range resources.Locales {
		for _, r := range list {
			if r.ID == "" || r.Contact == "" || r.VerifiedOn == "" {
				return nil, ErrAssets
			}
			if _, err := time.Parse("2006-01-02", r.VerifiedOn); err != nil {
				return nil, ErrAssets
			}
			svc.contacts[normalizeContact(r.Contact)] = struct{}{}
		}
	}
	// The bundle loader verifies clinical approval and validity windows;
	// accept only its immutable snapshot, never arbitrary injected contacts.
	raw, err := source.Artifact(ctx, "config/thresholds.yaml")
	if err != nil || len(raw) == 0 || !utf8.Valid(raw) || len(raw) > 256<<10 {
		return nil, ErrAssets
	}
	var cfg struct {
		Schema    string             `yaml:"schema"`
		Validator map[string]float32 `yaml:"validator"`
	}
	if err := yaml.Unmarshal(raw, &cfg); err != nil || cfg.Schema != "thresholds.v1" {
		return nil, ErrAssets
	}
	for name, val := range cfg.Validator {
		if _, exists := scoreToRule[name]; !exists || math.IsNaN(float64(val)) || math.IsInf(float64(val), 0) || val <= 0 || val > 1 {
			return nil, ErrAssets
		}
		svc.thresholds[name] = val
	}
	if opt.Classifier != nil {
		for key := range scoreToRule {
			if svc.thresholds[key] == 0 {
				return nil, ErrAssets
			}
		}
	}
	if opt.ClassifierTimeout < 0 || opt.ClassifierTimeout > 10*time.Second {
		return nil, ErrAssets
	}
	svc.classifierTimeout = opt.ClassifierTimeout
	if svc.classifierTimeout == 0 {
		svc.classifierTimeout = 800 * time.Millisecond
	}
	return svc, nil
}

func (s *Service) Reference() contract.BundleRef {
	if s == nil {
		return contract.BundleRef{}
	}
	out := contract.BundleRef{Version: s.ref.Version, PromptSHAs: map[string]string{}}
	for k, v := range s.ref.PromptSHAs {
		out.PromptSHAs[k] = v
	}
	return out
}

func read(ctx context.Context, source ports.Bundle, filename string, output any) error {
	b, err := source.Artifact(ctx, filename)
	if err != nil || len(b) == 0 || len(b) > 256<<10 || !utf8.Valid(b) {
		return ErrAssets
	}
	decoder := yaml.NewDecoder(bytes.NewReader(b))
	decoder.KnownFields(true)
	if err := decoder.Decode(output); err != nil {
		return fmt.Errorf("%w: %s", ErrAssets, filename)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return ErrAssets
	}
	return nil
}

func compileRules(v lexicon) (rules, error) {
	out := rules{patterns: map[string][]*regexp.Regexp{}}
	if len(v.Hedges) == 0 || len(v.Questions) == 0 || len(v.ClinicalLabels) == 0 {
		return rules{}, ErrAssets
	}
	// Every static classifier backstop is mandatory, even when the optional
	// LLM classifier has not been configured.
	for key := range supportedPatternKeys {
		if len(v.Patterns[key]) == 0 {
			return rules{}, ErrAssets
		}
	}
	for _, h := range v.Hedges {
		if h == "" || len(h) > 100 {
			return rules{}, ErrAssets
		}
		out.hedges = append(out.hedges, strings.ToLower(strings.TrimSpace(h)))
	}
	for _, label := range v.ClinicalLabels {
		if label == "" || len(label) > 100 {
			return rules{}, ErrAssets
		}
		out.clinicalLabels = append(out.clinicalLabels, strings.ToLower(label))
	}
	compile := func(pattern string) (*regexp.Regexp, error) {
		if len(pattern) == 0 || len(pattern) > 512 {
			return nil, ErrAssets
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, ErrAssets
		}
		return re, nil
	}
	for _, p := range v.Questions {
		re, err := compile(p)
		if err != nil {
			return rules{}, err
		}
		out.questions = append(out.questions, re)
	}
	for kind, pats := range v.Patterns {
		if _, ok := supportedPatternKeys[kind]; !ok {
			return rules{}, ErrAssets
		}
		for _, p := range pats {
			re, err := compile(p)
			if err != nil {
				return rules{}, err
			}
			out.patterns[kind] = append(out.patterns[kind], re)
		}
	}
	return out, nil
}

func normalizeContact(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.TrimSuffix(s, "/")
	if strings.Contains(s, "://") || strings.Contains(s, "@") {
		return s
	}
	return strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, s)
}

func (s *Service) forLocale(locale string) (rules, bool) {
	if s == nil || locale == "" {
		return rules{}, false
	}
	if r, ok := s.locales[locale]; ok {
		return r, true
	}
	base := strings.Split(locale, "-")[0]
	r, ok := s.locales[base]
	return r, ok
}
