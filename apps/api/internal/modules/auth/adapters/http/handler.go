package httpadapter

import (
	"context"
	"crypto/hmac"
	"database/sql"
	"errors"
	"mime"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	auth "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/auth"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/auth/adapters/mssql"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/auth/oidc"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/config"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/httpx"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/principal"
)

type Handler struct {
	service  *auth.Service
	settings config.AuthConfig
}

// New discovers both fixed Keycloak issuers. Failure is fatal only if real
// authentication was explicitly enabled, avoiding a silent insecure fallback.
func New(ctx context.Context, db *sql.DB, cfg config.AuthConfig, localSynthetic bool) (*Handler, error) {
	if !cfg.Enabled || db == nil {
		return nil, errors.New("BFF requires enabled configuration and SQL Server")
	}
	// Require two independent gates: an operator-reviewed SMS-only realm
	// configuration AND a signed Keycloak ID-token session-note proof.
	// Either missing gate leaves the User surface unavailable on startup.
	if cfg.UserLoginEnabled && (!cfg.UserSMSFlowVerified || !cfg.UserIssuer.RequireSMSProof) {
		return nil, errors.New("User OIDC login requires reviewed SMS flow and signed OTP proof")
	}
	userProvider, err := oidc.New(ctx, cfg.UserIssuer)
	if err != nil {
		return nil, err
	}
	staffProvider, err := oidc.New(ctx, cfg.StaffIssuer)
	if err != nil {
		return nil, err
	}
	store := mssql.New(db, localSynthetic)
	service, err := auth.New(store, cfg.Key, map[auth.Kind]auth.Realm{
		auth.User:  {Issuer: cfg.UserIssuer.Issuer, Origin: cfg.UserOrigin, RedirectURI: cfg.UserIssuer.RedirectURI, Provider: userProvider},
		auth.Staff: {Issuer: cfg.StaffIssuer.Issuer, Origin: cfg.StaffOrigin, RedirectURI: cfg.StaffIssuer.RedirectURI, Provider: staffProvider},
	})
	if err != nil {
		return nil, err
	}
	return &Handler{service: service, settings: cfg}, nil
}

// NewForTests allows in-memory repositories and fake providers only in Go tests.
func NewForTests(service *auth.Service, cfg config.AuthConfig) *Handler {
	return &Handler{service: service, settings: cfg}
}

func (h *Handler) realm(kind auth.Kind) (origin string, enabled bool) {
	if kind == auth.Staff {
		return h.settings.StaffOrigin, h.settings.StaffLoginEnabled
	}
	return h.settings.UserOrigin, h.settings.UserLoginEnabled
}

func peerTrusted(remote string, proxies []netip.Prefix) bool {
	host, _, err := net.SplitHostPort(remote)
	if err != nil {
		return false
	}
	ip, err := netip.ParseAddr(host)
	if err != nil {
		return false
	}
	for _, p := range proxies {
		if p.Contains(ip.Unmap()) {
			return true
		}
	}
	return false
}

func (h *Handler) expectedRequest(c *gin.Context, kind auth.Kind) bool {
	origin, _ := h.realm(kind)
	parsed, err := url.Parse(origin)
	if err != nil || !strings.EqualFold(c.Request.Host, parsed.Host) {
		return false
	}
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	} else if peerTrusted(c.Request.RemoteAddr, h.settings.TrustedProxyCIDRs) {
		// Only a specifically trusted edge may assert the public scheme. The
		// upstream Host must already match our configured origin; forwarded
		// headers cannot be used to select a different identity host.
		proto := c.Request.Header.Values("X-Forwarded-Proto")
		forwardedHost := c.Request.Header.Values("X-Forwarded-Host")
		if len(proto) > 1 || (len(proto) == 1 && proto[0] != "http" && proto[0] != "https") ||
			len(forwardedHost) > 1 || (len(forwardedHost) == 1 && !strings.EqualFold(forwardedHost[0], parsed.Host)) {
			return false
		}
		if len(proto) == 1 {
			scheme = proto[0]
		}
	}
	return scheme == parsed.Scheme
}

func (h *Handler) enabledOrFail(c *gin.Context, kind auth.Kind) bool {
	_, enabled := h.realm(kind)
	if !enabled {
		httpx.Failure(c, http.StatusServiceUnavailable, "AUTH_NOT_CONFIGURED")
		return false
	}
	if !h.expectedRequest(c, kind) {
		httpx.Failure(c, http.StatusBadRequest, "AUTH_ORIGIN_MISMATCH")
		return false
	}
	return true
}

func (h *Handler) cookieName(kind auth.Kind, login bool) string {
	base := "theia_user_session"
	if kind == auth.Staff {
		base = "theia_staff_session"
	}
	if login {
		base = strings.TrimSuffix(base, "_session") + "_login"
	}
	origin, _ := h.realm(kind)
	if strings.HasPrefix(origin, "https://") {
		return "__Host-" + base
	}
	// __Host- cookies demand Secure. HTTP is allowed only on explicit local
	// loopback origins; use a different cookie name to avoid weakened prod flags.
	return base + "_local"
}
func (h *Handler) setCookie(c *gin.Context, kind auth.Kind, login bool, value string, maxAge int) {
	origin, _ := h.realm(kind)
	http.SetCookie(c.Writer, &http.Cookie{
		Name: h.cookieName(kind, login), Value: value, Path: "/", Secure: strings.HasPrefix(origin, "https://"),
		HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: maxAge,
	})
}

func (h *Handler) Login(kind auth.Kind) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !h.enabledOrFail(c, kind) {
			return
		}
		// Preserve deep links without open redirects. The return path is
		// validated once here and sealed in the one-time login transaction.
		// Callback query parameters cannot change the destination.
		query, err := url.ParseQuery(c.Request.URL.RawQuery)
		if err != nil || len(query["return_to"]) > 1 {
			httpx.Failure(c, http.StatusBadRequest, "AUTH_RETURN_PATH_INVALID")
			return
		}
		returnTo := "/"
		if len(query["return_to"]) == 1 {
			returnTo = query["return_to"][0]
		}
		if _, err := auth.SafeReturnPath(returnTo); err != nil {
			httpx.Failure(c, http.StatusBadRequest, "AUTH_RETURN_PATH_INVALID")
			return
		}
		state, destination, err := h.service.BeginWithReturn(c.Request.Context(), kind, returnTo)
		if err != nil {
			httpx.Failure(c, http.StatusServiceUnavailable, "AUTH_UNAVAILABLE")
			return
		}
		h.setCookie(c, kind, true, state, int(auth.LoginTTL.Seconds()))
		c.Header("Cache-Control", "no-store")
		c.Redirect(http.StatusFound, destination)
	}
}

func (h *Handler) Callback(kind auth.Kind) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !h.enabledOrFail(c, kind) {
			return
		}
		// A callback must contain exactly one state and one code. Taking
		// the first of repeated parameters can make reverse proxies and
		// identity providers disagree about which transaction was validated.
		query := c.Request.URL.Query()
		states, hasState := query["state"]
		codes, hasCode := query["code"]
		validQuery := hasState && hasCode && len(states) == 1 && len(codes) == 1 &&
			len(states[0]) > 0 && len(codes[0]) > 0
		state, code := "", ""
		if validQuery {
			state, code = states[0], codes[0]
		}
		cookie, err := c.Request.Cookie(h.cookieName(kind, true))
		h.setCookie(c, kind, true, "", -1)
		// Record only code-owned check names, never callback values or cookies.
		failedCheck := ""
		switch {
		case len(query["error"]) > 0:
			failedCheck = "callback_provider_error"
		case !validQuery:
			failedCheck = "callback_parameters_invalid"
		case err != nil || cookie.Value == "":
			failedCheck = "callback_login_cookie_missing"
		case len(state) > 100:
			failedCheck = "callback_state_invalid"
		case !hmac.Equal([]byte(cookie.Value), []byte(state)):
			failedCheck = "callback_state_mismatch"
		}
		if failedCheck != "" {
			c.Set("failed_check", failedCheck)
			httpx.Failure(c, http.StatusUnauthorized, "AUTH_CALLBACK_INVALID")
			return
		}
		token, s, returnTo, err := h.service.CompleteWithReturn(c.Request.Context(), kind, state, code)
		if err != nil {
			if errors.Is(err, auth.ErrInactive) {
				code := "IDENTITY_NOT_AUTHORIZED"
				if kind == auth.Staff {
					code = "STAFF_NOT_AUTHORIZED"
				}
				httpx.Failure(c, http.StatusForbidden, code)
				return
			}
			if errors.Is(err, auth.ErrInvalid) || errors.Is(err, auth.ErrNotFound) {
				httpx.Failure(c, http.StatusUnauthorized, "AUTH_CALLBACK_INVALID")
				return
			}
			httpx.Failure(c, http.StatusServiceUnavailable, "AUTH_UNAVAILABLE")
			return
		}
		// The session cookie never contains OIDC ID/access/refresh tokens.
		h.setCookie(c, kind, false, token, int(time.Until(s.ExpiresAt).Seconds()))
		c.Header("Cache-Control", "no-store")
		c.Redirect(http.StatusSeeOther, returnTo)
	}
}

func (h *Handler) load(c *gin.Context, kind auth.Kind) (auth.Session, string, bool) {
	if !h.enabledOrFail(c, kind) {
		return auth.Session{}, "", false
	}
	cookie, err := c.Request.Cookie(h.cookieName(kind, false))
	if err != nil || cookie.Value == "" {
		httpx.Failure(c, http.StatusUnauthorized, "SESSION_REQUIRED")
		return auth.Session{}, "", false
	}
	session, err := h.service.Authenticate(c.Request.Context(), kind, cookie.Value)
	if err != nil {
		if errors.Is(err, auth.ErrInvalid) || errors.Is(err, auth.ErrNotFound) {
			httpx.Failure(c, http.StatusUnauthorized, "SESSION_INVALID")
		} else {
			httpx.Failure(c, http.StatusServiceUnavailable, "SESSION_UNAVAILABLE")
		}
		return auth.Session{}, "", false
	}
	if kind != session.Kind {
		httpx.Failure(c, http.StatusUnauthorized, "SESSION_INVALID")
		return auth.Session{}, "", false
	}
	return session, cookie.Value, true
}

func (h *Handler) csrf(c *gin.Context, kind auth.Kind, token string) bool {
	origin, _ := h.realm(kind)
	if sent := c.GetHeader("Origin"); sent != "" && sent != origin {
		httpx.Failure(c, http.StatusForbidden, "CSRF_ORIGIN_DENIED")
		return false
	}
	if site := c.GetHeader("Sec-Fetch-Site"); site == "cross-site" {
		httpx.Failure(c, http.StatusForbidden, "CSRF_ORIGIN_DENIED")
		return false
	}
	if !h.service.VerifyCSRF(token, c.GetHeader("X-CSRF-Token")) {
		httpx.Failure(c, http.StatusForbidden, "CSRF_TOKEN_INVALID")
		return false
	}
	return true
}
func unsafe(method string) bool {
	return method != http.MethodGet && method != http.MethodHead && method != http.MethodOptions
}

func (h *Handler) Session(kind auth.Kind) gin.HandlerFunc {
	return func(c *gin.Context) {
		s, token, ok := h.load(c, kind)
		if !ok {
			return
		}
		response := gin.H{"authenticated": true, "principalType": string(kind), "principalId": s.PrincipalID.String(), "expiresAt": s.ExpiresAt.UTC().Format(time.RFC3339), "csrfToken": h.service.CSRFToken(token)}
		if kind == auth.Staff {
			response["role"] = "admin"
		}
		c.Header("Cache-Control", "no-store")
		c.Header("Vary", "Cookie")
		c.JSON(http.StatusOK, response)
	}
}
func (h *Handler) Logout(kind auth.Kind) gin.HandlerFunc {
	return func(c *gin.Context) {
		_, token, ok := h.load(c, kind)
		if !ok {
			return
		}
		if !h.csrf(c, kind, token) {
			return
		}
		if err := h.service.Revoke(c.Request.Context(), kind, token); err != nil {
			httpx.Failure(c, http.StatusServiceUnavailable, "SESSION_UNAVAILABLE")
			return
		}
		h.setCookie(c, kind, false, "", -1)
		// Local app session ends here. Keycloak SSO logout/backchannel revocation
		// is intentionally a separate AUTH-08 integration/release gate.
		c.Status(http.StatusNoContent)
	}
}

// keycloakLogoutURL uses only fixed, deployment-validated configuration.
// The ID token is not persisted, so Keycloak may require browser confirmation.
// The frontend must navigate the top-level browser to this URL after receiving
// the response; a server-side HTTP request cannot end a browser SSO cookie.
func (h *Handler) keycloakLogoutURL(kind auth.Kind) (string, error) {
	origin, _ := h.realm(kind)
	issuer := h.settings.UserIssuer
	if kind == auth.Staff {
		issuer = h.settings.StaffIssuer
	}
	if issuer.ClientID == "" || issuer.Issuer == "" {
		return "", auth.ErrInvalid
	}
	parsed, err := url.Parse(issuer.Issuer)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" || parsed.RawQuery != "" ||
		(parsed.Scheme != "https" && parsed.Scheme != "http") {
		return "", auth.ErrInvalid
	}
	target, err := url.Parse(origin)
	if err != nil || target.Host == "" || (target.Scheme != "https" && target.Scheme != "http") ||
		target.User != nil || target.RawQuery != "" || target.Fragment != "" || target.Path != "" {
		return "", auth.ErrInvalid
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/protocol/openid-connect/logout"
	q := url.Values{}
	q.Set("client_id", issuer.ClientID)
	q.Set("post_logout_redirect_uri", origin+"/")
	parsed.RawQuery = q.Encode()
	return parsed.String(), nil
}

// LogoutWithSSO invalidates the Theia session(s) before returning a fixed
// Keycloak end-session URL. Browser navigation to the URL is still required.
// all=true affects all *Theia* sessions, not remote Keycloak SSO sessions.
func (h *Handler) LogoutWithSSO(kind auth.Kind, all bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		_, token, ok := h.load(c, kind)
		if !ok {
			return
		}
		if !h.csrf(c, kind, token) {
			return
		}
		destination, err := h.keycloakLogoutURL(kind)
		if err != nil {
			httpx.Failure(c, http.StatusServiceUnavailable, "AUTH_UNAVAILABLE")
			return
		}
		if all {
			err = h.service.RevokeAll(c.Request.Context(), kind, token)
		} else {
			err = h.service.Revoke(c.Request.Context(), kind, token)
		}
		if err != nil {
			httpx.Failure(c, http.StatusServiceUnavailable, "SESSION_UNAVAILABLE")
			return
		}
		h.setCookie(c, kind, false, "", -1)
		c.Header("Cache-Control", "no-store")
		c.Header("Referrer-Policy", "no-referrer")
		scope := "current"
		if all {
			scope = "all"
		}
		c.JSON(http.StatusOK, gin.H{"logoutUrl": destination, "theiaSessionsRevoked": scope,
			"keycloakSSOStatus": "browser_action_required"})
	}
}

// Backchannel is a server-to-server, signed-JWT endpoint. It deliberately
// does not use browser cookies, Host-based identity or CSRF; Keycloak reaches
// it on the private Docker/Kubernetes service address. Validation is always
// fixed to this realm and the Keycloak signing keys.
func (h *Handler) Backchannel(kind auth.Kind) gin.HandlerFunc {
	return func(c *gin.Context) {
		ct, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
		if err != nil || ct != "application/x-www-form-urlencoded" {
			httpx.Failure(c, http.StatusUnsupportedMediaType, "AUTH_BACKCHANNEL_INVALID")
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 16*1024)
		if c.Request.ParseForm() != nil || len(c.Request.PostForm) != 1 ||
			len(c.Request.PostForm["logout_token"]) != 1 ||
			len(c.Request.PostForm.Get("logout_token")) == 0 ||
			len(c.Request.PostForm.Get("logout_token")) > 16*1024 {
			httpx.Failure(c, http.StatusBadRequest, "AUTH_BACKCHANNEL_INVALID")
			return
		}
		_, err = h.service.BackchannelLogout(c.Request.Context(), kind, c.Request.PostForm.Get("logout_token"))
		if err != nil {
			if errors.Is(err, auth.ErrInvalid) {
				httpx.Failure(c, http.StatusBadRequest, "AUTH_BACKCHANNEL_INVALID")
				return
			}
			httpx.Failure(c, http.StatusServiceUnavailable, "AUTH_BACKCHANNEL_UNAVAILABLE")
			return
		}
		c.Header("Cache-Control", "no-store")
		c.Status(http.StatusOK) // Replays are successful no-ops.
	}
}

func (h *Handler) guard(kind auth.Kind) gin.HandlerFunc {
	return func(c *gin.Context) {
		s, token, ok := h.load(c, kind)
		if !ok {
			c.Abort()
			return
		}
		if unsafe(c.Request.Method) && !h.csrf(c, kind, token) {
			c.Abort()
			return
		}
		if kind == auth.User {
			principal.SetUserID(c, s.PrincipalID)
		} else {
			c.Set("staffPrincipalID", s.PrincipalID)
			c.Set("staffRole", "admin")
			// Immutable projection from a verified, active SQL-backed Staff session.
			// Admin handlers must never accept caller-supplied principal metadata.
			c.Set("verifiedStaffSession", VerifiedStaffPrincipal{ID: s.PrincipalID, ExpiresAt: s.ExpiresAt})
		}
		c.Next()
	}
}
func (h *Handler) RequireUser() gin.HandlerFunc  { return h.guard(auth.User) }
func (h *Handler) RequireStaff() gin.HandlerFunc { return h.guard(auth.Staff) }

// API boundary can retrieve the verified staff ID, never from JSON/headers.
func StaffID(c *gin.Context) uuid.UUID {
	v, ok := c.Get("staffPrincipalID")
	if !ok {
		return uuid.Nil
	}
	id, _ := v.(uuid.UUID)
	return id
}

// VerifiedStaffPrincipal exposes only the minimal authenticated Staff context,
// without leaking OIDC subject, Keycloak session ID or session-token hashes.
type VerifiedStaffPrincipal struct {
	ID        uuid.UUID
	ExpiresAt time.Time
}

// VerifiedStaff is present only after RequireStaff validates an active Staff
// session against SQL Server. No request header/body can populate this value.
func VerifiedStaff(c *gin.Context) (VerifiedStaffPrincipal, bool) {
	v, ok := c.Get("verifiedStaffSession")
	if !ok {
		return VerifiedStaffPrincipal{}, false
	}
	p, ok := v.(VerifiedStaffPrincipal)
	if !ok || p.ID == uuid.Nil || !p.ExpiresAt.After(time.Now()) {
		return VerifiedStaffPrincipal{}, false
	}
	return p, true
}
