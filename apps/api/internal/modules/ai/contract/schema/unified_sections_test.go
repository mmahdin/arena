package schema_test

import (
	"encoding/json"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract/schema"
	"testing"
)

func TestUnifiedInvalidOptionalSectionsDoNotDestroyValidReply(t *testing.T) {
	s, e := schema.New(vocabulary(t))
	if e != nil {
		t.Fatal(e)
	}
	raw := []byte(`{"schema_version":"unified_turn.v2","reply":{"text":"Synthetic response.","conversation":{"intent":"other","objective":"reflect"},"used_context_ids":[],"depends_on":[]},"state_proposals":[{"id":"bad_summary","kind":"session_intent","value":{"mode":"invented","answered_pending":false},"depends_on":[]}],"background_requests":[{"id":"bad","kind":"delete_all_data"}],"future_hints":[{"topic":"future","resource_ids":[],"ttl_turns":99}]}`)
	u, e := s.DecodeUnified(raw)
	if e != nil || u.Reply != "Synthetic response." || u.Session != nil || len(u.BackgroundRequests) != 0 || len(u.FutureHints) != 0 || len(u.Rejected) != 3 {
		t.Fatal(u, e)
	}
	u.Session = &c.SessionIntent{AnsweredPending: true}
	u.Proposals = nil
	encoded, e := json.Marshal(u)
	if e != nil {
		t.Fatal(e)
	}
	round, e := s.DecodeUnified(encoded)
	if e != nil || round.Session == nil || !round.Session.AnsweredPending {
		t.Fatal(round, e)
	}
}
