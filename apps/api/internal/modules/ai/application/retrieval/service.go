package retrieval

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

// Service implements ports.Retriever. It is speculative and deterministic: it
// uses only cheap lexical/temporal signals and, at most, one embedding call for
// the current message. It never calls a reasoning model and never writes state.
type Service struct {
	store     Store
	policy    ports.BundleRegistry
	embedding Embedder
	now       clock
}

var _ ports.Retriever = (*Service)(nil)

const (
	// maxQueryChars bounds the text sent to the embedding provider. It is a
	// transport guard, not a content policy.
	maxQueryChars = 2000
	// strategyCandidateLimit bounds each adapter read so no strategy can build
	// an unbounded candidate pool (§7.3 "bounded candidate pool").
	strategyCandidateLimit = 40
	// maxSecondHopItems is the contract's absolute hard cap (§4.3, §3.4).
	maxSecondHopItems = 7
)

// Embedder is the additive WP-04 embedding companion, narrowed to what
// retrieval needs. A nil implementation disables semantic scoring; candidates
// then rank on recency/importance/thread match only, and no similarity is
// invented.
type Embedder interface {
	Embed(ctx context.Context, text string) (vector []float32, model string, err error)
}

// New constructs the retrieval service.
func New(store Store, registry ports.BundleRegistry, embedding Embedder, opts ...Option) (*Service, error) {
	if store == nil {
		return nil, errors.New("WP-09 retrieval requires a storage read port")
	}
	if registry == nil {
		return nil, errors.New("WP-09 retrieval requires a Knowledge Bundle registry")
	}
	s := &Service{store: store, policy: registry, embedding: embedding, now: time.Now}
	for _, opt := range opts {
		if opt != nil {
			opt(s)
		}
	}
	return s, nil
}

// Option is a narrow per-instance test seam; it is not a global override.
type Option func(*Service)

// WithClock makes recency and temporal windows deterministic in tests.
func WithClock(now func() time.Time) Option {
	return func(s *Service) {
		if now != nil {
			s.now = now
		}
	}
}

// Retrieve implements ports.Retriever. A returned error means the caller must
// set Degraded; the service never returns fabricated items alongside a healthy
// metadata block.
func (s *Service) Retrieve(ctx context.Context, req c.RetrievalRequest) (c.RetrievalResult, error) {
	started := s.now()
	if err := ctx.Err(); err != nil {
		return degraded(hopCount(req)), err
	}
	if !c.ValidID(req.UserID) {
		return c.RetrievalResult{}, newError(c.ErrForbidden, "retrieval owner identity required")
	}
	if !c.ValidID(req.Turn.MessageID) || strings.TrimSpace(req.Turn.Text) == "" {
		return c.RetrievalResult{}, newError(c.ErrValidation, "turn message identity and text required")
	}
	cfg, err := LoadConfig(ctx, s.policy, req.UserID)
	if err != nil {
		return c.RetrievalResult{}, newError(c.ErrBundle, "retrieval configuration unavailable")
	}
	q := buildLexicalQuery(req.Turn.Text)
	loc, locOK := s.userLocation(ctx, req.UserID)
	exclude := make(map[c.ID]bool, len(req.Exclude))
	for _, id := range req.Exclude {
		if c.ValidID(id) {
			exclude[id] = true
		}
	}
	limit := requestedLimit(req.MaxItems, cfg)
	// The bundle budgets retrieval as a whole AND per strategy (§7.3 timeouts).
	// Each strategy therefore gets its own bounded attempt inside the total.
	totalCtx, totalCancel := context.WithTimeout(ctx, cfg.Timeout)
	defer totalCancel()

	strategies := s.chooseStrategies(req, cfg, q, loc)
	meta := c.RetrievalMeta{Strategies: make([]string, 0, len(strategies)), Hops: hopCount(req)}
	var (
		cands     []candidate
		ambiguous bool
		resolved  bool
		exactSet  bool
		exact     bool
		failures  int
	)
	for _, strategy := range strategies {
		meta.Strategies = append(meta.Strategies, string(strategy))
		if err := totalCtx.Err(); err != nil {
			meta.Degraded = true
			failures++
			continue
		}
		attemptCtx, attemptCancel := context.WithTimeout(totalCtx, cfg.Strategy)
		out, amb, found, serr := s.run(attemptCtx, strategy, req, cfg, q, limit, loc, locOK)
		attemptCancel()
		if serr != nil {
			// Ownership and validation failures are not availability problems:
			// they abort the request instead of being reported as a degraded but
			// successful retrieval.
			if fatalRetrievalError(serr) {
				return c.RetrievalResult{}, serr
			}
			// One failing strategy must not discard the others, but the
			// overall result is degraded so the Brain cannot claim full recall.
			meta.Degraded = true
			failures++
			continue
		}
		if amb {
			ambiguous = true
		}
		if strategy == c.StratTemporal {
			exact, exactSet = found, true
		}
		if found && resolvesReferences(strategy) {
			// Only a strategy that actually looks for the referent may clear the
			// ambiguity flag. A generic hybrid hit is unrelated content, not a
			// resolution.
			resolved = true
		}
		cands = append(cands, out...)
	}
	if failures == len(strategies) && len(cands) == 0 && len(strategies) > 0 {
		return c.RetrievalResult{Meta: c.RetrievalMeta{
			Strategies: meta.Strategies, Degraded: true, Hops: meta.Hops,
		}}, newError(c.ErrRetrievalDegraded, "all retrieval strategies failed")
	}

	cands = s.attachSimilarity(ctx, req, cfg, cands)
	if q.Anaphora {
		applyAnaphoraBoost(cfg, cands, s.now())
	}
	items := rank(cfg, cands, exclude, s.now(), limit, contextBudget(cfg, limit))
	if exactSet {
		v := exact
		meta.ExactMatchFound = &v
	}
	if ambiguous && !resolved {
		// Nothing resolved the reference: the Brain must clarify, not guess.
		meta.AmbiguousReference = true
	}
	return c.RetrievalResult{Items: items, Meta: meta, Took: s.now().Sub(started)}, nil
}

// resolvesReferences reports whether a strategy can settle a reference to past
// content, and may therefore clear an "ambiguous reference" flag.
func resolvesReferences(strategy c.RetrievalStrategy) bool {
	switch strategy {
	case c.StratThread, c.StratAnaphora, c.StratEntity, c.StratTemporal:
		return true
	default:
		return false
	}
}

// chooseStrategies is the deterministic router from Architecture §3.4. Caller
// strategies are honoured when valid; otherwise the rules select the method
// that fits the question instead of always running semantic search.
func (s *Service) chooseStrategies(req c.RetrievalRequest, cfg Config, q lexicalQuery, loc *time.Location) []c.RetrievalStrategy {
	if req.Extra != nil {
		// A second hop is one targeted retrieval, never a fan-out.
		switch req.Extra.Kind {
		case c.ContextEntity:
			return []c.RetrievalStrategy{c.StratEntity}
		case c.ContextTimeWindow:
			return []c.RetrievalStrategy{c.StratTemporal}
		default:
			return []c.RetrievalStrategy{c.StratHybrid}
		}
	}
	if len(req.Strategies) > 0 {
		out := make([]c.RetrievalStrategy, 0, len(req.Strategies))
		seen := map[c.RetrievalStrategy]bool{}
		for _, st := range req.Strategies {
			if !st.Valid() || seen[st] {
				continue
			}
			seen[st] = true
			out = append(out, st)
		}
		if len(out) > 0 {
			return out
		}
	}
	var out []c.RetrievalStrategy
	// Temporal first: exact recall must not be answered by a semantic
	// near-match ("no semantic substitution", §4.4 scenario 2).
	if hasTemporalSignal(req, cfg, q, s.now(), loc) {
		out = append(out, c.StratTemporal)
	}
	if req.FocusThread != nil || q.Anaphora {
		out = append(out, c.StratThread)
	}
	if q.Anaphora {
		out = append(out, c.StratAnaphora)
	}
	if len(q.Roles) > 0 {
		out = append(out, c.StratEntity)
	}
	out = append(out, c.StratHybrid)
	return dedupeStrategies(out)
}

// hasTemporalSignal decides whether the exact/temporal strategy is justified:
// the Gateway already extracted a window, or the message both asks to recall
// something and names a parseable time.
func hasTemporalSignal(req c.RetrievalRequest, cfg Config, q lexicalQuery, now time.Time, loc *time.Location) bool {
	if len(req.Turn.TemporalRefs) > 0 {
		return true
	}
	if req.Extra != nil && req.Extra.Kind == c.ContextTimeWindow {
		return true
	}
	if !q.RecallVerb {
		return false
	}
	_, ok := detectTemporalPhrase(req.Turn.Text, NewTemporalParser(cfg.Temporal.WeekStartsOn), now, loc)
	return ok
}

func dedupeStrategies(in []c.RetrievalStrategy) []c.RetrievalStrategy {
	seen := map[c.RetrievalStrategy]bool{}
	out := in[:0]
	for _, st := range in {
		if seen[st] {
			continue
		}
		seen[st] = true
		out = append(out, st)
	}
	return out
}

// run dispatches one strategy. The returned (ambiguous, found) pair is only
// meaningful for strategies that can resolve a reference.
func (s *Service) run(ctx context.Context, strategy c.RetrievalStrategy, req c.RetrievalRequest, cfg Config, q lexicalQuery, limit int, loc *time.Location, locOK bool) ([]candidate, bool, bool, error) {
	switch strategy {
	case c.StratTemporal:
		if !locOK {
			return nil, false, false, newError(c.ErrRetrievalDegraded, "user time zone unavailable")
		}
		return s.temporal(ctx, req, cfg, limit, loc)
	case c.StratThread:
		return s.thread(ctx, req, cfg, limit)
	case c.StratAnaphora:
		return s.anaphora(ctx, req, cfg, q, limit)
	case c.StratEntity:
		return s.entity(ctx, req, cfg, q)
	case c.StratHybrid:
		return s.hybrid(ctx, req, cfg, q, limit)
	default:
		return nil, false, false, fmt.Errorf("unsupported retrieval strategy %q", strategy)
	}
}

// ---- strategies ----

// temporal performs exact recall over raw messages in the user's local window.
// It returns verbatim messages only — never a summary and never a semantic
// substitute. An empty window is a successful "found nothing" answer.
func (s *Service) temporal(ctx context.Context, req c.RetrievalRequest, cfg Config, limit int, loc *time.Location) ([]candidate, bool, bool, error) {
	parser := NewTemporalParser(cfg.Temporal.WeekStartsOn)
	windows := make([]c.TemporalRef, 0, len(req.Turn.TemporalRefs)+1)
	for _, ref := range req.Turn.TemporalRefs {
		if ref.From.IsZero() || ref.To.Before(ref.From) {
			continue
		}
		windows = append(windows, ref)
	}
	if req.Extra != nil && req.Extra.Kind == c.ContextTimeWindow && req.Extra.Query != "" {
		m, ok := parser.Parse(req.Extra.Query, s.now(), loc)
		if !ok {
			// The Brain asked for a window we cannot parse; report ambiguity
			// rather than substituting a semantically similar day.
			return nil, true, false, nil
		}
		windows = append(windows[:0], c.TemporalRef{Phrase: m.Phrase, From: m.From.UTC(), To: m.To.UTC(), RecallVerb: true})
	}
	if len(windows) == 0 {
		m, ok := detectTemporalPhrase(req.Turn.Text, parser, s.now(), loc)
		if !ok {
			// No parseable window: the caller must not receive an invented one.
			return nil, false, false, nil
		}
		// detectTemporalPhrase already evaluated the phrase in `loc`, so the
		// window is reused as-is rather than re-parsed in another zone.
		windows = append(windows, c.TemporalRef{Phrase: m.Phrase, From: m.From.UTC(), To: m.To.UTC(), RecallVerb: true})
	}
	roles := messageRoles(cfg)
	max := cfg.Temporal.MaxMessages
	if max > limit {
		max = limit
	}
	if max > strategyCandidateLimit {
		max = strategyCandidateLimit
	}
	var out []candidate
	for _, w := range windows {
		rows, err := s.store.MessagesInWindow(ctx, MessageWindowQuery{
			UserID: req.UserID, From: w.From.UTC(), To: w.To.UTC(), Roles: roles, Limit: max,
		})
		if err != nil {
			return nil, false, false, err
		}
		for _, row := range rows {
			// Owner scoping is enforced by SQL; this is defence in depth
			// against a misbehaving adapter.
			if !c.ValidID(row.ID) || (row.Role != c.RoleUser && row.Role != c.RoleAssistant) {
				continue
			}
			if row.ID == req.Turn.MessageID || row.ID == req.Turn.TurnID {
				continue // the triggering message is already in the turn view
			}
			out = append(out, candidate{Message: &messageRecord{
				ID: row.ID, TurnID: row.TurnID, Role: row.Role, Text: row.Text, CreatedAt: row.CreatedAt,
			}, Strategy: c.StratTemporal, Exact: true, Match: 1})
			if len(out) == max {
				break
			}
		}
	}
	return out, false, len(out) > 0, nil
}

// thread pulls a focused thread's linked memory items (§3.4 thread-based
// retrieval). The last episode and active hypotheses arrive through the same
// thread link; no separate hypothesis store is queried.
func (s *Service) thread(ctx context.Context, req c.RetrievalRequest, cfg Config, limit int) ([]candidate, bool, bool, error) {
	var thread c.ThreadRecord
	if req.FocusThread != nil {
		if !c.ValidID(*req.FocusThread) {
			return nil, false, false, newError(c.ErrValidation, "focus thread identity invalid")
		}
		t, err := s.store.Thread(ctx, req.UserID, *req.FocusThread)
		if err != nil {
			// A focus thread that does not resolve to an owner-visible row is a
			// forbidden request, not an outage. Any other error stays an
			// availability problem so it can be reported as degraded.
			if errors.Is(err, ErrNotVisible) {
				return nil, false, false, newError(c.ErrForbidden, "focus thread is not accessible to this owner")
			}
			return nil, false, false, err
		}
		if t.ID != *req.FocusThread || t.UserID != req.UserID {
			return nil, false, false, newError(c.ErrForbidden, "focus thread owner mismatch")
		}
		thread = t
	} else {
		threads, err := s.store.ActiveThreads(ctx, req.UserID, 1)
		if err != nil {
			return nil, false, false, err
		}
		if len(threads) == 0 {
			return nil, false, false, nil
		}
		if threads[0].UserID != req.UserID {
			return nil, false, false, newError(c.ErrForbidden, "thread owner mismatch")
		}
		thread = threads[0]
	}
	items, err := s.store.MemoryByThread(ctx, req.UserID, thread.ID, nil, strategyCandidateLimitStep(limit))
	if err != nil {
		return nil, false, false, err
	}
	out := make([]candidate, 0, len(items))
	for i := range items {
		item := items[i]
		if item.UserID != req.UserID || !visibleForAI(item) {
			continue
		}
		if !filterHypothesis(cfg, &item, req) {
			continue
		}
		linked := threadMatch(&item, thread.ID)
		strength := 0.0
		if linked {
			// A stored thread link is an exact structural match.
			strength = 1
		}
		out = append(out, candidate{Item: &item, ThreadMatch: linked, Strategy: c.StratThread, Match: strength})
	}
	return out, false, len(out) > 0, nil
}

// anaphora resolves reference phrases ("same feeling as before") by boosting
// recent emotion/topic-matching items inside the configured window. If nothing
// matches, the reference is reported ambiguous so the Brain can clarify.
func (s *Service) anaphora(ctx context.Context, req c.RetrievalRequest, cfg Config, q lexicalQuery, limit int) ([]candidate, bool, bool, error) {
	since := s.now().AddDate(0, 0, -cfg.Anaphora.RecentWindowDays)
	kinds := []c.MemoryKind{c.MemEpisode, c.MemEvent, c.MemFact, c.MemHypothesis}
	items, err := s.store.MemoryCandidates(ctx, CandidateQuery{
		UserID: req.UserID, Terms: q.Tokens, Kinds: kinds, Since: since,
		Limit: strategyCandidateLimitStep(limit), RecentDays: cfg.Anaphora.RecentWindowDays,
	})
	if err != nil {
		return nil, false, false, err
	}
	out := make([]candidate, 0, len(items))
	for i := range items {
		item := items[i]
		if item.UserID != req.UserID || !visibleForAI(item) {
			continue
		}
		if !filterHypothesis(cfg, &item, req) {
			continue
		}
		if at := candidateTime(candidate{Item: &item}); at.Before(since) {
			continue // bounded to the configured recent window
		}
		emotionHit := q.Emotion != "" && (strings.Contains(strings.ToLower(item.Content), q.Emotion) || hasTag(item.Tags, q.Emotion))
		terms := overlappingTerms(item.Content, q.Tokens)
		if !emotionHit && len(terms) == 0 {
			continue
		}
		// An emotion/topic tag hit is the strongest anaphora signal; plain
		// lexical overlap is weaker and scored as such.
		strength := 0.6
		if emotionHit {
			strength = 0.8
		}
		out = append(out, candidate{Item: &item, ThreadMatch: inThreads(&item, req), Strategy: c.StratAnaphora, Match: strength})
	}
	return out, len(out) == 0, len(out) > 0, nil
}

// entity looks up stored entity metadata (§7.3 Memory.ByEntity). Only the
// literal role word the user typed is matched; aliases and names are [OPEN] in
// v1.2, so a capitalized token is never treated as a person.
func (s *Service) entity(ctx context.Context, req c.RetrievalRequest, cfg Config, q lexicalQuery) ([]candidate, bool, bool, error) {
	roles := q.Roles
	if req.Extra != nil && req.Extra.Kind == c.ContextEntity {
		roles = buildLexicalQuery(req.Extra.Query).Roles
		if len(roles) == 0 {
			// The Brain asked about an entity that maps to no stored role: say
			// so instead of guessing a person.
			return nil, true, false, nil
		}
	}
	if len(roles) == 0 {
		return nil, false, false, nil
	}
	var out []candidate
	for _, role := range roles {
		items, err := s.store.MemoryByEntity(ctx, EntityQuery{
			UserID: req.UserID, Role: role, Limit: strategyCandidateLimitStep(requestedLimit(req.MaxItems, cfg)),
		})
		if err != nil {
			return nil, false, false, err
		}
		for i := range items {
			item := items[i]
			if item.UserID != req.UserID || !visibleForAI(item) {
				continue
			}
			if !filterHypothesis(cfg, &item, req) {
				continue
			}
			// The stored entity role matched the role the user named exactly.
			out = append(out, candidate{Item: &item, ThreadMatch: inThreads(&item, req), Strategy: c.StratEntity, Match: 1})
		}
	}
	return out, len(out) == 0, len(out) > 0, nil
}

// hybrid is the default strategy: a bounded candidate pool reranked in Go by
// similarity + recency + importance + thread match (no pgvector, no ANN).
func (s *Service) hybrid(ctx context.Context, req c.RetrievalRequest, cfg Config, q lexicalQuery, limit int) ([]candidate, bool, bool, error) {
	query := CandidateQuery{
		UserID: req.UserID, Terms: q.Tokens, Kinds: retrievableKinds(),
		Statuses: retrievalStatuses(), Limit: strategyCandidateLimitStep(limit),
	}
	if req.FocusThread != nil {
		query.ThreadID = req.FocusThread
	}
	items, err := s.store.MemoryCandidates(ctx, query)
	if err != nil {
		return nil, false, false, err
	}
	// A second bounded pass by most-significant term keeps a long message from
	// starving a distinctive keyword inside one capped query.
	if terms := q.Tokens; len(terms) > 0 {
		head := terms
		if len(head) > 4 {
			head = head[:4]
		}
		extra, xerr := s.store.MemoryCandidates(ctx, CandidateQuery{
			UserID: req.UserID, Terms: head, Kinds: retrievableKinds(),
			Statuses: retrievalStatuses(), Limit: strategyCandidateLimitStep(limit), MinImport: 0.5,
		})
		if xerr == nil {
			items = mergeItems(items, extra)
		}
	}
	out := make([]candidate, 0, len(items))
	for i := range items {
		item := items[i]
		if item.UserID != req.UserID || !visibleForAI(item) {
			continue
		}
		if !filterHypothesis(cfg, &item, req) {
			continue
		}
		out = append(out, candidate{Item: &item, ThreadMatch: inThreads(&item, req), Strategy: c.StratHybrid})
	}
	return out, false, len(out) > 0, nil
}

// ---- similarity ----

// attachSimilarity embeds the current message once and cosine-ranks stored
// vectors in Go (§7.3 Memory.Hybrid). A missing provider, missing embeddings, a
// model/dimension mismatch or a provider error leaves similarity at zero: the
// service degrades ranking quality, never the truthfulness of the result.
func (s *Service) attachSimilarity(ctx context.Context, req c.RetrievalRequest, cfg Config, cands []candidate) []candidate {
	_ = cfg
	if len(cands) == 0 || s.embedding == nil {
		return cands
	}
	text := req.Turn.Text
	if len(text) > maxQueryChars {
		text = text[:maxQueryChars]
	}
	vector, model, err := s.embedding.Embed(ctx, text)
	if err != nil || len(vector) == 0 || model == "" {
		return cands
	}
	refs := make([]EmbeddingRef, 0, len(cands))
	for _, cand := range cands {
		switch {
		case cand.Item != nil:
			refs = append(refs, EmbeddingRef{Kind: cand.Item.Kind, ID: cand.Item.ID})
		case cand.Message != nil:
			refs = append(refs, EmbeddingRef{IsMsg: true, ID: cand.Message.ID})
		}
	}
	stored, err := s.store.Embeddings(ctx, req.UserID, refs)
	if err != nil {
		return cands
	}
	byRef := make(map[c.ID]float64, len(stored))
	for _, e := range stored {
		if e.ModelID != model || e.Dimensions != len(vector) || e.CodecVer != 1 {
			continue // a different model/dimension is not comparable
		}
		if sim, ok := cosine(vector, e.Vector); ok {
			byRef[e.RefID] = sim
		}
	}
	for i := range cands {
		if sim, ok := byRef[candidateID(cands[i])]; ok {
			cands[i].Similarity = sim
		}
	}
	return cands
}

// applyAnaphoraBoost adds the configured emotion/topic boost to recent items.
// The boost only lifts an item over the threshold; it never invents one.
func applyAnaphoraBoost(cfg Config, cands []candidate, now time.Time) {
	since := now.AddDate(0, 0, -cfg.Anaphora.RecentWindowDays)
	for i := range cands {
		if cands[i].Item == nil || cands[i].Strategy != c.StratAnaphora {
			continue
		}
		if candidateTime(cands[i]).Before(since) {
			continue
		}
		boost := 0.0
		if len(cands[i].Item.Tags) > 0 {
			boost += cfg.Anaphora.TopicBoost
		}
		if cands[i].ThreadMatch {
			boost += cfg.Anaphora.EmotionBoost
		}
		cands[i].Boost = boost
	}
}

// ---- helpers ----

// userLocation resolves the user's persisted time zone (tz_source=user_state).
// A missing record defaults to UTC; an unusable stored zone is reported so the
// temporal strategy can degrade instead of silently shifting every window.
func (s *Service) userLocation(ctx context.Context, user c.ID) (*time.Location, bool) {
	tz, err := s.store.UserTimeZone(ctx, user)
	if err != nil {
		return time.UTC, false
	}
	if strings.TrimSpace(tz) == "" {
		return time.UTC, true
	}
	loc, lerr := time.LoadLocation(tz)
	if lerr != nil {
		return time.UTC, false
	}
	return loc, true
}

// detectTemporalPhrase finds the longest temporal phrase in free text using the
// closed grammar above. Unknown expressions simply do not match.
func detectTemporalPhrase(text string, parser *TemporalParser, now time.Time, loc *time.Location) (temporalMatch, bool) {
	if loc == nil {
		loc = time.UTC
	}
	words := strings.Fields(strings.Map(func(r rune) rune {
		switch r {
		case '?', '!', ',', ';', ':', '"', '\n', '\t':
			return ' '
		}
		return r
	}, strings.ToLower(text)))
	for size := 4; size >= 1; size-- {
		for i := 0; i+size <= len(words); i++ {
			phrase := strings.Join(words[i:i+size], " ")
			if m, ok := parser.Parse(phrase, now, loc); ok {
				return m, true
			}
		}
	}
	return temporalMatch{}, false
}

func hopCount(req c.RetrievalRequest) int {
	if req.Extra != nil {
		return 1
	}
	return 0
}

func requestedLimit(max int, cfg Config) int {
	if max < 1 {
		max = cfg.MaxItems
	}
	if max > cfg.HardCap {
		max = cfg.HardCap
	}
	if max > maxSecondHopItems {
		max = maxSecondHopItems
	}
	return max
}

// contextBudget is the total token allowance for retrieved items. The contract
// expresses a per-item budget and a WorkingContext target; the effective
// allowance is the smaller of the two.
func contextBudget(cfg Config, limit int) int {
	perItem := limit * cfg.PerItemTok
	if cfg.ContextBudget > 0 && cfg.ContextBudget < perItem {
		return cfg.ContextBudget
	}
	return perItem
}

func strategyCandidateLimitStep(limit int) int {
	if limit < 1 {
		limit = 1
	}
	if limit*4 > strategyCandidateLimit {
		return strategyCandidateLimit
	}
	return limit * 4
}

func inThreads(item *c.MemoryItem, req c.RetrievalRequest) bool {
	if item == nil || req.FocusThread == nil {
		return false
	}
	return threadMatch(item, *req.FocusThread)
}

func threadMatch(item *c.MemoryItem, thread c.ID) bool {
	for _, id := range item.ThreadIDs {
		if id == thread {
			return true
		}
	}
	return false
}

func messageRoles(cfg Config) []c.Role {
	out := make([]c.Role, 0, len(cfg.Temporal.DefaultRoles))
	for _, r := range cfg.Temporal.DefaultRoles {
		switch c.Role(r) {
		case c.RoleUser, c.RoleAssistant:
			out = append(out, c.Role(r))
		}
	}
	if len(out) == 0 {
		out = []c.Role{c.RoleUser}
	}
	return out
}

func retrievableKinds() []c.MemoryKind {
	return []c.MemoryKind{c.MemFact, c.MemPreference, c.MemEpisode, c.MemGoal, c.MemEvent}
}

// retrievalStatuses excludes rejected/superseded/stale from default retrieval.
// Hypotheses use a separate status set and additionally require an
// active-thread match, per §3.4's anti-excess rule.
func retrievalStatuses() []c.MemoryStatus {
	return []c.MemoryStatus{c.MemActive, c.MemContested}
}

// visibleForAI applies the eligibility filters of §3.4/§7.7. It is a
// conservative engineering filter, not a new clinical policy: highly sensitive
// content is excluded from default retrieval, terminal statuses are excluded,
// and hypotheses must be proposed/supported.
func visibleForAI(item c.MemoryItem) bool {
	switch item.Status {
	case c.MemRejected, c.MemSuperseded, c.MemStale:
		return false
	}
	if item.Sensitivity == c.SensHighly {
		return false
	}
	if item.Kind == c.MemHypothesis && item.Status != c.MemProposed && item.Status != c.MemSupported {
		return false
	}
	if item.Kind == c.MemSafetyEvt && item.Sensitivity != c.SensNormal {
		return false
	}
	return true
}

func filterHypothesis(cfg Config, item *c.MemoryItem, req c.RetrievalRequest) bool {
	if item.Kind != c.MemHypothesis {
		return true
	}
	allowed := false
	for _, st := range cfg.Hypotheses.IncludeStatuses {
		if item.Status == st {
			allowed = true
			break
		}
	}
	if !allowed {
		return false
	}
	// "Hypotheses are retrieved only if status ∈ {supported, proposed} AND
	// relevant to the active thread" (§3.4).
	return !cfg.Hypotheses.RequireThreadMatch || inThreads(item, req)
}

func mergeItems(a, b []c.MemoryItem) []c.MemoryItem {
	seen := make(map[c.ID]bool, len(a)+len(b))
	out := make([]c.MemoryItem, 0, len(a)+len(b))
	for _, list := range [][]c.MemoryItem{a, b} {
		for _, item := range list {
			if seen[item.ID] {
				continue
			}
			seen[item.ID] = true
			out = append(out, item)
		}
	}
	return out
}

func hasTag(tags []string, want string) bool {
	for _, tag := range tags {
		if strings.EqualFold(strings.TrimSpace(tag), want) {
			return true
		}
	}
	return false
}

func degraded(hops int) c.RetrievalResult {
	return c.RetrievalResult{Meta: c.RetrievalMeta{Degraded: true, Hops: hops, Strategies: []string{}}}
}

// retrievalError is a sanitized internal error carrying a contract code. It
// never embeds user content or SQL text.
type retrievalError struct {
	code c.Code
	msg  string
}

func newError(code c.Code, msg string) error { return &retrievalError{code: code, msg: msg} }

func (e *retrievalError) Error() string { return e.msg }

// Code exposes the contract error code for the orchestrator/trace layer.
func (e *retrievalError) Code() c.Code { return e.code }

// fatalRetrievalError reports whether a strategy failure must abort the whole
// request. Ownership mismatches and contract violations are never degraded into
// a partial success.
func fatalRetrievalError(err error) bool {
	var re *retrievalError
	if errors.As(err, &re) {
		return re.code == c.ErrForbidden || re.code == c.ErrValidation
	}
	return false
}

// AsContractError converts any retrieval failure into the sanitized contract
// error shape used by callers that must not log internal detail.
func AsContractError(err error) *c.Error {
	var re *retrievalError
	if errors.As(err, &re) {
		return &c.Error{Code: re.code, Step: "retrieval", Degrades: true, Retryable: re.code == c.ErrRetrievalDegraded}
	}
	if err == nil {
		return nil
	}
	return &c.Error{Code: c.ErrRetrievalDegraded, Step: "retrieval", Degrades: true, Retryable: true}
}
