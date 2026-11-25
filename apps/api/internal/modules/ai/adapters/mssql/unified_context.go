package mssql

import (
	"context"
	"database/sql"
	mem "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/retrieval"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"time"
)

// UnifiedContextVerifier reuses the source/version/privacy verification shared
// with historical records, without invoking legacy reasoning components.
type UnifiedContextVerifier struct {
	sources     *SourceVerifier
	hintTimeout time.Duration
}

func NewUnifiedContextVerifier(db *sql.DB, use retrieval.UsePolicy, cipher mem.Cipher, timeout ...time.Duration) (*UnifiedContextVerifier, error) {
	v, e := NewSourceVerifier(db, use, cipher)
	if e != nil {
		return nil, e
	}
	var limit time.Duration
	if len(timeout) > 0 {
		limit = timeout[0]
	}
	return &UnifiedContextVerifier{sources: v, hintTimeout: limit}, nil
}
func (v *UnifiedContextVerifier) VerifyContext(ctx context.Context, user, turn c.ID, w c.WorkingContext) error {
	deny := func(cause error) error { return &c.Error{Code: c.ErrForbidden, Step: "unified_context", Cause: cause} }
	var active int
	if e := v.sources.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM dbo.users WHERE id=@p1 AND deleted_at IS NULL`, user).Scan(&active); e != nil || active != 1 {
		return deny(e)
	}
	repo := &RetrievalRepository{v.sources.db, v.sources.use, v.sources.cipher}
	q := retrieval.Query{UserID: user, TriggerID: w.Turn.MessageID}
	hypotheses := []c.HypView{}
	if w.FocusThread != nil {
		hypotheses = append(hypotheses, w.FocusThread.Hypotheses...)
	}
	for _, thread := range w.ActiveThreads {
		hypotheses = append(hypotheses, thread.Hypotheses...)
	}
	for _, h := range hypotheses {
		item, found, e := repo.Lookup(ctx, q, retrieval.SourceMemory, h.ID)
		if e != nil || !found || item.Kind != c.MemHypothesis || item.Epistemic != c.EpInferred || item.Status != h.Status || item.SourceVersion != h.SourceVersion || !sourcePrefix(item.Text, h.Text) {
			return deny(e)
		}
	}
	forms := append([]c.FormulationBrief{}, w.Formulations...)
	if w.Formulation != nil {
		forms = append(forms, *w.Formulation)
	}
	for _, f := range forms {
		item, found, e := repo.Lookup(ctx, q, retrieval.SourceFormulation, f.ID)
		if e != nil || !found || item.SourceVersion != f.SourceVersion || !sourcePrefix(item.Text, f.Paragraph) {
			return deny(e)
		}
	}
	check := w
	check.Retrieved = append(append([]c.MemoryView{}, w.Retrieved...), w.Persistent.Items...)
	// All hypotheses have source metadata in the baseline. Avoid the historical
	// plaintext-only hypothesis verifier; encrypted sources use source.Open.
	if check.FocusThread != nil {
		copy := *check.FocusThread
		copy.Hypotheses = nil
		check.FocusThread = &copy
	}
	return v.sources.VerifyContext(ctx, user, turn, check)
}

// Optional stale background recommendations never prevent a conversation turn.
func (v *UnifiedContextVerifier) FilterHints(ctx context.Context, user c.ID, hints []c.FutureHint) []c.FutureHint {
	if v.hintTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, v.hintTimeout)
		defer cancel()
	}
	out := []c.FutureHint{}
	repo := &RetrievalRepository{v.sources.db, v.sources.use, v.sources.cipher}
	for _, hint := range hints {
		if hint.JobID == nil {
			out = append(out, hint)
			continue
		}
		valid := len(hint.Sources) > 0
		for _, source := range hint.Sources {
			item, found, e := repo.source(ctx, user, retrieval.SourceKind(source.Kind), source.ID)
			if e != nil || !found || item.SourceVersion != source.Version || item.Sensitivity == c.SensHighly {
				valid = false
				break
			}
		}
		if valid {
			out = append(out, hint)
		}
	}
	return out
}
