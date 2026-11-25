package bundle

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Signature is an Ed25519 signature over the canonical JSON manifest including
// the content-addressed artifact checksums. Private signing keys stay offline.
type Signature struct {
	Algorithm string `json:"algorithm"`
	KeyID     string `json:"key_id"`
	Signature string `json:"signature"`
}

func manifestDigest(m Manifest) string {
	m.Checksums.Algo = ""
	m.Checksums.ManifestSHA256 = ""
	m.Checksums.Artifacts = nil
	serialized, _ := json.Marshal(m)
	return hexSHA(serialized)
}
func signedPayload(m Manifest) []byte {
	encoded, _ := json.Marshal(m)
	return encoded
}
func validateIntegrity(files map[string][]byte, m *Manifest, opt Options, r *Report) {
	if m.Checksums.Algo != "sha256" {
		r.fail("bundle.yaml", "checksum_algorithm", "only sha256 checksums are supported")
	}
	if m.Checksums.ManifestSHA256 == "" || len(m.Checksums.Artifacts) == 0 {
		if !m.SyntheticOnly || opt.Release {
			r.fail("bundle.yaml", "checksums_missing", "bundle must include release-generated checksums")
		} else {
			r.warn("bundle.yaml", "checksums_pending", "synthetic fixture has no release checksums")
		}
	} else {
		if m.Checksums.ManifestSHA256 != manifestDigest(*m) {
			r.fail("bundle.yaml", "manifest_hash", "manifest checksum mismatch")
		}
		wanted := 0
		for p, b := range files {
			if p == "bundle.yaml" || p == "bundle.sig" {
				continue
			}
			wanted++
			if m.Checksums.Artifacts[p] != hexSHA(b) {
				r.fail(p, "artifact_hash", "artifact checksum mismatch or missing")
			}
		}
		if len(m.Checksums.Artifacts) != wanted {
			r.fail("bundle.yaml", "artifact_index", "checksum table has extra/missing entries")
		}
	}
	required := map[string]bool{}
	categories := map[string][]string{
		"protocols":            filesWithPrefix(files, "protocols/"),
		"boundary_policy":      {"policy/boundary_policy.yaml"},
		"safety_policy":        {"policy/safety_policy.yaml"},
		"communication_policy": {"policy/communication_policy.md"},
		"crisis_templates":     {"templates/crisis_templates.yaml"},
		"validator":            append(filesWithPrefix(files, "lexicons/validator."), "config/thresholds.yaml", "prompts/validator_classifier.system.tmpl"),
	}
	if _, ok := files["prompts/unified.system.tmpl"]; ok {
		required["prompts/unified.system.tmpl"] = true
	}
	for _, category := range m.ClinicalReview.RequiredFor {
		paths, ok := categories[category]
		if !ok {
			r.fail("bundle.yaml", "review_category", "unknown clinical review category: "+category)
			continue
		}
		for _, p := range paths {
			required[p] = true
		}
	}
	// Explicit multilingual manifests bind all language safety/resource assets
	// to review hashes. A status label alone cannot authorize production use.
	if len(m.ConversationLanguages) > 0 {
		for _, prefix := range []string{"lexicons/safety_l0.", "lexicons/validator.", "templates/"} {
			for _, p := range filesWithPrefix(files, prefix) {
				required[p] = true
			}
		}
		for _, p := range []string{"resources/crisis_resources.yaml", "prompts/safety_l1.system.tmpl", "prompts/validator_classifier.system.tmpl", "config/vocab/forbidden_claims.yaml"} {
			required[p] = true
		}
	}
	approvals := map[string]bool{}
	at := opt.At
	if at.IsZero() {
		at = time.Now().UTC()
	}
	for _, a := range m.ClinicalReview.Approvals {
		if _, ok := files[a.Artifact]; !ok {
			r.fail("bundle.yaml", "approval_target", "approval refers to absent artifact")
			continue
		}
		if approvals[a.Artifact] {
			r.fail("bundle.yaml", "approval_duplicate", "multiple approvals for same artifact")
		}
		approvals[a.Artifact] = true
		if a.ReviewerID == "" || len(a.SHA256) != 64 || a.SHA256 != hexSHA(files[a.Artifact]) {
			r.fail("bundle.yaml", "approval_hash", "reviewer/hash missing or approval is stale")
		}
		date, err := time.Parse("2006-01-02", a.Date)
		if err != nil || date.After(at.Add(24*time.Hour)) {
			r.fail("bundle.yaml", "approval_date", "invalid/future clinical approval date")
		}
	}
	for p := range required {
		if !approvals[p] {
			if m.SyntheticOnly && !opt.Release {
				r.warn(p, "approval_pending", "synthetic fixture is not clinically approved")
			} else {
				r.fail(p, "approval_missing", "missing matching clinical review hash")
			}
		}
	}
	if !opt.Release {
		return
	}
	raw, ok := files["bundle.sig"]
	if !ok {
		r.fail("bundle.sig", "signature_missing", "release requires detached signature")
		return
	}
	var sig Signature
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&sig); err != nil {
		r.fail("bundle.sig", "signature_schema", "invalid signature envelope")
		return
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		r.fail("bundle.sig", "signature_schema", "extra signature data is not allowed")
		return
	}
	if sig.Algorithm != "ed25519" || sig.KeyID == "" {
		r.fail("bundle.sig", "signature_schema", "unsupported signature metadata")
		return
	}
	key, ok := opt.TrustedKeys[sig.KeyID]
	if !ok || len(key) != ed25519.PublicKeySize {
		r.fail("bundle.sig", "signature_key", "signing key not in trusted key ring")
		return
	}
	decoded, err := base64.StdEncoding.DecodeString(sig.Signature)
	if err != nil || !ed25519.Verify(key, signedPayload(*m), decoded) {
		r.fail("bundle.sig", "signature_verify", "invalid Ed25519 signature")
	}
}

// SignRelease produces bundle checksums and a detached release signature.
// It never creates or stores keys. The caller must supply a separately managed
// private key and a unique, reviewed non-synthetic bundle version.
func SignRelease(ctx context.Context, root, coreVersion, keyID string, privateKey ed25519.PrivateKey) error {
	if strings.TrimSpace(keyID) == "" || len(privateKey) != ed25519.PrivateKeySize {
		return fmt.Errorf("%w: signing key", ErrInvalid)
	}
	files, err := readFiles(ctx, root)
	if err != nil {
		return fmt.Errorf("%w: inspect bundle: %v", ErrInvalid, err)
	}
	var m Manifest
	if err := strictYAML(files["bundle.yaml"], &m); err != nil {
		return fmt.Errorf("%w: manifest", ErrInvalid)
	}
	if m.SyntheticOnly {
		return fmt.Errorf("%w: synthetic bundle may not be signed", ErrInvalid)
	}
	m.Checksums.Algo = "sha256"
	m.Checksums.Artifacts = map[string]string{}
	for p, b := range files {
		if p != "bundle.yaml" && p != "bundle.sig" {
			m.Checksums.Artifacts[p] = hexSHA(b)
		}
	}
	m.Checksums.ManifestSHA256 = manifestDigest(m)
	manifestBytes, err := yaml.Marshal(&m)
	if err != nil {
		return err
	}
	sig := Signature{Algorithm: "ed25519", KeyID: keyID, Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, signedPayload(m)))}
	signatureBytes, err := json.MarshalIndent(sig, "", "  ")
	if err != nil {
		return err
	}
	// Validate the would-be release (including clinical approvals, freshness,
	// locale gates, checksums, and signature) before touching the source tree.
	staged, err := os.MkdirTemp("", "theia-kb-release-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(staged)
	for p, b := range files {
		if p == "bundle.yaml" || p == "bundle.sig" {
			continue
		}
		target := filepath.Join(staged, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			return err
		}
		if err := os.WriteFile(target, b, 0600); err != nil {
			return err
		}
	}
	if err := os.WriteFile(filepath.Join(staged, "bundle.yaml"), manifestBytes, 0600); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(staged, "bundle.sig"), signatureBytes, 0600); err != nil {
		return err
	}
	_, report, err := Read(ctx, staged, Options{CoreVersion: coreVersion, Release: true, TrustedKeys: map[string]ed25519.PublicKey{keyID: privateKey.Public().(ed25519.PublicKey)}})
	if err != nil {
		first := "release preflight failed"
		if len(report.Errors) > 0 {
			first = report.Errors[0].Path + ": " + report.Errors[0].Code
		}
		return fmt.Errorf("%w: %s", ErrInvalid, first)
	}
	originalManifest := files["bundle.yaml"]
	originalSig, sigExisted := files["bundle.sig"]
	// File updates are validated before install. On failure, restore originals;
	// distribution should still publish the directory as an atomic archive.
	if err := replaceFile(filepath.Join(root, "bundle.yaml"), manifestBytes); err != nil {
		return err
	}
	if err := replaceFile(filepath.Join(root, "bundle.sig"), signatureBytes); err != nil {
		_ = replaceFile(filepath.Join(root, "bundle.yaml"), originalManifest)
		if sigExisted {
			_ = replaceFile(filepath.Join(root, "bundle.sig"), originalSig)
		}
		return err
	}
	return nil
}
func replaceFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	file, err := os.CreateTemp(dir, ".wp03-release-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if err := file.Chmod(0644); err != nil {
		file.Close()
		return err
	}
	if _, err := file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}
