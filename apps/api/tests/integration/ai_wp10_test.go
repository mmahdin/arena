package integration

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/fake"
	aidb "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/mssql"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/assembler"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/brain"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/retrieval"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/state"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

type wp10SQLBundle map[string][]byte

func (b wp10SQLBundle) Reference() c.BundleRef {
	return c.BundleRef{Version: "kb@0.9.0", PromptSHAs: map[string]string{"brain": "synthetic"}}
}
func (b wp10SQLBundle) SyntheticOnly() bool { return true }
func (b wp10SQLBundle) ArtifactNames() []string {
	a := []string{}
	for p := range b {
		a = append(a, p)
	}
	return a
}
func (b wp10SQLBundle) Artifact(_ context.Context, p string) ([]byte, error) {
	v, ok := b[p]
	if !ok {
		return nil, errors.New("missing synthetic asset")
	}
	return v, nil
}

type wp10SQLRegistry struct{ b wp10SQLBundle }

func (r wp10SQLRegistry) Active(context.Context, c.ID) (ports.Bundle, error) { return r.b, nil }

type wp10SQLUse struct{ allowed bool }

func (u wp10SQLUse) CanSearch(context.Context, c.ID, c.ID) (bool, error) { return u.allowed, nil }
func (u wp10SQLUse) CanUse(context.Context, c.ID, retrieval.SourceKind, c.ID) (bool, error) {
	return u.allowed, nil
}

// This tests the actual WP-04/08/10 modules and WP-02 SQL Server transaction,
// with synthetic model output. No real psychological data or external API.
func TestWP10SQLStateEvidenceCooldownAndIsolation(t *testing.T) {
	db := aiDatabase(t)
	aiMigrate(t, db)
	f := aiSetup(t, db)
	ctx := context.Background()
	data, e := os.ReadFile("../../internal/modules/ai/application/brain/testdata/assets.json")
	if e != nil {
		t.Fatal(e)
	}
	var files map[string]string
	if json.Unmarshal(data, &files) != nil {
		t.Fatal("invalid test assets")
	}
	b := wp10SQLBundle{}
	for p, s := range files {
		b[p] = []byte(s)
	}
	thresholds, e := os.ReadFile("../../internal/modules/ai/application/orchestrator/testdata/wp03/config/thresholds.yaml")
	if e != nil {
		t.Fatal(e)
	}
	b["config/thresholds.yaml"] = thresholds
	retrievalConfig, e := os.ReadFile("../../internal/modules/ai/application/retrieval/testdata/retrieval.yaml")
	if e != nil {
		t.Fatal(e)
	}
	b["config/retrieval.yaml"] = retrievalConfig
	reg := wp10SQLRegistry{b}
	assets, e := brain.Load(ctx, b, brain.LoadOptions{AllowSyntheticForTests: true})
	if e != nil {
		t.Fatal(e)
	}
	reader, e := aidb.NewContextReader(db)
	if e != nil {
		t.Fatal(e)
	}
	facts, e := aidb.NewBrainFacts(db)
	if e != nil {
		t.Fatal(e)
	}
	verifier, e := aidb.NewBrainContextVerifier(db, wp10SQLUse{true})
	if e != nil {
		t.Fatal(e)
	}
	asm, e := assembler.New(reader, f.store, reg)
	if e != nil {
		t.Fatal(e)
	}
	req := f.req()
	req.Locale = "en"
	req.TZ = "UTC"
	a := admit(t, f, req)
	wc, snap, e := asm.Build(ctx, req, c.SafetySignal{Level: c.RiskLow}, c.RetrievalResult{})
	if e != nil {
		t.Fatal(e)
	}
	provider := fake.New()
	client := wp05Client(t, provider)
	svc, e := brain.New(client, assets, reader, brain.Options{Facts: facts, Verifier: verifier})
	if e != nil {
		t.Fatal(e)
	}
	d := svc.Fallback(wc)
	d.Decision.Confidence = c.ConfHigh
	d.TurnAnalysis.ThreadRelation = c.RelNew
	topic := "Synthetic work event"
	d.TurnAnalysis.NewThreadTopic = &topic
	d.Understanding.Slots = []c.SlotUpdate{{Slot: "situation", Value: "Synthetic input", Status: c.EpReported, Evidence: []c.ID{a.UserMessageID}}}
	raw, e := json.Marshal(d)
	if e != nil {
		t.Fatal(e)
	}
	provider.QueueStructured("wp05-brain", fake.StructuredReply{JSON: raw})
	callCtx := brain.WithCallMeta(ctx, c.CallMeta{UserID: f.a, TurnID: a.TurnID, Bundle: b.Reference()})
	out, e := svc.Decide(callCtx, ports.BrainInput{WCtx: wc, Safety: c.SafetySignal{Level: c.RiskLow}, Bundle: b.Reference()})
	if e != nil || len(out.Decision.Understanding.Slots) != 1 || out.Decision.Understanding.Slots[0].Status != c.EpReported {
		t.Fatal(out, e)
	}
	delta := c.StateDelta{TurnID: a.TurnID, UserID: f.a, ExpectedVersions: snap.Versions, Session: c.SessionPatch{LastObjective: out.Decision.Decision.Primary, TurnCountInc: 1}}
	brain.ProposeDelta(&delta, out.Decision, wc, snap)
	if delta.Thread == nil || delta.Thread.Create == nil {
		t.Fatal("missing new-thread proposal")
	}
	tid := delta.Thread.ThreadID
	commit, e := state.New(reader, reg)
	if e != nil {
		t.Fatal(e)
	}
	checked, rejections, e := commit.ValidateDelta(ctx, delta)
	if e != nil || len(rejections) != 0 {
		t.Fatal(e, rejections)
	}
	preparedTurn := prepared(a)
	preparedTurn.State = checked
	preparedTurn.Jobs = nil
	finalizer, e := state.NewValidatedFinalizer(f.store, commit)
	if e != nil {
		t.Fatal(e)
	}
	result, e := finalizer.Complete(ctx, a, preparedTurn)
	if e != nil || result.Status != c.TurnCompleted {
		t.Fatal(result, e)
	}
	if aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_threads WHERE id=@p1 AND user_id=@p2 AND JSON_VALUE(slots_json,'$.situation.status')=N'reported'`, tid, f.a) != 1 {
		t.Fatal("reported slot not atomically written")
	}
	if aiCount(t, db, `SELECT COUNT(*) FROM dbo.messages WHERE turn_id=@p1`, a.TurnID) != 2 {
		t.Fatal("raw message store duplicated")
	}
	// Check live policy evidence in another admitted turn; no cooldown is written
	// by the Brain. Tests insert the explicit suppression fixture themselves.
	req2 := f.req()
	req2.ConversationID = f.ca2
	req2.Locale = "en"
	req2.TZ = "UTC"
	a2 := admit(t, f, req2)
	allowed, e := facts.HypothesisAllowed(ctx, f.a, a2.TurnID, &tid, "synthetic_pattern")
	if e != nil || allowed {
		t.Fatal("memory pause ignored", allowed, e)
	}
	aiExec(t, db, `UPDATE dbo.ai_user_state SET memory_paused=0 WHERE user_id=@p1`, f.a)
	allowed, e = facts.HypothesisAllowed(ctx, f.a, a2.TurnID, &tid, "synthetic_pattern")
	if e != nil || !allowed {
		t.Fatal("eligible hypothesis denied", allowed, e)
	}
	allowed, e = facts.HypothesisAllowed(ctx, f.a, a2.TurnID, nil, "synthetic_pattern")
	if e != nil || !allowed {
		t.Fatal("new-thread global cooldown check", allowed, e)
	}
	aiExec(t, db, `INSERT dbo.ai_hypothesis_cooldowns(user_id,tag,thread_id,until_at) VALUES(@p1,N'synthetic_pattern',@p2,DATEADD(day,1,SYSUTCDATETIME()))`, f.a, tid)
	allowed, e = facts.HypothesisAllowed(ctx, f.a, a2.TurnID, &tid, "synthetic_pattern")
	if e != nil || allowed {
		t.Fatal("cooldown ignored", allowed, e)
	}
	allowed, e = facts.HypothesisAllowed(ctx, f.b, a2.TurnID, &tid, "other_tag")
	if e != nil || allowed {
		t.Fatal("cross-owner hypothesis allowed", allowed, e)
	}
	allowed, e = facts.OfferAllowed(ctx, f.a, a2.TurnID, 6, 1)
	if e != nil || !allowed {
		t.Fatal("offer quota read", allowed, e)
	}
	allowed, e = facts.OfferAllowed(ctx, f.b, a2.TurnID, 6, 1)
	if e != nil || allowed {
		t.Fatal("cross-owner offer allowed", allowed, e)
	}
	wc2, _, e := asm.Build(ctx, req2, c.SafetySignal{Level: c.RiskLow}, c.RetrievalResult{})
	if e != nil {
		t.Fatal(e)
	}
	// Real WP-09 exact retrieval feeds the real WP-10 schema/gate. Evidence
	// remains raw user messages, never semantic substitutions or summaries.
	repo, e := aidb.NewRetrievalRepository(db)
	if e != nil {
		t.Fatal(e)
	}
	retriever, e := retrieval.New(repo, nil, reg, wp10SQLUse{true})
	if e != nil {
		t.Fatal(e)
	}
	wc2.Turn.TemporalRefs = []c.TemporalRef{{Phrase: "synthetic window", From: time.Now().Add(-time.Hour), To: time.Now().Add(time.Hour), RecallVerb: true}}
	retrieved, e := retriever.Retrieve(ctx, c.RetrievalRequest{UserID: f.a, Turn: wc2.Turn, Strategies: []c.RetrievalStrategy{c.StratTemporal}, MaxItems: 5})
	if e != nil || len(retrieved.Items) != 1 || retrieved.Items[0].ID != a.UserMessageID || !retrieved.Items[0].Verbatim {
		t.Fatal("exact retrieval handoff", retrieved, e)
	}
	wc2.Retrieved = retrieved.Items
	wc2.RetrievalMeta = retrieved.Meta
	wc2.Eligibility.MemoryAvailable = true
	recall := svc.Fallback(wc2)
	recall.TurnAnalysis.Intent = c.IntentAskRecall
	recall.Decision.Primary = c.ObjAnswerDirect
	recall.Plan.PermittedContext = []c.ID{a.UserMessageID}
	raw, _ = json.Marshal(recall)
	provider.QueueStructured("wp05-brain", fake.StructuredReply{JSON: raw})
	recallCtx := brain.WithCallMeta(ctx, c.CallMeta{UserID: f.a, TurnID: a2.TurnID, Bundle: b.Reference()})
	answer, e := svc.Decide(recallCtx, ports.BrainInput{WCtx: wc2, Safety: c.SafetySignal{Level: c.RiskLow}, Bundle: b.Reference()})
	if e != nil || answer.Decision.Decision.Primary != c.ObjAnswerDirect || len(answer.Decision.Plan.PermittedContext) != 1 {
		t.Fatal("sourced recall decision", answer, e)
	}
	// Foreign text and future queued input cannot be sent to the provider.
	foreignReq := f.req()
	foreignReq.UserID = f.b
	foreignReq.ConversationID = f.cb
	foreign := admit(t, f, foreignReq)
	forged := wc2
	forged.Recent = []c.MsgView{{ID: foreign.UserMessageID, Role: c.RoleUser, Text: foreignReq.Text}}
	if verifier.VerifyContext(ctx, f.a, a2.TurnID, forged) == nil {
		t.Fatal("foreign recent source accepted")
	}
	futureReq := f.req()
	future := admit(t, f, futureReq)
	forged.Recent = []c.MsgView{{ID: future.UserMessageID, Role: c.RoleUser, Text: futureReq.Text}}
	if verifier.VerifyContext(ctx, f.a, a2.TurnID, forged) == nil {
		t.Fatal("future queued source accepted")
	}
	// Source content is also verified, not merely the opaque UUID.
	forged = wc2
	forged.Turn.Text = "Invented private facts"
	if verifier.VerifyContext(ctx, f.a, a2.TurnID, forged) == nil {
		t.Fatal("tampered source text accepted")
	}
	deny, _ := aidb.NewBrainContextVerifier(db, wp10SQLUse{false})
	forged = wc2
	forged.Retrieved = []c.MemoryView{{ID: a.UserMessageID, Verbatim: true, Text: req.Text}}
	if deny.VerifyContext(ctx, f.a, a2.TurnID, forged) == nil {
		t.Fatal("explicit use denial ignored")
	}
	// Soft delete hides the resource and blocks this turn's sources, preserving
	// original raw text, slots, references, cooldowns and every derived record.
	aiExec(t, db, `UPDATE dbo.conversations SET deleted_at=SYSUTCDATETIME(),deleted_by=@p2,deletion_reason=N'user_request' WHERE id=@p1 AND user_id=@p2`, f.ca2, f.a)
	if verifier.VerifyContext(ctx, f.a, a2.TurnID, wc2) == nil {
		t.Fatal("hidden conversation accepted")
	}
	if aiCount(t, db, `SELECT COUNT(*) FROM dbo.messages WHERE id=@p1 AND content=@p2`, a2.UserMessageID, req2.Text) != 1 || aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_threads WHERE id=@p1`, tid) != 1 || aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_hypothesis_cooldowns WHERE user_id=@p1`, f.a) != 1 {
		t.Fatal("soft deletion erased original/derived data")
	}
}
