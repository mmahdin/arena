package integration

import (
	"context"
	aidb "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/mssql"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/assembler"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/retrieval"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"testing"
)

func TestUnifiedPersistentAndRecoveryHistorySQL(t *testing.T) {
	db := aiDatabase(t)
	aiMigrate(t, db)
	f := aiSetup(t, db)
	r := wp12New(t, db)
	previous := wp12Complete(t, f)
	wp12Enable(t, r, f.a)
	report, e := r.service.Apply(r.ctx(f.a, previous.TurnID), f.a, []c.MemoryOpProposal{wp12Op(previous.UserMessageID, "Synthetic stable fact.")}, wp12Origin(r))
	if e != nil || len(report.Applied) != 1 {
		t.Fatal(report, e)
	}
	id := report.Applied[0]
	aiExec(t, db, `UPDATE dbo.ai_memory_items SET pinned=1,importance=0.9 WHERE id=@p1 AND user_id=@p2`, id, f.a)
	req := f.req()
	req.Text = "unrelated synthetic current turn"
	req.Locale = "en"
	admission := admit(t, f, req)
	repo, e := aidb.NewRetrievalRepository(db, syntheticUsePolicy{true}, r.cipher)
	if e != nil {
		t.Fatal(e)
	}
	baseline, e := repo.Baseline(context.Background(), f.a, admission.UserMessageID)
	if e != nil || len(baseline) != 1 || baseline[0].ID != id || len(baseline[0].Evidence) == 0 {
		t.Fatal("pinned context unavailable without semantic search", baseline, e)
	}
	if _, e = repo.Baseline(context.Background(), f.b, admission.UserMessageID); e == nil {
		t.Fatal("foreign trigger accepted")
	}
	// Admit more future messages than the historical Recent(32) limit. The
	// recovering earlier turn still sees its own message and earlier history.
	for i := 0; i < 80; i++ {
		future := f.req()
		future.Text = "FUTURE SYNTHETIC MESSAGE"
		if _, e = f.store.AdmitUserMessage(context.Background(), future); e != nil {
			t.Fatal(e)
		}
	}
	if aiCount(t, db, `SELECT COUNT(*) FROM sys.triggers WHERE name=N'ai_search_lexical_preserve_delete_guard' AND is_disabled=0`) != 1 {
		t.Fatal("lexical index delete guard missing")
	}
	if _, e = db.Exec(`INSERT dbo.ai_search_lexical(user_id,source_kind,source_id,source_version,doc_length,content_key,signature,terms_json) SELECT @p1,N'memory',id,CONVERT(binary(8),search_version),1,REPLICATE('0',64),CONVERT(binary(32),0),N'{}' FROM dbo.ai_memory_items WHERE id=@p2`, f.b, id); e == nil {
		t.Fatal("foreign source index accepted")
	}
	reader, _ := aidb.NewContextReader(db)
	assembly, e := assembler.New(reader, f.store, wp14Registry{internalBundle(t)})
	if e != nil {
		t.Fatal(e)
	}
	wc, _, e := assembly.Build(context.Background(), req, c.SafetySignal{Level: c.RiskNone}, c.RetrievalResult{Meta: c.RetrievalMeta{Degraded: true}})
	if e != nil || wc.Turn.MessageID != admission.UserMessageID || wc.Turn.Text != req.Text {
		t.Fatal("earlier turn lost to future admission", wc.Turn, e)
	}
	for _, message := range wc.Recent {
		if message.Text == "FUTURE SYNTHETIC MESSAGE" {
			t.Fatal("future message leaked into earlier turn")
		}
	}
	// A baseline source cannot be expanded into another user's raw message.
	if _, found, e := repo.Lookup(context.Background(), retrieval.Query{UserID: f.b, TriggerID: admission.UserMessageID}, retrieval.SourceMemory, id); e != nil || found {
		t.Fatal("foreign evidence lookup disclosed data", found, e)
	}
}

func TestUnifiedRollingSummarySQL(t *testing.T) {
	db := aiDatabase(t)
	aiMigrate(t, db)
	f := aiSetup(t, db)
	a := admit(t, f, f.req())
	aiExec(t, db, `UPDATE dbo.ai_sessions SET summary=REPLICATE(N'old ',120) WHERE id=@p1 AND user_id=@p2`, a.SessionID, f.a)
	p := prepared(a)
	p.Jobs = nil
	p.State.SchemaVersion = "state_delta.v2"
	summary := "A corrected current synthetic summary."
	p.State.Session.SummaryReplace = &summary
	if _, e := f.store.Complete(context.Background(), a, p); e != nil {
		t.Fatal(e)
	}
	var actual string
	if e := db.QueryRow(`SELECT summary FROM dbo.ai_sessions WHERE id=@p1 AND user_id=@p2`, a.SessionID, f.a).Scan(&actual); e != nil || actual != summary {
		t.Fatal("rolling replacement appended or froze", actual, e)
	}
}

func TestUnifiedHistoricalRetryCannotReadLaterStateSQL(t *testing.T) {
	db := aiDatabase(t)
	aiMigrate(t, db)
	f := aiSetup(t, db)
	first := f.req()
	a := admit(t, f, first)
	if e := f.store.Fail(context.Background(), a, c.ErrProvider); e != nil {
		t.Fatal(e)
	}
	second := f.req()
	second.Text = "later synthetic state"
	later := admit(t, f, second)
	p := prepared(later)
	p.Jobs = nil
	if _, e := f.store.Complete(context.Background(), later, p); e != nil {
		t.Fatal(e)
	}
	if e := f.store.Retry(context.Background(), f.a, first.ClientTurnID); e == nil {
		t.Fatal("historical retry admitted after later completion")
	}
	reader, _ := aidb.NewContextReader(db)
	if _, e := reader.ForTurn(context.Background(), f.a, a.TurnID); e == nil {
		t.Fatal("later state leaked into historical read")
	}
}
