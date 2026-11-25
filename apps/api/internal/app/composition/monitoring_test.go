package composition

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/llm"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/metrics"
)

type testJournal struct{ fail bool }

func (j *testJournal) BeginAttempt(context.Context, llm.AttemptStart) error { return nil }
func (j *testJournal) FinishAttempt(context.Context, llm.AttemptFinish) error {
	if j.fail {
		return errors.New("SECRET SQL error")
	}
	return nil
}

type testPulse struct{ fail bool }

func (p testPulse) Heartbeat(context.Context, contract.ID, string, string) error {
	if p.fail {
		return errors.New("SECRET SQL error")
	}
	return nil
}

func TestMetricsOnlyAfterDurableJournalAndHeartbeat(t *testing.T) {
	m := metrics.New()
	j := &testJournal{fail: true}
	wrapped := meteredAttemptJournal{next: j, meter: m}
	e := llm.AttemptFinish{Status: "succeeded", CostStatus: "estimated", Duration: time.Second, UsageKnown: true, InTokens: 100, OutTokens: 20, CostMicroUSD: 50}
	if err := wrapped.FinishAttempt(context.Background(), e); err == nil {
		t.Fatal("SQL failure ignored")
	}
	p := meteredWorkerPulse{next: testPulse{fail: true}, meter: m}
	if err := p.Heartbeat(context.Background(), contract.NewID(), "memory", "running"); err == nil {
		t.Fatal("pulse failure ignored")
	}
	r := httptest.NewRecorder()
	m.Handler().ServeHTTP(r, httptest.NewRequest("GET", "/metrics", nil))
	if strings.Contains(r.Body.String(), `theia_llm_attempts_total{`) || strings.Contains(r.Body.String(), `theia_worker_running{`) {
		t.Fatal("invented metrics after SQL error")
	}
	j.fail = false
	if err := wrapped.FinishAttempt(context.Background(), e); err != nil {
		t.Fatal(err)
	}
	p = meteredWorkerPulse{next: testPulse{}, meter: m}
	if err := p.Heartbeat(context.Background(), contract.NewID(), "memory", "running"); err != nil {
		t.Fatal(err)
	}
	r = httptest.NewRecorder()
	m.Handler().ServeHTTP(r, httptest.NewRequest("GET", "/metrics", nil))
	if !strings.Contains(r.Body.String(), `theia_llm_attempts_total{`) || !strings.Contains(r.Body.String(), `theia_worker_running{kind="memory"} 1`) {
		t.Fatal("successful durable actions not exposed")
	}
}
