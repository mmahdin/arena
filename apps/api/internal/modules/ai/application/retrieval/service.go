package retrieval

import (
	"context"
	"errors"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/telemetry"
	"strings"
	"time"
)

type Service struct {
	store   Repository
	embed   ports.EmbeddingClient
	bundles ports.BundleRegistry
	use     UsePolicy
	now     func() time.Time
}

func New(store Repository, embed ports.EmbeddingClient, bundles ports.BundleRegistry, policy UsePolicy) (*Service, error) {
	if store == nil || embed == nil || bundles == nil || policy == nil {
		return nil, errors.New("hybrid retrieval requires repository, embedder, bundle and use policy")
	}
	return &Service{store, embed, bundles, policy, time.Now}, nil
}
func (s *Service) WithClock(fn func() time.Time) *Service {
	if fn != nil {
		s.now = fn
	}
	return s
}
func (s *Service) Retrieve(ctx context.Context, req c.RetrievalRequest) (out c.RetrievalResult, err error) {
	ctx, span := trace.Start(ctx, "application/retrieval/service.Retrieve", func() any { return req })
	started := time.Now()
	out.Items = []c.MemoryView{}
	out.Meta.Strategies = []string{}
	defer func() { out.Took = time.Since(started); span.Complete(func() any { return out }, err, recover()) }()
	fail := func(reason string, e error) (c.RetrievalResult, error) { return failedWithError(out, reason, e) }
	if !c.ValidID(req.UserID) || !c.ValidID(req.Turn.TurnID) || !c.ValidID(req.Turn.MessageID) || len(req.Turn.Text) > 20000 || len(req.Exclude) > 200 {
		return fail("invalid_retrieval_request", errors.New("invalid retrieval request"))
	}
	for _, sens := range req.IncludeSensitivity {
		if !sens.Valid() {
			return fail("invalid_sensitivity", errors.New("invalid sensitivity"))
		}
	}
	allowed, e := s.use.CanSearch(ctx, req.UserID, req.Turn.TurnID)
	if e != nil {
		return fail("search_policy_check_failed", e)
	}
	if !allowed {
		return failedReason(out, "search_policy_denied"), nil
	}
	handle, e := pinnedOrActive(ctx, s.bundles, req.UserID)
	if e != nil {
		return fail("knowledge_bundle_unavailable", e)
	}
	cfg, e := loadConfig(ctx, handle)
	if e != nil {
		return fail("retrieval_config_invalid", e)
	}
	budget := time.Duration(cfg.Timeouts.TotalMS) * time.Millisecond
	if b, ok := c.BudgetFromContext(ctx); ok {
		if b.RetrievalTimeout > 0 && b.RetrievalTimeout < budget {
			budget = b.RetrievalTimeout
		}
		if !b.TotalDeadline.IsZero() && time.Until(b.TotalDeadline) < budget {
			budget = time.Until(b.TotalDeadline)
		}
	}
	ctx, cancel := context.WithTimeout(ctx, budget)
	defer cancel()
	retrievalState, e := s.store.UserState(ctx, req.UserID)
	if e != nil {
		return fail("user_state_read_failed", e)
	}
	// Initial query is always the current user message. Recent context supplies
	// exclusions in the orchestrator; no intent classifier or query LLM is called.
	query := req.Turn.Text
	hints := Hints{Focus: req.FocusThread}
	if reader, ok := s.store.(HintReader); ok && !req.Background {
		loaded, err := reader.RetrievalHints(ctx, req.UserID, req.Turn.MessageID)
		if err == nil {
			hints = loaded
		} else {
			out.Meta.Degraded = true
			out.Meta.DegradedReason = "recent_context_unavailable"
		}
	}
	refs := req.Turn.TemporalRefs
	if len(refs) == 0 {
		refs = timeWindows(query, retrievalState.TZ, s.now())
	}
	// Both deterministic queries finish before any conversational model call.
	// They share the existing retrieval deadline and scan the authorized corpus once.
	contextText := hints.RecentContext
	if contextText != "" {
		hints.ContextQuery = contextText + "\nuser: " + query
	}
	queries := []string{query}
	if hints.ContextQuery != "" {
		queries = append(queries, hints.ContextQuery)
	}
	// Only a successfully validated query embedding can supply a semantic branch.
	hints.ContextVector = nil
	type embeddingResult struct {
		response ports.EmbeddingResponse
		err      error
	}
	channels := make([]chan embeddingResult, len(queries))
	for i, text := range queries {
		consumer := "retrieval.current"
		if i > 0 {
			consumer = "retrieval.contextual"
		}
		if req.Background {
			consumer = "retrieval.background"
		}
		channels[i] = make(chan embeddingResult, 1)
		go func(ch chan embeddingResult, text string) {
			queryCtx, cancel := context.WithTimeout(ctx, time.Duration(cfg.Timeouts.PerStrategyMS)*time.Millisecond)
			defer cancel()
			response, err := s.embed.Embed(queryCtx, ports.EmbeddingRequest{Consumer: consumer, Tier: ports.ComponentEmbedding, Input: text, Timeout: time.Duration(cfg.Timeouts.PerStrategyMS) * time.Millisecond, Meta: c.CallMeta{UserID: req.UserID, TurnID: req.Turn.TurnID, Bundle: handle.Reference()}})
			ch <- embeddingResult{response, err}
		}(channels[i], text)
	}
	var vector []float32
	model := ""
	dimension := 1
	for i, ch := range channels {
		var result embeddingResult
		select {
		case result = <-ch:
		case <-ctx.Done():
			return fail("retrieval_context_failed", ctx.Err())
		}
		v, err := normalized(result.response.Vector)
		kind := "current"
		if i > 0 {
			kind = "contextual"
		}
		detail := c.RetrievalQuery{Kind: kind, Text: queries[i], Strategies: []string{}}
		if result.err != nil || err != nil || result.response.Model == "" || model != "" && (result.response.Model != model || len(v) != dimension) {
			out.Meta.Degraded = true
			out.Meta.DegradedReason = "semantic_unavailable"
			detail.FailedStrategies = []string{"vector"}
			out.Meta.FailedStrategies = append(out.Meta.FailedStrategies, kind+"_vector")
		} else {
			model = result.response.Model
			dimension = len(v)
			detail.Strategies = append(detail.Strategies, "vector")
			if i == 0 {
				vector = v
			} else {
				hints.ContextVector = v
			}
		}
		out.Meta.Queries = append(out.Meta.Queries, detail)
	}
	if len(vector) > 0 || len(hints.ContextVector) > 0 {
		out.Meta.Strategies = append(out.Meta.Strategies, "vector")
	}
	if len(out.Meta.FailedStrategies) > 0 {
		trace.Mark(ctx, "degraded", out.Meta.DegradedReason)
	}
	trace.Capture(ctx, "retrieval_queries", "retrieval_queries", "deterministic_dual_query", out.Meta.Queries)
	if telemetry.ContentEnabled(ctx) {
		telemetry.RAGEvent(ctx, "query", map[string]any{"queries": out.Meta.Queries, "background": req.Background})
	}
	sens := map[c.Sensitivity]bool{c.SensNormal: true, c.SensSensitive: true}
	if len(req.IncludeSensitivity) > 0 {
		sens = map[c.Sensitivity]bool{}
		for _, v := range req.IncludeSensitivity {
			sens[v] = true
		}
	}
	excluded := map[c.ID]bool{req.Turn.MessageID: true}
	for _, id := range req.Exclude {
		excluded[id] = true
	}
	var docs []Candidate
	// With no validated query vector, the scan can still serve independent
	// metadata signals. This lookup identity never enables a semantic branch.
	if model == "" {
		model = "__semantic_unavailable__"
	}
	q := Query{UserID: req.UserID, TriggerID: req.Turn.MessageID, Model: model, Dimension: dimension}
	if rel, ok := s.store.(Relationships); ok {
		hints.Hydrate = func(ctx context.Context, item Candidate) (Candidate, bool, error) {
			return rel.Lookup(ctx, q, item.Source, item.ID)
		}
	}
	_, e = trace.Call(ctx, "retrieval.repository.Scan", func() any { return q }, func(callCtx context.Context) (bool, error) {
		scan := s.store.Scan
		authorized := false
		if repository, ok := s.store.(interface {
			ScanAuthorized(context.Context, Query, func(Candidate) error) error
		}); ok {
			scan = repository.ScanAuthorized
			authorized = true
		}
		scanErr := scan(callCtx, q, func(item Candidate) error {
			if item.UserID != req.UserID {
				return errors.New("retrieval owner mismatch")
			}
			if !c.ValidID(item.ID) || !item.Source.Valid() || excluded[item.ID] || !sens[item.Sensitivity] || strings.TrimSpace(item.Text) == "" && item.Lexical == nil {
				return nil
			}
			if item.Status != c.MemActive && item.Status != c.MemProposed && item.Status != c.MemSupported {
				return nil
			}
			if item.Source == SourceMemory && item.EvidenceN < 1 {
				return nil
			}
			ok, e := true, error(nil)
			if !authorized {
				ok, e = s.use.CanUse(callCtx, req.UserID, item.Source, item.ID)
			}
			if e != nil {
				return e
			}
			if !ok {
				return nil
			}
			trace.Capture(callCtx, "candidate", "retrieval_candidate", "repository", item)
			docs = append(docs, item)
			return nil
		})
		if scanErr == nil {
			recordCandidates(callCtx, docs)
		}
		return true, scanErr
	})
	if e != nil {
		return fail("hybrid_candidate_search_failed", e)
	}
	limit := cfg.Defaults.MaxItems
	if req.MaxItems > 0 && req.MaxItems < limit {
		limit = req.MaxItems
	}
	// Date hints add an independent ranking vote; they do not discard older relevant evidence.
	hints.Windows = refs
	out = s.rankWithHints(ctx, out, docs, query, vector, limit, cfg.Defaults.PerItemTokens, cfg.Defaults.TotalTokens, hints)
	if len(refs) > 0 {
		found := len(temporalCandidates(docs, refs)) > 0
		out.Meta.ExactMatchFound = &found
		out.Meta.Strategies = append(out.Meta.Strategies, "temporal")
	}
	out = s.expand(ctx, q, out, limit, cfg.Defaults.TotalTokens)
	if e = ctx.Err(); e != nil {
		return fail("retrieval_context_failed", e)
	}
	recordSelectedContext(ctx, out)
	return out, nil
}
func (s *Service) rank(ctx context.Context, out c.RetrievalResult, docs []Candidate, query string, vector []float32, limit, perItem, total int) c.RetrievalResult {
	return s.rankWithHints(ctx, out, docs, query, vector, limit, perItem, total, Hints{})
}
func (s *Service) rankWithHints(ctx context.Context, out c.RetrievalResult, docs []Candidate, query string, vector []float32, limit, perItem, total int, hints Hints) c.RetrievalResult {
	ctx, span := trace.Start(ctx, "retrieval.rank", func() any { return map[string]any{"candidates": docs, "query": query} })
	defer func() { span.Complete(nil, nil, recover()) }()
	// BM25 is temporarily disconnected from active retrieval. Keep its
	// implementation/indexes intact; only observed semantic/metadata branches vote.
	rankings := make([]ragRanking, 0, 4)
	if len(vector) > 0 {
		semantic := vectorSearch(vector, docs)
		rankings = append(rankings, ragRanking{"vector", semantic})
		if trace.Enabled(ctx) {
			trace.Capture(ctx, "vector_ranks", "retrieval_rank", "cosine", traceRanks(semantic, docs))
		}
	}
	seen := map[string]bool{}
	if hints.ContextQuery != "" && len(hints.ContextVector) > 0 {
		contextualSemantic := vectorSearch(hints.ContextVector, docs)
		rankings = append(rankings, ragRanking{"contextual_vector", contextualSemantic})
		out.Meta.Strategies = append(out.Meta.Strategies, "contextual_vector")
		trace.Capture(ctx, "contextual_vector_ranks", "retrieval_rank", "cosine", traceRanks(contextualSemantic, docs))
	}
	if hints.Focus != nil {
		rankings = append(rankings, ragRanking{"thread", threadRanks(docs, hints.Focus)})
		out.Meta.Strategies = append(out.Meta.Strategies, "thread")
	}
	if len(hints.Windows) > 0 {
		hits := []ranked{}
		for i, d := range docs {
			if len(temporalCandidates([]Candidate{d}, hints.Windows)) > 0 {
				hits = append(hits, ranked{i, 1})
			}
		}
		rankings = append(rankings, ragRanking{"temporal", order(hits, docs)})
	}
	lists := make([][]ranked, 0, len(rankings))
	for _, ranking := range rankings {
		lists = append(lists, ranking.hits)
	}
	fused := fuse(docs, lists...)
	trace.Capture(ctx, "merged_ranks", "retrieval_merge", "reciprocal_rank_fusion", traceRanks(fused, docs))
	recordRankings(ctx, docs, rankings, fused)
	fullTrace := telemetry.ContentEnabled(ctx)
	var decisions []map[string]any
	stopReason := ""
	initialTokens, initialItems := total, len(out.Items)
	// Select in RRF order; only exact source identities and configured limits filter hits.
	for position, hit := range fused {
		if len(out.Items) >= limit || total <= 0 {
			stopReason = "not_considered_limit"
			if total <= 0 {
				stopReason = "not_considered_budget"
			}
			break
		}
		item := docs[hit.index]
		key := string(item.Source) + ":" + item.ID.String()
		if seen[key] {
			if fullTrace {
				decisions = append(decisions, rankDecision(item, position, hit, "duplicate_source"))
			}
			continue
		}
		if item.Text == "" && hints.Hydrate != nil {
			hydrated, found, e := hints.Hydrate(ctx, item)
			if e != nil || !found || hydrated.SourceVersion != item.SourceVersion {
				out.Meta.Degraded = true
				out.Meta.DegradedReason = "source_hydration_failed"
				if fullTrace {
					decisions = append(decisions, rankDecision(item, position, hit, "source_hydration_failed"))
				}
				continue
			}
			item = hydrated
		}
		text := truncate(item.Text, min(perItem, total))
		if text == "" {
			if fullTrace {
				decisions = append(decisions, rankDecision(item, position, hit, "empty_after_budget"))
			}
			continue
		}
		// Track only selected items, so duplicate-heavy corpora do not grow
		// deduplication state or per-item trace artifacts with every scanned hit.
		seen[key] = true
		remainingBefore := total
		total -= approximateTokens(text)
		if fullTrace {
			decision := rankDecision(item, position, hit, "selected")
			decision["context_tokens_before"] = remainingBefore
			decision["context_tokens_after"] = total
			decision["text_truncated"] = text != item.Text
			decisions = append(decisions, decision)
		}
		trace.Capture(ctx, "score", "retrieval_score", key, hit.score)
		view := c.MemoryView{ID: item.ID, Source: string(item.Source), SourceVersion: item.SourceVersion, SourceCreatedAt: item.CreatedAt.UTC().Format(time.RFC3339Nano), SourceConversationID: item.ConversationID, Kind: item.Kind, Epistemic: item.Epistemic, Status: item.Status, Text: text, Age: age(s.now(), item.CreatedAt), EvidenceN: item.EvidenceN, Sensitivity: item.Sensitivity, Verbatim: item.Source == SourceMessage, Score: float32(hit.score), ThreadID: item.ThreadID, Evidence: item.Evidence}
		if item.ValidFrom != nil {
			at := item.ValidFrom.UTC().Format("2006-01-02")
			view.ValidAt = &at
		}
		out.Items = append(out.Items, view)
		trace.Capture(ctx, "selected", "retrieval_selection", key, view)
	}
	if fullTrace {
		telemetry.RAGEvent(ctx, "selection", map[string]any{
			"items": decisions, "selected_count": len(out.Items) - initialItems,
			"ranked_count": len(fused), "unconsidered_count": len(fused) - len(decisions),
			"stop_reason": stopReason, "max_items": limit, "per_item_tokens": perItem,
			"initial_context_tokens": initialTokens, "remaining_context_tokens": total,
		})
	}
	return out
}
func failedWithError(out c.RetrievalResult, reason string, cause error) (c.RetrievalResult, error) {
	if errors.Is(cause, context.DeadlineExceeded) {
		reason = "retrieval_timeout"
	}
	if errors.Is(cause, context.Canceled) {
		reason = "retrieval_cancelled"
	}
	return failedReason(out, reason), cause
}
func failedReason(out c.RetrievalResult, reason string) c.RetrievalResult {
	out.Items = []c.MemoryView{}
	out.Meta.Degraded = true
	out.Meta.DegradedReason = reason
	out.Meta.ExactMatchFound = nil
	return out
}

type bundleContextKey struct{}
type scopedBundle struct {
	user   c.ID
	handle ports.Bundle
}

func PinBundle(ctx context.Context, user c.ID, handle ports.Bundle) context.Context {
	return context.WithValue(ctx, bundleContextKey{}, scopedBundle{user, handle})
}
func pinnedOrActive(ctx context.Context, registry ports.BundleRegistry, user c.ID) (ports.Bundle, error) {
	if p, ok := ctx.Value(bundleContextKey{}).(scopedBundle); ok {
		if p.user != user || p.handle == nil {
			return nil, errors.New("retrieval bundle owner mismatch")
		}
		return p.handle, nil
	}
	return registry.Active(ctx, user)
}

const rankTracePreviewLimit = 5

func traceRanks(hits []ranked, docs []Candidate) []map[string]any {
	hits = hits[:min(len(hits), rankTracePreviewLimit)]
	out := make([]map[string]any, 0, len(hits))
	for rank, hit := range hits {
		item := docs[hit.index]
		out = append(out, map[string]any{"source": item.Source, "id": item.ID, "rank": rank + 1, "score": hit.score})
	}
	return out
}
