package embedding

import (
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

// Invalidation cancels matching computations and fences publication/delivery.
// The bounded flight itself is its generation marker: no user tombstone map
// grows with account churn. Persisted vectors and source records are untouched.
func (s *Service) InvalidateEmbeddingUser(user c.ID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, e := range s.cache.entries {
		if k.user == user {
			s.cache.remove(e)
		}
	}
	for k, f := range s.flights {
		if k.user == user {
			f.invalidated = true
			f.cancel(ErrInvalidated)
		}
	}
}

// Close is idempotent, rejects new work, and joins every service-owned goroutine.
// Existing provider contracts must observe context cancellation.
func (s *Service) Close() {
	s.mu.Lock()
	s.cancel()
	for _, f := range s.flights {
		f.cancel(ErrClosed)
	}
	s.mu.Unlock()
	s.wg.Wait()
	s.mu.Lock()
	s.cache = newCache()
	s.mu.Unlock()
}
