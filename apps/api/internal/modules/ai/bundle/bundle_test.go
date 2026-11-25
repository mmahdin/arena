package bundle

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

func fixtureDir(t *testing.T) string {
	t.Helper()
	root := filepath.Join("..", "..", "..", "..", "..", "..", "knowledge")
	if _, err := os.Stat(filepath.Join(root, "bundle.yaml")); err != nil {
		t.Fatal(err)
	}
	return root
}
func copyFixture(t *testing.T) string {
	t.Helper()
	source := fixtureDir(t)
	dest := t.TempDir()
	files, err := readFiles(context.Background(), source)
	if err != nil {
		t.Fatal(err)
	}
	for name, b := range files {
		path := filepath.Join(dest, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	return dest
}
func rewrite(t *testing.T, root, name, from, to string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(name))
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(b, []byte(from)) {
		t.Fatalf("fixture %s lacks %q", name, from)
	}
	b = bytes.Replace(b, []byte(from), []byte(to), 1)
	if err := os.WriteFile(p, b, 0600); err != nil {
		t.Fatal(err)
	}
}
func containsCode(issues []Issue, code string) bool {
	for _, item := range issues {
		if item.Code == code {
			return true
		}
	}
	return false
}
func TestSyntheticReadAndSnapshotIsolation(t *testing.T) {
	ctx := context.Background()
	snapshot, report, err := Read(ctx, fixtureDir(t), Options{CoreVersion: "1.7.1"})
	if err != nil || !report.OK() {
		t.Fatalf("load synthetic fixture: %v %+v", err, report.Errors)
	}
	if snapshot.Version() != "kb@0.1.0" || !snapshot.SyntheticOnly() || snapshot.Fingerprint() == "" {
		t.Fatalf("unexpected snapshot metadata")
	}
	if !containsCode(report.Warnings, "synthetic_only") {
		t.Fatalf("expected non-production warnings: %+v", report.Warnings)
	}
	ref := snapshot.Reference()
	if len(ref.PromptSHAs) != 8 {
		t.Fatalf("want eight prompt SHAs, got %d", len(ref.PromptSHAs))
	}
	original := ref.PromptSHAs["prompts/unified.system.tmpl"]
	ref.PromptSHAs["prompts/unified.system.tmpl"] = "fake"
	if snapshot.Reference().PromptSHAs["prompts/unified.system.tmpl"] != original {
		t.Fatal("mutable prompt SHA escaped")
	}
	first, err := snapshot.Artifact(ctx, "policy/communication_policy.md")
	if err != nil || len(first) == 0 {
		t.Fatal(err)
	}
	first[0] ^= 0xff
	second, _ := snapshot.Artifact(ctx, "policy/communication_policy.md")
	if bytes.Equal(first, second) {
		t.Fatal("artifact bytes aliased internal snapshot")
	}
	for _, path := range []string{"../bundle.yaml", "/bundle.yaml", "prompts/../bundle.yaml", "missing.yaml", "evals/golden/synthetic_envelope.yaml"} {
		if _, err := snapshot.Artifact(ctx, path); !errors.Is(err, ErrArtifact) {
			t.Fatalf("expected artifact isolation for %q, got %v", path, err)
		}
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := snapshot.Artifact(cancelled, "bundle.yaml"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
func TestRegistryPinRollbackAndFailSafe(t *testing.T) {
	root := copyFixture(t)
	r := NewRegistry(Options{CoreVersion: "1.0.0"}, true)
	if _, err := r.Active(context.Background(), uuid.New()); !errors.Is(err, ErrUnavailable) {
		t.Fatal("registry should start empty")
	}
	if report, err := r.Activate(context.Background(), root); err != nil || !report.OK() {
		t.Fatalf("activate: %v %+v", err, report.Errors)
	}
	pinned, err := r.Active(context.Background(), uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	pinnedBytes, _ := pinned.Artifact(context.Background(), "prompts/unified.system.tmpl")
	rewrite(t, root, "config/memory.yaml", "allow_hard_delete: false", "allow_hard_delete: true")
	if report, err := r.Activate(context.Background(), root); err == nil || !containsCode(report.Errors, "soft_delete") {
		t.Fatalf("unsafe update accepted: %+v %v", report.Errors, err)
	}
	active, _ := r.Active(context.Background(), uuid.New())
	if active.Reference().Version != "kb@0.1.0" {
		t.Fatal("failed activation changed pointer")
	}
	rewrite(t, root, "config/memory.yaml", "allow_hard_delete: true", "allow_hard_delete: false")
	rewrite(t, root, "bundle.yaml", "version: 0.1.0", "version: 0.2.0")
	// Change the prompt bytes without coupling this registry test to persona wording.
	promptPath := filepath.Join(root, "prompts", "unified.system.tmpl")
	promptBytes, err := os.ReadFile(promptPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(promptPath, append(promptBytes, []byte("\n# Synthetic snapshot revision\n")...), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Activate(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	next, _ := r.Active(context.Background(), uuid.New())
	if next.Reference().Version != "kb@0.2.0" {
		t.Fatal("activation did not publish new snapshot")
	}
	old, _ := pinned.Artifact(context.Background(), "prompts/unified.system.tmpl")
	if !bytes.Equal(old, pinnedBytes) {
		t.Fatal("pinned snapshot changed after reload")
	}
	prev, err := r.ByVersion(context.Background(), "kb@0.1.0")
	if err != nil || prev.Reference().Version != "kb@0.1.0" {
		t.Fatal("prior version not retained")
	}
	if err := r.Rollback(context.Background(), "kb@0.1.0"); err != nil {
		t.Fatal(err)
	}
	after, _ := r.Active(context.Background(), uuid.New())
	if after.Reference().Version != "kb@0.1.0" {
		t.Fatal("rollback did not restore known snapshot")
	}
	if err := r.Rollback(context.Background(), "kb@9.9.9"); !errors.Is(err, ErrUnavailable) {
		t.Fatal("unknown rollback should fail")
	}
	rewrite(t, root, "bundle.yaml", "version: 0.2.0", "version: 0.1.0")
	report, err := r.Activate(context.Background(), root)
	if err == nil || !containsCode(report.Errors, "immutable_version") {
		t.Fatalf("version collision must fail: %v %+v", err, report.Errors)
	}
	after, _ = r.Active(context.Background(), uuid.New())
	if after.Reference().Version != "kb@0.1.0" {
		t.Fatal("collision changed active bundle")
	}
}
func TestRegistryConcurrentPins(t *testing.T) {
	root := copyFixture(t)
	registry := NewRegistry(Options{CoreVersion: "1.0.0"}, true)
	if _, err := registry.Activate(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 30; j++ {
				active, err := registry.Active(context.Background(), uuid.New())
				if err != nil || active.Reference().Version != "kb@0.1.0" {
					t.Errorf("unsafe active: %v", err)
					return
				}
				data, err := active.Artifact(context.Background(), "bundle.yaml")
				if err != nil || len(data) == 0 {
					t.Errorf("unsafe pinned read: %v", err)
					return
				}
			}
		}()
	}
	wg.Wait()
}
func TestRejectProductionActivationOfSynthetic(t *testing.T) {
	r := NewRegistry(Options{CoreVersion: "1.0.0"}, false)
	report, err := r.Activate(context.Background(), fixtureDir(t))
	if err == nil || !containsCode(report.Errors, "synthetic_release") {
		t.Fatalf("production must fail closed: %+v %v", report.Errors, err)
	}
}
func TestLintBrokenBundles(t *testing.T) {
	cases := []struct{ name, path, from, to, code string }{
		{"core_compatibility", "bundle.yaml", ">=1.0.0 <2.0.0", ">=9.0.0 <10.0.0", "core_compatibility"},
		{"unknown_yaml_field", "config/retrieval.yaml", "schema: retrieval.v2", "schema: retrieval.v2\nextra_key: forbidden", "yaml_schema"},
		{"unknown_skill", "skills/reflect_emotion.yaml", "- validate_experience", "- unknown_skill", "unknown_skill"},
		{"skill_budget", "skills/reflect_emotion.yaml", "Name one plausible feeling", strings.Repeat("Too many words ", 180), "skill_token_budget"},
		{"missing_protocol_step", "protocols/thought_record.yaml", "on_success: thought", "on_success: nonexistent_step", "PR-001"},
		{"missing_distress", "protocols/thought_record.yaml", "on_distress: stop", "on_distress: ''", "PR-002"},
		{"unknown_predicate", "protocols/thought_record.yaml", "success_criteria: situation_specific_enough", "success_criteria: unknown_predicate", "PR-005"},
		{"unknown_adaptation", "protocols/thought_record.yaml", "adapt: ''", "adapt: nonexistent_adaptation", "PR-006"},
		{"missing_offer_text", "protocols/thought_record.yaml", "- thought_feels_certain", "- unreviewed_reason", "offer_reference"},
		{"unknown_boundary_script", "policy/boundary_policy.yaml", "response_script_id: boundary_no_diagnosis", "response_script_id: missing_script", "boundary_script_reference"},
		{"unknown_forbidden_claim_pattern", "config/vocab/forbidden_claims.yaml", `(?i)\byou (?:definitely|certainly) have (?:depression|bipolar disorder|ptsd)\b`, "'[invalid'", "claim_regex"},
		{"unknown_flag", "protocols/thought_record.yaml", "- psychosis_signs", "- unreviewed_flag", "PR-007"},
		{"invalid_refs", "config/vocab/avoid_tags.yaml", "- advice", "- advice\n- advice", "vocab_duplicate"},
		{"fallback_interpolation", "templates/fallback_by_objective.yaml", "It sounds like a lot is happening", "Hello {{.UserText}}", "fallback_interpolation"},
		{"locale_coverage", "bundle.yaml", "conversation_languages: [en, fa]", "conversation_languages: [en, fa, de]", "locale_coverage"},
		{"illegal_delete", "config/memory.yaml", "allow_hard_delete: false", "allow_hard_delete: true", "soft_delete"},
		{"invalid_retrieval_limit", "config/retrieval.yaml", "max_items: 5", "max_items: 0", "retrieval_ranges"},
		{"unknown_vocab", "config/vocab/problem_tags.yaml", "- anxiety_social", "- other_problem_tag", "vocab_reference"},
		{"unrecognized_schema", "policy/boundary_policy.yaml", "boundary.v1", "boundary.v5", "schema_version"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := copyFixture(t)
			rewrite(t, root, tc.path, tc.from, tc.to)
			_, report, err := Read(context.Background(), root, Options{CoreVersion: "1.0.0"})
			if err == nil || !containsCode(report.Errors, tc.code) {
				t.Fatalf("expected %s, got %v %+v", tc.code, err, report.Errors)
			}
		})
	}
}
func TestCommunicationPolicyWithinExpandedCapIsAccepted(t *testing.T) {
	root := copyFixture(t)
	path := filepath.Join(root, "policy", "communication_policy.md")
	// More than 600 estimated lint tokens but below the expanded 1200 limit.
	if err := os.WriteFile(path, []byte(strings.Repeat("Clear and bounded synthetic policy. ", 85)), 0600); err != nil {
		t.Fatal(err)
	}
	_, report, err := Read(context.Background(), root, Options{CoreVersion: "1.0.0"})
	if err != nil || !report.OK() {
		t.Fatalf("moderately expanded policy rejected: %v %+v", err, report.Errors)
	}
}

func TestCommunicationPolicyAboveExpandedCapIsRejected(t *testing.T) {
	root := copyFixture(t)
	path := filepath.Join(root, "policy", "communication_policy.md")
	if err := os.WriteFile(path, []byte(strings.Repeat("Oversized synthetic text. ", 500)), 0600); err != nil {
		t.Fatal(err)
	}
	_, report, err := Read(context.Background(), root, Options{CoreVersion: "1.0.0"})
	if err == nil || !containsCode(report.Errors, "token_budget") {
		t.Fatalf("oversized communication policy accepted: %v %+v", err, report.Errors)
	}
}

func TestInvalidFilesystemAndCancellation(t *testing.T) {
	root := copyFixture(t)
	if err := os.Symlink(filepath.Join(root, "bundle.yaml"), filepath.Join(root, "skills", "evil.yaml")); err != nil {
		t.Fatal(err)
	}
	_, report, err := Read(context.Background(), root, Options{CoreVersion: "1.0.0"})
	if err == nil || !containsCode(report.Errors, "bundle_files") {
		t.Fatalf("symlink should be rejected: %v %+v", err, report.Errors)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := Read(cancelled, root, Options{}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
func TestReleaseGuardsAndSignatureVerification(t *testing.T) {
	root := copyFixture(t)
	pub, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(root, "bundle.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := SignRelease(context.Background(), root, "1.0.0", "test", private); !errors.Is(err, ErrInvalid) {
		t.Fatalf("synthetic signing must fail: %v", err)
	}
	after, err := os.ReadFile(filepath.Join(root, "bundle.yaml"))
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("failed release mutated source")
	}
	if _, err := os.Stat(filepath.Join(root, "bundle.sig")); !os.IsNotExist(err) {
		t.Fatal("failed release wrote signature")
	}
	files, err := readFiles(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	var m Manifest
	if err := strictYAML(files["bundle.yaml"], &m); err != nil {
		t.Fatal(err)
	}
	// Pure cryptographic gate test uses synthetic fixture content only; it does
	// not try to create a release or pretend a clinician approved anything.
	m.Checksums.Artifacts = map[string]string{}
	for p, b := range files {
		if p != "bundle.yaml" && p != "bundle.sig" {
			m.Checksums.Artifacts[p] = hexSHA(b)
		}
	}
	m.Checksums.ManifestSHA256 = manifestDigest(m)
	message := signedPayload(m)
	signature := ed25519.Sign(private, message)
	if !ed25519.Verify(pub, message, signature) {
		t.Fatal("Ed25519 gate broken")
	}
	message = append(append([]byte(nil), message...), byte('!'))
	if ed25519.Verify(pub, message, signature) {
		t.Fatal("signature accepted modified manifest")
	}
	if compatible(">=1.0.0 <2.0.0", "2.0.0") || !compatible(">=1.0.0 <2.0.0", "1.9.0") {
		t.Fatal("semver compatibility gate")
	}
	if !time.Now().After(time.Time{}) {
		t.Fatal("clock unavailable")
	}
}

func TestChecksumAndClinicalApprovalTampering(t *testing.T) {
	root := copyFixture(t)
	files, err := readFiles(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	var m Manifest
	if err := strictYAML(files["bundle.yaml"], &m); err != nil {
		t.Fatal(err)
	}
	m.Checksums.Artifacts = map[string]string{}
	for p, b := range files {
		if p != "bundle.yaml" && p != "bundle.sig" {
			m.Checksums.Artifacts[p] = hexSHA(b)
		}
	}
	m.Checksums.ManifestSHA256 = manifestDigest(m)
	m.ClinicalReview.Approvals = []Approval{{Artifact: "policy/communication_policy.md", ReviewerID: "synthetic_reviewer", Date: "2026-01-01", SHA256: hexSHA(files["policy/communication_policy.md"])}}
	// Approval was added after manifest hash generation: this must invalidate
	// the canonical manifest hash even if every artifact remains unchanged.
	var report Report
	validateIntegrity(files, &m, Options{At: time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC)}, &report)
	if !containsCode(report.Errors, "manifest_hash") {
		t.Fatalf("missing manifest-hash gate: %+v", report.Errors)
	}
	m.Checksums.ManifestSHA256 = manifestDigest(m)
	report = Report{}
	validateIntegrity(files, &m, Options{At: time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC)}, &report)
	if containsCode(report.Errors, "approval_hash") || containsCode(report.Errors, "manifest_hash") || containsCode(report.Errors, "artifact_hash") {
		t.Fatalf("valid matching hashes rejected: %+v", report.Errors)
	}
	files["policy/communication_policy.md"] = append(files["policy/communication_policy.md"], byte('!'))
	report = Report{}
	validateIntegrity(files, &m, Options{}, &report)
	if !containsCode(report.Errors, "artifact_hash") || !containsCode(report.Errors, "approval_hash") {
		t.Fatalf("tampering not caught: %+v", report.Errors)
	}
}
func TestReleaseSignatureGateWithSyntheticCryptographicFixture(t *testing.T) {
	files, err := readFiles(context.Background(), fixtureDir(t))
	if err != nil {
		t.Fatal(err)
	}
	var manifest Manifest
	if err := strictYAML(files["bundle.yaml"], &manifest); err != nil {
		t.Fatal(err)
	}
	manifest.Checksums.Artifacts = map[string]string{}
	for p, b := range files {
		if p != "bundle.yaml" && p != "bundle.sig" {
			manifest.Checksums.Artifacts[p] = hexSHA(b)
		}
	}
	manifest.Checksums.ManifestSHA256 = manifestDigest(manifest)
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signature := Signature{Algorithm: "ed25519", KeyID: "test_key", Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(priv, signedPayload(manifest)))}
	sigBytes, err := json.Marshal(signature)
	if err != nil {
		t.Fatal(err)
	}
	files["bundle.sig"] = sigBytes
	opt := Options{Release: true, TrustedKeys: map[string]ed25519.PublicKey{"test_key": pub}}
	// Release still fails for missing clinical content; the cryptographic
	// verifier itself must accept the unchanged, correctly signed digest.
	report := Report{}
	validateIntegrity(files, &manifest, opt, &report)
	if containsCode(report.Errors, "signature_verify") || containsCode(report.Errors, "signature_key") || containsCode(report.Errors, "manifest_hash") {
		t.Fatalf("valid signature rejected: %+v", report.Errors)
	}
	signature.Signature = base64.StdEncoding.EncodeToString(ed25519.Sign(priv, []byte("wrong manifest")))
	files["bundle.sig"], _ = json.Marshal(signature)
	report = Report{}
	validateIntegrity(files, &manifest, opt, &report)
	if !containsCode(report.Errors, "signature_verify") {
		t.Fatalf("tampered signature accepted: %+v", report.Errors)
	}
	opt.TrustedKeys = map[string]ed25519.PublicKey{}
	report = Report{}
	validateIntegrity(files, &manifest, opt, &report)
	if !containsCode(report.Errors, "signature_key") {
		t.Fatal("untrusted signer accepted")
	}
}
