package protocol

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/state"
	kb "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/bundle"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/bundle/testfixture"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

type productRegistry struct{ b ports.Bundle }

func (r productRegistry) Active(ctx context.Context, _ c.ID) (ports.Bundle, error) {
	return r.b, ctx.Err()
}

func TestPersianRequestedExerciseStartsWithoutIntakeOrRepeatedConsent(t *testing.T) {
	ctx := context.Background()
	b, report, err := kb.Read(ctx, filepath.Join("..", "..", "..", "..", "..", "..", "..", "knowledge"), kb.Options{})
	if err != nil || !report.OK() {
		t.Fatal(err, report.Errors)
	}
	assets, err := Load(ctx, b, LoadOptions{AllowSyntheticForTests: true})
	if err != nil {
		t.Fatal(err)
	}
	f := setup(t)
	f.e.assets = assets
	f.in.Bundle = b.Reference()
	f.r.snapshot.User.Prefs.Language = "fa"
	f.r.snapshot.Threads = nil
	f.w.FocusThread = nil
	f.w.Turn.Text = "یه تمرین ACT برای فاصله گرفتن از فکرهام می‌خوام"
	f.w.Prefs.Language = "fa"
	f.in.WCtx = f.w
	list, err := f.e.Shortlist(ctx, f.w, nil, f.r.snapshot.User)
	if err != nil || len(list) != 7 {
		t.Fatal("library hidden behind intake/readiness/inventory", list, err)
	}
	id := "defusion_phrase"
	f.in.Objective = c.ObjOfferIntervention
	f.in.SelectedContentIDs = selectedContentIDs(list)
	f.in.UserAffirmed = true
	spec, patch, err := f.resolve(&c.ProtocolDirective{Action: c.PAOffer, ProtocolID: &id})
	if err != nil || spec.StepID != "phrase" || spec.Offer != nil || patch.Create == nil || patch.Create.ThreadID != nil || patch.ConsentMessageID == nil || *patch.ConsentMessageID != f.in.UserMessageID {
		t.Fatal(spec, patch, err)
	}
	requireStatus(t, patch, c.PIActive)
	committer, err := state.New(f.r, productRegistry{testfixture.WithRuntimeSettings(t, b)})
	if err != nil {
		t.Fatal(err)
	}
	delta := c.StateDelta{UserID: f.u.UserID, TurnID: f.in.TurnID, ExpectedVersions: c.Versions{Session: 1, User: 1}, Session: c.SessionPatch{TurnCountInc: 1, LastObjective: c.ObjContinueIntervention}, Protocol: &patch}
	clean, rejected, err := committer.ValidateDelta(ctx, delta)
	if err != nil || len(rejected) > 0 || clean.Protocol == nil || clean.Protocol.NewStatus == nil || *clean.Protocol.NewStatus != c.PIActive {
		t.Fatal("atomic direct start rejected", clean.Protocol, rejected, err)
	}
	// The same API never turns a mere optional offer into an automatic start.
	f.in.UserAffirmed = false
	offer, offered, err := f.resolve(&c.ProtocolDirective{Action: c.PAOffer, ProtocolID: &id})
	if err != nil || offer.Offer == nil || offered.ConsentMessageID != nil || offered.NewStatus != nil {
		t.Fatal("unsolicited offer started", offer, offered, err)
	}
}

func TestProtocolProgressionUsesMeaningAndRespectsChoice(t *testing.T) {
	f := setup(t)
	f.active()
	f.in.WCtx.Turn.Text = "It was a difficult conversation with a colleague."
	step, patch, err := f.resolve(directive(c.PAAdvance, c.OutSuccess, map[string]string{"situation": "difficult work conversation"}))
	if err != nil || step.StepID != "last" || patch.Inputs["situation"] == "" {
		t.Fatal("paraphrase could not advance", step, patch, err)
	}
	f.in.Active.StuckCount = 9
	f.r.snapshot.Protocol.StuckCount = 9
	step, patch, err = f.resolve(directive(c.PAAdapt, c.OutStuck, nil))
	if err != nil || step.Instruction == "" || patch.NewStatus != nil {
		t.Fatal("difficulty forced exit", step, patch, err)
	}
	_, patch, err = f.resolve(directive(c.PAPause, c.OutPartial, nil))
	if err != nil {
		t.Fatal(err)
	}
	requireStatus(t, patch, c.PIPaused)
	f.in.Active.Status = c.PIPaused
	f.r.snapshot.Protocol.Status = c.PIPaused
	f.in.UserAffirmed = true
	f.in.WCtx.Turn.Text = "بله، حالا دوست دارم همون تمرین رو ادامه بدیم"
	step, patch, err = f.resolve(directive(c.PAAdvance, c.OutSuccess, nil))
	if err != nil || step.Instruction == "" {
		t.Fatal("natural Persian agreement refused", step, patch, err)
	}
	requireStatus(t, patch, c.PIActive)
	f.in.Active.Status = c.PIActive
	f.r.snapshot.Protocol.Status = c.PIActive
	f.in.Risk = c.RiskHigh
	step, patch, err = f.resolve(directive(c.PARepeat, c.OutPartial, nil))
	if err != nil || step.Instruction != "" {
		t.Fatal("risk continued exercise", step, patch, err)
	}
	requireStatus(t, patch, c.PIStopped)
}
