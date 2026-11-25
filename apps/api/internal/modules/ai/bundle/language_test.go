package bundle

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestMultilingualCoverageRequiresResourcesNotKnowledgeCopies(t *testing.T) {
	root := fixtureDir(t)
	files, err := readFiles(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	var manifest Manifest
	if err := strictYAML(files["bundle.yaml"], &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Knowledge() != "en" || manifest.ValidateLanguages() != nil || !manifest.ProtocolLanguage("fa") || !manifest.ProtocolLanguage("en") {
		t.Fatal("invalid language metadata")
	}
	if _, exists := files["policy/communication_policy.fa.md"]; exists {
		t.Fatal("knowledge duplication")
	}
	for _, path := range []string{"lexicons/safety_l0.fa.yaml", "lexicons/validator.fa.yaml"} {
		t.Run(path, func(t *testing.T) {
			copy := copyFixture(t)
			if err := os.Remove(filepath.Join(copy, path)); err != nil {
				t.Fatal(err)
			}
			_, report, err := Read(context.Background(), copy, Options{})
			if err == nil || report.OK() {
				t.Fatal("missing safety coverage accepted")
			}
		})
	}
	t.Run("missing_crisis_translation", func(t *testing.T) {
		copy := copyFixture(t)
		rewrite(t, copy, "templates/crisis_templates.yaml", "    fa:", "    xx:")
		_, report, err := Read(context.Background(), copy, Options{})
		if err == nil || !containsCode(report.Errors, "locale_coverage") {
			t.Fatal("missing crisis language accepted", err, report)
		}
	})
	for _, change := range []func(*Manifest){
		func(m *Manifest) { m.KnowledgeLanguage = "fa" },
		func(m *Manifest) { delete(m.LanguageResources, "fa") },
		func(m *Manifest) { m.SyntheticOnly = false },
	} {
		var m Manifest
		if err := strictYAML(files["bundle.yaml"], &m); err != nil {
			t.Fatal(err)
		}
		change(&m)
		if m.ValidateLanguages() == nil {
			t.Fatal("invalid language declaration accepted")
		}
	}
	// A claimed reviewed status cannot substitute for actual artifact approvals.
	manifest.SyntheticOnly = false
	for lang, r := range manifest.LanguageResources {
		r.ReviewStatus = "reviewed"
		manifest.LanguageResources[lang] = r
	}
	var report Report
	validateIntegrity(files, &manifest, Options{}, &report)
	for _, path := range []string{"lexicons/safety_l0.fa.yaml", "lexicons/validator.fa.yaml", "resources/crisis_resources.yaml", "templates/fallback_by_objective.yaml", "templates/boundary_scripts.yaml"} {
		found := false
		for _, issue := range report.Errors {
			if issue.Path == path && issue.Code == "approval_missing" {
				found = true
			}
		}
		if !found {
			t.Errorf("no approval gate for %s", path)
		}
	}
}

func TestLegacyManifestSignaturePayloadIsStable(t *testing.T) {
	m := Manifest{Version: "0.9.0", Locales: []string{"en"}, SyntheticOnly: true}
	got, err := json.Marshal(m)
	const want = `{"Name":"","Version":"0.9.0","RequiresCore":"","EmitsSchemas":null,"Locales":["en"],"SyntheticOnly":true,"ClinicalReview":{"RequiredFor":null,"Approvals":null},"Checksums":{"Algo":"","ManifestSHA256":"","Artifacts":null},"Changelog":""}`
	if err != nil || string(got) != want {
		t.Fatalf("legacy signed payload changed: %s %v", got, err)
	}
}
