package trace

import (
	"context"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

// EmbeddingInfo is local Inspector correlation metadata, never provider usage.
// A reuse node may reference another Turn file without merging their graphs.
type EmbeddingInfo struct {
	ComputationID     string `json:"computation_id"`
	Provider          string `json:"provider"`
	Model             string `json:"model"`
	Dimensions        int    `json:"dimensions"`
	InputHash         string `json:"input_hash"`
	Consumer          string `json:"consumer,omitempty"`
	CacheOutcome      string `json:"cache_outcome"`
	Outcome           string `json:"outcome,omitempty"`
	OriginTraceID     string `json:"origin_trace_id,omitempty"`
	OriginNodeID      string `json:"origin_node_id,omitempty"`
	OriginTurnID      string `json:"origin_turn_id,omitempty"`
	WaitDurationNS    int64  `json:"wait_duration_ns,omitempty"`
	ComputeDurationNS int64  `json:"compute_duration_ns,omitempty"`
	ProviderAttempts  int    `json:"provider_attempts"`
	Reason            string `json:"reason,omitempty"`
}

func RecordEmbedding(ctx context.Context, info EmbeddingInfo) {
	sc := scopeFrom(ctx)
	if sc.node == nil {
		return
	}
	sc.session.mu.Lock()
	defer sc.session.mu.Unlock()
	sc.node.Embedding = &info
}

// InspectorIdentity returns only captured local IDs. Callers never invent
// execution/data edges across files; external reuse stays explicit metadata.
func InspectorIdentity(ctx context.Context) (traceID, nodeID, turnID string) {
	sc := scopeFrom(ctx)
	if sc.node == nil {
		return
	}
	sc.session.mu.Lock()
	defer sc.session.mu.Unlock()
	if c.ValidID(sc.session.doc.TurnID) {
		turnID = sc.session.doc.TurnID.String()
	}
	return sc.session.doc.TraceID, sc.node.ID, turnID
}

func EmbeddingAttempts(ctx context.Context) int {
	sc := scopeFrom(ctx)
	if sc.node == nil {
		return 0
	}
	sc.session.mu.Lock()
	defer sc.session.mu.Unlock()
	descendants := map[string]bool{sc.node.ID: true}
	count := 0
	for _, n := range sc.session.doc.Nodes {
		if descendants[n.ParentID] {
			descendants[n.ID] = true
			if n.Operation == "provider.Embed" {
				count++
			}
		}
	}
	return count
}

// StartAsync reuses the existing span/reference aggregation, while allowing a
// detached child to extend the envelope's observed end beyond its response root.
// It does not change the synchronous pipeline status or create another file.
func StartAsync(ctx context.Context, operation string, input func() any) (context.Context, *Span) {
	ctx, span := Start(ctx, operation, input)
	if span != nil {
		span.ownerRoot = true
	}
	return ctx, span
}
