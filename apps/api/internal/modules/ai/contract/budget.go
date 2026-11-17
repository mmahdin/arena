package contract

import "context"

type budgetKey struct{}

// WithBudget carries limits only; the orchestrator must enforce both these
// limits and the caller's context deadline. It does not extend that deadline.
func WithBudget(ctx context.Context, budget Budget) context.Context {
	return context.WithValue(ctx, budgetKey{}, budget)
}

func BudgetFromContext(ctx context.Context) (Budget, bool) {
	b, ok := ctx.Value(budgetKey{}).(Budget)
	return b, ok
}
