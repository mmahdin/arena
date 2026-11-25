package mssql

import (
	"context"
	mem "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
)

// Reuse the same source/version/privacy boundary before either background
// model sees personal history. It works for completed, owner-scoped triggers.
func (r *MemoryRepository) VerifyResearch(ctx context.Context, user, trigger c.ID, in ports.BackgroundResearch) (err error) {
	ctx, span := trace.Start(ctx, "background.VerifySources", func() any { return in })
	defer func() { span.Complete(nil, err, recover()) }()
	if r.searchUse == nil || len(in.Items) > 6 {
		return mem.ErrDenied
	}
	v, err := NewSourceVerifier(r.db, r.searchUse, r.cipher)
	if err != nil {
		return err
	}
	for _, item := range in.Items {
		if err = v.verifyRetrieved(ctx, user, trigger, item); err != nil {
			return err
		}
	}
	return nil
}
