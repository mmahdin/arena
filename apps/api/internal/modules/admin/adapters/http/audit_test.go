package httpadapter

import (
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
	auth "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/auth"
)

func TestAuditRequiresStaffAndRecordsAuthorizedReads(t *testing.T) {
	repo := &fakeAuditStore{}
	r, store := setup(t, repo)
	path := "/api/v1/admin/audit"
	if got := call(r, http.MethodGet, "admin.localhost:8080", path); got.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous audit accepted: %d", got.Code)
	}
	user := login(t, r, auth.User)
	if got := call(r, http.MethodGet, "admin.localhost:8080", path, user); got.Code != http.StatusUnauthorized {
		t.Fatalf("user session accepted: %d", got.Code)
	}
	staff := login(t, r, auth.Staff)
	if got := call(r, http.MethodGet, "app.localhost:8080", path, staff); got.Code != http.StatusBadRequest {
		t.Fatalf("wrong origin accepted: %d", got.Code)
	}
	if len(repo.events) != 0 {
		t.Fatalf("unauthorized requests appended %d events", len(repo.events))
	}

	me := call(r, http.MethodGet, "admin.localhost:8080", "/api/v1/admin/me", staff)
	if me.Code != http.StatusOK {
		t.Fatalf("me: %d %s", me.Code, me.Body.String())
	}
	if len(repo.events) != 1 || repo.events[0].ActionCode != "admin.me.read" || repo.events[0].RequestID.String() != me.Header().Get("X-Request-ID") {
		t.Fatal("me access not recorded with server-issued correlation")
	}
	got := call(r, http.MethodGet, "admin.localhost:8080", path, staff)
	if got.Code != http.StatusOK || got.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("audit read: %d %s", got.Code, got.Body.String())
	}
	var data struct {
		Items []struct {
			ActorStaffID string `json:"actorStaffId"`
			ActionCode   string `json:"actionCode"`
			RequestID    string `json:"requestId"`
		} `json:"items"`
		NextCursor *string `json:"nextCursor"`
	}
	if err := json.Unmarshal(got.Body.Bytes(), &data); err != nil || len(data.Items) < 1 || data.Items[0].ActorStaffID != store.staffID.String() {
		t.Fatalf("bad audit response: %s %v", got.Body.String(), err)
	}
	if len(repo.events) != 2 || repo.events[1].ActionCode != "admin.audit.read" || repo.events[1].RequestID.String() != got.Header().Get("X-Request-ID") {
		t.Fatal("audit viewer did not audit its own access")
	}
	for _, keyword := range []string{"test-subject", "password", "Bearer", "sessionToken", "mobile", "privateMessage", "test-keycloak-sid"} {
		if strings.Contains(got.Body.String(), keyword) {
			t.Fatalf("sensitive value in response: %s", keyword)
		}
	}
	store.allowStaff = false
	before := len(repo.events)
	if got := call(r, http.MethodGet, "admin.localhost:8080", path, staff); got.Code != http.StatusUnauthorized || len(repo.events) != before {
		t.Fatal("inactive staff audit allowed")
	}
}

func TestAuditFilterValidation(t *testing.T) {
	repo := &fakeAuditStore{}
	r, _ := setup(t, repo)
	staff := login(t, r, auth.Staff)
	from := time.Now().UTC().Add(-48 * time.Hour).Format(time.RFC3339)
	to := time.Now().UTC().Format(time.RFC3339)
	invalid := []string{
		"pageSize=0", "pageSize=101", "pageSize=x", "pageSize=", "pageSize=10&pageSize=20",
		"from=" + url.QueryEscape(from),
		"from=" + url.QueryEscape(from) + "&to=" + url.QueryEscape("garbage"),
		"from=" + url.QueryEscape(time.Now().UTC().Add(-32*24*time.Hour).Format(time.RFC3339)) + "&to=" + url.QueryEscape(to),
		"actionCode=unknown", "actorStaffId=secret", "cursor=%25", "token=stolen", "cursor=",
	}
	for _, query := range invalid {
		got := call(r, http.MethodGet, "admin.localhost:8080", "/api/v1/admin/audit?"+query, staff)
		if got.Code != http.StatusBadRequest || !strings.Contains(got.Body.String(), "AUDIT_FILTER_INVALID") {
			t.Fatalf("invalid filter %q: %d %s", query, got.Code, got.Body.String())
		}
	}
	if len(repo.events) != 0 {
		t.Fatal("invalid filters touched audit store")
	}
}

func TestAuditDatabaseFailureFailsClosed(t *testing.T) {
	repo := &fakeAuditStore{appendErr: errors.New("SQL connection lost: might include credentials")}
	r, _ := setup(t, repo)
	staff := login(t, r, auth.Staff)
	for _, route := range []string{"/api/v1/admin/me", "/api/v1/admin/audit"} {
		got := call(r, http.MethodGet, "admin.localhost:8080", route, staff)
		if got.Code != http.StatusServiceUnavailable || !strings.Contains(got.Body.String(), "ADMIN_AUDIT_UNAVAILABLE") || strings.Contains(got.Body.String(), "SQL connection lost") {
			t.Fatalf("failed audit repository leaked data on %s: %d %s", route, got.Code, got.Body.String())
		}
	}
	repo.appendErr = nil
	repo.listErr = errors.New("read error")
	got := call(r, http.MethodGet, "admin.localhost:8080", "/api/v1/admin/audit", staff)
	if got.Code != http.StatusServiceUnavailable || len(repo.events) != 1 {
		t.Fatalf("list error not fail-closed: %d %s", got.Code, got.Body.String())
	}
}

func TestDirectAuditHandlerNoVerifiedPrincipal(t *testing.T) {
	r := gin.New()
	r.GET("/audit", New(application.NewWithAudit(&fakeAuditStore{})).Audit)
	got := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/audit?role=admin&staffId="+uuid.NewString(), nil)
	req.Host = "admin.localhost:8080"
	r.ServeHTTP(got, req)
	if got.Code != http.StatusUnauthorized {
		t.Fatalf("unguarded audit handler trusted request: %d %s", got.Code, got.Body.String())
	}
}
