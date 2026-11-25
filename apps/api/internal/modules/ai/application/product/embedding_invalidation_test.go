package product

import (
	"context"
	"errors"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"testing"
)

type privacyAccounts struct{ err error }

func (a *privacyAccounts) HideAccount(context.Context, c.ID) error { return a.err }

type embeddingInvalidator struct{ users []c.ID }

func (i *embeddingInvalidator) InvalidateEmbeddingUser(user c.ID) { i.users = append(i.users, user) }
func TestAccountHideInvalidatesEmbeddingCacheAfterCommit(t *testing.T) {
	account := &privacyAccounts{}
	cache := &embeddingInvalidator{}
	user := c.NewID()
	s, e := New(&productTurns{}, productMem{}, &productRepo{}, account, Options{EmbeddingInvalidator: cache})
	if e != nil {
		t.Fatal(e)
	}
	if e = s.HideAccount(context.Background(), user); e != nil {
		t.Fatal(e)
	}
	if len(cache.users) != 1 || cache.users[0] != user {
		t.Fatal("cache did not follow owner deletion", cache.users)
	}
	account.err = errors.New("synthetic deletion rollback")
	if e = s.HideAccount(context.Background(), c.NewID()); e == nil {
		t.Fatal("deletion failure hidden")
	}
	if len(cache.users) != 1 {
		t.Fatal("failed deletion treated as committed")
	}
}
