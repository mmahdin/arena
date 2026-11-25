package protocol

import (
	"context"
	"strings"
	"testing"

	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

func TestLanguageChangeStopsProtocolAndPreventsDirectOffer(t *testing.T) {
	f := setup(t)
	f.e.assets.manifest.Locales = []string{"en"}
	f.active()
	f.r.snapshot.User.Prefs.Language = "fa"
	f.in.WCtx.Prefs.Language = "fa"
	spec, patch, err := f.resolve(directive(c.PAAdvance, c.OutSuccess, map[string]string{"situation": "synthetic"}))
	if err != nil || spec.Instruction != "" || !strings.Contains(patch.Reason, "language_resources_unavailable") {
		t.Fatal(spec, patch, err)
	}
	requireStatus(t, patch, c.PIStopped)

	f = setup(t)
	f.e.assets.manifest.Locales = []string{"en"}
	f.r.snapshot.User.Prefs.Language = "fa"
	// Even a stale English WCtx/shortlist cannot authorize a new offer.
	_, _, err = f.resolve(directive(c.PAOffer, "", nil))
	if err == nil {
		t.Fatal("direct offer bypassed saved-language coverage")
	}

	f = setup(t)
	f.e.assets.manifest.Locales = []string{"en"}
	f.w.Prefs.Language = "FA-ir"
	list, err := f.e.ShortlistWithContext(context.Background(), ShortlistContext{WCtx: f.w, User: f.u, Thread: &f.t, Readiness: ReadinessHigh, FlagsVerified: true, Now: f.now})
	if err != nil || len(list) != 0 {
		t.Fatal("unreviewed protocol language shortlisted", list, err)
	}
}
