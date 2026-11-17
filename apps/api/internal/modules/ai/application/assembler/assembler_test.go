package assembler

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/contextstate"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/bundle"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

type stateReader struct{ snapshot contextstate.Snapshot }

func (r *stateReader) Read(_ context.Context, _, _ c.ID) (contextstate.Snapshot, error) {
	return r.snapshot, nil
}
func (r *stateReader) ForTurn(_ context.Context, _, _ c.ID) (contextstate.Snapshot, error) {
	return r.snapshot, nil
}
func (r *stateReader) UserEvidence(_ context.Context, _ c.ID, ids []c.ID) (map[c.ID]bool, error) {
	return nil, nil
}
func (r *stateReader) MissingFinalizations(_ context.Context, _ c.ID) (int, error) { return 0, nil }
func (r *stateReader) ExistingHypotheses(_ context.Context, _ c.ID, ids []c.ID) (map[c.ID]bool, error) {
	return map[c.ID]bool{}, nil
}
func (r *stateReader) Completed(_ context.Context, _, _ c.ID) (bool, error) { return false, nil }

type messages struct {
	result  c.TurnResult
	history []c.Message
}

func (m messages) ValidateOwner(context.Context, c.ID, c.ID) error { return nil }
func (m messages) AdmitUserMessage(context.Context, c.TurnRequest) (c.Admission, error) {
	return c.Admission{}, nil
}
func (m messages) Recent(context.Context, c.ID, c.ID, int) ([]c.Message, error) {
	return m.history, nil
}
func (m messages) GetTurn(context.Context, c.ID, c.ID) (c.TurnResult, error) { return m.result, nil }

type testRegistry struct {
	root string
	fail bool
}

func (r testRegistry) Active(_ context.Context, _ c.ID) (ports.Bundle, error) {
	if r.fail {
		return nil, errors.New("bundle failure")
	}
	return fixtureBundle{root: r.root}, nil
}

type fixtureBundle struct{ root string }

func (f fixtureBundle) Reference() c.BundleRef { return c.BundleRef{Version: "kb@0.1.0"} }
func (f fixtureBundle) Artifact(_ context.Context, path string) ([]byte, error) {
	return os.ReadFile(filepath.Join(f.root, path))
}
func (f fixtureBundle) SyntheticOnly() bool { return true }
func (f fixtureBundle) Version() string     { return "kb@0.1.0" }

func knowledgeRoot(t *testing.T) string {
	t.Helper()
	dir, _ := os.Getwd()
	for i := 0; i < 12; i++ {
		probe := filepath.Join(dir, "knowledge", "bundle.yaml")
		if _, e := os.Stat(probe); e == nil {
			return filepath.Dir(probe)
		}
		dir = filepath.Dir(dir)
	}
	t.Fatal("missing WP-03 knowledge fixture")
	return ""
}
func fixture(t *testing.T) (*Service, *stateReader, c.TurnRequest, c.ID, time.Time) {
	t.Helper()
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	user, conv, client, turn, msg, session, thread := c.NewID(), c.NewID(), c.NewID(), c.NewID(), c.NewID(), c.NewID(), c.NewID()
	state := contextstate.Snapshot{
		Session: c.SessionRecord{ID: session, UserID: user, ConversationID: conv, Mode: c.ModeExploring, FocusThreadID: &thread, Version: 3,
			StartedAt: now.Add(-5 * time.Hour), LastActivityAt: now.Add(-3 * time.Hour), TurnCount: 5},
		User: c.UserState{UserID: user, Locale: "en", TZ: "UTC", Version: 4, ComfortBudget: .5, Risk: c.RiskState{Level: c.RiskNone},
			Prefs: c.PrefsView{Length: c.LengthShort, Tone: c.ToneGentle}},
		Threads: []c.ThreadRecord{{ID: thread, UserID: user, Topic: "Synthetic topic", Phase: c.PhaseExploring,
			Slots: map[string]c.SlotView{"situation": {Slot: "situation", Status: c.EpUnknown}}, LastActiveAt: now.Add(-2 * time.Hour), Version: 2}},
	}
	reader := &stateReader{snapshot: state}
	history := []c.Message{
		{ID: c.NewID(), UserID: user, ConvID: conv, Role: c.RoleUser, Content: "Earlier text", CreatedAt: now.Add(-2 * time.Hour)},
		{ID: c.NewID(), UserID: user, ConvID: conv, Role: c.RoleAssistant, Content: "Earlier answer", CreatedAt: now.Add(-2*time.Hour + time.Minute)},
		{ID: msg, UserID: user, ConvID: conv, Role: c.RoleUser, Content: "Now", CreatedAt: now.Add(-time.Minute)},
	}
	service, err := New(reader, messages{result: c.TurnResult{TurnID: turn, UserMessageID: msg, Status: c.TurnRunning}, history: history}, testRegistry{root: knowledgeRoot(t)})
	if err != nil {
		t.Fatal(err)
	}
	service.WithClock(func() time.Time { return now })
	req := c.TurnRequest{UserID: user, ConversationID: conv, ClientTurnID: client, Text: "Now", Locale: "en", TZ: "UTC", Origin: c.OriginUser}
	return service, reader, req, msg, now
}
func TestWorkingContextFromWP02AndPinnedWP03(t *testing.T) {
	a, reader, req, trigger, now := fixture(t)
	ctx, snap, err := a.Build(context.Background(), req, c.SafetySignal{Level: c.RiskLow}, c.RetrievalResult{})
	if err != nil {
		t.Fatal(err)
	}
	if ctx.SchemaVersion != "wctx.v1" || ctx.Turn.Text != "Now" || ctx.Turn.MessageID != trigger || ctx.Turn.TurnID == req.ClientTurnID {
		t.Fatalf("wrong current turn: %+v", ctx.Turn)
	}
	if len(ctx.Recent) != 2 || ctx.Recent[0].Role != c.RoleUser || ctx.Recent[1].Role != c.RoleAssistant {
		t.Fatalf("unexpected history: %+v", ctx.Recent)
	}
	if ctx.Turn.SinceLastMsg < time.Hour || ctx.Risk.Current != c.RiskLow || !ctx.Session.Resumed || ctx.Resumption.Band != "short" {
		t.Fatalf("context state: %+v", ctx)
	}
	if ctx.FocusThread == nil || len(ctx.FocusThread.Slots) != 3 || ctx.FocusThread.Slots[0].Slot != "situation" {
		t.Fatalf("schema completeness: %+v", ctx.FocusThread)
	}
	if len(ctx.InfoNeeds) != 2 || ctx.InfoNeeds[0].Slot != "situation" || ctx.InfoNeeds[1].Slot != "emotions" {
		t.Fatalf("information needs: %+v", ctx.InfoNeeds)
	}
	if ctx.Eligibility.ProtocolsPossible || ctx.Eligibility.MayVoiceHypothesis {
		t.Fatalf("synthetic bundle exposed protocols: %+v", ctx.Eligibility)
	}
	if snap.Session.ID != reader.snapshot.Session.ID || snap.Versions.User != 4 || snap.Versions.Session != 3 || snap.Versions.Thread != 2 {
		t.Fatalf("snapshot versions: %+v", snap)
	}
	if ctx.TokenEstimate <= 0 || ctx.Turn.LocalTime != "Mon 12:00" {
		t.Fatalf("bad metadata %+v / now=%s", ctx.Turn, now)
	}
}
func TestNeedsSuppressionAndMemoryPause(t *testing.T) {
	a, reader, req, _, _ := fixture(t)
	reader.snapshot.Session.Mode = c.ModeVenting
	reader.snapshot.User.MemoryPaused = true
	mem := c.MemoryView{ID: c.NewID(), Kind: c.MemFact, Status: c.MemActive, Epistemic: c.EpReported, Text: "do-not-use"}
	wc, _, err := a.Build(context.Background(), req, c.SafetySignal{Level: c.RiskElevated}, c.RetrievalResult{Items: []c.MemoryView{mem}})
	if err != nil {
		t.Fatal(err)
	}
	if len(wc.InfoNeeds) != 0 || len(wc.Retrieved) != 0 || wc.Eligibility.MemoryAvailable || wc.Risk.Current != c.RiskElevated {
		t.Fatalf("no consent/distress gate: %+v", wc)
	}
	reader.snapshot.Session.Mode = c.ModeExploring
	reader.snapshot.User.MemoryPaused = false
	past := time.Date(2026, 11, 15, 18, 0, 0, 0, time.UTC)
	reader.snapshot.Threads[0].Slots["emotions"] = c.SlotView{Slot: "emotions", Status: c.EpUnknown, Asked: &past}
	reader.snapshot.Threads[0].Slots["situation"] = c.SlotView{Slot: "situation", Status: c.EpUnknown, Declined: true}
	wc, _, err = a.Build(context.Background(), req, c.SafetySignal{Level: c.RiskNone}, c.RetrievalResult{Meta: c.RetrievalMeta{Degraded: true}, Items: []c.MemoryView{mem}})
	if err != nil {
		t.Fatal(err)
	}
	if len(wc.InfoNeeds) != 0 || len(wc.Retrieved) != 0 || wc.Eligibility.MemoryAvailable {
		t.Fatalf("declines/recent ask/retrieval outage: %+v", wc)
	}
}
func TestWP03RegistryCompatibilityAndFailureClosed(t *testing.T) {
	// Real WP-03 registry (dev synthetic mode), not a new Knowledge loader.
	root := knowledgeRoot(t)
	reg := bundle.NewRegistry(bundle.Options{CoreVersion: "1.0.0"}, true)
	report, err := reg.Activate(context.Background(), root)
	if err != nil || !report.OK() {
		t.Fatalf("WP03 activate failed: %v %+v", err, report)
	}
	p, err := contextstate.LoadPolicy(context.Background(), reg, c.NewID())
	if err != nil || p.Problem.Schema != "problem.v1" || p.Ref.Version == "" || !p.Synthetic {
		t.Fatalf("real bundle integration failed: %+v %v", p, err)
	}
	a, _, req, _, _ := fixture(t)
	a.bundles = testRegistry{fail: true}
	if _, _, err = a.Build(context.Background(), req, c.SafetySignal{Level: c.RiskNone}, c.RetrievalResult{}); err == nil {
		t.Fatal("missing bundle accepted")
	}
}
func TestInvalidOwnerAndCancellation(t *testing.T) {
	a, r, req, _, _ := fixture(t)
	r.snapshot.Session.UserID = c.NewID()
	if _, _, err := a.Build(context.Background(), req, c.SafetySignal{Level: c.RiskNone}, c.RetrievalResult{}); err == nil {
		t.Fatal("cross-user state accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := a.Build(ctx, req, c.SafetySignal{}, c.RetrievalResult{}); !errors.Is(err, context.Canceled) {
		t.Fatal("cancellation ignored", err)
	}
}
func TestRecentIsBoundedAndUnicodeSafe(t *testing.T) {
	now := time.Now().UTC()
	user, conv := c.NewID(), c.NewID()
	in := []c.Message{}
	for i := 0; i < 30; i++ {
		in = append(in, c.Message{ID: c.NewID(), UserID: user, ConvID: conv, Role: c.RoleUser, Content: string(make([]rune, 1200)) + "سلام 🌙", CreatedAt: now})
	}
	out := buildRecent(in, now)
	if len(out) > 8 {
		t.Fatalf("too many recent entries: %d", len(out))
	}
	sum := 0
	for _, msg := range out {
		sum += (len([]byte(msg.Text)) + 3) / 4
	}
	if sum > recentTokenBudget {
		t.Fatalf("token budget exceeded: %d", sum)
	}
	if got := clipRunes("سلام 🌙", 10); got == "" {
		t.Fatal("unicode clipping returned empty")
	}
}

func TestFutureQueuedMessageNotIncludedInContext(t *testing.T) {
	a, _, req, _, now := fixture(t)
	m := a.conversations.(messages)
	later := c.Message{ID: c.NewID(), UserID: req.UserID, ConvID: req.ConversationID, Role: c.RoleUser, Content: "queued later turn private", CreatedAt: now}
	m.history = append(m.history, later)
	a.conversations = m
	wc, _, err := a.Build(context.Background(), req, c.SafetySignal{Level: c.RiskLow}, c.RetrievalResult{})
	if err != nil {
		t.Fatal(err)
	}
	if len(wc.Recent) != 2 || wc.Recent[len(wc.Recent)-1].Text == later.Content {
		t.Fatalf("future queued message leaked: %+v", wc.Recent)
	}
}
