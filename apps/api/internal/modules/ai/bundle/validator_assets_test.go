package bundle

import (
	"os"
	"path/filepath"
	"testing"
)

// The WP-03 linter must recognize WP-07 assets. These are synthetic fixtures
// and do not imply clinical coverage or permission to activate a bundle.
func TestWP07ValidatorLexiconAndThresholdsAcceptedByBundleLinter(t *testing.T) {
	lex, err := os.ReadFile(filepath.Join("..", "validator", "testdata", "validator.en.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	const name = "lexicons/validator.en.yaml"
	files := map[string][]byte{name: lex}
	var report Report
	validatePolicyAssets(files, &Manifest{Locales: []string{"en"}}, nil, nil, &report, Options{})
	for _, issue := range report.Errors {
		if issue.Path == name {
			t.Fatalf("WP-07 lexicon rejected by WP-03: %+v", issue)
		}
	}
	thresholds := []byte("schema: thresholds.v1\nvalidator:\n  directive_advice: 0.8\n  certainty_about_inference: 0.8\n  dependency_encouraging: 0.8\n  invalidating: 0.8\n")
	var parsed Thresholds
	if err = strictYAML(thresholds, &parsed); err != nil || len(parsed.Validator) != 4 {
		t.Fatalf("WP-07 thresholds unsupported by strict YAML decoder: %v %+v", err, parsed.Validator)
	}
	var configReport Report
	validateConfig(map[string][]byte{"config/thresholds.yaml": thresholds}, nil, &configReport)
	if containsCode(configReport.Errors, "validator_threshold") {
		t.Fatalf("valid threshold rejected: %+v", configReport.Errors)
	}
	thresholds = append(thresholds, []byte("  unsupported_classification: 0.7\n")...)
	var badReport Report
	validateConfig(map[string][]byte{"config/thresholds.yaml": thresholds}, nil, &badReport)
	if !containsCode(badReport.Errors, "validator_threshold") {
		t.Fatalf("unknown WP-07 threshold accepted: %+v", badReport.Errors)
	}
}

func TestWP07ReleaseRequiresClinicalReviewCategory(t *testing.T) {
	files := map[string][]byte{"lexicons/validator.en.yaml": []byte("fixture")}
	manifest := &Manifest{Version: "0.1.0", Name: "theia-kb", RequiresCore: ">=1.0.0", Locales: []string{"en"}, Changelog: "CHANGELOG.md"}
	manifest.ClinicalReview.RequiredFor = []string{"protocols", "boundary_policy", "safety_policy", "communication_policy", "crisis_templates"}
	manifest.EmitsSchemas = map[string]string{"brain": "brain.v1", "memory_writer": "memwriter.v1", "reflection": "reflection.v1", "safety_l1": "safety.v1"}
	var report Report
	validateManifest(files, manifest, Options{CoreVersion: "1.0.0", Release: true}, &report)
	if !containsCode(report.Errors, "validator_review") {
		t.Fatalf("missing validator clinical review category permitted: %+v", report.Errors)
	}
}
