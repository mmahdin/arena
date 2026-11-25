package metrics

import "context"

type runtimeKey struct{}

// WithRuntime is a code-owned dependency-injection channel, never from an HTTP
// header or user-supplied field.
func WithRuntime(ctx context.Context, r *Runtime) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, runtimeKey{}, r)
}

func FromContext(ctx context.Context) *Runtime {
	if ctx == nil {
		return nil
	}
	m, _ := ctx.Value(runtimeKey{}).(*Runtime)
	return m
}
