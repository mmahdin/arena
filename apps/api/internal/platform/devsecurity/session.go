package devsecurity

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/httpx"
)

const CookieName = "wellbeing_dev_session"
const SessionTTL = 12 * time.Hour

// Claims are signed, but valid claims alone are insufficient: sessions can be revoked.
type Claims struct {
	UserID    uuid.UUID
	SessionID uuid.UUID
}

type payload struct {
	UserID    string `json:"userId"`
	SessionID string `json:"sid"`
	Exp       int64  `json:"exp"`
}

type Signer struct {
	key    []byte
	secure bool
}

func NewSigner(secret string, secure bool) (*Signer, error) {
	if len(secret) < 32 {
		return nil, errors.New("secret must be >=32 bytes")
	}
	return &Signer{key: []byte(secret), secure: secure}, nil
}

func (s *Signer) Issue(userID, sessionID uuid.UUID, now time.Time) string {
	data, _ := json.Marshal(payload{UserID: userID.String(), SessionID: sessionID.String(), Exp: now.Add(SessionTTL).Unix()})
	plain := base64.RawURLEncoding.EncodeToString(data)
	mac := hmac.New(sha256.New, s.key)
	_, _ = mac.Write([]byte(plain))
	return plain + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (s *Signer) Verify(token string, now time.Time) (Claims, error) {
	chunks := strings.Split(token, ".")
	if len(chunks) != 2 {
		return Claims{}, errors.New("malformed token")
	}
	signature, err := base64.RawURLEncoding.DecodeString(chunks[1])
	if err != nil {
		return Claims{}, err
	}
	mac := hmac.New(sha256.New, s.key)
	_, _ = mac.Write([]byte(chunks[0]))
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return Claims{}, errors.New("invalid signature")
	}
	raw, err := base64.RawURLEncoding.DecodeString(chunks[0])
	if err != nil {
		return Claims{}, err
	}
	var p payload
	if err := json.Unmarshal(raw, &p); err != nil {
		return Claims{}, err
	}
	if p.Exp <= now.Unix() {
		return Claims{}, errors.New("expired")
	}
	uid, err := uuid.Parse(p.UserID)
	if err != nil || uid == uuid.Nil {
		return Claims{}, errors.New("invalid user")
	}
	sid, err := uuid.Parse(p.SessionID)
	if err != nil || sid == uuid.Nil {
		return Claims{}, errors.New("invalid session")
	}
	return Claims{UserID: uid, SessionID: sid}, nil
}

func (s *Signer) Cookie(token string) *http.Cookie {
	return &http.Cookie{Name: CookieName, Value: token, Path: "/api/v1", HttpOnly: true, Secure: s.secure, SameSite: http.SameSiteLaxMode, MaxAge: int(SessionTTL.Seconds())}
}
func (s *Signer) ExpireCookie() *http.Cookie {
	return &http.Cookie{Name: CookieName, Value: "", Path: "/api/v1", HttpOnly: true, Secure: s.secure, SameSite: http.SameSiteLaxMode, MaxAge: -1}
}

// SessionStore checks a session in persistent SQL Server storage, including user eligibility.
type SessionStore interface {
	IsActive(ctx context.Context, userID, sessionID uuid.UUID) (bool, error)
}

const userContextKey = "resolvedUserID"

func Guard(signer *Signer, sessions SessionStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		if signer == nil || sessions == nil {
			httpx.Failure(c, http.StatusUnauthorized, "SESSION_REQUIRED")
			c.Abort()
			return
		}
		cookie, err := c.Request.Cookie(CookieName)
		if err != nil {
			httpx.Failure(c, http.StatusUnauthorized, "SESSION_REQUIRED")
			c.Abort()
			return
		}
		claims, err := signer.Verify(cookie.Value, time.Now())
		if err != nil {
			httpx.Failure(c, http.StatusUnauthorized, "SESSION_INVALID")
			c.Abort()
			return
		}
		ok, err := sessions.IsActive(c.Request.Context(), claims.UserID, claims.SessionID)
		if err != nil {
			httpx.Failure(c, http.StatusServiceUnavailable, "SESSION_UNAVAILABLE")
			c.Abort()
			return
		}
		if !ok {
			httpx.Failure(c, http.StatusUnauthorized, "SESSION_INVALID")
			c.Abort()
			return
		}
		c.Set(userContextKey, claims.UserID)
		c.Next()
	}
}

func UserID(c *gin.Context) uuid.UUID {
	v, ok := c.Get(userContextKey)
	if !ok {
		return uuid.Nil
	}
	id, ok := v.(uuid.UUID)
	if !ok {
		return uuid.Nil
	}
	return id
}
