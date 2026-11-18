package mssql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"time"

	mem "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

// Control is the authenticated command path, independent of AI-use consent.
// Every command has an immutable receipt; retries never extend a cooldown or
// reapply a visibility change. All writes share WP-02's durable owner lock.
func (r *MemoryRepository) Control(ctx context.Context, m mem.Meta, command mem.Control) (out mem.ControlResult, err error) {
	defer func() { err = memorySQLError(err) }()
	if _, e := mem.Auth(mem.WithMeta(ctx, m), m.UserID); e != nil {
		return out, e
	}
	hash := mem.Hash(command)
	// Load encrypted metadata before the short transaction, then fence its version.
	var target *mem.Record
	if command.Kind == "correct" || command.Kind == "pin" {
		m.Targets = []c.ID{command.Target}
		snap, e := r.Snapshot(ctx, m.UserID, m, nil)
		if e != nil {
			return out, e
		}
		for _, v := range snap.Records {
			if v.Item.ID == command.Target {
				copy := v
				target = &copy
			}
		}
		if target == nil {
			return out, mem.ErrDenied
		}
	}
	tx, e := r.begin(ctx, m.UserID)
	if e != nil {
		return out, e
	}
	defer tx.Rollback()
	var oldHash []byte
	var oldJSON string
	e = tx.QueryRowContext(ctx, `SELECT request_hash,result_json FROM dbo.ai_memory_batches WHERE user_id=@p1 AND operation_id=@p2`, m.UserID, m.OperationID).Scan(&oldHash, &oldJSON)
	if e == nil {
		if !reflect.DeepEqual(oldHash, hashBytes(hash)) {
			return out, mem.ErrConflict
		}
		if json.Unmarshal([]byte(oldJSON), &out) != nil {
			return out, mem.ErrDenied
		}
		return out, nil
	}
	if !errors.Is(e, sql.ErrNoRows) {
		return out, e
	}
	switch command.Kind {
	case "retry_job":
		result, e := tx.ExecContext(ctx, `UPDATE dbo.ai_jobs SET status=N'queued',max_attempts=max_attempts+6,run_after=SYSUTCDATETIME() WHERE id=@p1 AND user_id=@p2 AND kind IN(N'memory_write',N'memory_embed',N'memory_feedback',N'memory_maintain',N'reflection') AND status IN(N'failed',N'cancelled') AND max_attempts<=94`, command.Target, m.UserID)
		if e != nil {
			return out, e
		}
		n, _ := result.RowsAffected()
		if n != 1 {
			return out, mem.ErrDenied
		}
	case "correct", "pin":
		if e = archiveMemory(ctx, tx, m.UserID, command.Target); e != nil {
			return out, e
		}
		var res sql.Result
		if command.Kind == "pin" {
			if command.Pinned == nil {
				return out, mem.ErrDenied
			}
			res, e = tx.ExecContext(ctx, `UPDATE dbo.ai_memory_items SET pinned=@p4,version=version+1,updated_at=SYSUTCDATETIME() WHERE id=@p1 AND user_id=@p2 AND version=@p3 AND deleted_at IS NULL`, command.Target, m.UserID, target.Version, *command.Pinned)
		} else {
			if !command.Correction.Valid() || command.CooldownDays < 1 {
				return out, mem.ErrDenied
			}
			status := c.MemContested
			if target.Item.Kind == c.MemHypothesis || command.Correction == "reject" {
				status = c.MemRejected
			}
			res, e = tx.ExecContext(ctx, `UPDATE dbo.ai_memory_items SET status=@p4,confidence=N'low',contradict_n=contradict_n+1,version=version+1,updated_at=SYSUTCDATETIME() WHERE id=@p1 AND user_id=@p2 AND version=@p3 AND deleted_at IS NULL`, command.Target, m.UserID, target.Version, status)
			if e == nil && target.Item.Kind == c.MemHypothesis {
				for _, tag := range target.Item.Tags {
					if e = saveCooldown(ctx, tx, m.UserID, mem.Cooldown{Tag: tag, Item: command.Target, Until: time.Now().UTC().AddDate(0, 0, command.CooldownDays)}); e != nil {
						break
					}
				}
			}
			if e == nil && status == c.MemContested {
				_, e = tx.ExecContext(ctx, `IF NOT EXISTS(SELECT 1 FROM dbo.ai_open_loops WHERE user_id=@p1 AND kind=N'confirm_fact' AND ref_id=@p2 AND status=N'open') INSERT dbo.ai_open_loops(id,user_id,kind,text,ref_id) VALUES(@p3,@p1,N'confirm_fact',N'confirm_fact',@p2)`, m.UserID, command.Target, c.NewID())
			}
		}
		if e != nil {
			return out, e
		}
		n, _ := res.RowsAffected()
		if n != 1 {
			return out, mem.ErrConflict
		}
		// Retain an actor-bound correction source even for controls outside chat.
		if command.Kind == "correct" {
			_, e = tx.ExecContext(ctx, `INSERT dbo.ai_turn_events(id,user_id,turn_id,type,actor,delta_json) VALUES(@p1,@p2,@p3,N'memory_corrected',N'user',@p4); INSERT dbo.ai_memory_evidence(user_id,item_id,ref_kind,ref_id,role,evidence_at) VALUES(@p2,@p5,N'event',@p1,N'contradicts',SYSUTCDATETIME())`, m.OperationID, m.UserID, memoryNullableID(m.TurnID), memoryJSON(struct{ Target c.ID }{command.Target}), command.Target)
			if e != nil {
				return out, e
			}
		}
	case "delete":
		out, e = softDeleteMemory(ctx, tx, m, command)
		if e != nil {
			return out, e
		}
	case "controls":
		if len(command.Consent) > 32 {
			return out, mem.ErrDenied
		}
		var raw string
		var state c.UserState
		if e = loadUser(ctx, tx, m.UserID, &state); e != nil {
			return out, e
		}
		if e = tx.QueryRowContext(ctx, `SELECT consent_json FROM dbo.ai_user_state WHERE user_id=@p1`, m.UserID).Scan(&raw); e != nil {
			return out, e
		}
		consent := map[string]bool{}
		if json.Unmarshal([]byte(raw), &consent) != nil {
			return out, mem.ErrDenied
		}
		for key, value := range command.Consent {
			if strings.TrimSpace(key) == "" || len(key) > 80 {
				return out, mem.ErrDenied
			}
			consent[key] = value
		}
		if command.Pause != nil {
			state.MemoryPaused = *command.Pause
		}
		if command.Prefs != nil {
			state.Prefs = *command.Prefs
		}
		if len(memoryJSON(state.Prefs)) > 4096 {
			return out, mem.ErrDenied
		}
		_, e = tx.ExecContext(ctx, `UPDATE dbo.ai_user_state SET memory_paused=@p2,consent_json=@p3,prefs_json=@p4,version=version+1 WHERE user_id=@p1`, m.UserID, state.MemoryPaused, memoryJSON(consent), memoryJSON(state.Prefs))
		if e != nil {
			return out, e
		}
	case "suppress":
		if len(hashBytes(command.ClaimHash)) != 32 {
			return out, mem.ErrDenied
		}
		_, e = tx.ExecContext(ctx, `INSERT dbo.ai_memory_tombstones(id,user_id,reason,claim_hash,suppress_until) VALUES(@p1,@p2,N'explicit_suppression',@p3,@p4)`, c.NewID(), m.UserID, hashBytes(command.ClaimHash), command.SuppressUntil)
		if e != nil {
			return out, e
		}
	default:
		return out, mem.ErrDenied
	}
	eventType := "memory_control"
	if command.Kind == "correct" {
		eventType = string(c.EvMemoryCorrected)
	}
	if command.Kind == "delete" {
		eventType = string(c.EvMemoryDeleted)
	}
	// IDs and technical command names only; no memory body in the audit outbox.
	_, e = tx.ExecContext(ctx, `INSERT dbo.ai_memory_batches(user_id,operation_id,request_hash,result_json) VALUES(@p1,@p2,@p3,@p4); UPDATE dbo.ai_user_state SET memory_revision=memory_revision+1 WHERE user_id=@p1; INSERT dbo.ai_outbox(id,user_id,type,payload_json) VALUES(@p5,@p1,@p7,@p6)`, m.UserID, m.OperationID, hashBytes(hash), memoryJSON(out), c.NewID(), memoryJSON(struct {
		Operation, Target c.ID
		Kind              string
	}{m.OperationID, command.Target, command.Kind}), eventType)
	if e != nil {
		return out, e
	}
	return out, tx.Commit()
}

func softDeleteMemory(ctx context.Context, tx *sql.Tx, m mem.Meta, command mem.Control) (out mem.ControlResult, err error) {
	s := command.Scope
	selectors := 0
	if len(s.ItemIDs) > 0 {
		selectors++
	}
	if len(s.MessageIDs) > 0 {
		selectors++
	}
	if s.Category != "" {
		selectors++
	}
	if s.All {
		selectors++
	}
	if selectors != 1 || len(s.ItemIDs) > 100 || len(s.MessageIDs) > 100 || (s.Reason != "" && s.Reason != "user_request") || (s.Category != "" && !c.MemoryKind(s.Category).Valid()) {
		return out, mem.ErrDenied
	}
	if len(s.MessageIDs) > 0 {
		for _, id := range s.MessageIDs {
			var found int
			if !c.ValidID(id) {
				return out, mem.ErrDenied
			}
			if e := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM dbo.messages m JOIN dbo.conversations cv ON cv.id=m.conversation_id WHERE m.id=@p1 AND cv.user_id=@p2`, id, m.UserID).Scan(&found); e != nil {
				return out, e
			}
			if found != 1 {
				return out, mem.ErrDenied
			}
		}
		res, e := tx.ExecContext(ctx, `UPDATE m SET deleted_at=SYSUTCDATETIME(),deleted_by=@p1,deletion_reason=N'user_request' FROM dbo.messages m JOIN dbo.conversations cv ON cv.id=m.conversation_id AND cv.user_id=@p1 WHERE m.deleted_at IS NULL AND EXISTS(SELECT 1 FROM OPENJSON(@p2) ids WHERE TRY_CONVERT(uniqueidentifier,ids.value)=m.id)`, m.UserID, arrayJSON(s.MessageIDs))
		if e != nil {
			return out, e
		}
		n, _ := res.RowsAffected()
		out.Deleted.DeletedMessages = int(n)
		return out, nil
	}
	for _, id := range s.ItemIDs {
		var found int
		if !c.ValidID(id) {
			return out, mem.ErrDenied
		}
		if e := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM dbo.ai_memory_items WHERE id=@p1 AND user_id=@p2`, id, m.UserID).Scan(&found); e != nil {
			return out, e
		}
		if found != 1 {
			return out, mem.ErrDenied
		}
	}
	rows, e := tx.QueryContext(ctx, `SELECT CONVERT(varchar(36),id) FROM dbo.ai_memory_items WHERE user_id=@p1 AND deleted_at IS NULL AND (@p2=1 OR kind=@p3 OR EXISTS(SELECT 1 FROM OPENJSON(@p4) ids WHERE TRY_CONVERT(uniqueidentifier,ids.value)=id))`, m.UserID, s.All, s.Category, arrayJSON(s.ItemIDs))
	if e != nil {
		return out, e
	}
	ids := []c.ID{}
	for rows.Next() {
		var id c.ID
		if e = rows.Scan(&id); e != nil {
			rows.Close()
			return out, e
		}
		ids = append(ids, id)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return out, e
	}
	for _, id := range ids {
		if e = archiveMemory(ctx, tx, m.UserID, id); e != nil {
			return out, e
		}
		if _, e = tx.ExecContext(ctx, `UPDATE dbo.ai_memory_items SET deleted_at=SYSUTCDATETIME(),deleted_by=@p2,deletion_reason=N'user_request',version=version+1 WHERE id=@p1 AND user_id=@p2 AND deleted_at IS NULL`, id, m.UserID); e != nil {
			return out, e
		}
	}
	out.Deleted.DeletedItems = len(ids)
	// Evidence, embeddings, derived items, jobs and consent are deliberately retained.
	return out, nil
}
