package eval

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRuntimePreflightAlwaysBlocksUnknownReplayInputs(t *testing.T) {
	for _, flags := range []struct {
		bundle, versions, encrypted bool
	}{
		{false, false, false},
		{true, false, false},
		{true, true, true},
	} {
		got := CheckRuntimeCoverage(flags.bundle, flags.versions, flags.encrypted)
		if got.Replayable || len(got.Blockers) < 2 {
			t.Fatalf("metadata must never authorize replay: %+v", got)
		}
		b, err := json.Marshal(got)
		if err != nil || strings.Contains(string(b), "sensitive content") || strings.Contains(string(b), "turn_id") || strings.Contains(string(b), "user_id") {
			t.Fatal("unexpected user data in metadata-only report")
		}
		contains := func(code string) bool {
			for _, found := range got.Blockers {
				if found == code {
					return true
				}
			}
			return false
		}
		if !contains("EVAL_RUNTIME_TYPED_STAGES_UNVERIFIED") || !contains("EVAL_RUNTIME_REPLAY_POLICY_UNAPPROVED") {
			t.Fatalf("suppressed privacy/replay blockers: %+v", got)
		}
		if flags.encrypted && !contains("EVAL_RUNTIME_ENCRYPTED_PAYLOAD_UNVERIFIED") {
			t.Fatal("opaque ciphertext mistaken for typed evidence")
		}
		if !flags.encrypted && !contains("EVAL_RUNTIME_ENCRYPTED_PAYLOAD_MISSING") {
			t.Fatal("absent encrypted payload not detected")
		}
	}
}
