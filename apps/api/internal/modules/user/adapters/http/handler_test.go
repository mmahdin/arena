package httpadapter

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/user/application"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/user/domain"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/devsecurity"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/httpx"
)

type fakeProfileDB struct{ profiles map[uuid.UUID]domain.Profile }

func (r *fakeProfileDB) Get(_ context.Context, uid uuid.UUID) (domain.Profile, error) {
	p, ok := r.profiles[uid]
	if !ok {
		return domain.Profile{}, httpx.ErrNotFound
	}
	return p, nil
}
func (r *fakeProfileDB) Save(_ context.Context, p domain.Profile) error {
	if p.OnboardingAnswers == nil {
		p.OnboardingAnswers = r.profiles[p.UserID].OnboardingAnswers
	}
	r.profiles[p.UserID] = p
	return nil
}

func TestExplicitOnboardingAnswersAreScopedValidatedAndPreserved(t *testing.T) {
	a, b := uuid.New(), uuid.New()
	db := &fakeProfileDB{profiles: map[uuid.UUID]domain.Profile{
		a: {UserID: a, ConversationStyle: domain.Balanced},
		b: {UserID: b, ConversationStyle: domain.Balanced},
	}}
	// Use the same signed session guard as the production composition.
	signer, _ := devsecurity.NewSigner("0123456789abcdef0123456789abcdef", false)
	sid := uuid.New()
	r := gin.New()
	g := r.Group("/api/v1", devsecurity.Guard(signer, validSession{user: a, sid: sid}))
	h := New(application.New(db))
	g.GET("/me", h.Get)
	g.PATCH("/me/profile", h.Update)
	call := func(method, path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{Name: devsecurity.CookieName, Value: signer.Issue(a, sid, time.Now())})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}
	for _, answers := range []string{`{"mood":5}`, `{"mood":1.5}`, `{"identity":"invalid"}`, `{"userId":"other"}`, `{"concerns":["bad\ncommand"]}`} {
		w := call("PATCH", "/api/v1/me/profile", `{"displayName":"Synthetic","conversationStyle":"balanced","onboardingAnswers":`+answers+`}`)
		if w.Code != 400 || db.profiles[a].OnboardingCompletedAt != nil {
			t.Fatal("invalid answers changed the profile")
		}
	}
	w := call("PATCH", "/api/v1/me/profile", `{"displayName":"Synthetic","conversationStyle":"balanced","onboardingAnswers":{"mood":0,"identity":"تمایل ندارم بگم","concerns":["کار یا درس"],"focus":["بهتر بخوابم"]}}`)
	if w.Code != 200 || db.profiles[a].OnboardingAnswers == nil || *db.profiles[a].OnboardingAnswers.Mood != 0 {
		t.Fatal("explicit answers were not saved")
	}
	if db.profiles[b].OnboardingAnswers != nil || db.profiles[b].OnboardingCompletedAt != nil {
		t.Fatal("another user's profile was modified")
	}
	w = call("PATCH", "/api/v1/me/profile", `{"displayName":"Updated","conversationStyle":"listening"}`)
	if w.Code != 200 || db.profiles[a].OnboardingAnswers.Focus[0] != "بهتر بخوابم" {
		t.Fatal("ordinary profile edit erased onboarding answers")
	}
	w = call("GET", "/api/v1/me", "")
	var p domain.Profile
	if json.Unmarshal(w.Body.Bytes(), &p) != nil || p.OnboardingAnswers == nil || p.OnboardingAnswers.Concerns[0] != "کار یا درس" {
		t.Fatal("profile did not return saved answers")
	}
	w = call("PATCH", "/api/v1/me/profile", `{"displayName":"Updated","conversationStyle":"balanced","onboardingAnswers":{}}`)
	if w.Code != 200 || len(db.profiles[a].OnboardingAnswers.Concerns) != 0 {
		t.Fatal("explicit empty answers did not clear context")
	}
}
func (r *fakeProfileDB) SaveOnboardingName(_ context.Context, uid uuid.UUID, name string) error {
	p, ok := r.profiles[uid]
	if !ok {
		return httpx.ErrNotFound
	}
	if p.OnboardingCompletedAt != nil {
		return domain.ErrOnboardingAlreadyComplete
	}
	p.DisplayName = name
	r.profiles[uid] = p
	return nil
}

type validSession struct {
	user uuid.UUID
	sid  uuid.UUID
}

func (s validSession) IsActive(_ context.Context, u, session uuid.UUID) (bool, error) {
	return s.user == u && s.sid == session, nil
}

func TestOnboardingAndProfileAreScopedToSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	a, b := uuid.New(), uuid.New()
	db := &fakeProfileDB{profiles: map[uuid.UUID]domain.Profile{
		a: {UserID: a, DisplayName: "", ConversationStyle: domain.Balanced},
		b: {UserID: b, DisplayName: "Tester B", ConversationStyle: domain.Balanced},
	}}
	signer, err := devsecurity.NewSigner("0123456789abcdef0123456789abcdef", false)
	if err != nil {
		t.Fatal(err)
	}
	sid := uuid.New()
	r := gin.New()
	g := r.Group("/api/v1", devsecurity.Guard(signer, validSession{user: a, sid: sid}))
	h := New(application.New(db))
	g.GET("/me", h.Get)
	g.PATCH("/me/profile", h.Update)
	cookie := &http.Cookie{Name: devsecurity.CookieName, Value: signer.Issue(a, sid, time.Now())}
	call := func(method, body string, withCookie bool) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, "/api/v1/me", nil)
		if method == http.MethodPatch {
			req = httptest.NewRequest(method, "/api/v1/me/profile", bytes.NewBufferString(body))
			req.Header.Set("Content-Type", "application/json")
		}
		if withCookie {
			req.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}
	if w := call("GET", "", false); w.Code != 401 {
		t.Fatalf("anonymous read: %d", w.Code)
	}
	if w := call("GET", "", true); w.Code != 200 {
		t.Fatalf("session read: %d", w.Code)
	}
	for _, body := range []string{
		`{ "displayName": "   ", "conversationStyle": "balanced" }`,
		`{ "displayName": "Test", "conversationStyle": "invalid" }`,
		`{ "displayName": "Name\nInjected", "conversationStyle": "balanced" }`,
	} {
		w := call("PATCH", body, true)
		if w.Code != 400 {
			t.Fatalf("invalid input accepted: %d %s", w.Code, w.Body.String())
		}
	}
	if db.profiles[a].OnboardingCompletedAt != nil {
		t.Fatal("invalid requests completed onboarding")
	}
	w := call("PATCH", `{"displayName":"  نیلو  ","conversationStyle":"listening"}`, true)
	if w.Code != 200 {
		t.Fatalf("valid onboarding: %d %s", w.Code, w.Body.String())
	}
	var p domain.Profile
	if err := json.Unmarshal(w.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	if p.DisplayName != "نیلو" || p.OnboardingCompletedAt == nil || p.ConversationStyle != domain.Listening {
		t.Fatalf("incorrect profile: %+v", p)
	}
	original := *p.OnboardingCompletedAt
	w = call("PATCH", `{"displayName":"Next","conversationStyle":"guidance"}`, true)
	if w.Code != 200 {
		t.Fatalf("edit: %d", w.Code)
	}
	if err := json.Unmarshal(w.Body.Bytes(), &p); err != nil || !p.OnboardingCompletedAt.Equal(original) {
		t.Fatalf("edit changed completion: %+v %v", p, err)
	}
	if got := db.profiles[b].DisplayName; got != "Tester B" {
		t.Fatalf("other user modified: %s", got)
	}
}

func TestSaveOnboardingNameBeforeCompletionAndRejectDuplicateEdits(t *testing.T) {
	gin.SetMode(gin.TestMode)
	a, b := uuid.New(), uuid.New()
	db := &fakeProfileDB{profiles: map[uuid.UUID]domain.Profile{
		a: {UserID: a, ConversationStyle: domain.Balanced},
		b: {UserID: b, DisplayName: "Another", ConversationStyle: domain.Balanced},
	}}
	signer, err := devsecurity.NewSigner("0123456789abcdef0123456789abcdef", false)
	if err != nil {
		t.Fatal(err)
	}
	sid := uuid.New()
	r := gin.New()
	g := r.Group("/api/v1", devsecurity.Guard(signer, validSession{user: a, sid: sid}))
	h := New(application.New(db))
	g.PATCH("/me/onboarding-name", h.SaveOnboardingName)
	g.PATCH("/me/profile", h.Update)
	cookie := &http.Cookie{Name: devsecurity.CookieName, Value: signer.Issue(a, sid, time.Now())}
	call := func(path, body string, auth bool) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPatch, path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		if auth {
			req.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}
	path := "/api/v1/me/onboarding-name"
	if w := call(path, `{"displayName":"مهسا"}`, false); w.Code != 401 {
		t.Fatalf("anonymous: %d", w.Code)
	}
	for _, input := range []string{`{"displayName":"  "}`, `{"displayName":"bad\u202e"}`, `{"displayName":"مهسا","conversationStyle":"guidance"}`} {
		if w := call(path, input, true); w.Code != 400 {
			t.Fatalf("invalid input: %s: %d", input, w.Code)
		}
	}
	if db.profiles[a].DisplayName != "" || db.profiles[a].ConversationStyle != domain.Balanced {
		t.Fatal("rejected input changed profile state")
	}
	w := call(path, `{"displayName":"  مهسا  "}`, true)
	if w.Code != 200 {
		t.Fatalf("draft save: %d %s", w.Code, w.Body.String())
	}
	if db.profiles[a].DisplayName != "مهسا" || db.profiles[a].OnboardingCompletedAt != nil {
		t.Fatalf("unexpected draft state: %+v", db.profiles[a])
	}
	if db.profiles[b].DisplayName != "Another" {
		t.Fatal("draft changed a second user's name")
	}
	w = call("/api/v1/me/profile", `{"displayName":"مهسا","conversationStyle":"balanced"}`, true)
	if w.Code != 200 || db.profiles[a].OnboardingCompletedAt == nil {
		t.Fatalf("completion failed: %d", w.Code)
	}
	w = call(path, `{"displayName":"دوباره"}`, true)
	if w.Code != 409 || db.profiles[a].DisplayName != "مهسا" {
		t.Fatalf("completed profile edited via draft endpoint: %d", w.Code)
	}
}
