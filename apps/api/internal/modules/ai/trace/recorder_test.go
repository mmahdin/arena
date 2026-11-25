package trace_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
)

func TestTraceNeverStoresPayloadsOrRawErrors(t *testing.T) {
	recorder := trace.NewRecorder()
	id := c.NewID()
	tr := recorder.Begin(id, c.CallMeta{TurnID: id, UserID: c.NewID()})
	tr.Record("generator", "private message must never be recorded")
	tr.Record("unapproved_step", trace.Observation{Count: 100})
	tr.Record("generator", trace.Observation{Duration: 11 * time.Millisecond, Count: 1, RuleID: "malicious raw route"})
	tr.Record("route", trace.Observation{RuleID: "RT-099"})
	tr.AddUsage(c.LLMUsage{Provider: "private-provider-label", Model: "private-model-label", InTokens: 7, OutTokens: 11, CostMicroUSD: 5, FinishReason: "raw user message"})
	tr.Fail("generator", &c.Error{Code: c.ErrGenFailed, Cause: context.Canceled})
	tr.SetRoute(c.RouteNormal)
	tr.Finish(context.Background())
	tr.Record("generator", trace.Observation{Count: 999})
	snap, ok := recorder.Snapshot(id)
	if !ok || !snap.Finished || len(snap.Steps) != 3 || snap.Steps[2].ErrorCode != c.ErrGenFailed || snap.Steps[0].RuleID != "" || snap.Steps[1].RuleID != "RT-099" || snap.Usage.InTokens != 7 {
		t.Fatalf("metadata trace invalid: %+v", snap)
	}
	raw, _ := json.Marshal(snap)
	for _, secret := range []string{"private message", "private-provider-label", "private-model-label", "raw user message"} {
		if strings.Contains(string(raw), secret) {
			t.Fatalf("private payload leaked: %s", secret)
		}
	}
}
