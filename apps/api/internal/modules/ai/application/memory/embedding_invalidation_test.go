package memory

import (
	"context"
	"errors"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"testing"
)

type invalidationRecorder struct{ users []c.ID }

func (r *invalidationRecorder) InvalidateEmbeddingUser(user c.ID) { r.users = append(r.users, user) }

type controlOnlyRepo struct {
	Repository
	err error
}

func (r *controlOnlyRepo) Control(context.Context, Meta, Control) (ControlResult, error) {
	return ControlResult{}, r.err
}
func TestCommittedMemoryRevocationInvalidatesOnlyOwnerEmbeddingCache(t *testing.T) {
	a := assets(t)
	_, meta, _, _ := fixture()
	meta.Bundle = a.Reference()
	repo := &controlOnlyRepo{}
	invalidator := &invalidationRecorder{}
	s, e := New(repo, a, Options{Policy: testPolicy{}, Cipher: invariantCipher{}, EmbeddingModel: "synthetic", Dimension: 2, EmbeddingInvalidator: invalidator})
	if e != nil {
		t.Fatal(e)
	}
	ctx := WithMeta(context.Background(), meta)
	if e = s.SetControls(ctx, meta.UserID, nil, map[string]bool{"memory": false}, nil); e != nil {
		t.Fatal(e)
	}
	if len(invalidator.users) != 1 || invalidator.users[0] != meta.UserID {
		t.Fatal("owner revocation did not invalidate cache", invalidator.users)
	}
	repo.err = errors.New("synthetic rollback")
	if e = s.SetControls(ctx, meta.UserID, nil, map[string]bool{"memory": false}, nil); e == nil {
		t.Fatal("control failure hidden")
	}
	if len(invalidator.users) != 1 {
		t.Fatal("failed transaction invalidated as committed")
	}
}
