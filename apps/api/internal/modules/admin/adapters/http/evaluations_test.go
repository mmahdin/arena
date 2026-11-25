package httpadapter

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/admin/application"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/admin/domain"
	auth "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/auth"
	authhttp "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/auth/adapters/http"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/config"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/httpx"
)

type handlerEvalStore struct {
	last  domain.EvalRunDetail
	audit *fakeAuditStore
	fail  error
	calls int
}

func (s *handlerEvalStore) Import(_ context.Context, d domain.EvalRunDetail) (domain.EvalRun, bool, error) {
	if len(s.audit.events) == 0 {
		panic("not audited")
	}
	s.calls++
	if s.fail != nil {
		return domain.EvalRun{}, false, s.fail
	}
	s.last = d
	return d.Run, true, nil
}
func (s *handlerEvalStore) Get(_ context.Context, id uuid.UUID) (domain.EvalRunDetail, error) {
	s.calls++
	if s.fail != nil {
		return domain.EvalRunDetail{}, s.fail
	}
	if s.last.Run.ID != id {
		return domain.EvalRunDetail{}, domain.ErrEvalNotFound
	}
	return s.last, nil
}
func (s *handlerEvalStore) List(_ context.Context, _ domain.EvalFilter) (domain.EvalPage, error) {
	s.calls++
	if s.fail != nil {
		return domain.EvalPage{}, s.fail
	}
	if s.last.Run.ID == uuid.Nil {
		return domain.EvalPage{Items: []domain.EvalRun{}}, nil
	}
	return domain.EvalPage{Items: []domain.EvalRun{s.last.Run}}, nil
}
func setupEvaluationHTTP(t *testing.T) (*gin.Engine, *fakeStore, *fakeAuditStore, *handlerEvalStore) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	store := &fakeStore{staffID: uuid.New(), userID: uuid.New(), allowStaff: true}
	svc, err := auth.New(store, []byte(strings.Repeat("s", 32)), map[auth.Kind]auth.Realm{
		auth.User:  {Origin: "http://app.localhost:8080", RedirectURI: "http://app.localhost:8080/auth/user/callback", Issuer: "https://issuer.test/users", Provider: fakeOIDC{}},
		auth.Staff: {Origin: "http://admin.localhost:8080", RedirectURI: "http://admin.localhost:8080/auth/staff/callback", Issuer: "https://issuer.test/staff", Provider: fakeOIDC{}},
	})
	if err != nil {
		t.Fatal(err)
	}
	bff := authhttp.NewForTests(svc, config.AuthConfig{Enabled: true, UserLoginEnabled: true, StaffLoginEnabled: true, UserOrigin: "http://app.localhost:8080", StaffOrigin: "http://admin.localhost:8080"})
	r := gin.New()
	r.Use(httpx.Middleware("http://admin.localhost:8080"))
	for _, kind := range []auth.Kind{auth.Staff, auth.User} {
		base := "/auth/" + string(kind)
		r.GET(base+"/login", bff.Login(kind))
		r.GET(base+"/callback", bff.Callback(kind))
	}
	r.GET("/auth/staff/session", bff.Session(auth.Staff))
	audit := &fakeAuditStore{}
	reader := &handlerEvalStore{audit: audit}
	h := New(application.NewWithEvaluationRepositories(audit, nil, nil, reader))
	group := r.Group("/api/v1/admin")
	group.Use(bff.RequireStaff())
	group.GET("/evaluations", h.Evaluations)
	group.GET("/evaluations/:runId", h.EvaluationDetail)
	group.POST("/evaluations/import", h.EvaluationImport)
	return r, store, audit, reader
}
func evalRequest(r *gin.Engine, method, path string, b []byte, cookie *http.Cookie, csrf string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(string(b)))
	req.Host = "admin.localhost:8080"
	req.Header.Set("Content-Type", "application/json")
	if cookie != nil {
		req.AddCookie(cookie)
	}
	if csrf != "" {
		req.Header.Set("Origin", "http://admin.localhost:8080")
		req.Header.Set("X-CSRF-Token", csrf)
	}
	digest := sha256.Sum256(b)
	req.Header.Set("X-Eval-SHA256", hex.EncodeToString(digest[:]))
	r.ServeHTTP(w, req)
	return w
}
func readEvaluationReportFixture(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile("../../../../../../../docs/quality/test-results/ai-core-eval.json")
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestEvaluationHTTPStaffCSRFAndRedaction(t *testing.T) {
	r, authStore, audit, reader := setupEvaluationHTTP(t)
	raw := readEvaluationReportFixture(t)
	url := "/api/v1/admin/evaluations/import"
	if w := evalRequest(r, "POST", url, raw, nil, ""); w.Code != 401 {
		t.Fatalf("anon %d", w.Code)
	}
	user := login(t, r, auth.User)
	if w := evalRequest(r, "POST", url, raw, user, ""); w.Code != 401 {
		t.Fatalf("user %d", w.Code)
	}
	staff := login(t, r, auth.Staff)
	if w := call(r, http.MethodGet, "app.localhost:8080", "/api/v1/admin/evaluations", staff); w.Code != http.StatusBadRequest {
		t.Fatalf("wrong host: %d", w.Code)
	}
	if w := call(r, http.MethodGet, "admin.localhost:8080", "/api/v1/admin/evaluations", &http.Cookie{Name: "wellbeing_dev_session", Value: "fake"}); w.Code != http.StatusUnauthorized {
		t.Fatalf("dev session accepted: %d", w.Code)
	}
	if w := evalRequest(r, "POST", url, raw, staff, ""); w.Code != 403 {
		t.Fatalf("missing csrf %d", w.Code)
	}
	session := call(r, http.MethodGet, "admin.localhost:8080", "/auth/staff/session", staff)
	var token struct {
		CSRFToken string `json:"csrfToken"`
	}
	if session.Code != 200 || json.Unmarshal(session.Body.Bytes(), &token) != nil || token.CSRFToken == "" {
		t.Fatal("missing CSRF token")
	}
	if w := evalRequest(r, "POST", url, []byte(`{"prompt":"private"}`), staff, token.CSRFToken); w.Code != 400 {
		t.Fatalf("untrusted content accepted: %d", w.Code)
	}
	if len(audit.events) != 0 || reader.calls != 0 {
		t.Fatal("invalid import reached store")
	}
	good := evalRequest(r, "POST", url, raw, staff, token.CSRFToken)
	if good.Code != 201 || reader.calls != 1 || len(audit.events) != 1 || audit.events[0].ActionCode != domain.ActionEvaluationImport {
		t.Fatalf("import: %d %s", good.Code, good.Body.String())
	}
	if strings.Contains(good.Body.String(), "PRIVATE") || strings.Contains(good.Body.String(), "clinicallyApproved\":true") {
		t.Fatal("unsafe data or approval")
	}
	list := call(r, http.MethodGet, "admin.localhost:8080", "/api/v1/admin/evaluations", staff)
	if list.Code != 200 || reader.calls != 2 {
		t.Fatalf("list %d", list.Code)
	}
	detail := call(r, http.MethodGet, "admin.localhost:8080", "/api/v1/admin/evaluations/"+reader.last.Run.ID.String(), staff)
	if detail.Code != 200 || reader.calls != 3 {
		t.Fatalf("detail %d", detail.Code)
	}
	if strings.Contains(detail.Body.String(), "prompt") || strings.Contains(detail.Body.String(), "PRIVATE") {
		t.Fatal("private report data")
	}
	audit.appendErr = errors.New("hidden SQL error")
	w := call(r, http.MethodGet, "admin.localhost:8080", "/api/v1/admin/evaluations", staff)
	if w.Code != 503 || !strings.Contains(w.Body.String(), "ADMIN_AUDIT_UNAVAILABLE") || reader.calls != 3 {
		t.Fatal("audit failure did not block data")
	}
	audit.appendErr = nil
	authStore.allowStaff = false
	if w = call(r, http.MethodGet, "admin.localhost:8080", "/api/v1/admin/evaluations", staff); w.Code != 401 {
		t.Fatal("inactive staff access")
	}
}
