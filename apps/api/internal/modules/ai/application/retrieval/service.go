// Package retrieval (application layer, WP-09) implements the deterministic
// Retrieval Service described in the Final Architecture §3.4 and Contract Pack
// §4.4/§7.3.
//
// Responsibilities
//   - choose strategies from cheap signals (temporal / thread / semantic /
//     anaphora / entity) and execute them through consumer-owned ports;
//   - integrate the embedding tier (WP-04 embedding port) for semantic recall;
//   - score, rank, deduplicate and budget the candidate set;
//   - report RetrievalMeta so the Brain can never imply memory it does not have.
//
// It uses no LLM. The only model use is the embedding lookup, which is cached
// per message/turn and reused by later phases.
package retrieval

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	domain "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/domain/retrieval"
)

// Sentinel errors. They are mapped to contract error codes by the orchestrator
// (WP-05) and never leak internal detail.
var (
	ErrUserRequired      = errors.New("retrieval requires a server-resolved user")
	ErrNoStore           = errors.New("retrieval store is not configured")
	ErrInvalidStrategy   = errors.New("unknown retrieval strategy")
	ErrEmbeddingDisabled = errors.New("embedding provider is not configured")
	ErrTimeout           = errors.New("retrieval exceeded its time budget")
)

// ---- Ports (consumer-owned; §6.1 Retriever, §7.3 read models) ----

// Store is the owner-scoped read port over SQL Server. Every method receives the
// server-resolved UserID and the adapter must filter on it in SQL; the port
// signature forces the parameter to exist so an unscoped query is a visible
// mistake rather than an omission. Implemented in adapters/mssql (WP-02 tables).
type Store interface {
	// RecentMemory returns at most limit owner-scoped items ordered by
	// last activity, used as the bounded candidate pool for hybrid ranking.
	// Excluded rows (soft-deleted) are never returned.
	RecentMemory(ctx context.Context, userID contract.ID, limit int) ([]contract.MemoryRecord, error)

	// MemoryByThread returns owner-scoped items linked to one thread.
	MemoryByThread(ctx context.Context, userID, threadID contract.ID, limit int) ([]contract.MemoryRecord, error)

	// MemoryByEntity returns owner-scoped items whose entities_json contains the
	// requested role/label. An empty query returns nothing.
	MemoryByEntity(ctx context.Context, userID contract.ID, q EntityFilter, limit int) ([]contract.MemoryRecord, error)

	// MemoryByIDs is the direct item read used by a second-hop `item` request and
	// by reference resolution.
	MemoryByIDs(ctx context.Context, userID contract.ID, ids []contract.ID) ([]contract.MemoryRecord, error)

	// MessageWindow is the exact/temporal read: owner-scoped raw messages inside
	// [from,to) for the given roles, newest last, capped by limit.
	MessageWindow(ctx context.Context, userID contract.ID, from, to time.Time, roles []contract.Role, terms []string, limit int) ([]contract.MessageRecord, error)

	// Embeddings returns stored vectors for the given refs. Missing rows are
	// omitted (a normal state: embedding is asynchronous). A user-scoped read of
	// another user's ref returns nothing.
	Embeddings(ctx context.Context, userID contract.ID, refs []contract.EmbeddingKey) (map[contract.EmbeddingKey]contract.Vector, error)

	// Preferences returns the always-on preference items (core set, §7.6).
	Preferences(ctx context.Context, userID contract.ID, limit int) ([]contract.MemoryRecord, error)

	// RecentMessages returns the most recent raw message texts of one
	// conversation, owner-scoped, newest first. It feeds the novelty filter
	// ("do not retrieve what the last N messages already say", §7.6) and never
	// returns rows of another user. Soft-deleted messages are excluded.
	RecentMessages(ctx context.Context, userID, conversationID contract.ID, limit int) ([]string, error)
}

// EntityFilter is the entity lookup filter for Store.MemoryByEntity.
type EntityFilter struct {
	Role  string
	Label string
}

// Embedder is the WP-04 embedding tier. The retriever only ever asks for one
// query embedding per request and tolerates failure (degraded hybrid).
type Embedder interface {
	Embed(ctx context.Context, texts []string) ([]contract.Vector, error)
	ModelID() string
}

// Clock is injected so recency and temporal windows are testable.
type Clock interface{ Now() time.Time }

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now().UTC() }

// Config is the constructor input.
type Config struct {
	Store    Store
	Embedder Embedder // optional: without it hybrid uses lexical similarity only
	Clock    Clock
	Tunables domain.Tunables
	// Location resolves relative time expressions to the user's timezone.
	// The assembler (WP-08) normally supplies a pre-resolved window; this is the
	// defensive path when it does not.
	Location *time.Location
	// WeekStartsOnMonday affects "last week" resolution.
	WeekStartsOnMonday bool
}

// Service is the WP-09 Retrieval Service.
type Service struct {
	store     Store
	embedder  Embedder
	clock     Clock
	tunables  domain.Tunables
	loc       *time.Location
	weekStart bool
}

// New builds the service. A nil store is a configuration error, surfaced on the
// first call (composition may wire retrieval before storage in tests).
func New(cfg Config) (*Service, error) {
	t, err := cfg.Tunables.Normalize()
	if err != nil {
		return nil, err
	}
	clock := cfg.Clock
	if clock == nil {
		clock = systemClock{}
	}
	loc := cfg.Location
	if loc == nil {
		loc = time.UTC
	}
	return &Service{store: cfg.Store, embedder: cfg.Embedder, clock: clock, tunables: t, loc: loc, weekStart: cfg.WeekStartsOnMonday}, nil
}

// Tunables exposes the normalized configuration (used by tests and traces).
func (s *Service) Tunables() domain.Tunables { return s.tunables }

// Retrieve implements the §6.1 Retriever port.
//
// Contract:
//   - never returns partial results as if they were complete: on timeout or store
//     failure the result is empty with Meta.Degraded=true (FB-002);
//   - always returns the strategies actually executed;
//   - enforces top-K, token budget, dedupe, sensitivity and soft-delete rules;
//   - respects ctx cancellation and the configured total timeout.
func (s *Service) Retrieve(ctx context.Context, req contract.RetrievalRequest) (contract.RetrievalResult, error) {
	start := s.clock.Now()
	res := contract.RetrievalResult{Items: []contract.MemoryView{}, Meta: contract.RetrievalMeta{Strategies: []string{}}}
	if !contract.ValidID(req.UserID) {
		return res, ErrUserRequired
	}
	if s.store == nil {
		res.Meta.Degraded = true
		return res, ErrNoStore
	}

	timeout := s.tunables.TotalTimeout
	if deadline, ok := ctx.Deadline(); ok {
		if remaining := time.Until(deadline); remaining < timeout && remaining > 0 {
			timeout = remaining
		}
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	hops := 0
	if req.Extra != nil {
		hops = 1
	}
	maxItems := req.MaxItems
	if maxItems <= 0 {
		maxItems = s.tunables.MaxItems
	}
	if cap := s.tunables.ItemCap(hops); maxItems > cap {
		maxItems = cap
	}

	text := strings.TrimSpace(req.Turn.Text)
	window := s.resolveWindow(req, text)
	plan := domain.Plan(domain.PlanInput{
		Requested:         strategyNames(req.Strategies),
		Text:              text,
		Window:            window,
		ForceTemporal:     req.Strategies != nil && containsStrategy(req.Strategies, contract.StratTemporal),
		ExplicitAnaphora:  req.Extra != nil && req.Extra.Kind == "topic",
		FocusThread:       idString(req.FocusThread),
		RelationContinues: true,
		Emotions:          nil,
		Hop:               hops,
		NeedsKind:         needsKind(req.Extra),
	})
	if req.Extra != nil && !domain.HopNeedsContextKind(req.Extra.Kind) {
		// An unknown hop kind would silently execute nothing; reject it instead.
		return res, fmt.Errorf("%w: needs_context kind %q", ErrInvalidStrategy, req.Extra.Kind)
	}
	if len(plan.Strategies) == 0 {
		return res, ErrInvalidStrategy
	}
	for _, requested := range req.Strategies {
		if !domain.IsKnownStrategy(string(requested)) {
			return res, fmt.Errorf("%w: %s", ErrInvalidStrategy, requested)
		}
	}

	ctxMeta := domain.ScoreContext{
		Now:      start,
		Anaphora: plan.Anaphora,
		Tunables: s.tunables,
	}
	if req.FocusThread != nil {
		ctxMeta.FocusThread = req.FocusThread.String()
	}

	query := text
	queryVector := s.queryEmbedding(runCtx, query)

	builder := newCandidateBuilder(req.UserID, s.tunables, s.clock.Now())
	var strategiesRun []string

	for _, strategy := range plan.Strategies {
		if err := runCtx.Err(); err != nil {
			res.Meta.Degraded = true
			break
		}
		err := s.runStrategy(runCtx, strategy, req, window, plan, queryVector, builder)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				res.Meta.Degraded = true
				break
			}
			// A failing strategy degrades the whole result: partial recall must
			// not be presented as complete (§6.4 retrieval_degraded row).
			res.Meta.Degraded = true
			break
		}
		strategiesRun = append(strategiesRun, strategy)
	}

	res.Meta.Strategies = strategiesRun
	res.Meta.Hops = hops

	if ranStrategy(strategiesRun, domain.StrategyTemporal) {
		// exact_match_found is the Brain's hard gate for recall answers (BR-016);
		// it is only meaningful when the temporal strategy actually executed.
		found := builder.temporalHits > 0
		res.Meta.ExactMatchFound = &found
	}

	if plan.Ambiguous {
		// An unresolved reference is exactly where hypotheses must not be guessed
		// at; drop them before budgeting so they cannot consume the item quota.
		builder.dropHypotheses()
	}

	ranked := domain.Rank(builder.candidates(), ctxMeta)
	ranked = domain.NoveltyFilter(ranked, s.recentTexts(runCtx, req), s.tunables.NoveltySimilarity, s.tunables.NoveltyRecentMessages)
	ranked = domain.Dedupe(ranked, s.tunables.NoveltySimilarity)
	ranked = domain.Budget(ranked, maxItems, s.tunables.TotalTokensBudget, s.tunables.PerItemTokens)

	for _, r := range ranked {
		res.Items = append(res.Items, toView(r, start))
	}
	res.Meta.AmbiguousReference = plan.Ambiguous

	if res.Meta.Degraded {
		// §6.4: on retrieval failure the caller must proceed state-only.
		res.Items = []contract.MemoryView{}
	}
	res.Took = s.clock.Now().Sub(start)
	return res, nil
}

// resolveWindow prefers the assembler-provided temporal reference (already
// timezone-correct) and falls back to in-service parsing.
func (s *Service) resolveWindow(req contract.RetrievalRequest, text string) *domain.Window {
	if len(req.Turn.TemporalRefs) > 0 {
		ref := req.Turn.TemporalRefs[0]
		if !ref.From.IsZero() && ref.To.After(ref.From) {
			return &domain.Window{Phrase: ref.Phrase, From: ref.From, To: ref.To}
		}
	}
	parser := domain.TemporalParser{Loc: s.loc, WeekStartsOnMonday: s.weekStart, Now: s.clock.Now()}
	if w, ok := parser.Parse(text); ok {
		return &w
	}
	return nil
}

// recentTexts loads the novelty window. A failure is not fatal: novelty is an
// anti-excess heuristic, so it is skipped rather than degrading real recall.
func (s *Service) recentTexts(ctx context.Context, req contract.RetrievalRequest) []string {
	if s.tunables.NoveltyRecentMessages <= 0 || !contract.ValidID(req.Turn.ConversationID) {
		return nil
	}
	noveltyCtx, cancel := context.WithTimeout(ctx, s.tunables.PerStrategyTimeout)
	defer cancel()
	texts, err := s.store.RecentMessages(noveltyCtx, req.UserID, req.Turn.ConversationID, s.tunables.NoveltyRecentMessages)
	if err != nil {
		return nil
	}
	return texts
}

func (s *Service) queryEmbedding(ctx context.Context, text string) *contract.Vector {
	if s.embedder == nil || strings.TrimSpace(text) == "" {
		return nil
	}
	embedCtx, cancel := context.WithTimeout(ctx, s.tunables.PerStrategyTimeout)
	defer cancel()
	vecs, err := s.embedder.Embed(embedCtx, []string{text})
	if err != nil || len(vecs) != 1 {
		return nil // degraded: hybrid falls back to lexical similarity
	}
	v := vecs[0]
	if len(v.Values) == 0 {
		return nil
	}
	if v.Dim == 0 {
		v.Dim = len(v.Values)
	}
	return &v
}
