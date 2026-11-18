// Package memorycrypto provides authenticated owner/purpose-bound encryption.
// Key provisioning, custody, rotation and retention remain composition policy.
package memorycrypto

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
	"strings"

	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

var ErrCipher = errors.New("memory encryption unavailable")

type Keyring struct {
	active string
	keys   map[string]cipher.AEAD
}

func New(active string, keys map[string][]byte) (*Keyring, error) {
	k := &Keyring{active: active, keys: map[string]cipher.AEAD{}}
	for id, key := range keys {
		if id == "" || len(id) > 64 || strings.Contains(id, ":") || len(key) != 32 {
			return nil, ErrCipher
		}
		block, e := aes.NewCipher(append([]byte(nil), key...))
		if e != nil {
			return nil, ErrCipher
		}
		aead, e := cipher.NewGCM(block)
		if e != nil {
			return nil, ErrCipher
		}
		k.keys[id] = aead
	}
	if k.keys[active] == nil {
		return nil, ErrCipher
	}
	return k, nil
}
func (k *Keyring) Seal(ctx context.Context, user c.ID, purpose string, plain []byte) ([]byte, error) {
	if ctx.Err() != nil || !c.ValidID(user) || purpose == "" || len(plain) > 256<<10 {
		return nil, ErrCipher
	}
	aead := k.keys[k.active]
	nonce := make([]byte, aead.NonceSize())
	if _, e := io.ReadFull(rand.Reader, nonce); e != nil {
		return nil, ErrCipher
	}
	prefix := []byte("m1:" + k.active + ":")
	return append(prefix, aead.Seal(nonce, nonce, plain, []byte(user.String()+":"+purpose))...), nil
}
func (k *Keyring) Open(ctx context.Context, user c.ID, purpose string, raw []byte) ([]byte, error) {
	if ctx.Err() != nil || !c.ValidID(user) || len(raw) > 300<<10 {
		return nil, ErrCipher
	}
	parts := strings.SplitN(string(raw), ":", 3)
	if len(parts) != 3 || parts[0] != "m1" {
		return nil, ErrCipher
	}
	aead := k.keys[parts[1]]
	if aead == nil {
		return nil, ErrCipher
	}
	payload := []byte(parts[2])
	if len(payload) < aead.NonceSize() {
		return nil, ErrCipher
	}
	nonce := payload[:aead.NonceSize()]
	plain, e := aead.Open(nil, nonce, payload[aead.NonceSize():], []byte(user.String()+":"+purpose))
	if e != nil {
		return nil, ErrCipher
	}
	return plain, nil
}
