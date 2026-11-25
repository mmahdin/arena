package composition

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/bundle"
)

func TestBundleInventoryFromPinnedSyntheticSnapshot(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..", "..", "knowledge")
	active, validation, err := bundle.Read(context.Background(), root, bundle.Options{CoreVersion: "1.0.0"})
	if err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(t.TempDir(), "missing-directory")
	report := inspectDeploymentBundles(context.Background(), active, validation, []string{root, missing}, true, nil)
	if report.Scope != "deployment_snapshot" || report.AsOf.IsZero() || report.Active == nil || report.Active.Status != "synthetic_draft" || report.Active.SignatureVerified || report.Active.Version != "kb@0.1.0" || !report.Active.Validation.Valid {
		t.Fatalf("draft presented as signed or missing: %+v", report.Active)
	}
	if len(report.Candidates) != 2 || report.Candidates[0].Status != "synthetic_draft" || report.Candidates[1].Status != "invalid" || report.Candidates[1].Validation.ErrorCount == 0 || report.Candidates[1].Version != "" {
		t.Fatalf("candidates incorrect: %+v", report.Candidates)
	}
	if report.Candidates[1].Validation.ErrorCodes[0] != "bundle_files" {
		t.Fatal("invalid candidate must return only stable code")
	}
	for _, candidate := range report.Candidates {
		for _, code := range candidate.Validation.ErrorCodes {
			if strings.Contains(code, missing) {
				t.Fatal("leaked local filesystem path")
			}
		}
	}
	// A public release verifier never calls Read in development-only mode.
	_, rejected, e := bundle.Read(context.Background(), root, bundle.Options{CoreVersion: "1.0.0", Release: true})
	if e == nil || len(rejected.Errors) == 0 {
		t.Fatal("unsigned synthetic bundle passed release validation")
	}
}

func TestBundleInventoryDoesNotRescanAfterStartup(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..", "..", "knowledge")
	active, validation, err := bundle.Read(context.Background(), root, bundle.Options{CoreVersion: "1.0.0"})
	if err != nil {
		t.Fatal(err)
	}
	// Candidate path exists and has a malformed manifest: only code leaks.
	invalidRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(invalidRoot, "bundle.yaml"), []byte("schema: secret\nunsupported: true\n"), 0600); err != nil {
		t.Fatal(err)
	}
	inventory := inspectDeploymentBundles(context.Background(), active, validation, []string{invalidRoot}, true, nil)
	if len(inventory.Candidates) != 1 || inventory.Candidates[0].Status != "invalid" || inventory.Candidates[0].Name != "" {
		t.Fatal("invalid candidate leaked manifest")
	}
	if err := os.RemoveAll(invalidRoot); err != nil {
		t.Fatal(err)
	}
	// No subsequent filesystem call is necessary to return inventory.
	if inventory.Active.Version != "kb@0.1.0" || inventory.Candidates[0].Status != "invalid" {
		t.Fatal("startup snapshot changed")
	}
}

func TestCandidateVersionCollisionHasNoActivationEffect(t *testing.T) {
	ctx := context.Background()
	source := filepath.Join("..", "..", "..", "..", "..", "knowledge")
	active, report, err := bundle.Read(ctx, source, bundle.Options{CoreVersion: "1.0.0"})
	if err != nil {
		t.Fatal(err)
	}
	clone := t.TempDir()
	err = filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, e := filepath.Rel(source, path)
		if e != nil {
			return e
		}
		dst := filepath.Join(clone, rel)
		if entry.IsDir() {
			return os.MkdirAll(dst, 0700)
		}
		b, e := os.ReadFile(path)
		if e != nil {
			return e
		}
		if rel == "CHANGELOG.md" {
			b = append(b, []byte("\nSynthetic version collision test.\n")...)
		}
		return os.WriteFile(dst, b, 0600)
	})
	if err != nil {
		t.Fatal(err)
	}
	candidate, rep, err := bundle.Read(ctx, clone, bundle.Options{CoreVersion: "1.0.0"})
	if err != nil {
		t.Fatalf("candidate must be individually valid: %v %+v", err, rep.Errors)
	}
	if candidate.Fingerprint() == active.Fingerprint() {
		t.Fatal("expected different bytes")
	}
	inventory := inspectDeploymentBundles(ctx, active, report, []string{clone}, true, nil)
	if inventory.Active == nil || !inventory.Active.Validation.Valid || inventory.Candidates[0].Status != "version_conflict" || inventory.Candidates[0].Validation.Valid || inventory.Candidates[0].Validation.ErrorCount != 1 {
		t.Fatalf("collision misreported: %+v", inventory)
	}
	if inventory.Active.FingerprintSHA256 != active.Fingerprint() {
		t.Fatal("active pointer was replaced")
	}
}
