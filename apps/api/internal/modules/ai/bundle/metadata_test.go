package bundle

import (
	"context"
	"testing"
)

func TestMetadataDoesNotClaimReleaseFromSyntheticLint(t *testing.T) {
	b, report, err := Read(context.Background(), fixtureDir(t), Options{CoreVersion: "1.0.0"})
	if err != nil || !report.OK() {
		t.Fatal(err)
	}
	m := b.Metadata()
	if m.Version != "kb@0.1.0" || !m.SyntheticOnly || m.SignatureVerified || m.SignatureKeyID != "" || m.ManifestSHA256 != "" || m.ApprovalHashCount != 0 || m.FingerprintSHA256 == "" {
		t.Fatalf("bad synthetic trust claims %+v", m)
	}
	m.EmitsSchemas["brain"] = "forged"
	m.ConversationLanguages[0] = "forged"
	again := b.Metadata()
	if again.EmitsSchemas["brain"] == "forged" || again.ConversationLanguages[0] == "forged" {
		t.Fatal("metadata alias escaped")
	}
}
