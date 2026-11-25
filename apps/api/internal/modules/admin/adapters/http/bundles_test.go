package httpadapter

import (
	"encoding/json"
	"errors"
	"net/http"
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

func setupBundleHTTP(t *testing.T) (*gin.Engine, *fakeStore, *fakeAuditStore) {
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
	r.GET("/auth/user/login", bff.Login(auth.User))
	r.GET("/auth/user/callback", bff.Callback(auth.User))
	r.GET("/auth/staff/login", bff.Login(auth.Staff))
	r.GET("/auth/staff/callback", bff.Callback(auth.Staff))
	audit := &fakeAuditStore{}
	svcAdmin := application.NewWithAudit(audit)
	synthetic := true
	svcAdmin.SetBundleInventory(domain.BundleInventory{Scope: "deployment_snapshot", AsOf: time.Now().UTC(),
		Active: &domain.BundleRecord{Slot: "active", Status: "synthetic_draft", Version: "kb@0.1.0", SyntheticOnly: &synthetic,
			Validation: domain.BundleValidation{Valid: true, WarningCodes: []string{"synthetic_only"}}}})
	adminAPI := r.Group("/api/v1/admin")
	adminAPI.Use(bff.RequireStaff())
	adminAPI.GET("/bundles", New(svcAdmin).Bundles)
	return r, store, audit
}

func TestBundleHTTPStaffBoundariesAndAudit(t *testing.T) {
	r, store, audit := setupBundleHTTP(t)
	endpoint := "/api/v1/admin/bundles"
	if w := call(r, http.MethodGet, "admin.localhost:8080", endpoint); w.Code != 401 {
		t.Fatalf("anonymous %d", w.Code)
	}
	user := login(t, r, auth.User)
	if w := call(r, http.MethodGet, "admin.localhost:8080", endpoint, user); w.Code != 401 {
		t.Fatalf("user %d", w.Code)
	}
	staff := login(t, r, auth.Staff)
	if w := call(r, http.MethodGet, "app.localhost:8080", endpoint, staff); w.Code != 400 {
		t.Fatalf("wrong origin %d", w.Code)
	}
	if w := call(r, http.MethodGet, "admin.localhost:8080", endpoint, &http.Cookie{Name: "wellbeing_dev_session", Value: "synthetic"}); w.Code != 401 {
		t.Fatalf("dev session %d", w.Code)
	}
	for _, path := range []string{endpoint + "?path=/etc/shadow", endpoint + "?version=kb%400.1.0", endpoint + "?", endpoint + "?candidate=1&candidate=2"} {
		if path == endpoint+"?" {
			continue
		} // empty query is no input
		if w := call(r, http.MethodGet, "admin.localhost:8080", path, staff); w.Code != 400 {
			t.Fatalf("untrusted query %s: %d", path, w.Code)
		}
	}
	if len(audit.events) != 0 {
		t.Fatal("unauthorized or invalid requests audited as authorized")
	}
	w := call(r, http.MethodGet, "admin.localhost:8080", endpoint, staff)
	if w.Code != 200 || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("staff %d %s", w.Code, w.Body.String())
	}
	var result domain.BundleInventory
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil || result.Active == nil || result.Active.Status != "synthetic_draft" || result.Active.SignatureVerified || result.AsOf.IsZero() {
		t.Fatalf("response %s: %v", w.Body.String(), err)
	}
	if len(audit.events) != 1 || audit.events[0].ActionCode != domain.ActionBundlesRead || audit.events[0].RequestID.String() != w.Header().Get("X-Request-ID") {
		t.Fatal("read not audited")
	}
	for _, private := range []string{"/etc/shadow", "bundleDirectory", "reviewer_id", "private_key", "prompts/", "test-subject", "test-keycloak-sid"} {
		if strings.Contains(w.Body.String(), private) {
			t.Fatalf("sensitive response %q", private)
		}
	}
	store.allowStaff = false
	if w := call(r, http.MethodGet, "admin.localhost:8080", endpoint, staff); w.Code != 401 || len(audit.events) != 1 {
		t.Fatal("inactive staff access")
	}
}

func TestBundleHTTPAuditDownAndDirectHandler(t *testing.T) {
	r, _, audit := setupBundleHTTP(t)
	staff := login(t, r, auth.Staff)
	audit.appendErr = errors.New("private SQL password")
	w := call(r, http.MethodGet, "admin.localhost:8080", "/api/v1/admin/bundles", staff)
	if w.Code != 503 || !strings.Contains(w.Body.String(), "ADMIN_AUDIT_UNAVAILABLE") || strings.Contains(w.Body.String(), "private SQL") {
		t.Fatalf("audit leak: %d %s", w.Code, w.Body.String())
	}
	bare := gin.New()
	bare.GET("/bundles", New(application.NewWithAudit(audit)).Bundles)
	if w := call(bare, http.MethodGet, "admin.localhost:8080", "/bundles"); w.Code != 401 {
		t.Fatalf("unguarded handler %d", w.Code)
	}
}
