package contextstate

import (
	"context"
	"errors"
	"testing"

	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

type testBundle struct{ paths []string }

func (b *testBundle) Reference() c.BundleRef { return c.BundleRef{Version: "kb@0.1.0"} }
func (b *testBundle) Artifact(_ context.Context, path string) ([]byte, error) {
	b.paths = append(b.paths, path)
	return nil, errors.New("synthetic bundle missing asset")
}
func TestProtocolGraphCannotTraverseBundlePaths(t *testing.T) {
	b := &testBundle{}
	for _, invalid := range []string{"../unsafe", "protocol/path", "UPPER", "a.b", ""} {
		if _, err := ProtocolGraph(context.Background(), b, invalid); err == nil {
			t.Fatalf("accepted unsafe protocol id %q", invalid)
		}
	}
	if len(b.paths) != 0 {
		t.Fatalf("requested arbitrary bundle paths: %+v", b.paths)
	}
}
