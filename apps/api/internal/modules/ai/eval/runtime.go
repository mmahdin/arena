package eval

// RuntimeTraceCoverage is a metadata-only preflight for persisted traces. It
// deliberately cannot replay a turn: ordinary SQL traces lack the typed model,
// safety, generator and state evidence required by Contract Pack v1.2 §12.1.
// This report contains no user identifiers, prompts, message text or drafts.
type RuntimeTraceCoverage struct {
	Replayable             bool     `json:"replayable"`
	HasBundleVersion       bool     `json:"has_bundle_version"`
	HasModelPromptVersions bool     `json:"has_model_prompt_versions"`
	HasEncryptedPayload    bool     `json:"has_encrypted_payload"`
	Blockers               []string `json:"blockers"`
}

// CheckRuntimeCoverage never treats opaque ciphertext or a trace summary as
// proof that typed replay inputs exist. A separate reviewed encrypted capture,
// retention and access contract is needed before real runtime replay is enabled.
func CheckRuntimeCoverage(bundleVersion, modelPromptVersions, encryptedPayload bool) RuntimeTraceCoverage {
	out := RuntimeTraceCoverage{
		HasBundleVersion:       bundleVersion,
		HasModelPromptVersions: modelPromptVersions,
		HasEncryptedPayload:    encryptedPayload,
		Blockers:               []string{},
	}
	if !bundleVersion {
		out.Blockers = append(out.Blockers, "EVAL_RUNTIME_BUNDLE_VERSION_MISSING")
	}
	if !modelPromptVersions {
		out.Blockers = append(out.Blockers, "EVAL_RUNTIME_MODEL_PROMPT_VERSIONS_MISSING")
	}
	if !encryptedPayload {
		out.Blockers = append(out.Blockers, "EVAL_RUNTIME_ENCRYPTED_PAYLOAD_MISSING")
	} else {
		// The inspector intentionally cannot decrypt or verify a ciphertext's
		// schema. Non-null bytes never establish permission to replay.
		out.Blockers = append(out.Blockers, "EVAL_RUNTIME_ENCRYPTED_PAYLOAD_UNVERIFIED")
	}
	out.Blockers = append(out.Blockers, "EVAL_RUNTIME_TYPED_STAGES_UNVERIFIED", "EVAL_RUNTIME_REPLAY_POLICY_UNAPPROVED")
	return out
}
