package generator

import (
	"context"
	"strings"
	"testing"

	ai "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/letters/application"
)

type fakeModel struct {
	answer   string
	messages []ai.ChatMessage
}

func (f *fakeModel) Generate(_ context.Context, messages []ai.ChatMessage) (string, error) {
	f.messages = messages
	return f.answer, nil
}
func TestGeneratorAcceptsStructuredNonClinicalOutput(t *testing.T) {
	m := &fakeModel{answer: `{"title":"یک تجربهٔ کوچک","salutation":null,"paragraphs":["حرف اول آزمایشی","حرف دوم آزمایشی"],"question":null}`}
	g := New(m)
	draft, err := g.Generate(context.Background(), []application.Turn{{Role: "user", Content: "synthetic chat"}})
	if err != nil || len(draft.Paragraphs) != 2 {
		t.Fatal(draft, err)
	}
	if len(m.messages) != 2 || !strings.Contains(m.messages[1].Content, "synthetic chat") {
		t.Fatal("model context missing")
	}
}
func TestGeneratorRejectsUnstructuredOrUnsafeOutput(t *testing.T) {
	for _, answer := range []string{
		`not json`,
		`{"title":"test","paragraphs":["first", "second"]} extra`,
		`{"title":"test","paragraphs":["first", "second"],"extra":"unexpected"}`,
		`{"title":"test","paragraphs":["من روانشناس تو هستم", "second"]}`,
	} {
		_, err := New(&fakeModel{answer: answer}).Generate(context.Background(), []application.Turn{{Role: "user", Content: "synthetic"}})
		if err == nil {
			t.Fatalf("untrusted content accepted: %s", answer)
		}
	}
}
