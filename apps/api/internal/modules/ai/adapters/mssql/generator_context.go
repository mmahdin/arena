package mssql

import (
	"context"
	"database/sql"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/generator"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/retrieval"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

// GeneratorContextVerifier rechecks source visibility/ownership at the last
// disclosure boundary. Reads only: deletion never mutates or clears content.
type GeneratorContextVerifier struct {
	sources *BrainContextVerifier
	use     retrieval.UsePolicy
}

var _ generator.SourceVerifier = (*GeneratorContextVerifier)(nil)

func NewGeneratorContextVerifier(db *sql.DB, use retrieval.UsePolicy) (*GeneratorContextVerifier, error) {
	sources, err := NewBrainContextVerifier(db, use)
	if err != nil {
		return nil, err
	}
	return &GeneratorContextVerifier{sources, use}, nil
}
func (v *GeneratorContextVerifier) VerifyGeneration(ctx context.Context, user c.ID, in ports.GenInput) error {
	deny := func() error { return &c.Error{Code: c.ErrForbidden, Step: "generator_context"} }
	w := c.WorkingContext{Turn: in.Turn, Recent: in.Recent}
	recent := map[c.ID]c.MsgView{}
	for _, m := range in.Recent {
		allowed, err := v.use.CanUse(ctx, user, retrieval.SourceMessage, m.ID)
		if err != nil || !allowed {
			return deny()
		}
		recent[m.ID] = m
	}
	for _, m := range in.Permitted {
		if m.ID == in.Turn.MessageID {
			if !m.Verbatim || m.Text != in.Turn.Text || m.Epistemic != c.EpReported || m.Kind != "" || m.Status != "" {
				return deny()
			}
			continue
		}
		if source, ok := recent[m.ID]; ok {
			ep := c.EpReported
			if source.Role == c.RoleAssistant {
				ep = c.EpSystemObserved
			}
			if !m.Verbatim || m.Text != source.Text || m.Epistemic != ep || m.Kind != "" || m.Status != "" {
				return deny()
			}
			continue
		}
		// WP-10's verifier checks historical raw excerpts as user evidence.
		// System-observed assistant prose may only arrive as recent history,
		// never as a claimed user fact or independent hypothesis support.
		if m.Verbatim {
			rawRecent := m.Kind == "" && m.Status == ""
			rawRetrieved := m.Kind == c.MemEpisode && m.Status == c.MemActive && m.EvidenceN == 1 && m.Sensitivity == c.SensNormal
			if m.Epistemic != c.EpReported || !rawRecent && !rawRetrieved {
				return deny()
			}
		}
		if !m.Verbatim {
			var evidence int
			var sensitivity c.Sensitivity
			err := v.sources.db.QueryRowContext(ctx, `SELECT (SELECT COUNT(*) FROM dbo.ai_memory_evidence ev WHERE ev.item_id=m.id AND ev.user_id=m.user_id),m.sensitivity FROM dbo.ai_memory_items m JOIN dbo.turns t ON t.id=@p3 AND t.user_id=m.user_id JOIN dbo.messages trigger_msg ON trigger_msg.id=t.user_message_id WHERE m.id=@p1 AND m.user_id=@p2 AND m.updated_at<=trigger_msg.created_at AND m.deleted_at IS NULL`, m.ID, user, in.Turn.TurnID).Scan(&evidence, &sensitivity)
			if err != nil || evidence < 1 || m.EvidenceN != evidence || m.Sensitivity != sensitivity {
				return deny()
			}
		}
		w.Retrieved = append(w.Retrieved, m)
	}
	return v.sources.VerifyContext(ctx, user, in.Turn.TurnID, w)
}
