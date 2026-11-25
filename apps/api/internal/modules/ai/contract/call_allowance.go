package contract

import (
	"context"
	"sync/atomic"
)

type callAllowanceKey struct{}
type callAllowance struct {
	used  atomic.Int64
	limit int64
}

// WithCallAllowance bounds reasoning/generation provider attempts across a
// single admitted turn, including a second Brain hop and regeneration. This
// complements Budget.MaxLLMCalls, which WP-04 also enforces per invocation.
// Classifier and embedding tiers have their own budgets. No wire type changes.
func WithCallAllowance(ctx context.Context, max int) context.Context {
	if max < 1 {
		return ctx
	}
	return context.WithValue(ctx, callAllowanceKey{}, &callAllowance{limit: int64(max)})
}

// ReserveLLMCall is used by WP-04 immediately before an eligible attempt.
// The shared counter is safe across derived contexts and concurrent calls.
func ReserveLLMCall(ctx context.Context) bool {
	a, ok := ctx.Value(callAllowanceKey{}).(*callAllowance)
	if !ok {
		return true
	}
	for {
		used := a.used.Load()
		if used >= a.limit {
			return false
		}
		if a.used.CompareAndSwap(used, used+1) {
			return true
		}
	}
}
