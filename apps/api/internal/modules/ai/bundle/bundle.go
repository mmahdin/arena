// Package bundle loads and validates immutable, versioned AI Knowledge Bundles.
// It has no dependency on prototype prompts, the LLM client, or runtime routing.
package bundle

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

const maxFileBytes = 256 << 10
const maxBundleBytes = 4 << 20

var (
	ErrInvalid     = errors.New("invalid knowledge bundle")
	ErrUnavailable = errors.New("knowledge bundle unavailable")
	ErrArtifact    = errors.New("knowledge bundle artifact unavailable")
	idPattern      = regexp.MustCompile(`^[a-z0-9_]+$`)
)

type Issue struct {
	Path    string `json:"path"`
	Code    string `json:"code"`
	Message string `json:"message"`
}
type Report struct {
	Errors   []Issue `json:"errors"`
	Warnings []Issue `json:"warnings"`
}

func (r *Report) fail(file, code, explanation string) {
	r.Errors = append(r.Errors, Issue{file, code, explanation})
}
func (r *Report) warn(file, code, explanation string) {
	r.Warnings = append(r.Warnings, Issue{file, code, explanation})
}
func (r Report) OK() bool { return len(r.Errors) == 0 }
func issueLess(a, b Issue) bool {
	if a.Path != b.Path {
		return a.Path < b.Path
	}
	if a.Code != b.Code {
		return a.Code < b.Code
	}
	return a.Message < b.Message
}

// Manifest is the strict release and compatibility contract in §8.2.
type Manifest struct {
	Name           string            `yaml:"name"`
	Version        string            `yaml:"version"`
	RequiresCore   string            `yaml:"requires_core"`
	EmitsSchemas   map[string]string `yaml:"emits_schemas"`
	Locales        []string          `yaml:"locales"`
	SyntheticOnly  bool              `yaml:"synthetic_only"`
	ClinicalReview struct {
		RequiredFor []string   `yaml:"required_for"`
		Approvals   []Approval `yaml:"approvals"`
	} `yaml:"clinical_review"`
	Checksums struct {
		Algo           string            `yaml:"algo"`
		ManifestSHA256 string            `yaml:"manifest_sha256"`
		Artifacts      map[string]string `yaml:"artifacts"`
	} `yaml:"checksums"`
	Changelog string `yaml:"changelog"`
}
type Approval struct {
	Artifact   string `yaml:"artifact"`
	ReviewerID string `yaml:"reviewer_id"`
	Date       string `yaml:"date"`
	SHA256     string `yaml:"sha256"`
}

// Options controls the linter; all warnings are non-blocking only in synthetic dev checks.
type Options struct {
	CoreVersion string
	Release     bool
	At          time.Time
	TrustedKeys map[string]ed25519.PublicKey
}

// Snapshot owns private copies of every file, including prompt hashes.
// No filesystem writes, mutable maps, or external network references escape it.
type Snapshot struct {
	manifest    Manifest
	artifacts   map[string][]byte
	ref         contract.BundleRef
	fingerprint string
}

var _ ports.Bundle = (*Snapshot)(nil)

func (s *Snapshot) Reference() contract.BundleRef {
	if s == nil {
		return contract.BundleRef{}
	}
	out := contract.BundleRef{Version: s.ref.Version, PromptSHAs: make(map[string]string, len(s.ref.PromptSHAs))}
	for k, v := range s.ref.PromptSHAs {
		out.PromptSHAs[k] = v
	}
	return out
}
func (s *Snapshot) Artifact(ctx context.Context, name string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || !fs.ValidPath(name) || strings.HasPrefix(name, "/") {
		return nil, ErrArtifact
	}
	b, exists := s.artifacts[name]
	if !exists {
		return nil, ErrArtifact
	}
	return append([]byte(nil), b...), nil
}
func (s *Snapshot) Version() string {
	if s == nil {
		return ""
	}
	return s.ref.Version
}
func (s *Snapshot) SyntheticOnly() bool { return s != nil && s.manifest.SyntheticOnly }

// Fingerprint identifies the exact immutable bytes of the pinned snapshot.
func (s *Snapshot) Fingerprint() string {
	if s == nil {
		return ""
	}
	return s.fingerprint
}

// Read validates a directory and builds a snapshot for callers to pin. A read
// does not make the bundle active; activation has independent release gates.
func Read(ctx context.Context, root string, opt Options) (*Snapshot, Report, error) {
	var report Report
	if err := ctx.Err(); err != nil {
		return nil, report, err
	}
	all, err := readFiles(ctx, root)
	if ctx.Err() != nil {
		return nil, report, ctx.Err()
	}
	if err != nil {
		report.fail(".", "bundle_files", err.Error())
		return nil, report, fmt.Errorf("%w: bundle files", ErrInvalid)
	}
	b, ok := all["bundle.yaml"]
	if !ok {
		report.fail("bundle.yaml", "manifest_missing", "bundle manifest required")
		return nil, report, ErrInvalid
	}
	var m Manifest
	if err := strictYAML(b, &m); err != nil {
		report.fail("bundle.yaml", "manifest_schema", err.Error())
		return nil, report, ErrInvalid
	}
	validateBundle(ctx, all, &m, opt, &report)
	sort.Slice(report.Errors, func(i, j int) bool { return issueLess(report.Errors[i], report.Errors[j]) })
	sort.Slice(report.Warnings, func(i, j int) bool { return issueLess(report.Warnings[i], report.Warnings[j]) })
	if ctx.Err() != nil {
		return nil, report, ctx.Err()
	}
	if !report.OK() {
		return nil, report, ErrInvalid
	}
	ref := contract.BundleRef{Version: "kb@" + m.Version, PromptSHAs: map[string]string{}}
	for p, data := range all {
		if strings.HasPrefix(p, "prompts/") {
			ref.PromptSHAs[p] = hexSHA(data)
		}
	}
	h := sha256.New()
	names := sortedKeys(all)
	for _, name := range names {
		h.Write([]byte(name))
		h.Write([]byte{0})
		h.Write(all[name])
		h.Write([]byte{0})
	}
	// Golden scenarios contribute to release integrity but are not loaded or
	// exposed as runtime prompt/knowledge assets (WP-15 owns their execution).
	runtimeFiles := make(map[string][]byte, len(all))
	for p, b := range all {
		if !strings.HasPrefix(p, "evals/golden/") {
			runtimeFiles[p] = b
		}
	}
	return &Snapshot{manifest: m, artifacts: runtimeFiles, ref: ref, fingerprint: hex.EncodeToString(h.Sum(nil))}, report, nil
}
func hexSHA(b []byte) string { sum := sha256.Sum256(b); return hex.EncodeToString(sum[:]) }
func sortedKeys[V any](items map[string]V) []string {
	names := make([]string, 0, len(items))
	for name := range items {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func readFiles(ctx context.Context, root string) (map[string][]byte, error) {
	info, err := os.Lstat(root)
	if err != nil {
		return nil, err
	}
	if info.Mode()&fs.ModeSymlink != 0 || !info.IsDir() {
		return nil, fmt.Errorf("bundle root must be a real directory")
	}
	files := map[string][]byte{}
	var total int64
	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}
		if !fs.ValidPath(rel) {
			return fmt.Errorf("unsafe bundle path")
		}
		if d.Type()&fs.ModeSymlink != 0 {
			return fmt.Errorf("symlink not permitted in bundle: %s", rel)
		}
		if d.IsDir() {
			return nil
		}
		if !d.Type().IsRegular() {
			return fmt.Errorf("nonregular artifact: %s", rel)
		}
		if !allowedPath(rel) {
			return fmt.Errorf("unrecognized bundle artifact: %s", rel)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.Size() > maxFileBytes {
			return fmt.Errorf("oversized artifact: %s", rel)
		}
		total += info.Size()
		if total > maxBundleBytes {
			return fmt.Errorf("bundle exceeds limit")
		}
		contents, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if int64(len(contents)) > maxFileBytes {
			return fmt.Errorf("oversized artifact: %s", rel)
		}
		if !utf8.Valid(contents) || strings.ContainsRune(string(contents), 0) {
			return fmt.Errorf("invalid UTF-8 or NUL: %s", rel)
		}
		files[rel] = contents
		return nil
	})
	return files, err
}
func allowedPath(p string) bool {
	if p == "bundle.yaml" || p == "CHANGELOG.md" || p == "README.md" || p == "bundle.sig" {
		return true
	}
	exact := map[string]bool{
		"policy/communication_policy.md": true, "policy/communication_digest.md": true, "policy/decision_policy.md": true,
		"policy/boundary_policy.yaml": true, "policy/safety_policy.yaml": true, "policy/continuity_policy.yaml": true,
		"schemas/user_information.yaml": true, "schemas/problem_understanding.yaml": true,
		"schemas/formulation_cbt.yaml": true, "schemas/formulation_five_p.yaml": true,
		"templates/fallback_by_objective.yaml": true, "templates/crisis_templates.yaml": true,
		"templates/protocol_offer.yaml": true, "templates/boundary_scripts.yaml": true,
		"resources/crisis_resources.yaml": true,
		"config/router.yaml":              true, "config/policy_gate.yaml": true, "config/retrieval.yaml": true,
		"config/budgets.yaml": true, "config/thresholds.yaml": true, "config/memory.yaml": true,
		"config/models.yaml": true, "config/sampling.yaml": true,
	}
	if exact[p] {
		return true
	}
	for _, prefix := range []string{"skills/", "protocols/", "lexicons/", "evals/golden/"} {
		if strings.HasPrefix(p, prefix) && !strings.Contains(strings.TrimPrefix(p, prefix), "/") && strings.HasSuffix(p, ".yaml") {
			return true
		}
	}
	if strings.HasPrefix(p, "config/vocab/") && !strings.Contains(strings.TrimPrefix(p, "config/vocab/"), "/") {
		for _, name := range []string{"emotions", "rationale_tags", "avoid_tags", "hypothesis_tags", "memory_tags", "problem_tags", "skill_tags", "protocol_flags", "protocol_predicates", "forbidden_claims"} {
			if p == "config/vocab/"+name+".yaml" {
				return true
			}
		}
	}
	for _, base := range []string{"communication_policy", "communication_digest", "decision_policy"} {
		prefix := "policy/" + base + "."
		if strings.HasPrefix(p, prefix) && strings.HasSuffix(p, ".md") && localeRE.MatchString(strings.TrimSuffix(strings.TrimPrefix(p, prefix), ".md")) {
			return true
		}
	}
	for _, name := range []string{"brain.system.tmpl", "generator.system.tmpl", "generator.crisis.tmpl", "memory_writer.system.tmpl", "reflection.system.tmpl", "safety_l1.system.tmpl", "validator_classifier.system.tmpl"} {
		if p == "prompts/"+name {
			return true
		}
	}

	return false
}
func checkRequired(files map[string][]byte, report *Report) {
	required := []string{"bundle.yaml", "CHANGELOG.md", "policy/communication_policy.md", "policy/communication_digest.md", "policy/decision_policy.md", "policy/boundary_policy.yaml", "policy/safety_policy.yaml", "policy/continuity_policy.yaml", "schemas/user_information.yaml", "schemas/problem_understanding.yaml", "schemas/formulation_cbt.yaml", "schemas/formulation_five_p.yaml", "templates/fallback_by_objective.yaml", "templates/crisis_templates.yaml", "templates/protocol_offer.yaml", "templates/boundary_scripts.yaml", "resources/crisis_resources.yaml", "config/router.yaml", "config/policy_gate.yaml", "config/retrieval.yaml", "config/budgets.yaml", "config/thresholds.yaml", "config/memory.yaml", "config/models.yaml", "config/sampling.yaml"}
	for _, p := range []string{"brain.system.tmpl", "generator.system.tmpl", "generator.crisis.tmpl", "memory_writer.system.tmpl", "reflection.system.tmpl", "safety_l1.system.tmpl", "validator_classifier.system.tmpl"} {
		required = append(required, "prompts/"+p)
	}
	for _, p := range []string{"emotions", "rationale_tags", "avoid_tags", "hypothesis_tags", "memory_tags", "problem_tags", "skill_tags", "protocol_flags", "protocol_predicates", "forbidden_claims"} {
		required = append(required, "config/vocab/"+p+".yaml")
	}
	for _, p := range required {
		if _, ok := files[p]; !ok {
			report.fail(p, "missing", "required bundle artifact missing")
		}
	}
	for _, prefix := range []string{"skills/", "protocols/", "lexicons/safety_l0."} {
		seen := false
		for p := range files {
			if strings.HasPrefix(p, prefix) && strings.HasSuffix(p, ".yaml") {
				seen = true
				break
			}
		}
		if !seen {
			report.fail(prefix, "missing", "at least one artifact required")
		}
	}
}

// Registry publishes whole snapshots with a single atomic pointer swap. A turn
// holds its snapshot even after rollback, preventing mixed-version reads.
type Registry struct {
	mu             sync.Mutex
	active         *Snapshot
	versions       map[string]*Snapshot
	opt            Options
	allowSynthetic bool
}

var _ ports.BundleRegistry = (*Registry)(nil)

func NewRegistry(opt Options, allowSynthetic bool) *Registry {
	trusted := make(map[string]ed25519.PublicKey, len(opt.TrustedKeys))
	for id, key := range opt.TrustedKeys {
		trusted[id] = append(ed25519.PublicKey(nil), key...)
	}
	opt.TrustedKeys = trusted
	return &Registry{opt: opt, versions: map[string]*Snapshot{}, allowSynthetic: allowSynthetic}
}
func (r *Registry) Active(ctx context.Context, _ contract.ID) (ports.Bundle, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.Lock()
	snap := r.active
	r.mu.Unlock()
	if snap == nil {
		return nil, ErrUnavailable
	}
	return snap, nil
}
func (r *Registry) Activate(ctx context.Context, root string) (Report, error) {
	opt := r.opt
	if !r.allowSynthetic {
		opt.Release = true
	}
	snap, report, err := Read(ctx, root, opt)
	if err != nil {
		return report, err
	}
	if snap.SyntheticOnly() && !r.allowSynthetic {
		report.fail("bundle.yaml", "synthetic_activation", "synthetic bundle cannot be activated")
		return report, ErrInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if prev, ok := r.versions[snap.Version()]; ok && prev.fingerprint != snap.fingerprint {
		report.fail("bundle.yaml", "immutable_version", "bundle version already registered with different content")
		return report, ErrInvalid
	}
	r.versions[snap.Version()] = snap
	r.active = snap
	return report, nil
}

// ByVersion allows reproducible replay of snapshots that remain in this process.
func (r *Registry) ByVersion(ctx context.Context, version string) (ports.Bundle, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.Lock()
	snap := r.versions[version]
	r.mu.Unlock()
	if snap == nil {
		return nil, ErrUnavailable
	}
	return snap, nil
}
func (r *Registry) Rollback(ctx context.Context, version string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	snap := r.versions[version]
	if snap == nil {
		return ErrUnavailable
	}
	r.active = snap
	return nil
}

// Helper used only for normalized in-bundle lookups; path traversal is refused.
func basenameID(p string) string { return strings.TrimSuffix(path.Base(p), ".yaml") }
