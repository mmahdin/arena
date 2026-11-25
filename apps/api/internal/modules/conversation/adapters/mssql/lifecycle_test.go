package mssql

import (
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"strings"
	"testing"
	"time"
)

func TestAdmissionIdentity(t *testing.T) {
	req := c.TurnRequest{UserID: c.NewID(), ConversationID: c.NewID(), ClientTurnID: c.NewID(), Text: "cafe\u0301"}
	normalized, hash, err := normalize(req)
	if err != nil || normalized.Text != "café" {
		t.Fatal(normalized.Text, err)
	}
	req.Text = "café"
	now := time.Now()
	req.ClientTS = &now
	_, retry, err := normalize(req)
	if err != nil || retry != hash {
		t.Fatal("canonical retry identity changed")
	}
	for _, change := range []func(*c.TurnRequest){func(r *c.TurnRequest) { r.ConversationID = c.NewID() }, func(r *c.TurnRequest) { r.Text = "different" }, func(r *c.TurnRequest) { r.Locale = "en" }, func(r *c.TurnRequest) { id := c.NewID(); r.ReplyToID = &id }} {
		r := req
		change(&r)
		_, other, err := normalize(r)
		if err != nil || other == hash {
			t.Fatal("changed input reused identity")
		}
	}
	for _, text := range []string{" ", string([]byte{0xff}), strings.Repeat("😀", 10001)} {
		r := req
		r.Text = text
		if _, _, err := normalize(r); err == nil {
			t.Fatal("invalid input admitted")
		}
	}
}
