package retrieval

import (
	"context"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	domain "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/domain/retrieval"
)

// runStrategy executes one strategy and appends its candidates to the builder.
// Any error fails the whole call (degraded), never a silent partial result.
func (s *Service) runStrategy(
	ctx context.Context,
	strategy string,
	req contract.RetrievalRequest,
	window *domain.Window,
	plan domain.StrategyPlan,
	queryVector *contract.Vector,
	b *candidateBuilder,
) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, s.tunables.PerStrategyTimeout)
	defer cancel()

	switch strategy {
	case domain.StrategyTemporal:
		return s.runTemporal(timeoutCtx, req, window, b)
	case domain.StrategyThread:
		return s.runThread(timeoutCtx, req, b)
	case domain.StrategyEntity:
		return s.runEntity(timeoutCtx, req, b)
	case domain.StrategyAnaphora:
		return s.runAnaphora(timeoutCtx, req, b)
	case domain.StrategyHybrid:
		return s.runHybrid(timeoutCtx, req, queryVector, plan, b)
	default:
		return ErrInvalidStrategy
	}
}

// runTemporal is exact recall: owner-scoped raw messages in a resolved window.
// It returns verbatim messages, never summaries (§3.4, Scenario 2).
func (s *Service) runTemporal(ctx context.Context, req contract.RetrievalRequest, window *domain.Window, b *candidateBuilder) error {
	if window == nil {
		// No resolvable window: nothing may be substituted semantically.
		b.noteStrategy(domain.StrategyTemporal)
		return nil
	}
	roles := make([]contract.Role, 0, len(s.tunables.TemporalRoles))
	for _, r := range s.tunables.TemporalRoles {
		roles = append(roles, contract.Role(r))
	}
	terms := domain.RecallTerms(req.Turn.Text)
	limit := s.tunables.TemporalMaxMessages
	if req.Extra != nil && req.Extra.Kind == "time_window" {
		// Second-hop time window: the Brain told us what to look for.
		terms = domain.RecallTerms(req.Extra.Query)
	}
	msgs, err := s.store.MessageWindow(ctx, req.UserID, window.From, window.To, roles, terms, limit)
	if err != nil {
		return err
	}
	for _, m := range msgs {
		if m.DeletedAt != nil || m.UserID != req.UserID {
			continue // soft-delete and cross-user defense in depth
		}
		b.addMessage(m, domain.StrategyTemporal)
	}
	b.temporalHits += len(msgs)
	b.noteStrategy(domain.StrategyTemporal)
	return nil
}

// runThread pulls the focus thread's linked items (§3.4 thread-based).
func (s *Service) runThread(ctx context.Context, req contract.RetrievalRequest, b *candidateBuilder) error {
	if req.FocusThread == nil {
		return nil
	}
	items, err := s.store.MemoryByThread(ctx, req.UserID, *req.FocusThread, s.tunables.CandidatePoolLimit)
	if err != nil {
		return err
	}
	for _, it := range items {
		b.addMemory(it, domain.StrategyThread, 1.0)
	}
	b.noteStrategy(domain.StrategyThread)
	return nil
}

// runEntity resolves a relational/entity mention against the entity index.
func (s *Service) runEntity(ctx context.Context, req contract.RetrievalRequest, b *candidateBuilder) error {
	var hopQuery string
	if req.Extra != nil && req.Extra.Kind == "entity" {
		hopQuery = req.Extra.Query
	}
	q := domain.EntityQueryFrom(req.Turn.Text, hopQuery)
	if q.Role == "" && q.Label == "" {
		b.noteStrategy(domain.StrategyEntity)
		return nil
	}
	items, err := s.store.MemoryByEntity(ctx, req.UserID, EntityFilter{Role: q.Role, Label: q.Label}, s.tunables.CandidatePoolLimit)
	if err != nil {
		return err
	}
	sim := s.tunables.MinScore + (1-s.tunables.MinScore)*0.5
	for _, it := range items {
		b.addMemory(it, domain.StrategyEntity, sim)
	}
	b.noteStrategy(domain.StrategyEntity)
	return nil
}

// runAnaphora boosts recent items whose tags overlap the current turn's theme.
// The tags come from what the caller already knows (focus thread topic/emotion
// labels); there is no LLM inference here.
func (s *Service) runAnaphora(ctx context.Context, req contract.RetrievalRequest, b *candidateBuilder) error {
	pool, err := s.store.RecentMemory(ctx, req.UserID, s.tunables.CandidatePoolLimit)
	if err != nil {
		return err
	}
	now := s.clock.Now()
	cutoff := now.Add(-s.tunables.AnaphoraWindow)
	terms := domain.RecallTerms(req.Turn.Text)
	matched := 0
	for _, it := range pool {
		if it.UpdatedAt.Before(cutoff) {
			continue
		}
		// An item is a candidate for "same thing as before" when it shares a
		// topic tag, shares a content term, or is thread-linked.
		share := false
		for _, tag := range it.Tags {
			for _, term := range terms {
				if domain.EqualFoldTolerant(tag, term) {
					share = true
				}
			}
		}
		if !share && req.FocusThread != nil {
			for _, tid := range it.ThreadIDs {
				if tid == *req.FocusThread {
					share = true
				}
			}
		}
		if !share {
			continue
		}
		matched++
		b.addMemory(it, domain.StrategyAnaphora, domain.LexicalSimilarity(req.Turn.Text, it.Content))
	}
	b.noteStrategy(domain.StrategyAnaphora)
	b.anaphoraMatched += matched
	return nil
}

// runHybrid is the default semantic path: a bounded owner-scoped candidate pool,
// cosine rerank on stored vectors plus a lexical fallback for items without an
// embedding (§7.3 Memory.Hybrid).
func (s *Service) runHybrid(ctx context.Context, req contract.RetrievalRequest, queryVector *contract.Vector, plan domain.StrategyPlan, b *candidateBuilder) error {
	pool, err := s.store.RecentMemory(ctx, req.UserID, s.tunables.CandidatePoolLimit)
	if err != nil {
		return err
	}
	prefs, err := s.store.Preferences(ctx, req.UserID, 3)
	if err != nil {
		return err
	}
	pool = append(pool, prefs...)
	if len(pool) == 0 {
		b.noteStrategy(domain.StrategyHybrid)
		return nil
	}

	refs := make([]contract.EmbeddingKey, 0, len(pool))
	for _, it := range pool {
		refs = append(refs, contract.EmbeddingKey{Kind: contract.RefKindMemory, ID: it.ID})
	}
	var vectors map[contract.EmbeddingKey]contract.Vector
	if s.embedder != nil {
		vectors, err = s.store.Embeddings(ctx, req.UserID, refs)
		if err != nil {
			return err
		}
	}

	missing := 0
	for _, it := range pool {
		sim := float32(0)
		if v, ok := vectors[contract.EmbeddingKey{Kind: contract.RefKindMemory, ID: it.ID}]; ok && queryVector != nil {
			if v.Dim != len(v.Values) || (queryVector.Dim != 0 && v.Dim != queryVector.Dim) || v.ModelID != queryVector.ModelID {
				// §7.3: never compare vectors from different models/dimensions.
				missing++
			} else if cos, err := domain.CosineSimilarity(queryVector.Values, v.Values); err == nil {
				sim = cos
			} else {
				missing++
			}
		} else {
			missing++
		}
		// Missing embedding: degrade this item to lexical similarity instead of
		// pretending it is irrelevant or inventing a vector.
		lex := domain.LexicalSimilarity(req.Turn.Text, it.Content)
		if sim == 0 && lex > 0 {
			sim = lex * 0.9
		}
		b.addMemory(it, domain.StrategyHybrid, sim)
	}
	b.missingEmbeddings += missing
	b.noteStrategy(domain.StrategyHybrid)
	return nil
}
