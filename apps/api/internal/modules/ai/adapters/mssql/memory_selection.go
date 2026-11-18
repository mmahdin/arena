package mssql

import (
	"context"
	mem "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

func (r *MemoryRepository) Selection(ctx context.Context, user, turn, message c.ID) (out mem.Snapshot, err error) {
	defer func() { err = memorySQLError(err) }()
	tx, e := r.begin(ctx, user)
	if e != nil {
		return out, e
	}
	defer tx.Rollback()
	if e = loadUser(ctx, tx, user, &out.User); e != nil {
		return out, mem.ErrDenied
	}
	out.Sources, e = memorySources(ctx, tx, user, turn, []c.ID{message})
	if e != nil {
		return out, e
	}
	source, ok := out.Sources[message]
	if !ok || source.Future || source.Message.TurnID != turn || source.Message.Role != c.RoleUser {
		return out, mem.ErrDenied
	}
	return out, tx.Commit()
}
