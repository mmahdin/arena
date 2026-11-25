package oidc

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type Settings struct {
	Issuer          string
	ClientID        string
	ClientSecret    string
	RedirectURI     string
	RequireSMSProof bool // User realm: require Keycloak's verified SMS session note in the signed ID token
	RequireSID      bool // AUTH-08: bind app sessions to a Keycloak SSO sid
}

type discovery struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	JWKSURI               string `json:"jwks_uri"`
}

type Provider struct {
	cfg       Settings
	discovery discovery
	client    *http.Client
	mu        sync.Mutex
	keys      []publicKey
	keysUntil time.Time
}

func New(ctx context.Context, cfg Settings) (*Provider, error) {
	if cfg.ClientID == "" || cfg.ClientSecret == "" || cfg.RedirectURI == "" || cfg.Issuer == "" {
		return nil, errors.New("incomplete OIDC settings")
	}
	issuer, err := url.Parse(cfg.Issuer)
	if err != nil || issuer.Hostname() == "" || issuer.User != nil || issuer.RawQuery != "" || issuer.Fragment != "" || (issuer.Scheme != "https" && issuer.Scheme != "http") || strings.HasSuffix(cfg.Issuer, "/") {
		return nil, errors.New("invalid OIDC issuer")
	}
	u, err := url.Parse(cfg.RedirectURI)
	if err != nil || u.Hostname() == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.Fragment != "" || u.RawQuery != "" {
		return nil, errors.New("invalid OIDC redirect")
	}
	p := &Provider{cfg: cfg, client: &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
	var d discovery
	if err := p.fetchJSON(ctx, cfg.Issuer+"/.well-known/openid-configuration", &d); err != nil {
		return nil, errors.New("OIDC discovery failed")
	}
	if d.Issuer != cfg.Issuer || !sameOrigin(d.AuthorizationEndpoint, cfg.Issuer) || !sameOrigin(d.TokenEndpoint, cfg.Issuer) || !sameOrigin(d.JWKSURI, cfg.Issuer) {
		return nil, errors.New("OIDC discovery mismatch")
	}
	p.discovery = d
	return p, nil
}

func sameOrigin(a, b string) bool {
	u, err := url.Parse(a)
	if err != nil || u.User != nil || u.Fragment != "" || u.RawQuery != "" || u.Hostname() == "" {
		return false
	}
	v, err := url.Parse(b)
	if err != nil {
		return false
	}
	return u.Scheme == v.Scheme && u.Host == v.Host && u.Path != ""
}

func (p *Provider) AuthorizationURL(state, nonce, verifier string) string {
	u, _ := url.Parse(p.discovery.AuthorizationEndpoint)
	q := u.Query()
	q.Set("response_type", "code")
	q.Set("scope", "openid")
	q.Set("client_id", p.cfg.ClientID)
	q.Set("redirect_uri", p.cfg.RedirectURI)
	q.Set("state", state)
	q.Set("nonce", nonce)
	digest := sha256.Sum256([]byte(verifier))
	q.Set("code_challenge", base64.RawURLEncoding.EncodeToString(digest[:]))
	q.Set("code_challenge_method", "S256")
	u.RawQuery = q.Encode()
	return u.String()
}

func (p *Provider) Exchange(ctx context.Context, code, verifier, nonce string) (subject, sid string, err error) {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {p.cfg.RedirectURI},
		"code_verifier": {verifier},
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, p.discovery.TokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", "", errors.New("OIDC exchange unavailable")
	}
	request.SetBasicAuth(p.cfg.ClientID, p.cfg.ClientSecret)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := p.client.Do(request)
	if err != nil {
		return "", "", errors.New("OIDC exchange unavailable")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", "", errors.New("OIDC exchange denied")
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, 24*1024+1))
	if err != nil || len(raw) > 24*1024 {
		return "", "", errors.New("OIDC response invalid")
	}
	var token struct {
		IDToken   string `json:"id_token"`
		TokenType string `json:"token_type"`
	}
	if json.Unmarshal(raw, &token) != nil || token.IDToken == "" || !strings.EqualFold(token.TokenType, "Bearer") {
		return "", "", errors.New("OIDC response invalid")
	}
	keys, err := p.publicKeys(ctx, false)
	if err != nil {
		return "", "", err
	}
	subject, sid, err = VerifyIDToken(token.IDToken, p.cfg.Issuer, p.cfg.ClientID, nonce, keys, time.Now(), p.cfg.RequireSMSProof)
	if err == nil {
		if p.cfg.RequireSID && (sid == "" || len(sid) > 200) {
			return "", "", errors.New("OIDC session identifier unavailable")
		}
		return subject, sid, nil
	}
	// Refresh on a signing-key rotation, not by blindly trusting a new key URL.
	keys, fetchErr := p.publicKeys(ctx, true)
	if fetchErr != nil {
		return "", "", fetchErr
	}
	subject, sid, err = VerifyIDToken(token.IDToken, p.cfg.Issuer, p.cfg.ClientID, nonce, keys, time.Now(), p.cfg.RequireSMSProof)
	if err == nil && p.cfg.RequireSID && (sid == "" || len(sid) > 200) {
		return "", "", errors.New("OIDC session identifier unavailable")
	}
	return subject, sid, err
}

func (p *Provider) publicKeys(ctx context.Context, refresh bool) ([]publicKey, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !refresh && len(p.keys) > 0 && time.Now().Before(p.keysUntil) {
		return p.keys, nil
	}
	var jwksDoc jwks
	if err := p.fetchJSON(ctx, p.discovery.JWKSURI, &jwksDoc); err != nil || len(jwksDoc.Keys) == 0 {
		return nil, errors.New("OIDC signing keys unavailable")
	}
	p.keys = jwksDoc.Keys
	p.keysUntil = time.Now().Add(5 * time.Minute)
	return p.keys, nil
}

func (p *Provider) fetchJSON(ctx context.Context, endpoint string, dst interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.New("provider discovery unavailable")
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 32*1024+1))
	if err != nil || len(b) > 32*1024 {
		return errors.New("provider document too large")
	}
	return json.Unmarshal(b, dst)
}

// VerifyBackchannel authenticates a Keycloak logout_token using the same
// issuer-pinned JWKS and key-rotation strategy as ID-token verification.
func (p *Provider) VerifyBackchannel(ctx context.Context, token string) (LogoutEvent, error) {
	keys, err := p.publicKeys(ctx, false)
	if err != nil {
		return LogoutEvent{}, err
	}
	event, err := VerifyLogoutToken(token, p.cfg.Issuer, p.cfg.ClientID, keys, time.Now())
	if err == nil {
		return event, nil
	}
	// A rotation may introduce a new kid; the refresh source is fixed by
	// discovery, never supplied in a JWT header or browser parameter.
	keys, fetchErr := p.publicKeys(ctx, true)
	if fetchErr != nil {
		return LogoutEvent{}, fetchErr
	}
	return VerifyLogoutToken(token, p.cfg.Issuer, p.cfg.ClientID, keys, time.Now())
}
