package mssql

import (
	"context"
	"database/sql"
	"errors"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
	"math"
	"reflect"
	"sort"
	"strings"

	mem "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

func (r *MemoryRepository) permit(ctx context.Context, s mem.Snapshot, m mem.Meta, action mem.Action, kind string, id c.ID) bool {
	permission := mem.Permission{Action: action, User: s.User, TurnID: m.TurnID, SourceKind: kind, SourceID: id, Origin: c.ProvenanceSrc{Origin: "memory_writer"}}
	if source, ok := s.Sources[id]; ok {
		permission.Evidence = []mem.Source{source}
	}
	for _, record := range s.Records {
		if record.Item.ID == id {
			copy := record.Item
			permission.Memory = &copy
			break
		}
	}
	ok, e := r.policy.Allow(ctx, permission)
	return e == nil && ok
}

func (r *MemoryRepository) WriterInput(ctx context.Context, m mem.Meta, task mem.Task) (out ports.MemWriterInput, err error) {
	ctx, inspectionSpan := trace.Start(ctx, "adapters/mssql/memory_writer.WriterInput", func() any { return map[string]any{"m": m, "task": task} })
	defer func() { inspectionSpan.Complete(func() any { return map[string]any{"out": out} }, err, recover()) }()

	defer func() { err = memorySQLError(err) }()
	ids := task.SourceIDs
	if !task.Consolidate {
		rows, e := r.db.QueryContext(ctx, `SELECT CONVERT(varchar(36),m.id) FROM dbo.messages m JOIN dbo.conversations cv ON cv.id=m.conversation_id AND cv.user_id=@p1 JOIN dbo.turns t ON t.id=m.turn_id AND t.user_id=cv.user_id WHERE t.id=@p2 AND t.status=N'completed' ORDER BY m.created_at,m.id`, m.UserID, m.TurnID)
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
	if len(ids) == 0 || len(ids) > 12 {
		return out, mem.ErrDenied
	}
	m.RequireCompleted = true
	s, e := r.Snapshot(ctx, m.UserID, m, ids)
	if e != nil {
		return out, e
	}
	if !r.permit(ctx, s, m, mem.Extract, "turn", m.TurnID) {
		return out, mem.ErrDenied
	}
	out = ports.MemWriterInput{UserID: m.UserID, Analysis: task.Analysis, Bundle: task.Bundle}
	for _, id := range ids {
		v, ok := s.Sources[id]
		if !ok || v.Future || v.Hidden && !r.permit(ctx, s, m, mem.UseRetainedSource, "message", id) || !r.permit(ctx, s, m, mem.UseSource, "message", id) {
			return ports.MemWriterInput{}, mem.ErrDenied
		}
		out.Messages = append(out.Messages, v.Message)
	}
	sort.SliceStable(out.Messages, func(i, j int) bool {
		if out.Messages[i].CreatedAt.Equal(out.Messages[j].CreatedAt) {
			return out.Messages[i].Role == c.RoleUser && out.Messages[j].Role != c.RoleUser
		}
		return out.Messages[i].CreatedAt.Before(out.Messages[j].CreatedAt)
	})
	// Reuse WP-09's cached owner-scoped message embedding when available;
	// deterministic lexical overlap is the no-provider fallback. Selection never
	// exposes unavailable or highly sensitive records to extraction.
	var queryVector []float32
	for _, message := range out.Messages {
		if message.Role != c.RoleUser {
			continue
		}
		var raw []byte
		var codec, dimension int
		e := r.db.QueryRowContext(ctx, `SELECT codec_version,dim,vector_bytes FROM dbo.ai_embeddings WHERE user_id=@p1 AND ref_kind=N'message' AND ref_id=@p2 AND model_id=@p3`, m.UserID, message.ID, r.model).Scan(&codec, &dimension, &raw)
		if e != nil && !errors.Is(e, sql.ErrNoRows) {
			return out, e
		}
		if e == nil {
			queryVector, e = DecodeEmbedding(codec, r.model, r.model, dimension, r.dimension, raw)
			if e != nil {
				return out, e
			}
			break
		}
	}
	text := ""
	for _, v := range out.Messages {
		if v.Role == c.RoleUser {
			text += " " + v.Content
		}
	}
	scores := map[c.ID]float64{}
	for _, record := range s.Records {
		scores[record.Item.ID] = memorySimilarity(queryVector, record.Vector, text, record.Item.Content)
	}
	sort.Slice(s.Records, func(i, j int) bool {
		if scores[s.Records[i].Item.ID] != scores[s.Records[j].Item.ID] {
			return scores[s.Records[i].Item.ID] > scores[s.Records[j].Item.ID]
		}
		if s.Records[i].Item.Importance == s.Records[j].Item.Importance {
			return s.Records[i].Item.ID.String() < s.Records[j].Item.ID.String()
		}
		return s.Records[i].Item.Importance > s.Records[j].Item.Importance
	})
	for _, v := range s.Records {
		if len(out.Neighbors) == 5 {
			break
		}
		if v.Item.Sensitivity == c.SensHighly || v.Item.Status == c.MemContested || !r.permit(ctx, s, m, mem.UseSource, "memory", v.Item.ID) {
			continue
		}
		out.Neighbors = append(out.Neighbors, v.Item)
	}
	for _, claim := range task.Claims {
		if claim.Source != nil {
			found := false
			for _, v := range out.Neighbors {
				if v.ID == *claim.Source {
					found = true
				}
			}
			if !found {
				continue
			}
		}
		out.Claims = append(out.Claims, claim)
	}
	return out, nil
}

func memorySimilarity(query, vector []float32, text, content string) float64 {
	if len(query) > 0 && len(query) == len(vector) {
		dot, xx, yy := 0., 0., 0.
		for i, x := range query {
			y := vector[i]
			dot += float64(x) * float64(y)
			xx += float64(x) * float64(x)
			yy += float64(y) * float64(y)
		}
		if xx > 0 && yy > 0 {
			return dot / math.Sqrt(xx*yy)
		}
	}
	words := map[string]bool{}
	for _, w := range strings.Fields(mem.Canonical(text)) {
		words[w] = true
	}
	score := 0.
	for _, w := range strings.Fields(mem.Canonical(content)) {
		if words[w] {
			score++
		}
	}
	return score / (1 + float64(len(strings.Fields(content))))
}

func (r *MemoryRepository) VerifyWriter(ctx context.Context, m mem.Meta, in ports.MemWriterInput) error {
	if in.UserID != m.UserID || !reflect.DeepEqual(in.Bundle, m.Bundle) {
		return mem.ErrDenied
	}
	ids := []c.ID{}
	for _, v := range in.Messages {
		ids = append(ids, v.ID)
	}
	for _, v := range in.Neighbors {
		m.Targets = append(m.Targets, v.ID)
	}
	m.RequireCompleted = true
	s, e := r.Snapshot(ctx, m.UserID, m, ids)
	if e != nil {
		return e
	}
	if !r.permit(ctx, s, m, mem.Extract, "turn", m.TurnID) {
		return mem.ErrDenied
	}
	for _, v := range in.Messages {
		actual, ok := s.Sources[v.ID]
		if !ok || actual.Future || actual.Hidden && !r.permit(ctx, s, m, mem.UseRetainedSource, "message", v.ID) || actual.Message.Content != v.Content || actual.Message.Role != v.Role || v.UserID != m.UserID || !r.permit(ctx, s, m, mem.UseSource, "message", v.ID) {
			return mem.ErrDenied
		}
	}
	if in.Research != nil {
		if len(in.Messages) == 0 {
			return mem.ErrDenied
		}
		if e := r.VerifyResearch(ctx, m.UserID, in.Messages[0].ID, *in.Research); e != nil {
			return e
		}
	}
	for _, v := range in.Neighbors {
		found := false
		for _, record := range s.Records {
			if record.Item.ID == v.ID && reflect.DeepEqual(record.Item, v) && record.Item.Status != c.MemContested && record.Item.Sensitivity != c.SensHighly && r.permit(ctx, s, m, mem.UseSource, "memory", v.ID) {
				found = true
			}
		}
		if !found {
			return mem.ErrDenied
		}
	}
	return nil
}
