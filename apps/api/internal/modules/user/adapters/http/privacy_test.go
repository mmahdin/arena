package httpadapter

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/user/application"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/devsecurity"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/httpx"
)

type privacyFake struct{ values map[uuid.UUID]bool }

func (f *privacyFake) GetPrivacy(_ context.Context, id uuid.UUID) (application.PrivacyPreference, error) {
	v, ok := f.values[id]
	if !ok {
		return application.PrivacyPreference{}, httpx.ErrNotFound
	}
	return application.PrivacyPreference{ConversationSharingEnabled: v}, nil
}
func (f *privacyFake) SetPrivacy(_ context.Context, id uuid.UUID, enabled bool) (application.PrivacyPreference, error) {
	if _, ok := f.values[id]; !ok {
		return application.PrivacyPreference{}, httpx.ErrNotFound
	}
	f.values[id] = enabled
	return application.PrivacyPreference{ConversationSharingEnabled: enabled}, nil
}
func TestPrivacyEndpointIsSessionScoped(t *testing.T) {
	gin.SetMode(gin.TestMode)
	a, b := uuid.New(), uuid.New()
	fake := &privacyFake{values: map[uuid.UUID]bool{a: true, b: true}}
	signer, err := devsecurity.NewSigner(strings.Repeat("s", 40), false)
	if err != nil {
		t.Fatal(err)
	}
	sid := uuid.New()
	router := gin.New()
	group := router.Group("/api/v1", devsecurity.Guard(signer, validSession{user: a, sid: sid}))
	handler := NewPrivacyHandler(application.NewPrivacyService(fake))
	group.GET("/me/privacy-preferences", handler.Get)
	group.PUT("/me/privacy-preferences", handler.Set)
	cookie := &http.Cookie{Name: devsecurity.CookieName, Value: signer.Issue(a, sid, time.Now())}
	call := func(method, body string, auth bool) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, "/api/v1/me/privacy-preferences", bytes.NewBufferString(body))
		if method == "PUT" {
			req.Header.Set("Content-Type", "application/json")
		}
		if auth {
			req.AddCookie(cookie)
		}
		result := httptest.NewRecorder()
		router.ServeHTTP(result, req)
		return result
	}
	if got := call("GET", "", false).Code; got != 401 {
		t.Fatal(got)
	}
	if got := call("GET", "", true); got.Code != 200 || !strings.Contains(got.Body.String(), `"conversationSharingEnabled":true`) {
		t.Fatal(got)
	}
	for _, body := range []string{`{}`, `{"conversationSharingEnabled":null}`, `{"conversationSharingEnabled":"true"}`, `{"conversationSharingEnabled":false,"userId":"` + b.String() + `"}`} {
		if got := call("PUT", body, true).Code; got != 400 {
			t.Fatal(body, got)
		}
	}
	if got := call("PUT", `{"conversationSharingEnabled":false}`, true); got.Code != 200 {
		t.Fatal(got)
	}
	if fake.values[a] || !fake.values[b] {
		t.Fatal("cross-user privacy state changed")
	}
	if got := call("GET", "", true); !strings.Contains(got.Body.String(), `"conversationSharingEnabled":false`) {
		t.Fatal(got)
	}
	if got := call("PUT", `{"conversationSharingEnabled":true}`, true); got.Code != 200 {
		t.Fatal(got)
	}
	if !fake.values[a] {
		t.Fatal("toggle did not persist")
	}
}
