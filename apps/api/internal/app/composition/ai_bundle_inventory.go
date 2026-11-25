package composition

import (
	"context"
	"crypto/ed25519"
	"regexp"
	"sort"
	"time"

	admindomain "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/admin/domain"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/bundle"
)

var safeBundleCode = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// issueCodes discards paths and free-form messages (which might include
// operator filesystem paths or knowledge text). Only stable validator codes
// survive. Lists are capped; total counts retain the full diagnostic count.
func issueCodes(issues []bundle.Issue) []string {
	seen := map[string]bool{}
	for _, issue := range issues {
		if safeBundleCode.MatchString(issue.Code) {
			seen[issue.Code] = true
		}
	}
	result := make([]string, 0, len(seen))
	for code := range seen {
		result = append(result, code)
	}
	sort.Strings(result)
	if len(result) > 24 {
		result = result[:24]
	}
	return result
}

func describeBundle(slot string, snap *bundle.Snapshot, report bundle.Report) admindomain.BundleRecord {
	validation := admindomain.BundleValidation{
		Valid:      report.OK() && snap != nil,
		ErrorCount: len(report.Errors), WarningCount: len(report.Warnings),
		ErrorCodes: issueCodes(report.Errors), WarningCodes: issueCodes(report.Warnings),
	}
	out := admindomain.BundleRecord{
		Slot: slot, Status: "invalid", ConversationLanguages: []string{}, EmitsSchemas: map[string]string{},
		ReviewEvidence: "unverified", Validation: validation,
	}
	if snap == nil {
		return out
	}
	meta := snap.Metadata()
	status := "synthetic_draft"
	evidence := "unverified"
	if meta.SignatureVerified {
		status = "verified_release"
		evidence = "hash_matched"
	}
	return admindomain.BundleRecord{
		Slot: slot, Status: status, Name: meta.Name, Version: meta.Version, RequiresCore: meta.RequiresCore,
		KnowledgeLanguage: meta.KnowledgeLanguage, ConversationLanguages: meta.ConversationLanguages,
		EmitsSchemas: meta.EmitsSchemas, SyntheticOnly: &meta.SyntheticOnly,
		FingerprintSHA256: meta.FingerprintSHA256, ManifestSHA256: meta.ManifestSHA256,
		ArtifactCount: meta.ArtifactCount, ApprovalHashCount: meta.ApprovalHashCount,
		SignatureVerified: meta.SignatureVerified, SignatureKeyID: meta.SignatureKeyID,
		ReviewEvidence: evidence, Validation: validation,
	}
}

// inspectDeploymentBundles captures a startup-only, metadata-only inventory.
// The pinned active snapshot is the EXACT one passed into AI Core. Candidate
// roots are explicit trusted deployment configuration; their file bytes are
// never exposed or activated, and validation failures never break the active
// bundle. Invalid candidates cannot masquerade as signed releases.
func inspectDeploymentBundles(ctx context.Context, active *bundle.Snapshot, activeReport bundle.Report, candidateRoots []string, localDraft bool, trusted map[string]ed25519.PublicKey) admindomain.BundleInventory {
	inventory := admindomain.BundleInventory{AsOf: time.Now().UTC(), Scope: "deployment_snapshot", Candidates: []admindomain.BundleRecord{}}
	observed := map[string]string{}
	if active != nil {
		current := describeBundle("active", active, activeReport)
		inventory.Active = &current
		observed[current.Version] = current.FingerprintSHA256
	}
	for i, root := range candidateRoots {
		slot := "candidate-" + string(rune('1'+i)) // bounded to five by deployment config
		snap, report, err := bundle.Read(ctx, root, bundle.Options{CoreVersion: "1.0.0", Release: !localDraft, TrustedKeys: trusted})
		if err != nil {
			inventory.Candidates = append(inventory.Candidates, describeBundle(slot, nil, report))
			continue
		}
		if localDraft && !snap.SyntheticOnly() {
			report.Errors = append(report.Errors, bundle.Issue{Code: "candidate_non_synthetic"})
			inventory.Candidates = append(inventory.Candidates, describeBundle(slot, nil, report))
			continue
		}
		candidate := describeBundle(slot, snap, report)
		if fingerprint, exists := observed[candidate.Version]; exists && fingerprint != candidate.FingerprintSHA256 {
			candidate.Status = "version_conflict"
			candidate.Validation.Valid = false
			candidate.Validation.ErrorCount++
			candidate.Validation.ErrorCodes = append(candidate.Validation.ErrorCodes, "immutable_version")
		}
		if _, exists := observed[candidate.Version]; !exists {
			observed[candidate.Version] = candidate.FingerprintSHA256
		}
		inventory.Candidates = append(inventory.Candidates, candidate)
	}
	// An inventory is informational and cannot grant independent release
	// permission: signatures prove package integrity, NOT reviewer identity.
	return inventory
}
