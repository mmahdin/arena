package retrieval

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/bundle"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

type Service struct {
	store   Repository
	embed   ports.EmbeddingClient
	bundles ports.BundleRegistry
	use     UsePolicy
	now     func() time.Time
}

var _ ports.Retriever = (*Service)(nil)

// New fails closed unless an explicitly reviewed AI-use policy has been
// supplied. A soft-delete predicate is NOT an authorization policy.
func New(store Repository, embed ports.EmbeddingClient, bundles ports.BundleRegistry, policy UsePolicy) (*Service, error) {
	if store == nil || bundles == nil || policy == nil {
		return nil, errors.New("WP-09 requires owner-scoped storage, bundles, and explicit AI-use policy")
	}
	return &Service{store: store, embed: embed, bundles: bundles, use: policy, now: time.Now}, nil
}

func (s *Service) WithClock(fn func() time.Time) *Service {
	if fn != nil {
		s.now = fn
	}
	return s
}

func (s *Service) Retrieve(ctx context.Context, req c.RetrievalRequest) (out c.RetrievalResult, err error) {
	start := time.Now()
	out.Items = []c.MemoryView{}
	out.Meta.Strategies = []string{}
	defer func() { out.Took = time.Since(start) }()
	if !c.ValidID(req.UserID) || !c.ValidID(req.Turn.TurnID) || !c.ValidID(req.Turn.MessageID) ||
		len(req.Turn.Text) > 20000 || len(req.Strategies) > 5 || len(req.Exclude) > 200 || len(req.IncludeSensitivity) > 3 {
		return failed(out), errors.New("invalid retrieval request")
	}
	if req.FocusThread != nil && !c.ValidID(*req.FocusThread) {
		return failed(out), errors.New("invalid focus thread")
	}
	if req.Extra != nil && (len(strings.Fields(req.Extra.Query)) > 12 || len(req.Extra.Query) > 400 || !req.Extra.Kind.Valid() || !req.Extra.Why.Valid()) {
		return failed(out), errors.New("invalid second-hop query")
	}
	for _, sens := range req.IncludeSensitivity {
		if !sens.Valid() {
			return failed(out), errors.New("invalid sensitivity")
		}
	}
	if err = ctx.Err(); err != nil {
		return failed(out), err
	}
	allowed, err := s.use.CanSearch(ctx, req.UserID, req.Turn.TurnID)
	if err != nil {
		return failed(out), fmt.Errorf("retrieval search eligibility: %w", err)
	}
	if !allowed {
		return failed(out), nil
	}
	handle, err := pinnedOrActive(ctx, s.bundles, req.UserID)
	if err != nil {
		return failed(out), err
	}
	cfg, err := loadConfig(ctx, handle)
	if err != nil {
		return failed(out), err
	}
	budget := time.Duration(cfg.Timeouts.TotalMS) * time.Millisecond
	if b, ok := c.BudgetFromContext(ctx); ok {
		if b.RetrievalTimeout > 0 && b.RetrievalTimeout < budget {
			budget = b.RetrievalTimeout
		}
		if !b.TotalDeadline.IsZero() {
			until := time.Until(b.TotalDeadline)
			if until <= 0 {
				return failed(out), context.DeadlineExceeded
			}
			if until < budget {
				budget = until
			}
		}
	}
	ctx, cancel := context.WithTimeout(ctx, budget)
	defer cancel()
	state, err := s.store.UserState(ctx, req.UserID)
	if err != nil {
		return failed(out), err
	}
	// An explicit memory pause blocks both memory items and raw historical
	// recall; no permission can be inferred from records being retained.
	if state.MemoryPaused {
		return failed(out), nil
	}
	if _, err = time.LoadLocation(state.TZ); err != nil {
		return failed(out), errors.New("invalid persisted timezone")
	}
	refs := req.Turn.TemporalRefs
	if len(refs) == 0 {
		refs, err = TemporalReferences(req.Turn.Text, s.now(), state.TZ)
	}
	if err != nil {
		return failed(out), err
	}
	if len(refs) > 3 {
		return failed(out), errors.New("too many temporal references")
	}
	for _, ref := range refs {
		if !ref.From.Before(ref.To) || ref.To.Sub(ref.From) > 32*24*time.Hour {
			return failed(out), errors.New("invalid temporal interval")
		}
	}
	selected := req.Strategies
	if len(selected) == 0 {
		selected = selectStrategies(req, len(refs) > 0)
	}
	seen := map[c.RetrievalStrategy]bool{}
	for _, strat := range selected {
		if seen[strat] {
			continue
		}
		seen[strat] = true
		switch strat {
		case c.StratTemporal, c.StratThread, c.StratHybrid, c.StratAnaphora, c.StratEntity:
			out.Meta.Strategies = append(out.Meta.Strategies, string(strat))
		default:
			return failed(out), errors.New("unknown retrieval strategy")
		}
	}
	if req.Extra != nil {
		out.Meta.Hops = 1
	}
	maxItems := cfg.Defaults.MaxItems
	if req.Extra != nil {
		maxItems++
	}
	if maxItems > cfg.Defaults.HardCapWithHop {
		maxItems = cfg.Defaults.HardCapWithHop
	}
	if req.MaxItems > 0 && req.MaxItems < maxItems {
		maxItems = req.MaxItems
	}
	if maxItems < 1 {
		maxItems = 1
	}

	sens := map[c.Sensitivity]bool{c.SensNormal: true, c.SensSensitive: true}
	if len(req.IncludeSensitivity) != 0 {
		sens = map[c.Sensitivity]bool{}
		for _, value := range req.IncludeSensitivity {
			sens[value] = true
		}
	}
	excluded := map[c.ID]bool{req.Turn.MessageID: true}
	for _, id := range req.Exclude {
		excluded[id] = true
	}

	// Exact date/role recall is never silently replaced by semantic matches.
	exact := seen[c.StratTemporal] && len(refs) > 0
	if seen[c.StratTemporal] {
		found := false
		out.Meta.ExactMatchFound = &found
	}
	recent, err := s.store.RecentBefore(ctx, req.UserID, req.Turn.MessageID, cfg.NoveltyFilter.SkipIfInRecentMessages)
	if err != nil {
		return failed(out), err
	}
	var queryVector []float32
	var model string
	if seen[c.StratHybrid] && s.embed != nil && strings.TrimSpace(req.Turn.Text) != "" {
		remaining := time.Duration(cfg.Timeouts.PerStrategyMS) * time.Millisecond
		embCtx, stop := context.WithTimeout(ctx, remaining)
		resp, e := s.embed.Embed(embCtx, ports.EmbeddingRequest{Tier: ports.TierEmbedding, Input: queryText(req), Timeout: remaining, Meta: c.CallMeta{UserID: req.UserID, TurnID: req.Turn.TurnID, Bundle: handle.Reference()}})
		stop()
		if e != nil {
			return failed(out), fmt.Errorf("embedding retrieval: %w", e)
		}
		queryVector, err = normalized(resp.Vector)
		if err != nil || resp.Model == "" {
			return failed(out), errors.New("invalid query embedding")
		}
		model = resp.Model
	}
	var all []Candidate
	collect := func(candidates []Candidate) error {
		for _, item := range candidates {
			if item.UserID != req.UserID {
				return errors.New("retrieval adapter owner mismatch")
			}
			if !c.ValidID(item.ID) || item.DeletedAt != nil || excluded[item.ID] {
				continue
			}
			if !sens[item.Sensitivity] || strings.TrimSpace(item.Text) == "" {
				continue
			}
			if item.Source == SourceMemory {
				if item.EvidenceN < 1 || !activeMemory(item, cfg, req.FocusThread) {
					continue
				}
			} else if item.Source != SourceMessage {
				continue
			}
			use, e := s.use.CanUse(ctx, req.UserID, item.Source, item.ID)
			if e != nil {
				return e
			}
			if !use {
				return errors.New("source AI-use authorization denied")
			}
			all = append(all, item)
		}
		return nil
	}
	search := func(q MemoryQuery) error {
		subctx, stop := context.WithTimeout(ctx, time.Duration(cfg.Timeouts.PerStrategyMS)*time.Millisecond)
		defer stop()
		q.UserID, q.TriggerID = req.UserID, req.Turn.MessageID
		q.Model, q.Dimension = model, len(queryVector)
		if q.Limit == 0 {
			q.Limit = 96
		}
		found, e := s.store.Memories(subctx, q)
		if e != nil {
			return e
		}
		return collect(found)
	}
	if seen[c.StratTemporal] {
		if len(refs) > 0 {
			for _, ref := range refs {
				subctx, stop := context.WithTimeout(ctx, time.Duration(cfg.Timeouts.PerStrategyMS)*time.Millisecond)
				msgs, e := s.store.Messages(subctx, MessageQuery{UserID: req.UserID, TriggerID: req.Turn.MessageID, From: ref.From, To: ref.To, Limit: cfg.Temporal.MaxMessages})
				stop()
				if e != nil {
					return failed(out), e
				}
				terms := subjectTerms(req.Turn.Text, ref.Phrase)
				for _, message := range msgs {
					if !textMatches(message.Text, terms) {
						continue
					}
					if e = collect([]Candidate{message}); e != nil {
						return failed(out), e
					}
				}
			}
		}
		// A supplied date reference is an exact lookup, not permission to
		// fill a missing answer with vaguely similar memories.
		if exact {
			out = s.rank(out, all, req, cfg, queryVector, recent, excluded, maxItems, true)
			found := len(out.Items) > 0
			out.Meta.ExactMatchFound = &found
			return out, nil
		}
	}
	if seen[c.StratThread] && req.FocusThread != nil {
		if err = search(MemoryQuery{FocusThread: req.FocusThread, ThreadOnly: true}); err != nil {
			return failed(out), err
		}
	}
	if seen[c.StratEntity] {
		role, label := targetEntity(req)
		if role != "" || label != "" {
			if err = search(MemoryQuery{EntityRole: role, EntityLabel: label}); err != nil {
				return failed(out), err
			}
		} else {
			out.Meta.AmbiguousReference = true
		}
	}
	if seen[c.StratAnaphora] {
		since := s.now().AddDate(0, 0, -cfg.Anaphora.RecentWindowDays)
		if err = search(MemoryQuery{FocusThread: req.FocusThread, Since: &since}); err != nil {
			return failed(out), err
		}
		if req.FocusThread == nil {
			out.Meta.AmbiguousReference = true
		}
	}
	if seen[c.StratHybrid] {
		if err = search(MemoryQuery{FocusThread: req.FocusThread}); err != nil {
			return failed(out), err
		}
		// SQL Server 2022 has no vector ANN. Complement bounded recent and
		// important pools with owner-scoped lexical candidates, if possible.
		for _, term := range firstTerms(queryText(req), 2) {
			if err = search(MemoryQuery{FocusThread: req.FocusThread, LexicalTerm: term, Limit: 48}); err != nil {
				return failed(out), err
			}
		}
	}
	out = s.rank(out, all, req, cfg, queryVector, recent, excluded, maxItems, false)
	if seen[c.StratAnaphora] && len(out.Items) != 1 {
		out.Meta.AmbiguousReference = true
	}
	if e := ctx.Err(); e != nil {
		return failed(out), e
	}
	return out, nil
}

func failed(out c.RetrievalResult) c.RetrievalResult {
	out.Items = []c.MemoryView{}
	out.Meta.Degraded = true
	out.Meta.ExactMatchFound = nil // cannot assert absence after an inaccessible/failed read
	return out
}

func selectStrategies(req c.RetrievalRequest, temporal bool) []c.RetrievalStrategy {
	if temporal {
		return []c.RetrievalStrategy{c.StratTemporal}
	}
	if req.Extra != nil {
		switch req.Extra.Kind {
		case c.ContextEntity:
			return []c.RetrievalStrategy{c.StratEntity, c.StratHybrid}
		case c.ContextTimeWindow:
			return []c.RetrievalStrategy{c.StratTemporal}
		case c.ContextItem, c.ContextTopic:
			return []c.RetrievalStrategy{c.StratHybrid, c.StratThread}
		}
	}
	if containsAnaphora(req.Turn.Text) || req.Turn.ReplyToPending {
		return []c.RetrievalStrategy{c.StratAnaphora, c.StratThread}
	}
	if role, label := targetEntity(req); role != "" || label != "" {
		return []c.RetrievalStrategy{c.StratEntity, c.StratHybrid}
	}
	if req.FocusThread != nil {
		return []c.RetrievalStrategy{c.StratThread, c.StratHybrid}
	}
	return []c.RetrievalStrategy{c.StratHybrid}
}

func activeMemory(item Candidate, cfg bundle.Retrieval, focus *c.ID) bool {
	switch item.Kind {
	case c.MemFact, c.MemPreference, c.MemEpisode, c.MemGoal, c.MemEvent:
		return item.Status == c.MemActive
	case c.MemHypothesis:
		if item.Status != c.MemProposed && item.Status != c.MemSupported {
			return false
		}
		allowed := false
		for _, v := range cfg.Hypotheses.IncludeStatuses {
			if v == string(item.Status) {
				allowed = true
			}
		}
		return allowed && (!cfg.Hypotheses.RequireThreadMatch || threadContains(item.ThreadIDs, focus))
	default:
		return false // includes safety_event, which cannot enter normal recall
	}
}

func threadContains(ids []c.ID, focus *c.ID) bool {
	if focus == nil {
		return false
	}
	for _, id := range ids {
		if id == *focus {
			return true
		}
	}
	return false
}

func queryText(req c.RetrievalRequest) string {
	if req.Extra != nil && strings.TrimSpace(req.Extra.Query) != "" {
		return req.Extra.Query
	}
	return req.Turn.Text
}

// This optional per-turn override is set only by NewIntegrated after pinning
// WP-03. Standalone readers still require an explicit registry dependency.
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

func (s *Service) rank(out c.RetrievalResult, candidates []Candidate, req c.RetrievalRequest, cfg bundle.Retrieval, embedding []float32, recent []string, excluded map[c.ID]bool, limit int, exact bool) c.RetrievalResult {
	type scored struct {
		candidate Candidate
		score     float64
	}
	sorted := make([]scored, 0, len(candidates))
	for _, cand := range candidates {
		if excluded[cand.ID] || (!exact && isNovel(cand.Text, recent, cfg.NoveltyFilter.SimilarityGTE)) {
			continue
		}
		termScore := topicalMatch(queryText(req), cand.Text)
		vectorScore, _ := cosine(embedding, cand.Vector)
		role, label := targetEntity(req)
		entityHit := entityContains(cand.Entities, role, label)
		threadHit := threadContains(cand.ThreadIDs, req.FocusThread)
		anchorScore := 0.0
		emotionMatch := false
		if containsAnaphora(req.Turn.Text) {
			for _, raw := range recent {
				anchorScore = maxFloat(anchorScore, topicalMatch(raw, cand.Text))
				emotionMatch = emotionMatch || explicitEmotionOverlap(raw, cand.Text)
			}
		}
		// High importance and freshness alone are NOT evidence of relevance.
		// An unresolved anaphor without an anchor should produce ambiguity,
		// not an unrelated highly ranked autobiographical claim.
		if !exact && termScore == 0 && vectorScore == 0 && !entityHit && !threadHit && anchorScore == 0 {
			continue
		}
		score := scoreCandidate(cand, queryText(req), embedding, req.FocusThread, s.now(), cfg, exact)
		if entityHit {
			score = maxFloat(score, 0.65)
		}
		if anchorScore > 0 {
			score += cfg.Anaphora.TopicBoost * anchorScore
		}
		if emotionMatch {
			score += cfg.Anaphora.EmotionBoost
		}
		if score > 1 {
			score = 1
		}
		if score < cfg.Defaults.MinScore {
			continue
		}
		sorted = append(sorted, scored{cand, score})
	}
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].score != sorted[j].score {
			return sorted[i].score > sorted[j].score
		}
		if !sorted[i].candidate.CreatedAt.Equal(sorted[j].candidate.CreatedAt) {
			return sorted[i].candidate.CreatedAt.After(sorted[j].candidate.CreatedAt)
		}
		return sorted[i].candidate.ID.String() < sorted[j].candidate.ID.String()
	})
	dedup := map[c.ID]bool{}
	cluster := map[c.ID]bool{}
	texts := []string{}
	remaining := 400
	for _, v := range sorted {
		if len(out.Items) >= limit || remaining <= 0 {
			break
		}
		item := v.candidate
		if dedup[item.ID] {
			continue
		}
		if item.ClusterID != nil && cluster[*item.ClusterID] {
			continue
		}
		duplicate := false
		for _, normalized := range texts {
			if similarity(normalized, item.Text) >= 0.92 {
				duplicate = true
				break
			}
		}
		if duplicate {
			continue
		}
		tokens := cfg.Defaults.PerItemTokens
		if tokens > remaining {
			tokens = remaining
		}
		text := truncate(item.Text, tokens)
		if text == "" {
			continue
		}
		dedup[item.ID] = true
		if item.ClusterID != nil {
			cluster[*item.ClusterID] = true
		}
		texts = append(texts, item.Text)
		remaining -= approximateTokens(text)
		mv := c.MemoryView{ID: item.ID, Kind: item.Kind, Epistemic: item.Epistemic, Status: item.Status,
			Text: text, Age: age(s.now(), item.CreatedAt), EvidenceN: item.EvidenceN,
			Sensitivity: item.Sensitivity, Verbatim: item.Source == SourceMessage, Score: float32(v.score)}
		if item.ValidFrom != nil {
			v := item.ValidFrom.UTC().Format("2006-01-02")
			mv.ValidAt = &v
		}
		out.Items = append(out.Items, mv)
	}
	return out
}
