package auth

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"
)

const LoginTTL = 5 * time.Minute
const UserIdle = 7 * 24 * time.Hour
const UserAbsolute = 30 * 24 * time.Hour
const StaffIdle = 30 * time.Minute
const StaffAbsolute = 12 * time.Hour

// Provider is an externally verified OIDC realm client, not a password verifier.
type Provider interface {
	AuthorizationURL(state, nonce, verifier string) string
	Exchange(ctx context.Context, code, verifier, nonce string) (subject, sid string, err error)
}

type Realm struct {
	Issuer      string
	RedirectURI string
	Origin      string
	Provider    Provider
}

type Service struct {
	store  Store
	key    []byte
	realms map[Kind]Realm
	now    func() time.Time
}

func New(store Store, key []byte, realms map[Kind]Realm) (*Service, error) {
	if store == nil || len(key) != 32 {
		return nil, errors.New("auth store and 256-bit key required")
	}
	for _, kind := range []Kind{User, Staff} {
		r, ok := realms[kind]
		if !ok || r.Provider == nil || r.Issuer == "" || r.Origin == "" || r.RedirectURI == "" {
			return nil, errors.New("both OIDC realms required")
		}
	}
	return &Service{store: store, key: append([]byte(nil), key...), realms: realms, now: time.Now}, nil
}

func random() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func hash(s string) []byte { sum := sha256.Sum256([]byte(s)); return sum[:] }

func (s *Service) seal(plain []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plain, []byte("theia-oidc-login-v1")), nil
}
func (s *Service) open(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	n := gcm.NonceSize()
	if len(ciphertext) <= n {
		return nil, ErrInvalid
	}
	out, err := gcm.Open(nil, ciphertext[:n], ciphertext[n:], []byte("theia-oidc-login-v1"))
	if err != nil {
		return nil, ErrInvalid
	}
	return out, nil
}

type loginSecrets struct {
	Verifier string `json:"verifier"`
	Nonce    string `json:"nonce"`
	ReturnTo string `json:"returnTo,omitempty"`
}

func (s *Service) Begin(ctx context.Context, kind Kind) (state, authorizationURL string, err error) {
	return s.BeginWithReturn(ctx, kind, "/")
}

// BeginWithReturn seals a validated, same-origin post-login path alongside
// PKCE and nonce. Caller-supplied callback parameters cannot override it.
func (s *Service) BeginWithReturn(ctx context.Context, kind Kind, returnTo string) (state, authorizationURL string, err error) {
	returnTo, err = SafeReturnPath(returnTo)
	if err != nil {
		return "", "", err
	}
	realm, ok := s.realms[kind]
	if !ok {
		return "", "", ErrInvalid
	}
	state, err = random()
	if err != nil {
		return "", "", err
	}
	nonce, err := random()
	if err != nil {
		return "", "", err
	}
	verifier, err := random()
	if err != nil {
		return "", "", err
	}
	raw, err := json.Marshal(loginSecrets{Verifier: verifier, Nonce: nonce, ReturnTo: returnTo})
	if err != nil {
		return "", "", err
	}
	sealed, err := s.seal(raw)
	if err != nil {
		return "", "", err
	}
	err = s.store.SaveLogin(ctx, LoginRecord{StateHash: hash(state), Kind: kind, ExpectedOrigin: realm.Origin, RedirectURI: realm.RedirectURI, SealedPayload: sealed, ExpiresAt: s.now().Add(LoginTTL)})
	if err != nil {
		return "", "", err
	}
	return state, realm.Provider.AuthorizationURL(state, nonce, verifier), nil
}

func (s *Service) Complete(ctx context.Context, kind Kind, state, code string) (sessionToken string, session Session, err error) {
	sessionToken, session, _, err = s.CompleteWithReturn(ctx, kind, state, code)
	return sessionToken, session, err
}

// CompleteWithReturn consumes the login record once and returns only the path
// sealed by BeginWithReturn. Older in-flight records default to app root.
func (s *Service) CompleteWithReturn(ctx context.Context, kind Kind, state, code string) (sessionToken string, session Session, returnTo string, err error) {
	realm, ok := s.realms[kind]
	if !ok || len(state) < 32 || len(state) > 100 || len(code) == 0 || len(code) > 4096 {
		return "", Session{}, "", ErrInvalid
	}
	record, err := s.store.ConsumeLogin(ctx, hash(state))
	if err != nil {
		return "", Session{}, "", err
	}
	if record.Kind != kind || record.ExpectedOrigin != realm.Origin || record.RedirectURI != realm.RedirectURI || !record.ExpiresAt.After(s.now()) {
		return "", Session{}, "", ErrInvalid
	}
	payload, err := s.open(record.SealedPayload)
	if err != nil {
		return "", Session{}, "", err
	}
	var secrets loginSecrets
	if json.Unmarshal(payload, &secrets) != nil || secrets.Nonce == "" || secrets.Verifier == "" {
		return "", Session{}, "", ErrInvalid
	}
	returnTo, err = SafeReturnPath(secrets.ReturnTo)
	if err != nil {
		return "", Session{}, "", ErrInvalid
	}
	sub, sid, err := realm.Provider.Exchange(ctx, code, secrets.Verifier, secrets.Nonce)
	if err != nil {
		return "", Session{}, "", ErrInvalid
	}
	if len(sub) == 0 || len(sub) > 200 {
		return "", Session{}, "", ErrInvalid
	}
	pid, err := s.store.ResolveIdentity(ctx, kind, realm.Issuer, sub)
	if err != nil {
		return "", Session{}, "", err
	}
	token, err := random()
	if err != nil {
		return "", Session{}, "", err
	}
	now := s.now()
	idle, absolute := UserIdle, UserAbsolute
	if kind == Staff {
		idle, absolute = StaffIdle, StaffAbsolute
	}
	session = Session{TokenHash: hash(token), Kind: kind, PrincipalID: pid, Issuer: realm.Issuer, Subject: sub, KeycloakSID: sid, CreatedAt: now, LoginStartedAt: record.ExpiresAt.Add(-LoginTTL), IdleExpiresAt: now.Add(idle), ExpiresAt: now.Add(absolute), IdleTTL: idle}
	if err = s.store.SaveSession(ctx, session); err != nil {
		return "", Session{}, "", err
	}
	// Store.SaveSession atomically records the successful login audit event.
	return token, session, returnTo, nil
}

func (s *Service) Authenticate(ctx context.Context, kind Kind, token string) (Session, error) {
	if len(token) != 43 {
		return Session{}, ErrInvalid
	}
	// Verify session in SQL on every request, including disabled/deleted principals.
	return s.store.ActiveSession(ctx, kind, hash(token))
}
func (s *Service) Revoke(ctx context.Context, kind Kind, token string) error {
	if len(token) != 43 {
		return ErrInvalid
	}
	return s.store.RevokeSession(ctx, kind, hash(token))
}
func (s *Service) CSRFToken(token string) string {
	m := hmac.New(sha256.New, s.key)
	_, _ = m.Write([]byte("theia-session-csrf-v1:" + token))
	return base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}
func (s *Service) VerifyCSRF(token, submitted string) bool {
	if submitted == "" || len(submitted) > 128 {
		return false
	}
	expected := s.CSRFToken(token)
	return hmac.Equal([]byte(expected), []byte(submitted))
}
