package safety

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

// All fixture strings are inert synthetic markers, not real crisis language,
// clinical policy, or valid phone numbers. JSON is also valid YAML 1.2.
type testBundle struct {
	files     map[string]string
	synthetic bool
}

func (b *testBundle) SyntheticOnly() bool { return b.synthetic }
func (b *testBundle) Reference() contract.BundleRef {
	return contract.BundleRef{Version: "kb@0.9.0", PromptSHAs: map[string]string{"prompts/safety_l1.system.tmpl": "synthetic-sha"}}
}
func (b *testBundle) Artifact(ctx context.Context, path string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	text, ok := b.files[path]
	if !ok {
		return nil, errors.New("artifact missing")
	}
	return []byte(text), nil
}
func syntheticBundle() *testBundle {
	return &testBundle{synthetic: true, files: map[string]string{
		"bundle.yaml":                          `{"name":"synthetic","version":"0.9.0","locales":["en","fa"],"synthetic_only":true}`,
		"policy/safety_policy.yaml":            `{"schema":"safety_policy.v1","categories":{"other":{"template_family":"synthetic_case","escalate_to":"high"}},"template_families":{"synthetic_case":{"required_elements":["acknowledge","care","ask_safety","resources","invite_continue"],"forbidden":["method_details","promises"]}},"risk_fsm":{"follow_up_hours":24,"deescalate":{"from_high_to_elevated":{"min_hours_since_signal":12,"require":["no_new_signals"]},"from_elevated_to_low":{"min_hours_since_signal":24,"require":["no_new_signals","user_statement_safe_or_topic_neutral_3_turns"]}}},"second_look":{"downgrade":{"from":"high","to":"elevated","when_context_flags_any":["quoted","fictional"],"never_when":["first_person_present_tense"]}}}`,
		"config/thresholds.yaml":               `{"schema":"thresholds.v1","safety":{"deescalate":{"require_user_statement":true}}}`,
		"lexicons/safety_l0.en.yaml":           `{"schema":"lexicon.v1","version":"synthetic-0.9.0","locale":"en","rules":[{"id":"synthetic.high.001","category":"other","level":"high","match":{"type":"literal","pattern":"TEST_HIGH_SIGNAL"},"second_look":true},{"id":"synthetic.imminent.001","category":"other","level":"imminent","match":{"type":"regex","pattern":"TEST_IMMINENT_SIGNAL"},"second_look":false}]}`,
		"lexicons/safety_l0.fa.yaml":           `{"schema":"lexicon.v1","version":"synthetic-fa-0.9.0","locale":"fa","rules":[{"id":"synthetic.fa.high.001","category":"other","level":"high","match":{"type":"literal","pattern":"TEST_FA_HIGH"},"second_look":true}]}`,
		"resources/crisis_resources.yaml":      `{"schema":"crisis_resources.v1","last_full_verification":"2026-11-15","stale_after_days":180,"fallback_locale":"en","locales":{"en":[{"id":"synthetic_generic_resource","name":"Synthetic Resource","kind":"chat","contact":"TEST-ONLY-CONTACT","hours":"test only","verified_on":"2026-11-15"}],"fa":[]}}`,
		"templates/crisis_templates.yaml":      `{"schema":"crisis_templates.v1","templates":{"synthetic_case":{"text":"SYNTHETIC TEST CRISIS SCRIPT (not for people)","locales":{"fa":"SYNTHETIC FA CRISIS SCRIPT"}},"resources_general_notice":{"text":"SYNTHETIC GENERAL RESOURCES NOTICE","locales":{"fa":"SYNTHETIC FA GENERAL NOTICE"}}}}`,
		"templates/fallback_by_objective.yaml": `{"schema":"fallback.v1","templates":{"reflect":{"id":"test_reflect","text":"SYNTHETIC NORMAL FALLBACK","locales":{"fa":"SYNTHETIC FA FALLBACK"}}}}`,
		"prompts/safety_l1.system.tmpl":        "SYNTHETIC UNIT TEST CLASSIFIER BLOCK: non-clinical marker only.",
	}}
}
func loadFixture(t *testing.T) *Assets {
	t.Helper()
	a, err := Load(context.Background(), syntheticBundle(), LoadOptions{AllowSyntheticForTests: true})
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	return a
}
func fixedTime() time.Time { return time.Date(2026, time.November, 16, 12, 0, 0, 0, time.UTC) }
