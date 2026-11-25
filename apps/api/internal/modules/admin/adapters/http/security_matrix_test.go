package httpadapter

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/admin/application"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/admin/domain"
	auth "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/auth"
	authhttp "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/auth/adapters/http"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/config"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/httpx"
)

// This test mounts the actual Admin handlers behind the actual AUTH-03/04
// session guard. It exercises their shared security boundary without an
// external IdP, external SQL Server, or fake "admin" request headers.
func setupBO10SecurityMatrix(t *testing.T) (*gin.Engine, *fakeStore, *fakeAuditStore, *fakeOverviewStore, *fakeInspector, *handlerEvalStore) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	store := &fakeStore{staffID: uuid.New(), userID: uuid.New(), allowStaff: true}
	provider := fakeOIDC{}
	authService, err := auth.New(store, []byte(strings.Repeat("s", 32)), map[auth.Kind]auth.Realm{
		auth.User:  {Origin: "http://app.localhost:8080", RedirectURI: "http://app.localhost:8080/auth/user/callback", Issuer: "https://issuer.test/users", Provider: provider},
		auth.Staff: {Origin: "http://admin.localhost:8080", RedirectURI: "http://admin.localhost:8080/auth/staff/callback", Issuer: "https://issuer.test/staff", Provider: provider},
	})
	if err != nil {
		t.Fatal(err)
	}
	bff := authhttp.NewForTests(authService, config.AuthConfig{
		Enabled: true, UserLoginEnabled: true, StaffLoginEnabled: true,
		UserOrigin: "http://app.localhost:8080", StaffOrigin: "http://admin.localhost:8080",
	})
	audit := &fakeAuditStore{}
	overview := &fakeOverviewStore{}
	inspector := &fakeInspector{id: uuid.New()}
	evaluations := &handlerEvalStore{audit: audit, last: domain.EvalRunDetail{Run: domain.EvalRun{ID: uuid.New()}}}
	service := application.NewWithEvaluationRepositories(audit, overview, inspector, evaluations)
	service.SetTraceReferenceStore(&fakeTraceReferenceReader{})
	service.SetBundleInventory(domain.BundleInventory{Scope: "not_configured", AsOf: time.Now().UTC()})
	h := New(service)
	r := gin.New()
	r.Use(httpx.Middleware("http://admin.localhost:8080"))
	for _, kind := range []auth.Kind{auth.User, auth.Staff} {
		path := "/auth/" + string(kind)
		r.GET(path+"/login", bff.Login(kind))
		r.GET(path+"/callback", bff.Callback(kind))
	}
	r.GET("/auth/staff/session", bff.Session(auth.Staff))
	admin := r.Group("/api/v1/admin")
	admin.Use(bff.RequireStaff())
	admin.GET("/me", h.Me)
	admin.GET("/audit", h.Audit)
	admin.GET("/overview", h.Overview)
	admin.GET("/turns", h.Turns)
	admin.GET("/turns/:turnId", h.TurnDetail)
	admin.GET("/turns/:turnId/trace-references", h.TraceReferences)
	admin.GET("/jobs", h.Jobs)
	admin.GET("/workers", h.Workers)
	admin.GET("/bundles", h.Bundles)
	admin.GET("/evaluations", h.Evaluations)
	admin.GET("/evaluations/:runId", h.EvaluationDetail)
	admin.POST("/evaluations/import", h.EvaluationImport)
	return r, store, audit, overview, inspector, evaluations
}

func TestBO10EveryAdminReadRequiresVerifiedStaffAndAudit(t *testing.T) {
	r, store, audit, overview, inspector, evals := setupBO10SecurityMatrix(t)
	paths := []struct{ path, action string }{
		{"/api/v1/admin/me", domain.ActionAdminMeRead},
		{"/api/v1/admin/audit", domain.ActionAuditRead},
		{"/api/v1/admin/overview", domain.ActionOverviewRead},
		{"/api/v1/admin/turns", domain.ActionTurnsRead},
		{"/api/v1/admin/turns/" + inspector.id.String(), domain.ActionTurnDetailRead},
		{"/api/v1/admin/turns/" + inspector.id.String() + "/trace-references", domain.ActionTurnTraceRefsRead},
		{"/api/v1/admin/jobs", domain.ActionJobsRead},
		{"/api/v1/admin/workers", domain.ActionWorkersRead},
		{"/api/v1/admin/bundles", domain.ActionBundlesRead},
		{"/api/v1/admin/evaluations", domain.ActionEvaluationsRead},
		{"/api/v1/admin/evaluations/" + evals.last.Run.ID.String(), domain.ActionEvaluationDetailRead},
	}
	user := login(t, r, auth.User)
	staff := login(t, r, auth.Staff)
	for _, tc := range paths {
		for _, cookie := range []*http.Cookie{nil, user, &http.Cookie{Name: "wellbeing_dev_session", Value: "synthetic"}} {
			var w *httptest.ResponseRecorder
			if cookie == nil {
				w = call(r, http.MethodGet, "admin.localhost:8080", tc.path)
			} else {
				w = call(r, http.MethodGet, "admin.localhost:8080", tc.path, cookie)
			}
			if w.Code != 401 {
				t.Fatalf("untrusted principal accessed %s: %d", tc.path, w.Code)
			}
		}
		if got := call(r, http.MethodGet, "app.localhost:8080", tc.path, staff); got.Code != 400 {
			t.Fatalf("cross-host %s: %d", tc.path, got.Code)
		}
	}
	if len(audit.events) != 0 || overview.calls != 0 || inspector.calls != 0 || evals.calls != 0 {
		t.Fatal("unauthorized request reached business data or privileged audit")
	}
	for i, tc := range paths {
		w := call(r, http.MethodGet, "admin.localhost:8080", tc.path, staff)
		if w.Code != 200 || w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("X-Request-ID") == "" {
			t.Fatalf("staff %s: %d %s", tc.path, w.Code, w.Body.String())
		}
		if len(audit.events) != i+1 || audit.events[i].ActionCode != tc.action || audit.events[i].ActorStaffID != store.staffID ||
			audit.events[i].RequestID.String() != w.Header().Get("X-Request-ID") {
			t.Fatalf("missing verified audit for %s: %+v", tc.path, audit.events)
		}
		for _, private := range []string{"PRIVATE DO NOT LEAK", "test-subject", "test-keycloak-sid", "wellbeing_dev_session", "Bearer forged"} {
			if strings.Contains(w.Body.String(), private) {
				t.Fatalf("private metadata leaked on %s: %q", tc.path, private)
			}
		}
	}
	readerCalls := overview.calls + inspector.calls + evals.calls
	audit.appendErr = errors.New("PRIVATE SQL PASSWORD MUST NOT LEAK")
	for _, tc := range paths {
		w := call(r, http.MethodGet, "admin.localhost:8080", tc.path, staff)
		if w.Code != 503 || !strings.Contains(w.Body.String(), "ADMIN_AUDIT_UNAVAILABLE") || strings.Contains(w.Body.String(), "PRIVATE SQL") {
			t.Fatalf("audit outage exposed %s: %d %s", tc.path, w.Code, w.Body.String())
		}
	}
	if overview.calls+inspector.calls+evals.calls != readerCalls {
		t.Fatal("privileged data queried during audit outage")
	}
	audit.appendErr = nil
	store.allowStaff = false
	for _, tc := range paths {
		if w := call(r, http.MethodGet, "admin.localhost:8080", tc.path, staff); w.Code != 401 {
			t.Fatalf("inactive staff accepted by %s: %d", tc.path, w.Code)
		}
	}
	if len(audit.events) != len(paths) {
		t.Fatal("inactive Staff reads created authorized audit events")
	}
}

func TestBO10AdminEvaluationImportRequiresCSRFOriginAndAudit(t *testing.T) {
	r, _, audit, _, _, evals := setupBO10SecurityMatrix(t)
	staff := login(t, r, auth.Staff)
	raw := readEvaluationReportFixture(t)
	post := func(origin, csrf string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/evaluations/import", strings.NewReader(string(raw)))
		req.Host = "admin.localhost:8080"
		req.AddCookie(staff)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", origin)
		req.Header.Set("X-CSRF-Token", csrf)
		sum := sha256.Sum256(raw)
		req.Header.Set("X-Eval-SHA256", hex.EncodeToString(sum[:]))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}
	session := call(r, http.MethodGet, "admin.localhost:8080", "/auth/staff/session", staff)
	var data struct {
		CSRFToken string `json:"csrfToken"`
	}
	if session.Code != 200 || json.Unmarshal(session.Body.Bytes(), &data) != nil || data.CSRFToken == "" {
		t.Fatalf("missing CSRF: %d %s", session.Code, session.Body.String())
	}
	if w := post("http://admin.localhost:8080", ""); w.Code != 403 {
		t.Fatalf("missing CSRF accepted: %d", w.Code)
	}
	if w := post("http://app.localhost:8080", data.CSRFToken); w.Code != 403 {
		t.Fatalf("wrong Origin accepted: %d", w.Code)
	}
	if len(audit.events) != 0 || evals.calls != 0 {
		t.Fatal("unverified import accessed storage")
	}
	audit.appendErr = errors.New("PRIVATE SQL PASSWORD")
	if w := post("http://admin.localhost:8080", data.CSRFToken); w.Code != 503 || !strings.Contains(w.Body.String(), "ADMIN_AUDIT_UNAVAILABLE") {
		t.Fatalf("audit failure not honored by import: %d %s", w.Code, w.Body.String())
	}
	if evals.calls != 0 {
		t.Fatal("import executed while audit was offline")
	}
	audit.appendErr = nil
	if w := post("http://admin.localhost:8080", data.CSRFToken); w.Code != 201 || len(audit.events) != 1 || audit.events[0].ActionCode != domain.ActionEvaluationImport || evals.calls != 1 {
		t.Fatalf("valid synthetic import rejected or unaudited: %d %s", w.Code, w.Body.String())
	}
}

var _ application.OverviewStore = (*fakeOverviewStore)(nil)
var _ application.InspectorStore = (*fakeInspector)(nil)
var _ application.EvaluationStore = (*handlerEvalStore)(nil)
var _ auth.Store = (*fakeStore)(nil)
