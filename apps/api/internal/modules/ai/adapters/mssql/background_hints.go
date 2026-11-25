package mssql

import (
	"context"
	"encoding/json"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/contextstate"
	mem "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"time"
)

type hintTurnKey struct{}

func (r *ContextReader) WithBackgroundHints(cipher mem.Cipher, timeout ...time.Duration) *ContextReader {
	r.hintsCipher = cipher
	if len(timeout) > 0 {
		r.hintsTimeout = timeout[0]
	}
	return r
}
func (r *ContextReader) backgroundHints(ctx context.Context, user c.ID, out *contextstate.Snapshot) error {
	// Background hints persisted in an earlier turn are refreshed from their
	// authoritative job/source record, never renewed from conversational carry.
	keep := []c.FutureHint{}
	for _, h := range out.FutureHints {
		if h.JobID == nil {
			keep = append(keep, h)
		}
	}
	out.FutureHints = keep
	if r.hintsCipher == nil || out.User.MemoryPaused {
		return nil
	}
	turn, _ := ctx.Value(hintTurnKey{}).(c.ID)
	rows, e := r.db.QueryContext(ctx, `SELECT TOP(4) CONVERT(varchar(36),h.id),h.hint_enc,h.sources_json FROM dbo.ai_background_hints h JOIN dbo.ai_jobs j ON j.id=h.job_id AND j.user_id=h.user_id AND j.status=N'completed' CROSS APPLY(SELECT COALESCE((SELECT m.created_at FROM dbo.turns t JOIN dbo.messages m ON m.id=t.user_message_id WHERE t.id=@p2 AND t.user_id=@p1),SYSUTCDATETIME()) as_of) point WHERE h.user_id=@p1 AND h.invalidated_at IS NULL AND h.expires_at>point.as_of AND h.created_at<point.as_of AND (h.thread_id IS NULL OR h.thread_id=@p3) AND (SELECT COUNT(*) FROM dbo.turns t WHERE t.user_id=@p1 AND t.status=N'completed' AND t.completed_at>h.created_at AND t.completed_at<point.as_of)<h.ttl_turns AND NOT EXISTS(SELECT 1 FROM OPENJSON(h.sources_json) refs WHERE NOT EXISTS(SELECT 1 FROM dbo.ai_search_sources s WHERE s.user_id=h.user_id AND s.source_kind=JSON_VALUE(refs.value,'$.kind') AND s.id=TRY_CONVERT(uniqueidentifier,JSON_VALUE(refs.value,'$.id')) AND s.source_version=TRY_CONVERT(binary(8),JSON_VALUE(refs.value,'$.version'),2))) ORDER BY h.created_at DESC,h.id`, user, memoryNullableID(turn), out.Session.FocusThreadID)
	if e != nil {
		return e
	}
	type body struct {
		id     c.ID
		sealed []byte
		refs   string
	}
	all := []body{}
	for rows.Next() {
		var b body
		if e = rows.Scan(&b.id, &b.sealed, &b.refs); e != nil {
			rows.Close()
			return e
		}
		all = append(all, b)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	for _, b := range all {
		raw, e := r.hintsCipher.Open(ctx, user, "background-hint:"+b.id.String(), b.sealed)
		if e != nil {
			return e
		}
		var hint c.FutureHint
		if json.Unmarshal(raw, &hint) != nil || json.Unmarshal([]byte(b.refs), &hint.Sources) != nil {
			return mem.ErrDenied
		}
		out.FutureHints = append(out.FutureHints, hint)
	}
	return nil
}
