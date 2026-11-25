package orchestrator

import (
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"testing"
)

func TestUnifiedFeedbackContainsOnlySevenRetrievedMemories(t *testing.T) {
	w := c.WorkingContext{}
	refs := []c.ID{}
	for _, source := range []string{"message", "thread", "formulation", "memory", "memory", "memory", "memory", "memory", "memory", "memory", "memory"} {
		item := c.MemoryView{ID: c.NewID(), Source: source}
		w.Retrieved = append(w.Retrieved, item)
		refs = append(refs, item.ID)
	}
	persistent := c.MemoryView{ID: c.NewID(), Source: "memory"}
	w.Persistent.Items = append(w.Persistent.Items, persistent)
	refs = append(refs, persistent.ID)
	w.Retrieved = append(w.Retrieved, w.Retrieved[3])
	got := memoryFeedback(c.StateDelta{SchemaVersion: "state_delta.v2"}, w, c.Extracted{ReferencedContext: refs})
	if len(got.Retrieved) != 7 || len(got.Referenced) != 7 {
		t.Fatalf("feedback exceeds worker contract: %#v", got)
	}
	for i, id := range got.Retrieved {
		if id != w.Retrieved[i+3].ID || got.Referenced[i] != id {
			t.Fatal("feedback includes unrelated state or persistent memory")
		}
	}
}
