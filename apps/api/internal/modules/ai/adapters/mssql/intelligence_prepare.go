package mssql

import (
	"context"
	"encoding/hex"
	"errors"
	bg "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/background"
	mem "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/retrieval"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
	"sort"
	"strings"
	"time"
)

func (r *IntelligenceRepository) Prepare(ctx context.Context, l mem.Lease, req bg.Request, retrieved []c.MemoryView) (out bg.Prepared, err error) {
	ctx, span := trace.Start(ctx, "background.Prepare", func() any {
		return map[string]any{"lease": l, "request": req, "retrieved": retrieved}
	})
	defer func() { span.Complete(func() any { return out }, err, recover()) }()
	if !req.Scope.Valid() || len(retrieved) > 16 || r.searchUse == nil {
		return out, mem.ErrDenied
	}
	bounded := []c.MemoryView{}
	for _, view := range retrieved {
		permitted, e := r.inScope(ctx, l.UserID, req.Scope, view.Source, view.ID)
		if e != nil {
			return out, e
		}
		if permitted {
			bounded = append(bounded, view)
		}
	}
	retrieved = bounded
	ids := append([]c.ID{}, req.MessageIDs...)
	targets := []c.ID{}
	if req.Origin == "periodic" {
		ids = nil
		for _, id := range req.MessageIDs {
			var pending int
			e := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM dbo.ai_memory_review_messages q JOIN dbo.messages m ON m.id=q.message_id WHERE q.user_id=@p1 AND q.message_id=@p2 AND q.job_id=@p3 AND (q.reviewed_at IS NULL OR q.source_version<>CONVERT(binary(8),m.search_version))`, l.UserID, id, l.ID).Scan(&pending)
			if e != nil {
				return out, e
			}
			if pending > 0 {
				ids = append(ids, id)
			}
		}
	}
	if req.Origin == "periodic" && len(ids) == 0 {
		snap, e := r.Snapshot(ctx, l.UserID, mem.Meta{UserID: l.UserID, TurnID: l.TurnID, OperationID: l.ID, OnlyTargets: true}, nil)
		if e != nil {
			return out, e
		}
		if snap.User.MemoryPaused {
			return out, bg.ErrPaused
		}
		out.Input = bg.Input{Schema: "background_input.v2", UserID: l.UserID, Request: req, Messages: []c.Message{}, Memories: []mem.Record{}, Records: []c.MemoryView{}, Threads: []bg.Thread{}}
		out.Snapshot = snap
		return out, nil
	}
	if req.Origin == "periodic" {
		req.MessageIDs = append([]c.ID{}, ids...)
	}
	// Recent dialogue complements explicit analysis, not every historical message.
	if req.Origin != "periodic" {
		rows, e := r.db.QueryContext(ctx, `SELECT TOP(12) CONVERT(varchar(36),m.id) FROM dbo.messages m JOIN dbo.conversations v ON v.id=m.conversation_id AND v.user_id=@p1 JOIN dbo.turns t ON t.id=m.turn_id AND t.user_id=@p1 WHERE m.deleted_at IS NULL AND v.deleted_at IS NULL AND t.status=N'completed' AND (@p2=N'user' OR EXISTS(SELECT 1 FROM dbo.ai_turn_events ev JOIN OPENJSON(@p3) scope ON TRY_CONVERT(uniqueidentifier,scope.value)=TRY_CONVERT(uniqueidentifier,JSON_VALUE(ev.delta_json,'$.thread.thread_id')) OR TRY_CONVERT(uniqueidentifier,scope.value)=TRY_CONVERT(uniqueidentifier,JSON_VALUE(ev.delta_json,'$.session.focus_thread')) WHERE ev.user_id=@p1 AND ev.turn_id=t.id)) ORDER BY t.admission_order DESC,m.message_order DESC`, l.UserID, req.Scope.Kind, arrayJSON(req.Scope.ThreadIDs))
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
	for _, v := range retrieved {
		if v.Source == "message" {
			ids = append(ids, v.ID)
		}
		if v.Source == "memory" {
			targets = append(targets, v.ID)
		}
	}
	// There is no fixed five-neighbor gate. Bound recent authorized candidates
	// independently, while always retaining requested/retrieved correction targets.
	rows, e := r.db.QueryContext(ctx, `SELECT TOP(16) CONVERT(varchar(36),id) FROM dbo.ai_memory_items WHERE user_id=@p1 AND deleted_at IS NULL AND status IN(N'active',N'proposed',N'supported',N'contested',N'stale') AND (@p2=N'user' OR EXISTS(SELECT 1 FROM OPENJSON(thread_ids_json) t JOIN OPENJSON(@p3) scope ON t.value=scope.value)) ORDER BY updated_at DESC,id`, l.UserID, req.Scope.Kind, arrayJSON(req.Scope.ThreadIDs))
	if e != nil {
		return out, e
	}
	for rows.Next() {
		var id c.ID
		if e = rows.Scan(&id); e != nil {
			rows.Close()
			return out, e
		}
		targets = append(targets, id)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return out, e
	}
	targets = uniqueBackgroundIDs(targets, 24)
	ids = uniqueBackgroundIDs(ids, 64)
	meta := mem.Meta{UserID: l.UserID, TurnID: l.TurnID, OperationID: l.ID, Bundle: c.BundleRef{}, Job: &l, Targets: targets, OnlyTargets: true, RequireCompleted: c.ValidID(l.TurnID)}
	snap, e := r.Snapshot(ctx, l.UserID, meta, ids)
	if e != nil {
		return out, e
	}
	if snap.User.MemoryPaused {
		return out, bg.ErrPaused
	}
	// Preserve the existing reflection.authorize risk boundary exactly.
	if req.Kind == "reflection" && (!snap.User.Risk.Level.Valid() || snap.User.Risk.Level.Rank() >= c.RiskElevated.Rank()) {
		return out, mem.ErrDenied
	}
	action := mem.Extract
	if req.Kind == "reflection" {
		action = mem.Reflect
	}
	allowed, e := r.policy.Allow(ctx, mem.Permission{Action: action, User: snap.User, TurnID: l.TurnID, Origin: c.ProvenanceSrc{Origin: map[bool]string{true: "reflection", false: "memory_writer"}[req.Kind == "reflection"]}})
	if e != nil || !allowed {
		return out, errors.Join(mem.ErrDenied, e)
	}
	repo := &RetrievalRepository{r.db, r.searchUse, r.cipher}
	out.Input = bg.Input{Schema: "background_input.v2", UserID: l.UserID, Request: req, Messages: []c.Message{}, Memories: []mem.Record{}, Records: []c.MemoryView{}, Threads: []bg.Thread{}, Cards: []c.KnowledgeCard{}, Knowledge: []c.KnowledgeResource{}, Retrieval: []c.RetrievalMeta{}}
	selectedSources := map[c.ID]mem.Source{}
	for _, id := range ids {
		if src, ok := snap.Sources[id]; ok {
			selectedSources[id] = src
		}
	}
	for _, record := range snap.Records {
		permit, e := r.policy.Allow(ctx, mem.Permission{Action: mem.UseSource, User: snap.User, SourceKind: "memory", SourceID: record.Item.ID, Memory: &record.Item})
		if e != nil {
			return out, e
		}
		if !permit {
			continue
		}
		if record.Item.Sensitivity == c.SensHighly {
			permit, e := r.policy.Allow(ctx, mem.Permission{Action: mem.Sensitive, User: snap.User, SourceKind: "memory", SourceID: record.Item.ID, Memory: &record.Item})
			if e != nil {
				return out, e
			}
			if !permit {
				continue
			}
		}
		if req.Scope.Kind == "threads" {
			scoped := false
			for _, id := range record.Item.ThreadIDs {
				for _, allowed := range req.Scope.ThreadIDs {
					if id == allowed {
						scoped = true
					}
				}
			}
			if !scoped {
				continue
			}
		}
		copy := bg.Clone(record)
		copy.Vector = nil
		copy.VectorModel = ""
		out.Input.Memories = append(out.Input.Memories, copy)
		current, found, e := repo.source(ctx, l.UserID, retrieval.SourceMemory, record.Item.ID)
		if e != nil {
			return out, e
		}
		if found {
			out.Sources = append(out.Sources, backgroundView(current))
		}
		// Expand primary provenance, retaining contradicting as well as supporting
		// user reports. Derived items never become independent raw evidence.
		for _, ev := range record.Item.Evidence {
			if ev.Kind == "message" && len(selectedSources) < 64 {
				if source, ok := snap.Sources[ev.Ref]; ok {
					selectedSources[ev.Ref] = source
				}
			}
		}
	}
	for id, src := range selectedSources {
		if src.Hidden || src.Future {
			continue
		}
		permit, e := r.policy.Allow(ctx, mem.Permission{Action: mem.UseSource, User: snap.User, TurnID: l.TurnID, SourceKind: "message", SourceID: id, Evidence: []mem.Source{src}})
		if e != nil {
			return out, e
		}
		if !permit {
			continue
		}
		candidate, found, e := repo.source(ctx, l.UserID, retrieval.SourceMessage, id)
		if e != nil {
			return out, e
		}
		if !found {
			continue
		}
		if c.ValidID(src.SessionID) {
			id := src.SessionID
			src.Message.SessionID = &id
		}
		out.Input.Messages = append(out.Input.Messages, src.Message)
		out.Sources = append(out.Sources, backgroundView(candidate))
	}
	sort.Slice(out.Input.Messages, func(i, j int) bool {
		a, b := out.Input.Messages[i], out.Input.Messages[j]
		if a.CreatedAt.Equal(b.CreatedAt) {
			return a.ID.String() < b.ID.String()
		}
		return a.CreatedAt.Before(b.CreatedAt)
	})
	for _, id := range ids {
		present := false
		for _, m := range out.Input.Messages {
			if m.ID == id {
				present = true
				break
			}
		}
		if !present {
			return out, mem.ErrDenied
		}
	}
	// Include relevant retrieved operational context with exact source revisions.
	seen := map[string]bool{}
	for _, view := range retrieved {
		if view.Source == "message" || view.Source == "memory" {
			continue
		}
		candidate, found, e := repo.source(ctx, l.UserID, retrieval.SourceKind(view.Source), view.ID)
		if e != nil {
			return out, e
		}
		if !found || candidate.SourceVersion != view.SourceVersion {
			return out, mem.ErrConflict
		}
		key := view.Source + view.ID.String()
		if !seen[key] {
			seen[key] = true
			out.Input.Records = append(out.Input.Records, backgroundView(candidate))
			out.Sources = append(out.Sources, backgroundView(candidate))
		}
	}
	rows, e = r.db.QueryContext(ctx, `SELECT TOP(4) CONVERT(varchar(36),id),version,COALESCE(formulation_version,0),topic FROM dbo.ai_threads WHERE user_id=@p1 AND (@p2=N'user' OR EXISTS(SELECT 1 FROM OPENJSON(@p3) scope WHERE TRY_CONVERT(uniqueidentifier,scope.value)=id)) ORDER BY last_active_at DESC,id`, l.UserID, req.Scope.Kind, arrayJSON(req.Scope.ThreadIDs))
	if e != nil {
		return out, e
	}
	for rows.Next() {
		var t bg.Thread
		if e = rows.Scan(&t.ID, &t.Version, &t.FormulationVersion, &t.Topic); e != nil {
			rows.Close()
			return out, e
		}
		out.Input.Threads = append(out.Input.Threads, t)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return out, e
	}
	if req.Scope.Kind == "threads" && len(out.Input.Threads) != len(req.Scope.ThreadIDs) {
		return out, mem.ErrDenied
	}
	// Existing exercise records are observational context, never consent or new
	// raw user evidence. They do not introduce another intervention component.
	rows, e = r.db.QueryContext(ctx, `SELECT TOP(2) CONVERT(varchar(36),id) FROM dbo.ai_protocol_instances WHERE user_id=@p1 AND (@p2=N'user' OR EXISTS(SELECT 1 FROM OPENJSON(@p3) scope WHERE TRY_CONVERT(uniqueidentifier,scope.value)=thread_id)) ORDER BY created_at DESC`, l.UserID, req.Scope.Kind, arrayJSON(req.Scope.ThreadIDs))
	if e != nil {
		return out, e
	}
	protocols := []c.ID{}
	for rows.Next() {
		var id c.ID
		if e = rows.Scan(&id); e != nil {
			rows.Close()
			return out, e
		}
		protocols = append(protocols, id)
	}
	rows.Close()
	for _, id := range protocols {
		candidate, found, e := repo.source(ctx, l.UserID, retrieval.SourceProtocol, id)
		if e != nil {
			return out, e
		}
		if found {
			view := backgroundView(candidate)
			out.Input.Records = append(out.Input.Records, view)
			out.Sources = append(out.Sources, view)
		}
	}
	// Keep only authorized model-visible sources in the commit snapshot.
	snap.Records = out.Input.Memories
	snap.Sources = map[c.ID]mem.Source{}
	for _, message := range out.Input.Messages {
		snap.Sources[message.ID] = selectedSources[message.ID]
	}
	out.Snapshot = snap
	return out, nil
}
func backgroundView(v retrieval.Candidate) c.MemoryView {
	text := v.Text
	if len([]rune(text)) > 1200 {
		text = string([]rune(text)[:1200]) + "…"
	}
	return c.MemoryView{ID: v.ID, Source: string(v.Source), SourceVersion: v.SourceVersion, Kind: v.Kind, Epistemic: v.Epistemic, Status: v.Status, Sensitivity: v.Sensitivity, Text: text, SourceCreatedAt: v.CreatedAt.UTC().Format(time.RFC3339Nano), SourceConversationID: v.ConversationID, EvidenceN: v.EvidenceN, Verbatim: v.Source == retrieval.SourceMessage}
}
func uniqueBackgroundIDs(ids []c.ID, maxItems int) []c.ID {
	out := []c.ID{}
	seen := map[c.ID]bool{}
	for _, id := range ids {
		if c.ValidID(id) && !seen[id] && len(out) < maxItems {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}
func (r *IntelligenceRepository) Verify(ctx context.Context, l mem.Lease, p bg.Prepared) (err error) {
	ctx, span := trace.Start(ctx, "background.Verify", func() any {
		return map[string]any{"lease": l, "prepared": p}
	})
	defer func() { span.Complete(func() any { return map[string]bool{"valid": err == nil} }, err, recover()) }()
	// Reopening the same bounded source set checks current ownership, visibility,
	// pause, use policy and complete primary content before disclosure/recovery.
	targets := []c.ID{}
	ids := []c.ID{}
	for _, v := range p.Input.Memories {
		targets = append(targets, v.Item.ID)
	}
	for _, v := range p.Input.Messages {
		ids = append(ids, v.ID)
	}
	fresh, e := r.Snapshot(ctx, l.UserID, mem.Meta{UserID: l.UserID, TurnID: l.TurnID, OperationID: l.ID, Targets: targets, OnlyTargets: true, RequireCompleted: c.ValidID(l.TurnID)}, ids)
	if e != nil {
		return e
	}
	if fresh.User.MemoryPaused {
		return bg.ErrPaused
	}
	if fresh.User.Version != p.Snapshot.User.Version || fresh.Revision != p.Snapshot.Revision {
		return mem.ErrConflict
	}
	if p.Input.Request.Kind == "reflection" && (!fresh.User.Risk.Level.Valid() || fresh.User.Risk.Level.Rank() >= c.RiskElevated.Rank()) {
		return mem.ErrDenied
	}
	action := mem.Extract
	if p.Input.Request.Kind == "reflection" {
		action = mem.Reflect
	}
	ok, e := r.policy.Allow(ctx, mem.Permission{Action: action, User: fresh.User, TurnID: l.TurnID})
	if e != nil || !ok {
		return errors.Join(mem.ErrDenied, e)
	}
	for _, message := range p.Input.Messages {
		actual, ok := fresh.Sources[message.ID]
		if !ok || actual.Hidden || actual.Future || actual.Message.Content != message.Content || actual.Message.Role != message.Role {
			return mem.ErrConflict
		}
		allow, e := r.policy.Allow(ctx, mem.Permission{Action: mem.UseSource, User: fresh.User, SourceKind: "message", SourceID: message.ID, Evidence: []mem.Source{actual}})
		if e != nil || !allow {
			return errors.Join(mem.ErrDenied, e)
		}
	}
	repo := &RetrievalRepository{r.db, r.searchUse, r.cipher}
	for _, source := range p.Sources {
		actual, found, e := repo.source(ctx, l.UserID, retrieval.SourceKind(source.Source), source.ID)
		if e != nil {
			return e
		}
		if !found || !strings.EqualFold(actual.SourceVersion, source.SourceVersion) {
			return mem.ErrConflict
		}
		if _, e = hex.DecodeString(source.SourceVersion); e != nil {
			return mem.ErrDenied
		}
	}
	for _, old := range p.Input.Memories {
		found := false
		for _, n := range fresh.Records {
			if n.Item.ID == old.Item.ID && n.Version == old.Version {
				found = true
				permit, e := r.policy.Allow(ctx, mem.Permission{Action: mem.UseSource, User: fresh.User, SourceKind: "memory", SourceID: n.Item.ID, Memory: &n.Item})
				if e != nil || !permit {
					return errors.Join(mem.ErrDenied, e)
				}
			}
		}
		if !found {
			return mem.ErrConflict
		}
	}
	for _, old := range p.Input.Threads {
		var version int64
		var fv int
		if e = r.db.QueryRowContext(ctx, `SELECT version,COALESCE(formulation_version,0) FROM dbo.ai_threads WHERE user_id=@p1 AND id=@p2`, l.UserID, old.ID).Scan(&version, &fv); e != nil {
			return e
		}
		if version != old.Version || fv != old.FormulationVersion {
			return mem.ErrConflict
		}
	}
	return nil
}

func (r *IntelligenceRepository) inScope(ctx context.Context, user c.ID, scope bg.Scope, kind string, id c.ID) (bool, error) {
	if scope.Kind == "user" {
		return true, nil
	}
	var count int
	query := `SELECT COUNT(*) FROM OPENJSON(@p3) scope WHERE TRY_CONVERT(uniqueidentifier,scope.value)=@p2`
	switch kind {
	case "message":
		query = `SELECT COUNT(*) FROM dbo.ai_turn_events e JOIN dbo.messages m ON m.turn_id=e.turn_id JOIN OPENJSON(@p3) scope ON TRY_CONVERT(uniqueidentifier,scope.value)=TRY_CONVERT(uniqueidentifier,JSON_VALUE(e.delta_json,'$.thread.thread_id')) OR TRY_CONVERT(uniqueidentifier,scope.value)=TRY_CONVERT(uniqueidentifier,JSON_VALUE(e.delta_json,'$.session.focus_thread')) WHERE e.user_id=@p1 AND m.id=@p2`
	case "memory":
		query = `SELECT COUNT(*) FROM dbo.ai_memory_items m CROSS APPLY OPENJSON(m.thread_ids_json) t JOIN OPENJSON(@p3) scope ON t.value=scope.value WHERE m.user_id=@p1 AND m.id=@p2`
	case "protocol":
		query = `SELECT COUNT(*) FROM dbo.ai_protocol_instances p JOIN OPENJSON(@p3) scope ON p.thread_id=TRY_CONVERT(uniqueidentifier,scope.value) WHERE p.user_id=@p1 AND p.id=@p2`
	case "formulation":
		query = `SELECT COUNT(*) FROM dbo.ai_formulations f JOIN OPENJSON(@p3) scope ON f.thread_id=TRY_CONVERT(uniqueidentifier,scope.value) WHERE f.user_id=@p1 AND f.id=@p2`
	case "session":
		query = `SELECT COUNT(*) FROM dbo.ai_sessions s JOIN OPENJSON(@p3) scope ON s.focus_thread_id=TRY_CONVERT(uniqueidentifier,scope.value) WHERE s.user_id=@p1 AND s.id=@p2`
	case "thread":
	default:
		return false, nil
	}
	e := r.db.QueryRowContext(ctx, query, user, id, arrayJSON(scope.ThreadIDs)).Scan(&count)
	return count > 0, e
}
