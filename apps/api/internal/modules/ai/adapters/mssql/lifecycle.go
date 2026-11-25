// Package mssql owns AI state and lease SQL. It participates in the
// Conversation-owned finalization transaction through composition injection.
package mssql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/observability"
	"time"

	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

type StateStore struct{}

func NewStateStore() *StateStore { return &StateStore{} }
func conflict() error            { return &c.Error{Code: c.ErrStateConflict, Step: "state"} }
func invalid() error             { return &c.Error{Code: c.ErrValidation, Step: "state"} }
func changed(r sql.Result, err error) error {
	if err != nil {
		return observability.DatabaseError(err, "state_write")
	}
	n, err := r.RowsAffected()
	if err != nil {
		return observability.DatabaseError(err, "state_write")
	}
	if n != 1 {
		return conflict()
	}
	return nil
}
func encode(v any) (string, error) { b, err := json.Marshal(v); return string(b), err }
func (s *StateStore) Admit(ctx context.Context, tx *sql.Tx, req c.TurnRequest, a *c.Admission) error {
	if err := correlateTransaction(ctx, tx, a.UserID, a.TurnID); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `IF NOT EXISTS(SELECT 1 FROM dbo.ai_user_state WHERE user_id=@p1) INSERT dbo.ai_user_state(user_id,tz,locale,memory_paused) VALUES(@p1,@p2,@p3,0);
 UPDATE dbo.ai_user_state SET tz=@p2,locale=@p3,version=version+1 WHERE user_id=@p1 AND (tz<>@p2 OR locale<>@p3);
 IF NOT EXISTS(SELECT 1 FROM dbo.ai_sessions WHERE user_id=@p1 AND conversation_id=@p4 AND closed_at IS NULL) INSERT dbo.ai_sessions(id,user_id,conversation_id) VALUES(@p5,@p1,@p4);`, req.UserID, req.TZ, req.Locale, req.ConversationID, c.NewID())
	if err != nil {
		return observability.DatabaseError(err, "state_write")
	}
	return tx.QueryRowContext(ctx, `SELECT CONVERT(varchar(36),s.id),s.version,u.version FROM dbo.ai_sessions s JOIN dbo.ai_user_state u ON u.user_id=s.user_id WHERE s.user_id=@p1 AND s.conversation_id=@p2 AND s.closed_at IS NULL`, req.UserID, req.ConversationID).Scan(&a.SessionID, &a.Versions.Session, &a.Versions.User)
}
func (s *StateStore) Acquire(ctx context.Context, tx *sql.Tx, user, turn c.ID, ttl time.Duration) (ports.LeaseHandle, error) {
	// Caller holds the user row lock. Expiry uses database time, not worker clocks.
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM dbo.ai_user_turn_leases WHERE user_id=@p1 AND lease_expires_at>SYSUTCDATETIME()`, user).Scan(&count); err != nil {
		return ports.LeaseHandle{}, observability.DatabaseError(err, "state_write")
	}
	if count != 0 {
		return ports.LeaseHandle{}, &c.Error{Code: c.ErrQueued, Step: "lease", Retryable: true}
	}

	h := ports.LeaseHandle{UserID: user, TurnID: turn, Token: c.NewID()}
	_, err := tx.ExecContext(ctx, `UPDATE dbo.ai_user_turn_leases SET turn_id=@p2,lease_token=@p3,lease_expires_at=DATEADD(millisecond,@p4,SYSUTCDATETIME()),updated_at=SYSUTCDATETIME() WHERE user_id=@p1;
 IF @@ROWCOUNT=0 INSERT dbo.ai_user_turn_leases(user_id,turn_id,lease_token,lease_expires_at) VALUES(@p1,@p2,@p3,DATEADD(millisecond,@p4,SYSUTCDATETIME()))`, user, turn, h.Token, ttl.Milliseconds())
	if err != nil {
		return h, observability.DatabaseError(err, "state_write")
	}
	err = tx.QueryRowContext(ctx, `SELECT lease_expires_at FROM dbo.ai_user_turn_leases WHERE user_id=@p1`, user).Scan(&h.ExpiresAt)
	return h, observability.DatabaseError(err, "state_write")
}
func (s *StateStore) Check(ctx context.Context, tx *sql.Tx, h ports.LeaseHandle) error {
	var n int
	err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM dbo.ai_user_turn_leases WHERE user_id=@p1 AND turn_id=@p2 AND lease_token=@p3 AND lease_expires_at>SYSUTCDATETIME()`, h.UserID, h.TurnID, h.Token).Scan(&n)
	if err != nil {
		return observability.DatabaseError(err, "state_write")
	}
	if n != 1 {
		return conflict()
	}
	return nil
}
func (s *StateStore) Release(ctx context.Context, tx *sql.Tx, h ports.LeaseHandle) error {
	return changed(tx.ExecContext(ctx, `UPDATE dbo.ai_user_turn_leases SET lease_expires_at=DATEADD(millisecond,-1,SYSUTCDATETIME()),updated_at=SYSUTCDATETIME() WHERE user_id=@p1 AND turn_id=@p2 AND lease_token=@p3 AND lease_expires_at>SYSUTCDATETIME()`, h.UserID, h.TurnID, h.Token))
}
func (s *StateStore) Renew(ctx context.Context, tx *sql.Tx, h ports.LeaseHandle, ttl time.Duration) (ports.LeaseHandle, error) {
	err := tx.QueryRowContext(ctx, `UPDATE dbo.ai_user_turn_leases SET lease_expires_at=DATEADD(millisecond,@p4,SYSUTCDATETIME()),updated_at=SYSUTCDATETIME() OUTPUT inserted.lease_expires_at WHERE user_id=@p1 AND turn_id=@p2 AND lease_token=@p3 AND lease_expires_at>SYSUTCDATETIME()`, h.UserID, h.TurnID, h.Token, ttl.Milliseconds()).Scan(&h.ExpiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		err = conflict()
	}
	return h, observability.DatabaseError(err, "state_write")
}

func (s *StateStore) Apply(ctx context.Context, tx *sql.Tx, a c.Admission, p c.PreparedTurn) error {
	if err := correlateTransaction(ctx, tx, a.UserID, a.TurnID); err != nil {
		return err
	}
	trace.Capture(ctx, "transaction_state", "transaction_pending", "StateStore.Apply", p)
	d := p.State
	if !c.ValidID(a.SessionID) || d.ExpectedVersions.Session < 1 || d.ExpectedVersions.User < 1 {
		return conflict()
	}
	if err := applyThread(ctx, tx, a, d); err != nil {
		return observability.DatabaseError(err, "state_write")
	}
	if err := applyProtocol(ctx, tx, a, d); err != nil {
		return observability.DatabaseError(err, "state_write")
	}
	if err := applySession(ctx, tx, a, d); err != nil {
		return observability.DatabaseError(err, "state_write")
	}
	if err := applyUser(ctx, tx, a, d); err != nil {
		return observability.DatabaseError(err, "state_write")
	}
	if err := applyOperations(ctx, tx, a, d); err != nil {
		return observability.DatabaseError(err, "state_write")
	}
	// Delta is restricted user data, separate from the redacted trace summary.
	delta, err := encode(d)
	if err != nil {
		return observability.DatabaseError(err, "state_write")
	}
	_, err = tx.ExecContext(ctx, `INSERT dbo.ai_turn_events(turn_id,user_id,type,actor,delta_json) VALUES(@p1,@p2,N'state_applied',N'finalizer',@p3)`, a.TurnID, a.UserID, delta)
	if err != nil {
		return observability.DatabaseError(err, "state_write")
	}
	for _, job := range p.Jobs {
		if job.Kind == "" || job.DedupeKey == "" || !json.Valid(job.Payload) {
			return invalid()
		}
		// Same dedupe key with different content is an error, never silent job loss.
		var payload string
		err = tx.QueryRowContext(ctx, `SELECT payload_json FROM dbo.ai_jobs WITH(FORCESEEK) WHERE user_id=@p1 AND kind=@p2 AND dedupe_key=@p3 AND status IN(N'queued',N'running')`, a.UserID, job.Kind, job.DedupeKey).Scan(&payload)
		if err == nil {
			if payload != string(job.Payload) {
				return conflict()
			}
			continue
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return observability.DatabaseError(err, "state_write")
		}
		jobID := c.NewID()
		if trace.Enabled(ctx) {
			trace.Capture(ctx, "job_insert", "transaction_pending", "ai_jobs", map[string]any{"id": jobID, "user_id": a.UserID, "turn_id": a.TurnID, "kind": job.Kind, "dedupe_key": job.DedupeKey, "payload_json": string(job.Payload)})
		}
		_, err = tx.ExecContext(ctx, `INSERT dbo.ai_jobs(id,user_id,turn_id,kind,dedupe_key,payload_json) VALUES(@p1,@p2,@p3,@p4,@p5,@p6)`, jobID, a.UserID, a.TurnID, job.Kind, job.DedupeKey, string(job.Payload))
		if err != nil {
			return observability.DatabaseError(err, "state_write")
		}
	}
	if p.Decisions != nil {
		carry, e := json.Marshal(p.Decisions.Carry)
		if e != nil {
			return e
		}
		resolution, e := json.Marshal(p.Decisions)
		if e != nil {
			return e
		}
		if len(carry) > 12000 || len(resolution) > 24000 {
			return invalid()
		}
		if _, e = tx.ExecContext(ctx, `INSERT dbo.ai_turn_decisions(turn_id,user_id,carry_json,resolution_json) VALUES(@p1,@p2,@p3,@p4)`, a.TurnID, a.UserID, string(carry), string(resolution)); e != nil {
			return observability.DatabaseError(e, "state_write")
		}
	}
	summary := c.TurnTraceSummary{TurnID: a.TurnID, BundleVersion: p.Trace.BundleVersion, Route: p.Route, Objective: p.Objective}
	raw, err := encode(summary)
	if err != nil {
		return observability.DatabaseError(err, "state_write")
	}
	versions, err := encode(d.ExpectedVersions)
	if err != nil {
		return observability.DatabaseError(err, "state_write")
	}
	_, err = tx.ExecContext(ctx, `INSERT dbo.ai_turn_context(turn_id,user_id,session_id,route,objective,bundle_version) VALUES(@p1,@p2,@p3,@p4,@p5,@p6);
 INSERT dbo.ai_turn_traces(turn_id,user_id,route,versions_json,summary_json) VALUES(@p1,@p2,@p4,@p7,@p8);`, a.TurnID, a.UserID, a.SessionID, string(p.Route), string(p.Objective), p.Trace.BundleVersion, versions, raw)
	if err != nil {
		return observability.DatabaseError(err, "state_write")
	}
	// Outbox payload is intentionally identifiers only, never arbitrary event text.
	events := append([]c.DomainEvent{{Type: c.EvTurnCompleted}}, d.Events...)
	trace.Capture(ctx, "outbox_events", "transaction_pending", "ai_outbox", events)
	for _, event := range events {
		if !event.Type.Valid() {
			return invalid()
		}
		payload, err := encode(struct {
			Schema string `json:"schema"`
			TurnID c.ID   `json:"turn_id"`
		}{"event.v1", a.TurnID})
		if err != nil {
			return observability.DatabaseError(err, "state_write")
		}
		outboxID := c.NewID()
		if trace.Enabled(ctx) {
			trace.Capture(ctx, "outbox_insert", "transaction_pending", "ai_outbox", map[string]any{"id": outboxID, "user_id": a.UserID, "type": event.Type, "payload_json": payload})
		}
		_, err = tx.ExecContext(ctx, `INSERT dbo.ai_outbox(id,user_id,type,payload_json) VALUES(@p1,@p2,@p3,@p4)`, outboxID, a.UserID, string(event.Type), payload)
		if err != nil {
			return observability.DatabaseError(err, "state_write")
		}
	}
	return nil
}
