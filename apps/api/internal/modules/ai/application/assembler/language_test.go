package assembler

import (
	"context"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"testing"
)

func TestResponsePreferenceDoesNotOverwriteInputLanguage(t *testing.T) {
	a, reader, req, _, _ := fixture(t)
	req.Text = "امروز روز سختی بود"
	req.Locale = "fa"
	// An old stored locale must not override this turn or an updated preference.
	reader.snapshot.User.Locale = "de"
	for _, tc := range []struct{ pref, want string }{{"en", "en"}, {"fa-IR", "fa"}, {"", "fa"}, {"de", "fa"}, {"en", "en"}} {
		reader.snapshot.User.Prefs.Language = tc.pref
		wc, _, err := a.Build(context.Background(), req, c.SafetySignal{Level: c.RiskLow}, c.RetrievalResult{})
		if err != nil || wc.Turn.Lang != "fa" || wc.Prefs.Language != tc.want {
			t.Fatalf("pref %q: lang=%q response=%q err=%v", tc.pref, wc.Turn.Lang, wc.Prefs.Language, err)
		}
		if reader.snapshot.User.Prefs.Language != tc.pref {
			t.Fatal("resolution mutated saved preference")
		}
	}
}
