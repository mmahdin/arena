package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestPrivateMetricsOptionsFailClosed(t *testing.T) {
	allowed := Options{Enabled: true, Address: "127.0.0.1:9100", Environment: "local"}
	if err := Validate(allowed); err != nil {
		t.Fatal(err)
	}
	for _, addr := range []string{"0.0.0.0:9100", ":9100", "admin.localhost:9100", "127.0.0.1:0", "127.0.0.1:abc", "127.0.0.1:70000", "192.168.1.1:9100"} {
		allowed.Address = addr
		if err := Validate(allowed); err == nil {
			t.Errorf("accepted %q", addr)
		}
	}
	allowed.Address = "127.0.0.1:9100"
	allowed.Environment = "stage"
	if err := Validate(allowed); err == nil {
		t.Fatal("metrics enabled outside local")
	}
	allowed.Enabled = false
	if err := Validate(allowed); err != nil {
		t.Fatal("disabled metrics should be a no-op", err)
	}
}

func TestNoUntrustedPrometheusLabels(t *testing.T) {
	gin.SetMode(gin.TestMode)
	meter := New()
	router := gin.New()
	router.Use(meter.Middleware())
	router.GET("/test/:id", func(c *gin.Context) { c.JSON(500, gin.H{"error": "none"}) })
	router.GET("/health/live", func(c *gin.Context) { c.Status(200) })
	private := "secret-person-1234"
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/test/"+private+"?token="+private, nil))
	if w.Code != 500 {
		t.Fatal(w.Code)
	}
	for _, path := range []string{"/health/live", "/some/" + private} {
		router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", path, nil))
	}
	if label := RouteLabel("/test/" + private + "?q=1"); label != "unmatched" {
		t.Fatalf("untrusted route accepted: %q", label)
	}
	got := httptest.NewRecorder()
	meter.Handler().ServeHTTP(got, httptest.NewRequest("GET", "/metrics", nil))
	body := got.Body.String()
	if got.Code != http.StatusOK || !strings.Contains(body, "theia_http_requests_total") || !strings.Contains(body, "theia_http_request_duration_seconds") {
		t.Fatalf("missing family or failed: %d", got.Code)
	}
	if strings.Contains(body, private) || strings.Contains(body, "?token=") || strings.Contains(body, "/health/live") {
		t.Fatal("private or health data leaked")
	}
	if !strings.Contains(body, `route="/test/:id"`) || !strings.Contains(body, `status_class="5xx"`) {
		t.Fatal("expected bounded labels")
	}
}

func TestLLMAttemptCountersAndWorkersHaveFixedLabels(t *testing.T) {
	m := New()
	m.ObserveLLM("FAILED secretly", "other secret", time.Second, true, 100, 30, 10, 500)
	m.ObserveLLM("succeeded", "estimated", time.Second, true, 100, 30, 10, 500)
	m.WorkerHeartbeat("reflection", "running", time.Unix(123, 0))
	m.WorkerHeartbeat("PRIVATE JOB ID", "running", time.Unix(123, 0))
	got := httptest.NewRecorder()
	m.Handler().ServeHTTP(got, httptest.NewRequest("GET", "/metrics", nil))
	body := got.Body.String()
	for _, bad := range []string{"FAILED secretly", "other secret", "PRIVATE JOB ID", "turn_id", "user_id", "conversation_id"} {
		if strings.Contains(body, bad) {
			t.Fatalf("unapproved value %q", bad)
		}
	}
	for _, expected := range []string{`status="unknown"`, `status="succeeded"`, `direction="input"`, `kind="reflection"`, `theia_llm_estimated_cost_micro_usd_total 500`} {
		if !strings.Contains(body, expected) {
			t.Fatalf("missing expected %q", expected)
		}
	}
}

// A configured AI Core has exactly three expected, code-owned worker kinds.
// The initial expected marker must exist before any heartbeat; otherwise a
// worker that never started cannot be alerted on. No marker is emitted by a
// metrics-only API instance lacking AI Core.
func TestExpectedWorkerKindsRequireExplicitCoreRegistration(t *testing.T) {
	m := New()
	read := func() string {
		out := httptest.NewRecorder()
		m.Handler().ServeHTTP(out, httptest.NewRequest("GET", "/metrics", nil))
		if out.Code != http.StatusOK {
			t.Fatalf("metrics: %d", out.Code)
		}
		return out.Body.String()
	}
	if strings.Contains(read(), "theia_worker_expected{") {
		t.Fatal("unconfigured AI Core must not emit worker expectations")
	}
	m.ExpectWorkers()
	got := read()
	for _, kind := range []string{"memory", "reflection", "scheduler"} {
		if !strings.Contains(got, "theia_worker_expected{kind=\""+kind+"\"} 1") {
			t.Fatalf("missing expected worker: %s", kind)
		}
	}
	if strings.Contains(got, "theia_worker_heartbeat_success_timestamp_seconds{") {
		t.Fatal("expected kinds should not fabricate a successful SQL heartbeat")
	}
}
