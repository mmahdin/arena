package mssql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"testing"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/retrieval"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

// No separate chat consent is needed for messages in the active conversation.
// This intentionally exercises the real source verifier with a policy that
// refuses all secondary message/memory use.
func TestUnifiedRecentChatDoesNotRequireMessageConsent(t *testing.T) {
	owner, turn, current, prior := c.NewID(), c.NewID(), c.NewID(), c.NewID()
	db := diagnosticDB(t,
		diagnosticQuery{values: []driver.Value{int64(20)}},
		diagnosticQuery{values: []driver.Value{"user", "continue", int64(20)}},
		diagnosticQuery{values: []driver.Value{"user", "hello", int64(10)},
			requireSQL: "m.conversation_id=(SELECT active.conversation_id FROM dbo.turns active WHERE active.id=@p3 AND active.user_id=@p2)"},
	)
	v, err := NewSourceVerifier(db, diagnosticPolicy{reason: "consent_missing"})
	if err != nil {
		t.Fatal(err)
	}
	in := c.WorkingContext{
		Turn:   c.TurnView{TurnID: turn, MessageID: current, Text: "continue"},
		Recent: []c.MsgView{{ID: prior, Role: c.RoleUser, Text: "hello"}},
	}
	if err := v.VerifyContext(context.Background(), owner, turn, in); err != nil {
		t.Fatalf("active-conversation history should not require consent: %v", err)
	}
}

func TestUnifiedRecentChatStillRequiresOwnerScopedVisibleSameConversation(t *testing.T) {
	owner, turn, current, prior := c.NewID(), c.NewID(), c.NewID(), c.NewID()
	// A deleted, foreign, or different-conversation message is not returned
	// by the owner-scoped and same-conversation query.
	db := diagnosticDB(t,
		diagnosticQuery{values: []driver.Value{int64(20)}},
		diagnosticQuery{values: []driver.Value{"user", "continue", int64(20)}},
		diagnosticQuery{err: sql.ErrNoRows,
			requireSQL: "m.conversation_id=(SELECT active.conversation_id FROM dbo.turns active WHERE active.id=@p3 AND active.user_id=@p2)"},
	)
	v, err := NewSourceVerifier(db, diagnosticPolicy{reason: "consent_missing"})
	if err != nil {
		t.Fatal(err)
	}
	err = v.VerifyContext(context.Background(), owner, turn, c.WorkingContext{
		Turn:   c.TurnView{TurnID: turn, MessageID: current, Text: "continue"},
		Recent: []c.MsgView{{ID: prior, Role: c.RoleUser, Text: "hello"}},
	})
	if got := diagnosticMap(err); got["reason"] != "resource_not_visible" || got["failed_check"] != "source_visibility" {
		t.Fatal(got)
	}
}

func TestSecondaryRetrievedHistoryStillChecksSourcePolicy(t *testing.T) {
	owner, turn, current, historical := c.NewID(), c.NewID(), c.NewID(), c.NewID()
	db := diagnosticDB(t,
		diagnosticQuery{values: []driver.Value{int64(20)}},
		diagnosticQuery{values: []driver.Value{"user", "continue", int64(20)}},
	)
	v, err := NewSourceVerifier(db, diagnosticPolicy{reason: "resource_not_visible"})
	if err != nil {
		t.Fatal(err)
	}
	err = v.VerifyContext(context.Background(), owner, turn, c.WorkingContext{
		Turn: c.TurnView{TurnID: turn, MessageID: current, Text: "continue"},
		Retrieved: []c.MemoryView{{ID: historical, Verbatim: true, Text: "old message", Epistemic: c.EpReported,
			Kind: c.MemEpisode, Status: c.MemActive, EvidenceN: 1, Sensitivity: c.SensNormal}},
	})
	if got := diagnosticMap(err); got["reason"] != "resource_not_visible" || got["failed_check"] != "retrieved_source_use" {
		t.Fatal(got)
	}
}

// Keep the owner/visibility source-use policy contract for secondary retrieval intact.
var _ retrieval.UsePolicy = diagnosticPolicy{}
