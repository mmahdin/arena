package retrieval

import (
	"context"
	"errors"
	"time"

	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

// ErrNotVisible is returned by a Store when a requested record does not resolve
// to a row visible to the supplied owner. It deliberately does not distinguish
// "does not exist" from "belongs to another user": telling those apart would be
// an information leak. The service maps it to a forbidden result, while any
// other error is treated as an availability problem and degrades the result.
var ErrNotVisible = errors.New("record not visible to this owner")

// Store is the consumer-owned read port implemented by the SQL Server adapter.
// Every method is owner-scoped: the implementation must filter by the supplied
// user_id and must never accept an owner from a caller-provided record. Reads
// are visibility-aware (soft deletion hides, never erases) but no method here
// writes, deletes or changes AI-use consent.
//
// A returned error is a degraded signal, not permission to fabricate content.
type Store interface {
	// MessagesInWindow returns owner-visible raw messages in [from,to) with the
	// requested roles, oldest first, capped by limit. Used by temporal_exact.
	MessagesInWindow(ctx context.Context, q MessageWindowQuery) ([]MessageRow, error)
	// RecentThreadMessages returns the most recent owner-visible messages in a
	// conversation (oldest first) for in-thread quote candidates.
	RecentThreadMessages(ctx context.Context, userID, conversationID c.ID, limit int) ([]MessageRow, error)
	// MemoryByThread returns owner-visible memory items linked to a thread.
	MemoryByThread(ctx context.Context, userID, threadID c.ID, kinds []c.MemoryKind, limit int) ([]c.MemoryItem, error)
	// MemoryByEntity matches EntityRef{Role,Label} in stored entity metadata,
	// owner-scoped and bounded.
	MemoryByEntity(ctx context.Context, q EntityQuery) ([]c.MemoryItem, error)
	// MemoryCandidates returns a bounded owner-scoped candidate pool: recent,
	// important, thread-linked, or lexically matching items. No full scan and no
	// unbounded vector search: v1.2 forbids promising ANN behaviour on SQL Server.
	MemoryCandidates(ctx context.Context, q CandidateQuery) ([]c.MemoryItem, error)
	// MemoryByIDs loads specific owner-visible items (second hop by item id).
	MemoryByIDs(ctx context.Context, userID c.ID, ids []c.ID) ([]c.MemoryItem, error)
	// Embeddings returns stored vectors for the requested refs. Refs without a
	// stored vector are simply absent from the result: a missing embedding is a
	// normal condition, not an error.
	Embeddings(ctx context.Context, userID c.ID, refs []EmbeddingRef) ([]StoredEmbedding, error)
	// Thread returns one owner-visible thread, used for thread_match scoring.
	Thread(ctx context.Context, userID, threadID c.ID) (c.ThreadRecord, error)
	// ActiveThreads returns the bounded set of non-resolved owner threads.
	ActiveThreads(ctx context.Context, userID c.ID, limit int) ([]c.ThreadRecord, error)
	// UserTimeZone returns the persisted IANA zone name for temporal windows
	// ("tz_source: user_state"). An empty string means "not configured" and is
	// not the same as an invalid zone.
	UserTimeZone(ctx context.Context, userID c.ID) (string, error)
}

// MessageWindowQuery is the exact/temporal read contract (§7.3
// `Messages.Window`). Bounds are UTC and half-open.
type MessageWindowQuery struct {
	UserID c.ID
	From   time.Time
	To     time.Time
	Roles  []c.Role
	Limit  int
}

type MessageRow struct {
	ID             c.ID
	ConversationID c.ID
	TurnID         c.ID
	Role           c.Role
	Text           string
	CreatedAt      time.Time
	Lang           string
}

// EntityQuery is the §7.3 `Memory.ByEntity` contract. An empty Label matches any
// label for the role; person alias matching is explicitly not implemented.
type EntityQuery struct {
	UserID c.ID
	Role   string
	Label  string
	Kind   c.MemoryKind
	Limit  int
}

// CandidateQuery is the §7.3 bounded candidate-pool contract.
type CandidateQuery struct {
	UserID     c.ID
	Terms      []string
	ThreadID   *c.ID
	Kinds      []c.MemoryKind
	Statuses   []c.MemoryStatus
	Since      time.Time
	Limit      int
	MinImport  float32
	RecentDays int
}

type EmbeddingRef struct {
	Kind c.MemoryKind // informational only; ref_kind is message|memory
	IsMsg bool
	ID   c.ID
}

type StoredEmbedding struct {
	RefID      c.ID
	IsMessage  bool
	ModelID    string
	CodecVer   int
	Dimensions int
	Vector     []float32 // decoded and validated by the adapter
}

// vectorizer is the additive WP-04 embedding companion port, narrowed to what
// retrieval needs. A nil implementation disables semantic scoring: candidates
// then rank on recency/importance/thread match only and metadata records that
// similarity was unavailable.
type vectorizer interface {
	Embed(ctx context.Context, text string) (vector []float32, model string, err error)
}

// clock is injectable so recency, decay and temporal windows are deterministic
// in tests. Production wiring passes time.Now.
type clock func() time.Time
