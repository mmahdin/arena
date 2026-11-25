package memory

import (
	"context"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

// Items is a bounded explicit owner view for WP-16 pagination, independent of
// AI-use consent. SQL/provenance/decryption stay behind the existing repository.
func (s *Service) Items(ctx context.Context, user c.ID, ids []c.ID) ([]c.MemoryItem, error) {
	m, e := Auth(ctx, user)
	if e != nil || len(ids) > 100 {
		return nil, ErrDenied
	}
	m.Targets = append([]c.ID{}, ids...)
	out := []c.MemoryItem{}
	if len(ids) == 0 {
		return out, nil
	}
	snap, e := s.repo.Snapshot(ctx, user, m, nil)
	if e != nil {
		return nil, e
	}
	found := map[c.ID]c.MemoryItem{}
	for _, r := range snap.Records {
		if r.DeletedAt == nil && r.Item.UserID == user {
			found[r.Item.ID] = r.Item
		}
	}
	for _, id := range ids {
		if v, ok := found[id]; ok {
			out = append(out, v)
		}
	}
	return out, nil
}
