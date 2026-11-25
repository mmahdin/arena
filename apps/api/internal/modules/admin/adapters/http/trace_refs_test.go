package httpadapter

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/admin/application"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/admin/domain"
	auth "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/auth"
)

type fakeTraceReferenceReader struct {
	calls int
	fail  error
}

func (r *fakeTraceReferenceReader) ListTraceReferences(_ context.Context, f domain.TraceReferenceFilter) (domain.TraceReferencePage, error) {
	r.calls++
	return domain.TraceReferencePage{TurnID: f.TurnID, Items: []domain.TraceReference{}}, r.fail
}

var _ application.TraceReferenceStore = (*fakeTraceReferenceReader)(nil)

func TestTraceReferencesHTTPGuardAuditAndFilters(t *testing.T) {
	id := uuid.New()
	reader := &fakeInspector{id: id}
	audit := &fakeAuditStore{}
	traceStore := &fakeTraceReferenceReader{}
	r, authStore := setupInspector(t, reader, audit, traceStore)
	path := "/api/v1/admin/turns/" + id.String() + "/trace-references"
	if w := call(r, http.MethodGet, "admin.localhost:8080", path); w.Code != 401 {
		t.Fatalf("anonymous: %d", w.Code)
	}
	user := login(t, r, auth.User)
	if w := call(r, http.MethodGet, "admin.localhost:8080", path, user); w.Code != 401 {
		t.Fatalf("user: %d", w.Code)
	}
	if traceStore.calls != 0 || len(audit.events) != 0 {
		t.Fatal("unauthorized read")
	}
	staff := login(t, r, auth.Staff)
	if w := call(r, http.MethodGet, "app.localhost:8080", path, staff); w.Code != 400 {
		t.Fatalf("wrong host: %d", w.Code)
	}
	bad := []string{path + "?pageSize=0", path + "?pageSize=51", path + "?cursor=xyz", path + "?pageSize=1&pageSize=2", path + "?status=failed", path + "?cursor=", "/api/v1/admin/turns/not-a-uuid/trace-references"}
	for _, p := range bad {
		w := call(r, http.MethodGet, "admin.localhost:8080", p, staff)
		if w.Code != 400 {
			t.Fatalf("invalid page %s: %d %s", p, w.Code, w.Body.String())
		}
	}
	if traceStore.calls != 0 || len(audit.events) != 0 {
		t.Fatal("invalid request reached privileged SQL")
	}
	w := call(r, http.MethodGet, "admin.localhost:8080", path+"?pageSize=10", staff)
	if w.Code != 200 || w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("Referrer-Policy") != "no-referrer" ||
		!strings.Contains(w.Body.String(), "\"items\":[]") || !strings.Contains(w.Body.String(), id.String()) || traceStore.calls != 1 ||
		len(audit.events) != 1 || audit.events[0].ActionCode != domain.ActionTurnTraceRefsRead || audit.events[0].ResourceID != id ||
		audit.events[0].RequestID.String() != w.Header().Get("X-Request-ID") {
		t.Fatalf("staff lookup or audit: %d %s %+v", w.Code, w.Body.String(), audit.events)
	}
	traceStore.fail = domain.ErrTurnNotFound
	if w = call(r, http.MethodGet, "admin.localhost:8080", path, staff); w.Code != 404 {
		t.Fatalf("missing Turn: %d", w.Code)
	}
	traceStore.fail = errors.New("PRIVATE SQL PASSWORD DO NOT LEAK")
	if w = call(r, http.MethodGet, "admin.localhost:8080", path, staff); w.Code != 503 || strings.Contains(w.Body.String(), "PRIVATE") {
		t.Fatalf("unsafe SQL leak: %d %s", w.Code, w.Body.String())
	}
	audit.appendErr = errors.New("PRIVATE audit database outage")
	count := traceStore.calls
	if w = call(r, http.MethodGet, "admin.localhost:8080", path, staff); w.Code != 503 || !strings.Contains(w.Body.String(), "ADMIN_AUDIT_UNAVAILABLE") || traceStore.calls != count {
		t.Fatal("audit outage returned data")
	}
	audit.appendErr = nil
	authStore.allowStaff = false
	if w = call(r, http.MethodGet, "admin.localhost:8080", path, staff); w.Code != 401 || traceStore.calls != count {
		t.Fatal("deactivated staff read")
	}
}

func TestTraceReferenceCursorDoesNotBecomePublicURL(t *testing.T) {
	f := domain.TraceReferenceFilter{TurnID: uuid.New(), Limit: 25, Cursor: &domain.InspectorCursor{At: time.Now().Add(-time.Hour), ID: uuid.New()}}
	if f.Validate() != nil {
		t.Fatal("valid cursor rejected")
	}
	if strings.Contains((domain.InspectorCursor{At: f.Cursor.At, ID: f.Cursor.ID}).Encode(), "/") {
		t.Fatal("cursor is not URL safe")
	}
}
