package httpadapter

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
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

type fakeInspector struct {
	calls int
	fail  error
	id    uuid.UUID
}

func (f *fakeInspector) ListTurns(_ context.Context, _ domain.InspectorFilter) (domain.TurnPage, error) {
	f.calls++
	return domain.TurnPage{Items: []domain.TurnRecord{{ID: f.id, Status: "failed"}}}, f.fail
}
func (f *fakeInspector) GetTurn(_ context.Context, id uuid.UUID) (domain.TurnDetail, error) {
	f.calls++
	return domain.TurnDetail{Turn: domain.TurnRecord{ID: id, Status: "failed"}, Stages: []domain.StageObservation{}, LLMAttempts: []domain.LLMAttempt{}, Jobs: []domain.JobRecord{}, Warnings: []string{"BEST_EFFORT_TELEMETRY_NOT_FULL_TRACE"}}, f.fail
}
func (f *fakeInspector) ListJobs(_ context.Context, _ domain.InspectorFilter) (domain.JobPage, error) {
	f.calls++
	return domain.JobPage{Items: []domain.JobRecord{}}, f.fail
}
func (f *fakeInspector) ListWorkers(_ context.Context, at time.Time) (domain.WorkerPage, error) {
	f.calls++
	return domain.WorkerPage{AsOf: at, Instances: []domain.WorkerInstance{}, Warnings: []string{"NO_RECENT_HEARTBEATS"}}, f.fail
}
func setupInspector(t *testing.T, reader *fakeInspector, audit *fakeAuditStore, traceStore ...application.TraceReferenceStore) (*gin.Engine, *fakeStore) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	store := &fakeStore{staffID: uuid.New(), userID: uuid.New(), allowStaff: true}
	oidc := fakeOIDC{}
	svc, err := auth.New(store, []byte(strings.Repeat("s", 32)), map[auth.Kind]auth.Realm{
		auth.User:  {Origin: "http://app.localhost:8080", RedirectURI: "http://app.localhost:8080/auth/user/callback", Issuer: "https://issuer.test/users", Provider: oidc},
		auth.Staff: {Origin: "http://admin.localhost:8080", RedirectURI: "http://admin.localhost:8080/auth/staff/callback", Issuer: "https://issuer.test/staff", Provider: oidc},
	})
	if err != nil {
		t.Fatal(err)
	}
	bff := authhttp.NewForTests(svc, config.AuthConfig{Enabled: true, UserLoginEnabled: true, StaffLoginEnabled: true, UserOrigin: "http://app.localhost:8080", StaffOrigin: "http://admin.localhost:8080"})
	r := gin.New()
	r.Use(httpx.Middleware("http://admin.localhost:8080"))
	for _, kind := range []auth.Kind{auth.User, auth.Staff} {
		base := "/auth/" + string(kind)
		r.GET(base+"/login", bff.Login(kind))
		r.GET(base+"/callback", bff.Callback(kind))
	}
	group := r.Group("/api/v1/admin")
	group.Use(bff.RequireStaff())
	service := application.NewWithOperationalRepositories(audit, &fakeOverviewStore{}, reader)
	if len(traceStore) > 0 {
		service.SetTraceReferenceStore(traceStore[0])
	}
	h := New(service)
	group.GET("/turns", h.Turns)
	group.GET("/turns/:turnId", h.TurnDetail)
	group.GET("/turns/:turnId/trace-references", h.TraceReferences)
	group.GET("/jobs", h.Jobs)
	group.GET("/workers", h.Workers)
	return r, store
}
func TestInspectorRoutesAuthenticatedAuditedAndPrivate(t *testing.T) {
	reader := &fakeInspector{id: uuid.New()}
	audit := &fakeAuditStore{}
	router, authStore := setupInspector(t, reader, audit)
	paths := []string{"/api/v1/admin/turns", "/api/v1/admin/turns/" + reader.id.String(), "/api/v1/admin/jobs", "/api/v1/admin/workers"}
	for _, path := range paths {
		if w := call(router, http.MethodGet, "admin.localhost:8080", path); w.Code != 401 {
			t.Fatalf("anon %s: %d", path, w.Code)
		}
	}
	user := login(t, router, auth.User)
	for _, path := range paths {
		if w := call(router, http.MethodGet, "admin.localhost:8080", path, user); w.Code != 401 {
			t.Fatalf("user %s: %d", path, w.Code)
		}
	}
	if reader.calls != 0 || len(audit.events) != 0 {
		t.Fatal("unauthenticated access reached storage")
	}
	staff := login(t, router, auth.Staff)
	for i, path := range paths {
		w := call(router, http.MethodGet, "admin.localhost:8080", path, staff)
		if w.Code != 200 || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("%s: %d %s", path, w.Code, w.Body.String())
		}
		if len(audit.events) != i+1 || audit.events[i].RequestID.String() != w.Header().Get("X-Request-ID") {
			t.Fatal("missing request-correlated audit")
		}
		for _, secret := range []string{"test-subject", "privateMessage", "payload_json", "password", "Bearer"} {
			if strings.Contains(w.Body.String(), secret) {
				t.Fatalf("leak %q", secret)
			}
		}
	}
	if audit.events[0].ActionCode != domain.ActionTurnsRead || audit.events[1].ResourceID != reader.id || audit.events[2].ActionCode != domain.ActionJobsRead || audit.events[3].ActionCode != domain.ActionWorkersRead {
		t.Fatal("audit action mismatch")
	}
	if reader.calls != 4 {
		t.Fatal("missing operational queries")
	}
	authStore.allowStaff = false
	if w := call(router, http.MethodGet, "admin.localhost:8080", paths[0], staff); w.Code != 401 || reader.calls != 4 {
		t.Fatal("inactive staff read")
	}
}
func TestInspectorRejectsBadFiltersAndFailClosed(t *testing.T) {
	reader := &fakeInspector{id: uuid.New()}
	audit := &fakeAuditStore{}
	router, _ := setupInspector(t, reader, audit)
	staff := login(t, router, auth.Staff)
	prefix := "/api/v1/admin/turns"
	now := time.Now().UTC().Truncate(time.Second)
	from := url.QueryEscape(now.Add(-time.Hour).Format(time.RFC3339))
	to := url.QueryEscape(now.Format(time.RFC3339))
	bad := []string{"?status=deleted", "?status=failed&status=failed", "?kind=reflection", "?cursor=bogus", "?pageSize=0", "?pageSize=101", "?from=" + from, "?from=" + from + "&to=" + to + "&x=1", "?from=" + from + "&to=" + to + "&from=" + from}
	for _, suffix := range bad {
		w := call(router, http.MethodGet, "admin.localhost:8080", prefix+suffix, staff)
		if w.Code != 400 || !strings.Contains(w.Body.String(), "ADMIN_INSPECTOR_FILTER_INVALID") {
			t.Fatalf("%s: %d %s", suffix, w.Code, w.Body.String())
		}
	}
	if reader.calls != 0 || len(audit.events) != 0 {
		t.Fatal("invalid filter touched storage")
	}
	for _, p := range []string{"/api/v1/admin/turns/invalid", "/api/v1/admin/turns/00000000-0000-0000-0000-000000000000", "/api/v1/admin/workers?foo=1"} {
		w := call(router, http.MethodGet, "admin.localhost:8080", p, staff)
		if w.Code != 400 {
			t.Fatalf("invalid %s: %d", p, w.Code)
		}
	}
	audit.appendErr = errors.New("secret in database")
	if w := call(router, http.MethodGet, "admin.localhost:8080", prefix, staff); w.Code != 503 || !strings.Contains(w.Body.String(), "ADMIN_AUDIT_UNAVAILABLE") || reader.calls != 0 {
		t.Fatal("audit failure did not block read")
	}
	audit.appendErr = nil
	reader.fail = errors.New("database password leaked")
	w := call(router, http.MethodGet, "admin.localhost:8080", prefix, staff)
	if w.Code != 503 || strings.Contains(w.Body.String(), "password") {
		t.Fatalf("SQL exception leaked: %d %s", w.Code, w.Body.String())
	}
	reader.fail = domain.ErrTurnNotFound
	w = call(router, http.MethodGet, "admin.localhost:8080", "/api/v1/admin/turns/"+reader.id.String(), staff)
	if w.Code != 404 {
		t.Fatalf("missing turn %d", w.Code)
	}
}
func TestInspectorNoGuardFailsClosed(t *testing.T) {
	h := New(application.NewWithOperationalRepositories(&fakeAuditStore{}, &fakeOverviewStore{}, &fakeInspector{id: uuid.New()}))
	r := gin.New()
	r.GET("/turns", h.Turns)
	r.GET("/jobs", h.Jobs)
	r.GET("/workers", h.Workers)
	r.GET("/turns/:turnId", h.TurnDetail)
	r.GET("/turns/:turnId/trace-references", h.TraceReferences)
	for _, path := range []string{"/turns", "/jobs", "/workers", "/turns/" + uuid.NewString(), "/turns/" + uuid.NewString() + "/trace-references"} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		r.ServeHTTP(w, req)
		if w.Code != 401 {
			t.Fatalf("unguarded %s: %d", path, w.Code)
		}
	}
}
func TestInspectorResponseContractShape(t *testing.T) {
	v := domain.TurnDetail{Turn: domain.TurnRecord{ID: uuid.New()}, Stages: []domain.StageObservation{}, LLMAttempts: []domain.LLMAttempt{}, Jobs: []domain.JobRecord{}, Warnings: []string{}}
	b, e := json.Marshal(v)
	if e != nil {
		t.Fatal(e)
	}
	for _, k := range []string{"\"turn\"", "\"stages\"", "\"llmAttempts\"", "\"jobs\"", "\"warnings\""} {
		if !strings.Contains(string(b), k) {
			t.Fatal(k)
		}
	}
}
