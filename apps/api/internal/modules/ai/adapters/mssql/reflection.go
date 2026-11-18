package mssql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"sort"
	"sync"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/contextstate"
	mem "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	rf "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/reflection"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/state"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

// ReflectionRepository extends the one memory writer transaction; it does not
// introduce a second memory store or an independent state writer.
type ReflectionRepository struct {
	*MemoryRepository
	policy      contextstate.Policy
	sweepMu     sync.Mutex
	sweepCursor c.ID
}

var _ rf.Repository = (*ReflectionRepository)(nil)

func NewReflectionRepository(memory *MemoryRepository, policy contextstate.Policy) (*ReflectionRepository, error) {
	if memory == nil || policy.Thresholds.Thread.MaxActive < 1 {
		return nil, mem.ErrAssets
	}
	return &ReflectionRepository{MemoryRepository: memory, policy: policy}, nil
}
func (r *ReflectionRepository) ClaimReflection(ctx context.Context, id string, ttl time.Duration, b c.BundleRef, quota int) (*mem.Lease, error) {
	return r.claimJobs(ctx, id, ttl, b, quota, true)
}
func (r *ReflectionRepository) Load(ctx context.Context, t rf.Task) (out rf.Snapshot, err error) {
	defer func() { err = memorySQLError(err) }()
	if !c.ValidID(t.UserID) || !c.ValidID(t.ThreadID) || !c.ValidID(t.SessionID) {
		return out, mem.ErrDenied
	}
	// Read candidate IDs and state under the SAME user lock used by finalization.
	tx, e := r.begin(ctx, t.UserID)
	if e != nil {
		return out, e
	}
	defer tx.Rollback()
	if e = loadUser(ctx, tx, t.UserID, &out.User); e != nil {
		return out, e
	}
	if e = tx.QueryRowContext(ctx, `SELECT memory_revision FROM dbo.ai_user_state WHERE user_id=@p1`, t.UserID).Scan(&out.Revision); e != nil {
		return out, e
	}
	var threads []c.ThreadRecord
	if e = loadThreads(ctx, tx, t.UserID, &threads); e != nil {
		return out, e
	}
	for _, th := range threads {
		if th.ID == t.ThreadID {
			out.Thread = th
		}
	}
	if out.Thread.ID != t.ThreadID {
		return out, mem.ErrDenied
	}
	var focus sql.NullString
	var closed sql.NullTime
	e = tx.QueryRowContext(ctx, `SELECT CONVERT(varchar(36),s.id),CONVERT(varchar(36),s.conversation_id),s.started_at,s.last_activity_at,s.closed_at,CONVERT(varchar(36),s.focus_thread_id),s.turn_count,s.version FROM dbo.ai_sessions s WHERE s.id=@p1 AND s.user_id=@p2`, t.SessionID, t.UserID).Scan(&out.Session.ID, &out.Session.ConversationID, &out.Session.StartedAt, &out.Session.LastActivityAt, &closed, &focus, &out.Session.TurnCount, &out.Session.Version)
	if e != nil {
		return out, mem.ErrDenied
	}
	out.Session.UserID = t.UserID
	if closed.Valid {
		out.Session.ClosedAt = &closed.Time
	}
	out.Session.FocusThreadID, e = nullableID(focus)
	if e != nil {
		return out, e
	}
	var eligible int
	e = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM dbo.ai_threads WHERE id=@p1 AND user_id=@p2`, t.ThreadID, t.UserID).Scan(&eligible)
	if e != nil || eligible != 1 {
		return out, mem.ErrDenied
	}
	ids := []c.ID{}
	for _, slot := range out.Thread.Slots {
		ids = append(ids, slot.Evidence...)
	}
	// Session transcripts are used for episode consolidation only, when the
	// closed session's focus is this thread. No cross-user or arbitrary history.
	if out.Session.FocusThreadID != nil && *out.Session.FocusThreadID == t.ThreadID {
		rows, e := tx.QueryContext(ctx, `SELECT TOP(12) CONVERT(varchar(36),m.id) FROM dbo.messages m JOIN dbo.ai_turn_context tc ON tc.turn_id=m.turn_id AND tc.user_id=@p1 JOIN dbo.turns tr ON tr.id=m.turn_id AND tr.user_id=tc.user_id WHERE tc.session_id=@p2 AND tr.status=N'completed' AND tc.state_committed=1 ORDER BY m.created_at DESC,m.id DESC`, t.UserID, t.SessionID)
		if e != nil {
			return out, e
		}
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
	}
	var fields, paragraph, framework string
	e = tx.QueryRowContext(ctx, `SELECT fields_json,paragraph,framework FROM dbo.ai_formulations WHERE user_id=@p1 AND thread_id=@p2 AND version=@p3 AND deleted_at IS NULL`, t.UserID, t.ThreadID, out.Thread.FormulationVersion).Scan(&fields, &paragraph, &framework)
	if e == nil {
		out.Previous = &c.FormulationDoc{Framework: framework, Paragraph: paragraph}
		if json.Unmarshal([]byte(fields), &out.Previous.Fields) != nil {
			return out, mem.ErrDenied
		}
	} else if !errors.Is(e, sql.ErrNoRows) {
		return out, e
	}
	if e = tx.Commit(); e != nil {
		return out, e
	}
	snapshot, e := r.MemoryRepository.Snapshot(ctx, t.UserID, mem.Meta{UserID: t.UserID, OperationID: t.ThreadID, ThreadID: &t.ThreadID, TurnID: t.TurnID, RequireCompleted: c.ValidID(t.TurnID)}, ids)
	if e != nil {
		return out, e
	}
	if snapshot.User.Version != out.User.Version || snapshot.Revision != out.Revision {
		return out, mem.ErrConflict
	}
	permit := func(a mem.Action, kind string, id c.ID, item *c.MemoryItem) bool {
		ok, e := r.MemoryRepository.policy.Allow(ctx, mem.Permission{Action: a, User: out.User, SourceKind: kind, SourceID: id, Memory: item, Origin: c.ProvenanceSrc{Origin: "reflection"}})
		return e == nil && ok
	}
	if out.User.MemoryPaused || !out.User.Risk.Level.Valid() || out.User.Risk.Level.Rank() >= c.RiskElevated.Rank() || !permit(mem.Reflect, "thread", t.ThreadID, nil) {
		return out, mem.ErrDenied
	}
	if out.Previous != nil && !permit(mem.UseSource, "formulation", t.ThreadID, nil) {
		out.Previous = nil
	}
	// Choose a deterministic bounded hot set; preserve counters/provenance.
	for _, record := range snapshot.Records {
		it := record.Item
		if !slices.Contains(it.ThreadIDs, t.ThreadID) || it.Sensitivity == c.SensHighly || it.Status == c.MemRejected || it.Status == c.MemSuperseded || !permit(mem.UseSource, "memory", it.ID, &it) {
			continue
		}
		out.Items = append(out.Items, it)
	}
	sort.Slice(out.Items, func(i, j int) bool {
		a, b := out.Items[i], out.Items[j]
		if !a.UpdatedAt.Equal(b.UpdatedAt) {
			return a.UpdatedAt.After(b.UpdatedAt)
		}
		return a.ID.String() < b.ID.String()
	})
	if len(out.Items) > 24 {
		out.Items = out.Items[:24]
	}
	selected := map[c.ID]bool{}
	for _, id := range ids {
		selected[id] = true
	}
	for _, it := range out.Items {
		for _, ev := range it.Evidence {
			if ev.Kind == "message" {
				selected[ev.Ref] = true
			}
		}
	}
	bySession := map[c.ID][]mem.Source{}
	for id := range selected {
		src, ok := snapshot.Sources[id]
		if !ok || src.Future || !c.ValidID(src.SessionID) || src.Message.Role != c.RoleUser && src.SessionID != t.SessionID {
			continue
		}
		if src.Hidden && !permit(mem.UseRetainedSource, "message", id, nil) || !permit(mem.UseSource, "message", id, nil) {
			continue
		}
		bySession[src.SessionID] = append(bySession[src.SessionID], src)
	}
	keys := []c.ID{}
	for k := range bySession {
		keys = append(keys, k)
		sort.Slice(bySession[k], func(i, j int) bool {
			a, b := bySession[k][i].Message, bySession[k][j].Message
			if a.CreatedAt.Equal(b.CreatedAt) {
				return a.ID.String() < b.ID.String()
			}
			return a.CreatedAt.Before(b.CreatedAt)
		})
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i].String() < keys[j].String() })
	// Round-robin across sessions preserves longitudinal diversity under the cap.
	for n := 0; len(out.Sources) < 48; n++ {
		added := false
		for _, k := range keys {
			if n < len(bySession[k]) && len(out.Sources) < 48 {
				out.Sources = append(out.Sources, bySession[k][n])
				added = true
			}
		}
		if !added {
			break
		}
	}
	sort.Slice(out.Sources, func(i, j int) bool {
		a, b := out.Sources[i].Message, out.Sources[j].Message
		if a.CreatedAt.Equal(b.CreatedAt) {
			return a.ID.String() < b.ID.String()
		}
		return a.CreatedAt.Before(b.CreatedAt)
	})
	included := map[c.ID]bool{}
	for _, src := range out.Sources {
		included[src.Message.ID] = true
		if src.Message.Role == c.RoleUser {
			// count below uses accepted state-event membership
		}
	}
	for k, v := range out.Thread.Slots {
		for _, id := range v.Evidence {
			if !included[id] {
				out.Thread.Slots[k] = c.SlotView{Slot: k, Status: c.EpUnknown, Value: "", Evidence: []c.ID{}}
				break
			}
		}
	}
	// Retained derived items have their OWN use authorization above. Withholding
	// raw sources must not silently revoke use of independently eligible memory.
	// New formulation fields still need the separately authorized evidence set.
	if e = r.db.QueryRowContext(ctx, `SELECT COUNT(DISTINCT e.turn_id) FROM dbo.ai_turn_events e JOIN dbo.turns tr ON tr.id=e.turn_id AND tr.user_id=e.user_id WHERE e.user_id=@p1 AND e.type=N'state_applied' AND tr.status=N'completed' AND (TRY_CONVERT(uniqueidentifier,JSON_VALUE(e.delta_json,'$.thread.thread_id'))=@p2 OR TRY_CONVERT(uniqueidentifier,JSON_VALUE(e.delta_json,'$.session.focus_thread'))=@p2)`, t.UserID, t.ThreadID).Scan(&out.ThreadTurns); e != nil {
		return out, e
	}
	out.Cooldowns = snapshot.Cooldowns
	var since sql.NullTime
	if e = r.db.QueryRowContext(ctx, `SELECT MAX(created_at) FROM dbo.ai_reflection_runs WHERE user_id=@p1 AND thread_id=@p2`, t.UserID, t.ThreadID).Scan(&since); e != nil {
		return out, e
	}
	for _, it := range out.Items {
		if !since.Valid || it.CreatedAt.After(since.Time) {
			out.NewItems++
		}
		if !since.Valid || it.UpdatedAt.After(since.Time) {
			out.Contradictions += it.ContradictN
		}
	}
	return out, nil
}
func (r *ReflectionRepository) Verify(ctx context.Context, t rf.Task, s rf.Snapshot) error {
	fresh, e := r.Load(ctx, t)
	if e != nil {
		return e
	}
	if !reflect.DeepEqual(fresh, s) {
		return mem.ErrConflict
	}
	return nil
}
func (r *ReflectionRepository) EnqueueReflection(ctx context.Context, t rf.Task) (err error) {
	defer func() { err = memorySQLError(err) }()
	tx, e := r.begin(ctx, t.UserID)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	key := t.ThreadID.String() + ":" + t.SessionID.String() + ":" + t.Trigger
	if t.Trigger != "session_closed" {
		var version int64
		if e = tx.QueryRowContext(ctx, `SELECT version FROM dbo.ai_threads WHERE id=@p1 AND user_id=@p2`, t.ThreadID, t.UserID).Scan(&version); e != nil {
			return e
		}
		key += ":" + fmt.Sprint(version)
		if t.Trigger == "nightly" {
			key += ":" + time.Now().UTC().Format("2006-01-02")
		}
	}
	_, e = tx.ExecContext(ctx, `IF NOT EXISTS(SELECT 1 FROM dbo.ai_jobs WHERE user_id=@p1 AND kind=N'reflection' AND dedupe_key=@p2) INSERT dbo.ai_jobs(id,user_id,kind,dedupe_key,payload_json) VALUES(@p3,@p1,N'reflection',@p2,@p4)`, t.UserID, key, c.NewID(), memoryJSON(t))
	if e != nil {
		return e
	}
	return tx.Commit()
}

type reflectionCommitKey struct{}

func (r *ReflectionRepository) WithCommit(ctx context.Context, v rf.Commit) context.Context {
	return context.WithValue(ctx, reflectionCommitKey{}, v)
}
func (r *ReflectionRepository) Save(ctx context.Context, m mem.Meta, s mem.Snapshot, p mem.Plan, hash string) error {
	v, ok := ctx.Value(reflectionCommitKey{}).(rf.Commit)
	if !ok {
		return r.MemoryRepository.Save(ctx, m, s, p, hash)
	}
	if m.Job == nil || m.Job.Kind != "reflection" || v.Task.UserID != m.UserID || v.Snapshot.User.Version != s.User.Version || v.Snapshot.Revision != s.Revision {
		return mem.ErrConflict
	}
	encrypted, e := r.cipher.Seal(ctx, m.UserID, "reflection-run:"+m.Job.ID.String(), []byte(memoryJSON(v)))
	if e != nil {
		return mem.ErrDenied
	}
	return r.save(ctx, m, s, p, hash, func(tx *sql.Tx) error { return r.commitReflection(ctx, tx, m, v, encrypted) })
}
func (r *ReflectionRepository) commitReflection(ctx context.Context, tx *sql.Tx, m mem.Meta, v rf.Commit, encrypted []byte) error {
	var version int64
	var formVersion int
	var sessionVersion int64
	if e := tx.QueryRowContext(ctx, `SELECT version,formulation_version FROM dbo.ai_threads WHERE id=@p1 AND user_id=@p2`, v.Task.ThreadID, m.UserID).Scan(&version, &formVersion); e != nil {
		return e
	}
	if version != v.Snapshot.Thread.Version || formVersion != v.Snapshot.Thread.FormulationVersion {
		return mem.ErrConflict
	}
	if e := tx.QueryRowContext(ctx, `SELECT version FROM dbo.ai_sessions WHERE id=@p1 AND user_id=@p2`, v.Task.SessionID, m.UserID).Scan(&sessionVersion); e != nil {
		return e
	}
	if sessionVersion != v.Snapshot.Session.Version {
		return mem.ErrConflict
	}
	// Verify source visibility/content under the owner lock after provider work.
	ids := []c.ID{}
	for _, src := range v.Snapshot.Sources {
		ids = append(ids, src.Message.ID)
	}
	fresh, e := memorySources(ctx, tx, m.UserID, c.ID{}, ids)
	if e != nil {
		return e
	}
	for _, src := range v.Snapshot.Sources {
		n, ok := fresh[src.Message.ID]
		if !ok || n.Hidden != src.Hidden || n.Message.Content != src.Message.Content {
			return mem.ErrConflict
		}
	}
	if form := v.Result.Formulation; form != nil {
		formVersion++
		_, e = tx.ExecContext(ctx, `INSERT dbo.ai_formulations(id,user_id,thread_id,version,framework,fields_json,paragraph,gaps_json) VALUES(@p1,@p2,@p3,@p4,@p5,@p6,@p7,@p8)`, c.NewID(), m.UserID, v.Task.ThreadID, formVersion, form.Framework, memoryJSON(form.Fields), form.Paragraph, arrayJSON(v.Result.InformationGaps))
		if e != nil {
			return e
		}
	}
	var threads []c.ThreadRecord
	if e = loadThreads(ctx, tx, m.UserID, &threads); e != nil {
		return e
	}
	snap := contextstate.Snapshot{User: v.Snapshot.User, Session: c.SessionRecord{UserID: m.UserID}, Threads: threads} // protect ANY current open session focus
	var focus sql.NullString
	e = tx.QueryRowContext(ctx, `SELECT TOP(1) CONVERT(varchar(36),focus_thread_id) FROM dbo.ai_sessions WHERE user_id=@p1 AND closed_at IS NULL ORDER BY last_activity_at DESC`, m.UserID).Scan(&focus)
	if e != nil && !errors.Is(e, sql.ErrNoRows) {
		return e
	}
	if focus.Valid {
		snap.Session.FocusThreadID, _ = nullableID(focus)
	}
	protected := map[c.ID]bool{}
	rows, e := tx.QueryContext(ctx, `SELECT CONVERT(varchar(36),focus_thread_id) FROM dbo.ai_sessions WHERE user_id=@p1 AND closed_at IS NULL AND focus_thread_id IS NOT NULL`, m.UserID)
	if e != nil {
		return e
	}
	for rows.Next() {
		var id c.ID
		if e = rows.Scan(&id); e != nil {
			rows.Close()
			return e
		}
		protected[id] = true
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	proposals := []c.ThreadProposal{}
	for _, proposal := range v.Result.ThreadProposals {
		if protected[proposal.ThreadID] {
			v.Rejected = append(v.Rejected, c.DeltaRejection{Path: "thread_proposals", RuleCode: "ST-006"})
			continue
		}
		proposals = append(proposals, proposal)
	}
	delta, rejected, e := state.ValidateReflection(ctx, snap, r.policy, proposals)
	if e != nil {
		return e
	}
	v.Rejected = append(v.Rejected, rejected...)
	if e = applyThread(ctx, tx, c.Admission{UserID: m.UserID}, delta); e != nil {
		return e
	}
	hypotheses := append([]c.ID{}, v.Snapshot.Thread.HypothesisIDs...)
	linked, e := tx.QueryContext(ctx, `SELECT CONVERT(varchar(36),id) FROM dbo.ai_memory_items WHERE user_id=@p1 AND kind=N'hypothesis' AND EXISTS(SELECT 1 FROM OPENJSON(thread_ids_json) ids WHERE TRY_CONVERT(uniqueidentifier,ids.value)=@p2) ORDER BY id`, m.UserID, v.Task.ThreadID)
	if e != nil {
		return e
	}
	for linked.Next() {
		var id c.ID
		if e = linked.Scan(&id); e != nil {
			linked.Close()
			return e
		}
		if !slices.Contains(hypotheses, id) {
			hypotheses = append(hypotheses, id)
		}
	}
	e = linked.Err()
	linked.Close()
	if e != nil {
		return e
	}
	if _, e = tx.ExecContext(ctx, `UPDATE dbo.ai_threads SET hypothesis_ids_json=@p3 WHERE id=@p1 AND user_id=@p2`, v.Task.ThreadID, m.UserID, arrayJSON(hypotheses)); e != nil {
		return e
	}
	_, e = tx.ExecContext(ctx, `UPDATE dbo.ai_threads SET formulation_version=@p3,version=version+1 WHERE id=@p1 AND user_id=@p2; INSERT dbo.ai_reflection_runs(id,user_id,thread_id,session_id,job_id,trigger_kind,thread_version,formulation_version,result_enc,provenance_json) VALUES(@p4,@p2,@p1,@p5,@p4,@p6,@p7,@p3,@p8,@p9); INSERT dbo.ai_turn_events(id,user_id,type,actor,delta_json) VALUES(@p10,@p2,N'reflection_committed',N'reflection',@p11)`, v.Task.ThreadID, m.UserID, formVersion, m.Job.ID, v.Task.SessionID, v.Task.Trigger, version, encrypted, memoryJSON(struct {
		Bundle c.BundleRef
		Model  string
	}{v.Task.Bundle, v.Usage.Model}), c.NewID(), memoryJSON(struct {
		Thread   c.ID
		Version  int
		Delta    c.StateDelta
		Rejected []c.DeltaRejection
	}{v.Task.ThreadID, formVersion, delta, v.Rejected}))
	return e
}

func (r *ReflectionRepository) Checkpoint(ctx context.Context, l mem.Lease, raw []byte) error {
	if l.Kind != "reflection" {
		return r.MemoryRepository.Checkpoint(ctx, l, raw)
	}
	return r.checkpoint(ctx, l, raw, true)
}
