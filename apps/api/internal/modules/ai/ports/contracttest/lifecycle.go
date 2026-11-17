// Package contracttest defines reusable acceptance tests for future AI Core
// adapters. WP-01 runs these against a test-only reference double; WP-02 must run
// them against SQL Server as well. These tests do not establish DB durability.
package contracttest

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

// Fixture must provision two synthetic owners, two conversations for UserA,
// and one for UserB. Every call to factory returns isolated state.
type Fixture struct {
	Store                                                          ports.ConversationStore
	Finalizer                                                      ports.TurnFinalizer
	Serialiser                                                     ports.TurnSerialiser
	UserA, UserB, ConversationA, OtherConversationA, ConversationB contract.ID
	AdvancePastLease                                               func()
	FailNextCompletion                                             func()
	// LeaseTurn provisions a durable queued turn for FK-backed implementations.
	LeaseTurn func(user contract.ID) contract.ID
}
type Factory func(*testing.T) Fixture

func request(f Fixture) contract.TurnRequest {
	return contract.TurnRequest{UserID: f.UserA, ConversationID: f.ConversationA, ClientTurnID: contract.NewID(), Text: "Synthetic contract test", Locale: "fa", TZ: "Asia/Tehran", Channel: "app_chat", Origin: contract.OriginUser}
}

func CheckLifecycle(t *testing.T, factory Factory) {
	t.Helper()
	ctx := context.Background()
	t.Run("concurrent_duplicate_admission", func(t *testing.T) {
		f := factory(t)
		req := request(f)
		var wg sync.WaitGroup
		ids := make(chan contract.ID, 8)
		for i := 0; i < 8; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				a, err := f.Store.AdmitUserMessage(ctx, req)
				if err != nil {
					t.Error(err)
					return
				}
				ids <- a.UserMessageID
			}()
		}
		wg.Wait()
		close(ids)
		var first contract.ID
		for id := range ids {
			if first == (contract.ID{}) {
				first = id
			}
			if id != first {
				t.Fatal("duplicate admission changed raw message ID")
			}
		}
		messages, err := f.Store.Recent(ctx, req.UserID, req.ConversationID, 10)
		if err != nil || len(messages) != 1 {
			t.Fatal("concurrent replay duplicated input")
		}
	})
	t.Run("stale_or_foreign_finalization_is_rejected", func(t *testing.T) {
		f := factory(t)
		req := request(f)
		a, err := f.Store.AdmitUserMessage(ctx, req)
		if err != nil {
			t.Fatal(err)
		}
		p := contract.PreparedTurn{TurnID: a.TurnID, UserMessageID: a.UserMessageID, Text: "Validated synthetic answer", Route: contract.RouteNormal, Objective: contract.ObjReflect, State: contract.StateDelta{TurnID: a.TurnID, UserID: req.UserID, ExpectedVersions: a.Versions, Session: contract.SessionPatch{LastObjective: contract.ObjReflect, TurnCountInc: 1}}}
		foreign := a
		foreign.UserID = f.UserB
		if _, err := f.Finalizer.Complete(ctx, foreign, p); err == nil {
			t.Fatal("foreign admission completed")
		}
		f.AdvancePastLease()
		if _, err := f.Serialiser.Acquire(ctx, f.UserA, a.TurnID); err != nil {
			t.Fatal(err)
		}
		if _, err := f.Finalizer.Complete(ctx, a, p); err == nil {
			t.Fatal("stale completion succeeded")
		}
		if err := f.Finalizer.Fail(ctx, a, contract.ErrGenFailed); err == nil {
			t.Fatal("stale failure changed turn")
		}
		messages, err := f.Store.Recent(ctx, req.UserID, req.ConversationID, 10)
		if err != nil || len(messages) != 1 {
			t.Fatal("invalid completion wrote an assistant message")
		}
	})
	t.Run("user_first_and_completed_replay", func(t *testing.T) {
		f := factory(t)
		req := request(f)
		a, err := f.Store.AdmitUserMessage(ctx, req)
		if err != nil {
			t.Fatal(err)
		}
		if !contract.ValidID(a.TurnID) || !contract.ValidID(a.UserMessageID) || !contract.ValidID(a.LeaseToken) || a.UserID != req.UserID || a.ConversationID != req.ConversationID {
			t.Fatal("missing fenced ownership")
		}
		messages, err := f.Store.Recent(ctx, req.UserID, req.ConversationID, 10)
		if err != nil || len(messages) != 1 || messages[0].Role != contract.RoleUser || messages[0].ID != a.UserMessageID {
			t.Fatal("raw input not admitted before generation")
		}
		duplicate, err := f.Store.AdmitUserMessage(ctx, req)
		if err != nil || duplicate.TurnID != a.TurnID || duplicate.UserMessageID != a.UserMessageID || duplicate.DuplicateOf == nil {
			t.Fatal("in-flight replay duplicated admission")
		}
		prepared := contract.PreparedTurn{TurnID: a.TurnID, UserMessageID: a.UserMessageID, Text: "Validated synthetic answer", Route: contract.RouteNormal, Objective: contract.ObjReflect, State: contract.StateDelta{TurnID: a.TurnID, UserID: req.UserID, ExpectedVersions: a.Versions, Session: contract.SessionPatch{LastObjective: contract.ObjReflect, TurnCountInc: 1}}}
		result, err := f.Finalizer.Complete(ctx, a, prepared)
		if err != nil || result.Status != contract.TurnCompleted || result.AssistantMessageID == nil {
			t.Fatalf("completion: %v", err)
		}
		replay, err := f.Store.GetTurn(ctx, req.UserID, req.ClientTurnID)
		if err != nil || replay.TurnID != result.TurnID || replay.Text != result.Text || replay.AssistantMessageID == nil || *replay.AssistantMessageID != *result.AssistantMessageID {
			t.Fatal("completed replay changed result")
		}
		if _, err = f.Store.AdmitUserMessage(ctx, req); err != nil {
			t.Fatal(err)
		}
		messages, err = f.Store.Recent(ctx, req.UserID, req.ConversationID, 10)
		if err != nil || len(messages) != 2 || messages[1].Role != contract.RoleAssistant {
			t.Fatal("replay added messages")
		}
		req.Text = "Changed synthetic input"
		if _, err = f.Store.AdmitUserMessage(ctx, req); err == nil {
			t.Fatal("idempotency key accepted changed input")
		}
		req.Text = "Synthetic contract test"
		req.ConversationID = f.OtherConversationA
		if _, err = f.Store.AdmitUserMessage(ctx, req); err == nil {
			t.Fatal("idempotency key moved conversations")
		}
	})
	t.Run("ownership_isolation", func(t *testing.T) {
		f := factory(t)
		req := request(f)
		if _, err := f.Store.AdmitUserMessage(ctx, req); err != nil {
			t.Fatal(err)
		}
		if err := f.Store.ValidateOwner(ctx, f.UserB, f.ConversationA); err == nil {
			t.Fatal("foreign owner accepted")
		}
		if _, err := f.Store.Recent(ctx, f.UserB, f.ConversationA, 10); err == nil {
			t.Fatal("foreign history exposed")
		}
		if _, err := f.Store.GetTurn(ctx, f.UserB, req.ClientTurnID); err == nil {
			t.Fatal("foreign replay exposed")
		}
		req.UserID = f.UserB
		if _, err := f.Store.AdmitUserMessage(ctx, req); err == nil {
			t.Fatal("foreign write accepted")
		}
	})
	t.Run("failed_turn_keeps_user_message", func(t *testing.T) {
		f := factory(t)
		req := request(f)
		a, err := f.Store.AdmitUserMessage(ctx, req)
		if err != nil {
			t.Fatal(err)
		}
		if err = f.Finalizer.Fail(ctx, a, contract.ErrGenFailed); err != nil {
			t.Fatal(err)
		}
		result, err := f.Store.GetTurn(ctx, req.UserID, req.ClientTurnID)
		if err != nil || result.Status != contract.TurnFailed || result.AssistantMessageID != nil || result.Text != "" {
			t.Fatal("failed turn exposed assistant draft")
		}
		messages, err := f.Store.Recent(ctx, req.UserID, req.ConversationID, 10)
		if err != nil || len(messages) != 1 || messages[0].ID != a.UserMessageID {
			t.Fatal("failure lost raw input")
		}
	})
	t.Run("completion_failure_has_no_partial_assistant", func(t *testing.T) {
		f := factory(t)
		req := request(f)
		a, err := f.Store.AdmitUserMessage(ctx, req)
		if err != nil {
			t.Fatal(err)
		}
		f.FailNextCompletion()
		_, err = f.Finalizer.Complete(ctx, a, contract.PreparedTurn{TurnID: a.TurnID, UserMessageID: a.UserMessageID, Text: "Validated synthetic answer", Route: contract.RouteNormal, Objective: contract.ObjReflect, State: contract.StateDelta{TurnID: a.TurnID, UserID: req.UserID, ExpectedVersions: a.Versions, Session: contract.SessionPatch{LastObjective: contract.ObjReflect, TurnCountInc: 1}}})
		if err == nil {
			t.Fatal("injected failure reported success")
		}
		result, err := f.Store.GetTurn(ctx, req.UserID, req.ClientTurnID)
		if err != nil || result.Status == contract.TurnCompleted || result.AssistantMessageID != nil {
			t.Fatal("partial completion")
		}
		messages, err := f.Store.Recent(ctx, req.UserID, req.ConversationID, 10)
		if err != nil || len(messages) != 1 {
			t.Fatal("partial assistant write")
		}
	})
	t.Run("cancelled_and_expired_contexts_do_not_write", func(t *testing.T) {
		f := factory(t)
		req := request(f)
		cancelled, cancel := context.WithCancel(ctx)
		cancel()
		expired, stop := context.WithDeadline(ctx, time.Now().Add(-time.Second))
		defer stop()
		for _, c := range []context.Context{cancelled, expired} {
			if _, err := f.Store.AdmitUserMessage(c, req); !errors.Is(err, c.Err()) {
				t.Fatalf("admission ignored context: %v", err)
			}
		}
		messages, err := f.Store.Recent(ctx, req.UserID, req.ConversationID, 10)
		if err != nil || len(messages) != 0 {
			t.Fatal("cancelled admission wrote")
		}
		a, err := f.Store.AdmitUserMessage(ctx, req)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = f.Finalizer.Complete(cancelled, a, contract.PreparedTurn{}); !errors.Is(err, context.Canceled) {
			t.Fatal("completion ignored cancellation")
		}
	})
}

func CheckLeases(t *testing.T, factory Factory) {
	t.Helper()
	t.Run("per_user_fencing_and_expiry", func(t *testing.T) {
		f := factory(t)
		ctx := context.Background()
		turnFor := func(user contract.ID) contract.ID {
			if f.LeaseTurn != nil {
				return f.LeaseTurn(user)
			}
			return contract.NewID()
		}
		first, err := f.Serialiser.Acquire(ctx, f.UserA, turnFor(f.UserA))
		if err != nil {
			t.Fatal(err)
		}
		if first.UserID != f.UserA || !contract.ValidID(first.Token) || first.ExpiresAt.IsZero() {
			t.Fatal("invalid lease")
		}
		// The serialiser has no conversation argument: the lock spans all chats.
		if _, err = f.Serialiser.Acquire(ctx, f.UserA, turnFor(f.UserA)); err == nil {
			t.Fatal("two active turns for one user")
		}
		if _, err = f.Serialiser.Acquire(ctx, f.UserB, turnFor(f.UserB)); err != nil {
			t.Fatal("different user blocked")
		}
		f.AdvancePastLease()
		second, err := f.Serialiser.Acquire(ctx, f.UserA, turnFor(f.UserA))
		if err != nil {
			t.Fatal(err)
		}
		if second.Token == first.Token {
			t.Fatal("recovered lease reused fencing token")
		}
		_ = f.Serialiser.Release(ctx, first)
		if _, err = f.Serialiser.Acquire(ctx, f.UserA, turnFor(f.UserA)); err == nil {
			t.Fatal("stale release removed new lease")
		}
		if err = f.Serialiser.Release(ctx, second); err != nil {
			t.Fatal(err)
		}
		if _, err = f.Serialiser.Acquire(ctx, f.UserA, turnFor(f.UserA)); err != nil {
			t.Fatal("lease not released")
		}
	})
}
