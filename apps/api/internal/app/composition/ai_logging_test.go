package composition

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	aidb "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/mssql"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/observability"
)

func TestAITracerEmitsOnlyApprovedMetadata(t *testing.T) {
	var b bytes.Buffer
	logger := observability.NewLogger(&b, slog.LevelDebug, time.UTC)
	id := uuid.New()
	tr := (loggingTracer{logger: logger}).Begin(id, c.CallMeta{UserID: uuid.New()})
	tr.Record("brain", trace.Observation{Duration: 5 * time.Millisecond, Degraded: true, RuleID: "SECRET_mental_health_text"})
	tr.Record("sensitive_user_input", "SECRET mental health text")
	tr.Record("generator", map[string]any{"prompt": "SECRET mental health text"})
	tr.SetRoute(c.RouteNormal)
	tr.Fail("brain", &c.Error{Code: c.ErrProvider, Step: "brain", Cause: errors.New("SECRET token")})
	tr.Finish(context.Background())
	log := b.String()
	if strings.Contains(log, "SECRET") || strings.Contains(log, id.String()) || !strings.Contains(log, "ai.turn.failed") || !strings.Contains(log, "brain") {
		t.Fatalf("unsafe or incomplete core trace: %s", log)
	}
	if !strings.Contains(log, observability.TurnRef(id)) {
		t.Fatalf("missing correlated turn ref: %s", log)
	}
}

func TestAITraceReportsCauseAttemptsAndValidationRules(t *testing.T) {
	var b bytes.Buffer
	logger := observability.NewLogger(&b, slog.LevelDebug, time.UTC)
	id := uuid.New()
	turn := (loggingTracer{logger: logger}).Begin(id, c.CallMeta{})
	cause := &c.Error{Code: c.ErrProvider, Step: "generator", Cause: context.DeadlineExceeded,
		Origin: "apps/api/internal/modules/ai/application/generator/service.go:108", Retryable: true}
	turn.Record("generator", trace.Observation{Duration: 25 * time.Millisecond, Degraded: true,
		Error: cause, Origin: "apps/api/internal/modules/ai/application/orchestrator/orchestrator.go:922",
		Operation: "generate", Attempt: 1})
	turn.Record("validator", trace.Observation{Degraded: true, Reason: "validation_failed", RuleID: "OV-012", Attempt: 2})
	turn.Record("validator", trace.Observation{Degraded: true, Reason: "SECRET private output", RuleID: "SECRET", Attempt: 3})
	turn.Fail("generator", cause)
	turn.Finish(context.Background())
	logs := b.String()
	for _, must := range []string{"ai.stage.degraded", "OV-012", "error_origin", "timeout", "\"attempt\":1", "ai.turn.failed"} {
		if !strings.Contains(logs, must) {
			t.Fatalf("missing %s in %s", must, logs)
		}
	}
	if strings.Contains(logs, "SECRET") || !strings.Contains(logs, "\"degraded\":true") {
		t.Fatalf("unsafe stage output: %s", logs)
	}
}

func TestContextualTracerLevelsOutcomesAndNoDuplicateFailure(t *testing.T) {
	for _, level := range []slog.Level{slog.LevelDebug, slog.LevelInfo, slog.LevelWarn, slog.LevelError} {
		var b bytes.Buffer
		logger := observability.NewLogger(&b, level, time.UTC)
		ctx := observability.WithRequestID(context.Background(), "abcdef12-1234-4abc-9abc-1234567890ab")
		tr := (loggingTracer{logger: logger}).BeginContext(ctx, uuid.New(), c.CallMeta{})
		tr.Record("generator", trace.Observation{Started: true, Operation: "generate"})
		tr.Record("generator", trace.Observation{Duration: time.Millisecond, Operation: "generate"})
		tr.Record("fallback", trace.Observation{Degraded: true, FallbackUsed: true, Reason: "policy_fallback"})
		tr.Fail("generator", &c.Error{Code: c.ErrForbidden, Step: "generator_sources", Reason: "consent_missing", Check: "recent_source_use"})
		tr.Fail("validator", errors.New("PRIVATE duplicate"))
		tr.Finish(ctx)
		counts := map[string]int{}
		for _, line := range strings.Split(strings.TrimSpace(b.String()), "\n") {
			var record map[string]any
			if err := json.Unmarshal([]byte(line), &record); err != nil {
				t.Fatal(err)
			}
			if record["request_id"] != observability.RequestID(ctx) || record["turn_ref"] == "" {
				t.Fatal("missing correlation", record)
			}
			counts[record["msg"].(string)]++
			if record["msg"] == "ai.turn.finished" && record["status"] != "failed" {
				t.Fatal(record)
			}
		}
		if counts["ai.turn.failed"] != 1 {
			t.Fatal("duplicate terminal event", counts)
		}
		if level == slog.LevelDebug && counts["ai.stage.started"] != 1 {
			t.Fatal(counts)
		}
		if level > slog.LevelDebug && counts["ai.stage.completed"] != 0 {
			t.Fatal(counts)
		}
		if level == slog.LevelError && len(counts) != 1 {
			t.Fatal(counts)
		}
	}
}

// BO-04: logging and SQL observability share the same approved stage vocabulary
// but the durable store must receive only typed numeric/allowlisted fields.
type capturedStageStore struct{ events []aidb.StageEvent }

func (s *capturedStageStore) RecordStage(_ context.Context, e aidb.StageEvent) error {
	s.events = append(s.events, e)
	return nil
}
func TestAITracerDurableStageMetadataOnly(t *testing.T) {
	var buffer bytes.Buffer
	sink := &capturedStageStore{}
	id := uuid.New()
	tr := (loggingTracer{logger: observability.NewLogger(&buffer, slog.LevelError, time.UTC), stages: sink}).Begin(id, c.CallMeta{})
	tr.Record("brain", trace.Observation{Duration: 10 * time.Millisecond, RuleID: "OV-012", Attempt: 2, Count: 3})
	tr.Record("private_user_message", map[string]any{"raw": "never store me"})
	tr.Record("validator", trace.Observation{Started: true})
	tr.Record("fallback", trace.Observation{Degraded: true, Reason: "SECRET private message", FallbackUsed: true})
	tr.Fail("unknown_private_stage", errors.New("SECRET vendor response"))
	tr.Finish(context.Background())
	if len(sink.events) != 4 {
		t.Fatalf("expected brain/fallback/fail/finish, got %+v", sink.events)
	}
	if sink.events[0].TurnID != id || sink.events[0].Stage != "brain" || sink.events[0].Status != "success" || sink.events[0].Attempt != 2 || sink.events[0].RuleID != "OV-012" {
		t.Fatalf("wrong stage metadata %+v", sink.events[0])
	}
	if sink.events[2].Stage != "turn" || sink.events[2].Status != "failed" || sink.events[3].Status != "failed" {
		t.Fatalf("invalid terminal status %+v", sink.events)
	}
	for _, e := range sink.events {
		if strings.Contains(e.Stage, "SECRET") || strings.Contains(e.RuleID, "SECRET") {
			t.Fatal("private payload leaked")
		}
	}
}
