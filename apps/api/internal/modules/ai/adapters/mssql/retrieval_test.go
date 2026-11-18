package mssql

import (
	"strings"
	"testing"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/retrieval"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

func TestWP09SQLQueryOwnerAndSoftDeleteGuards(t *testing.T) {
	for _, tc := range []struct {
		name, query string
		guards      []string
	}{
		{"memory", memorySQL, []string{"m.user_id=@p1", "m.deleted_at IS NULL", "trigger_conv.user_id=@p1", "m.updated_at<=trigger_msg.created_at", "ev.user_id=m.user_id", "v.user_id=m.user_id", "OPENJSON"}},
		{"messages", messagesSQL, []string{"v.user_id=@p1", "trigger_conv.user_id=@p1", "m.deleted_at IS NULL", "v.deleted_at IS NULL", "message_order<trigger_msg.message_order"}},
		{"novelty", recentBeforeSQL, []string{"v.user_id=@p1", "m.deleted_at IS NULL", "v.deleted_at IS NULL", "message_order<trigger_msg.message_order"}},
	} {
		for _, needle := range tc.guards {
			if !strings.Contains(tc.query, needle) {
				t.Errorf("%s missing %s", tc.name, needle)
			}
		}
		if strings.Contains(strings.ToUpper(tc.query), "DELETE FROM") {
			t.Errorf("%s contains a hard-delete", tc.name)
		}
	}
}
func TestWP09SQLInputValidation(t *testing.T) {
	if _, err := NewRetrievalRepository(nil); err == nil {
		t.Fatal("nil SQL pool accepted")
	}
	if _, err := NewRetrievalRepositoryTx(nil); err == nil {
		t.Fatal("nil SQL transaction accepted")
	}
	repo := &RetrievalRepository{}
	if _, err := repo.Memories(nil, retrieval.MemoryQuery{UserID: c.NewID(), TriggerID: c.NewID(), Limit: 1000}); err == nil {
		t.Fatal("unbounded candidate pool accepted")
	}
	if _, err := repo.Messages(nil, retrieval.MessageQuery{UserID: c.NewID(), TriggerID: c.NewID(), Limit: 100}); err == nil {
		t.Fatal("invalid date bounds accepted")
	}
}
