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
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/admin/application"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/admin/domain"
	auth "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/auth"
)

type fakeOverviewStore struct {
	calls      int
	err        error
	lastFilter domain.OverviewFilter
}

func (f *fakeOverviewStore) ReadOverview(_ context.Context, filter domain.OverviewFilter) (domain.Overview, error) {
	f.calls++
	f.lastFilter = filter
	if f.err != nil {
		return domain.Overview{}, f.err
	}
	return domain.Overview{Turns: domain.TurnSummary{TotalV1: 3, Completed: 2, Failed: 1, LatencySamples: 2},
		LLM:     domain.LLMSummary{Attempts: 2, UsageKnown: 1, EstimatedCostCount: 1, UnknownCostCount: 1},
		Workers: []domain.WorkerSummary{{Kind: "memory", Status: "running"}, {Kind: "reflection", Status: "stale"}, {Kind: "scheduler", Status: "unknown"}}}, nil
}

func TestOverviewProtectedAndAudited(t *testing.T) {
	overview := &fakeOverviewStore{}
	audit := &fakeAuditStore{}
	r, authStore := setupWithOverview(t, overview, audit)
	route := "/api/v1/admin/overview"
	if w := call(r, http.MethodGet, "admin.localhost:8080", route); w.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous: %d", w.Code)
	}
	user := login(t, r, auth.User)
	if w := call(r, http.MethodGet, "admin.localhost:8080", route, user); w.Code != http.StatusUnauthorized {
		t.Fatalf("user accepted: %d", w.Code)
	}
	dev := call(r, http.MethodGet, "admin.localhost:8080", route, &http.Cookie{Name: "wellbeing_dev_session", Value: "fake"})
	if dev.Code != http.StatusUnauthorized {
		t.Fatalf("dev accepted: %d", dev.Code)
	}
	staff := login(t, r, auth.Staff)
	if w := call(r, http.MethodGet, "app.localhost:8080", route, staff); w.Code != http.StatusBadRequest {
		t.Fatalf("wrong host: %d", w.Code)
	}
	if overview.calls != 0 || len(audit.events) != 0 {
		t.Fatal("unauthorized request reached store")
	}
	w := call(r, http.MethodGet, "admin.localhost:8080", route, staff)
	if w.Code != http.StatusOK || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("overview: %d %s", w.Code, w.Body.String())
	}
	var result domain.Overview
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil || result.Turns.TotalV1 != 3 || result.Turns.FailureRate == nil || !result.LLM.CostPartial || result.LLM.EstimatedMicroUSD != nil || result.LLM.UnknownUsage != 1 {
		t.Fatalf("bad result: %+v %v", result, err)
	}
	if overview.calls != 1 || len(audit.events) != 1 || audit.events[0].ActionCode != domain.ActionOverviewRead || audit.events[0].RequestID.String() != w.Header().Get("X-Request-ID") {
		t.Fatal("missing audit before query")
	}
	for _, private := range []string{"privateMessage", "password", "Bearer", "sessionToken", "test-subject"} {
		if strings.Contains(w.Body.String(), private) {
			t.Fatalf("leaked %s", private)
		}
	}
	authStore.allowStaff = false
	if w = call(r, http.MethodGet, "admin.localhost:8080", route, staff); w.Code != http.StatusUnauthorized || overview.calls != 1 {
		t.Fatal("inactive staff read")
	}
}

func TestOverviewRejectsInvalidWindowsAndErrors(t *testing.T) {
	overview := &fakeOverviewStore{}
	audit := &fakeAuditStore{}
	r, _ := setupWithOverview(t, overview, audit)
	staff := login(t, r, auth.Staff)
	now := time.Now().UTC().Truncate(time.Second)
	from := now.Add(-time.Hour).Format(time.RFC3339)
	to := now.Format(time.RFC3339)
	bad := []string{"from=" + url.QueryEscape(from), "from=x&to=" + url.QueryEscape(to),
		"from=" + url.QueryEscape(from) + "&to=" + url.QueryEscape(to) + "&token=bad",
		"from=" + url.QueryEscape(from) + "&from=" + url.QueryEscape(from) + "&to=" + url.QueryEscape(to),
		"from=" + url.QueryEscape(now.Add(-32*24*time.Hour).Format(time.RFC3339)) + "&to=" + url.QueryEscape(to),
		"from=" + url.QueryEscape(from) + "&to=" + url.QueryEscape(now.Add(2*time.Minute).Format(time.RFC3339)),
		"from=&to=", "unexpected=secret"}
	for _, query := range bad {
		w := call(r, http.MethodGet, "admin.localhost:8080", "/api/v1/admin/overview?"+query, staff)
		if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "OVERVIEW_FILTER_INVALID") {
			t.Fatalf("%q: %d %s", query, w.Code, w.Body.String())
		}
	}
	if overview.calls != 0 || len(audit.events) != 0 {
		t.Fatal("invalid filters hit storage")
	}
	overview.err = errors.New("SECRET SQL diagnostic")
	w := call(r, http.MethodGet, "admin.localhost:8080", "/api/v1/admin/overview", staff)
	if w.Code != http.StatusServiceUnavailable || !strings.Contains(w.Body.String(), "ADMIN_OVERVIEW_UNAVAILABLE") || strings.Contains(w.Body.String(), "SECRET") {
		t.Fatalf("SQL leak: %d %s", w.Code, w.Body.String())
	}
	overview.err = nil
	audit.appendErr = errors.New("audit write failed")
	w = call(r, http.MethodGet, "admin.localhost:8080", "/api/v1/admin/overview", staff)
	if w.Code != http.StatusServiceUnavailable || !strings.Contains(w.Body.String(), "ADMIN_AUDIT_UNAVAILABLE") || overview.calls != 1 {
		t.Fatal("audit failure did not block overview")
	}
}

func TestDirectOverviewHandlerRejectsUnverifiedPrincipal(t *testing.T) {
	r := gin.New()
	r.GET("/overview", New(application.NewWithRepositories(&fakeAuditStore{}, &fakeOverviewStore{})).Overview)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/overview?staffId=fake&role=admin", nil)
	req.Host = "admin.localhost:8080"
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("unguarded overview trusted caller: %d", w.Code)
	}
}
