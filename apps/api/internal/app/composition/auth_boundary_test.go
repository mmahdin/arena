package composition

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/config"
)

// A disposable ping-only driver tests readiness without starting SQL Server.
// No auth, user data or AI inference is invoked by these checks.
type readyTestDriver struct{}
type readyTestConn struct{}

func (readyTestDriver) Open(string) (driver.Conn, error) { return readyTestConn{}, nil }
func (readyTestConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("not implemented")
}
func (readyTestConn) Close() error              { return nil }
func (readyTestConn) Begin() (driver.Tx, error) { return nil, errors.New("not implemented") }
func (readyTestConn) Ping(context.Context) error {
	return nil
}

var readyTestDriverOnce sync.Once

func TestAUTH09ReadinessIndependentOfLoginConfiguration(t *testing.T) {
	readyTestDriverOnce.Do(func() { sql.Register("theia_ready_test", readyTestDriver{}) })
	for _, environment := range []string{"local", "staging"} {
		db, err := sql.Open("theia_ready_test", "")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = db.Close() })
		cfg := config.Config{Environment: environment}
		api, err := NewAPI(cfg, db)
		if err != nil {
			t.Fatal(err)
		}
		check := func(want int) {
			t.Helper()
			w := httptest.NewRecorder()
			api.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health/ready", nil))
			if w.Code != want {
				t.Fatalf("env=%s: readiness got %d, want %d: %s", environment, w.Code, want, w.Body.String())
			}
		}
		check(http.StatusServiceUnavailable) // no core
		api.core = &AIInternal{}             // injected test-only placeholder; not running
		check(http.StatusServiceUnavailable)
		api.healthy.Store(true) // runtime is reporting healthy (no model call)
		check(http.StatusOK)
		api.healthy.Store(false)
		check(http.StatusServiceUnavailable)
	}
}
