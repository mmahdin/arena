package mssql

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/llm"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

func TestTelemetryRejectsUntrustedLabelsAndInvalidStatus(t *testing.T) {
	if _, err := NewTelemetryStore(nil); err == nil {
		t.Fatal("nil storage accepted")
	}
	store := &TelemetryStore{db: &sql.DB{}} // validation happens BEFORE any SQL call
	ctx := context.Background()
	validStart := llm.AttemptStart{ID: contract.NewID(), Tier: ports.ComponentBrain, Kind: "structured", Provider: "provider", Model: "model-1", StartedAt: time.Now()}
	a := validStart
	a.Provider = "prompt with private text"
	if err := store.BeginAttempt(ctx, a); err == nil {
		t.Fatal("unsanitized provider accepted")
	}
	a = validStart
	a.InputRate = -1
	if err := store.BeginAttempt(ctx, a); err == nil {
		t.Fatal("negative price accepted")
	}
	a = validStart
	a.Kind = "arbitrary_payload"
	if err := store.BeginAttempt(ctx, a); err == nil {
		t.Fatal("arbitrary attempt kind accepted")
	}
	finish := llm.AttemptFinish{ID: contract.NewID(), FinishedAt: time.Now(), Status: "succeeded", CostStatus: "unknown", UsageKnown: true}
	finish.CostStatus = "estimated"
	finish.UsageKnown = false
	if err := store.FinishAttempt(ctx, finish); err == nil {
		t.Fatal("estimated cost without usage accepted")
	}
	if err := store.RecordStage(ctx, StageEvent{TurnID: contract.NewID(), Stage: "private message", Status: "success"}); err == nil {
		t.Fatal("unapproved stage accepted")
	}
	if err := store.RecordStage(ctx, StageEvent{TurnID: contract.NewID(), Stage: "brain", Status: "success", RuleID: "SECRET"}); err == nil {
		t.Fatal("unapproved rule accepted")
	}
	if err := store.Heartbeat(ctx, contract.NewID(), "privacy", "running"); err == nil {
		t.Fatal("unexpected worker type accepted")
	}
	if err := store.Heartbeat(ctx, contract.NewID(), "memory", "everything_is_fine"); err == nil {
		t.Fatal("unexpected worker status accepted")
	}
}
