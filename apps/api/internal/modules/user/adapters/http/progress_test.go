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

type progressFake struct{ state map[uuid.UUID]int }

func (f *progressFake) GetStep(_ context.Context, id uuid.UUID) (int, bool, error) {
	step, ok := f.state[id]
	if !ok {
		return 0, false, httpx.ErrNotFound
	}
	return step, false, nil
}
func (f *progressFake) AdvanceStep(_ context.Context, id uuid.UUID, target int) (int, bool, error) {
	old, ok := f.state[id]
	if !ok {
		return 0, false, httpx.ErrNotFound
	}
	if target > old+1 {
		return old, false, application.ErrProgressConflict
	}
	if target > old {
		f.state[id] = target
	}
	return f.state[id], false, nil
}
func TestProgressEndpointCookieIsolation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	a, b := uuid.New(), uuid.New()
	fake := &progressFake{state: map[uuid.UUID]int{a: 0, b: 3}}
	signer, err := devsecurity.NewSigner(strings.Repeat("z", 40), false)
	if err != nil {
		t.Fatal(err)
	}
	sid := uuid.New()
	router := gin.New()
	g := router.Group("/api/v1", devsecurity.Guard(signer, validSession{user: a, sid: sid}))
	handler := NewProgressHandler(application.NewProgressService(fake))
	g.GET("/me/onboarding-progress", handler.Get)
	g.PUT("/me/onboarding-progress", handler.Advance)
	cookie := &http.Cookie{Name: devsecurity.CookieName, Value: signer.Issue(a, sid, time.Now())}
	req := func(method, body string, authorized bool) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, "/api/v1/me/onboarding-progress", bytes.NewBufferString(body))
		if method == "PUT" {
			r.Header.Set("Content-Type", "application/json")
		}
		if authorized {
			r.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		return w
	}
	if got := req("GET", "", false).Code; got != 401 {
		t.Fatal(got)
	}
	if got := req("GET", "", true); got.Code != 200 || !strings.Contains(got.Body.String(), `"furthestStep":0`) {
		t.Fatal(got)
	}
	for _, body := range []string{`{}`, `{"furthestStep":null}`, `{"furthestStep":26}`, `{"furthestStep":-1}`, `{"furthestStep":0,"answers":{"health":"no"}}`} {
		if got := req("PUT", body, true).Code; got != 400 {
			t.Fatal(body, got)
		}
	}
	if got := req("PUT", `{"furthestStep":5,"userId":"`+b.String()+`"}`, true).Code; got != 400 {
		t.Fatal(got)
	}
	if got := req("PUT", `{"furthestStep":5}`, true).Code; got != 409 {
		t.Fatal(got)
	}
	if got := req("PUT", `{"furthestStep":1}`, true).Code; got != 200 {
		t.Fatal(got)
	}
	if fake.state[a] != 1 || fake.state[b] != 3 {
		t.Fatal("other user changed", fake.state)
	}
}
