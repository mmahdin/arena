package domain

import (
	"time"
)

// BundleValidation exposes only bounded validator codes; neither paths nor
// free-form messages from knowledge artifacts are returned over Admin HTTP.
type BundleValidation struct {
	Valid        bool     `json:"valid"`
	ErrorCount   int      `json:"errorCount"`
	WarningCount int      `json:"warningCount"`
	ErrorCodes   []string `json:"errorCodes"`
	WarningCodes []string `json:"warningCodes"`
}

type BundleRecord struct {
	Slot                  string            `json:"slot"`   // active, candidate-1 ... candidate-5
	Status                string            `json:"status"` // verified_release, synthetic_draft, invalid, version_conflict
	Name                  string            `json:"name,omitempty"`
	Version               string            `json:"version,omitempty"`
	RequiresCore          string            `json:"requiresCore,omitempty"`
	KnowledgeLanguage     string            `json:"knowledgeLanguage,omitempty"`
	ConversationLanguages []string          `json:"conversationLanguages"`
	EmitsSchemas          map[string]string `json:"emitsSchemas"`
	SyntheticOnly         *bool             `json:"syntheticOnly"` // nil for a rejected candidate
	FingerprintSHA256     string            `json:"fingerprintSha256,omitempty"`
	ManifestSHA256        string            `json:"manifestSha256,omitempty"`
	ArtifactCount         int               `json:"artifactCount"`
	ApprovalHashCount     int               `json:"approvalHashCount"`
	SignatureVerified     bool              `json:"signatureVerified"`
	SignatureKeyID        string            `json:"signatureKeyId,omitempty"`
	ReviewEvidence        string            `json:"reviewEvidence"` // hash_matched or unverified
	Validation            BundleValidation  `json:"validation"`
}

// BundleInventory is an immutable startup observation, NOT a release registry
// or a live filesystem scan. None of its fields grants permission to publish.
type BundleInventory struct {
	AsOf       time.Time      `json:"asOf"`
	Scope      string         `json:"scope"` // deployment_snapshot or not_configured
	Active     *BundleRecord  `json:"active"`
	Candidates []BundleRecord `json:"candidates"`
}

// CloneBundleInventory prevents callers from mutating the shared in-process
// description between audited requests.
func CloneBundleInventory(v BundleInventory) BundleInventory {
	copyRecord := func(in BundleRecord) BundleRecord {
		out := in
		out.ConversationLanguages = append([]string{}, in.ConversationLanguages...)
		out.EmitsSchemas = map[string]string{}
		for k, value := range in.EmitsSchemas {
			out.EmitsSchemas[k] = value
		}
		if in.SyntheticOnly != nil {
			b := *in.SyntheticOnly
			out.SyntheticOnly = &b
		}
		out.Validation.ErrorCodes = append([]string{}, in.Validation.ErrorCodes...)
		out.Validation.WarningCodes = append([]string{}, in.Validation.WarningCodes...)
		return out
	}
	result := BundleInventory{AsOf: v.AsOf, Scope: v.Scope, Candidates: make([]BundleRecord, 0, len(v.Candidates))}
	if v.Active != nil {
		a := copyRecord(*v.Active)
		result.Active = &a
	}
	for _, b := range v.Candidates {
		result.Candidates = append(result.Candidates, copyRecord(b))
	}
	return result
}
