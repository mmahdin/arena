// Package metrics provides bounded, metadata-only Prometheus instruments.
// It never accepts user IDs, turn/job IDs, prompt text or untrusted URL paths.
package metrics

import (
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Options allow local development scraping only. The HTTP API listener is
// separate: never mount /metrics on the user-facing or staff-facing Gin router.
type Options struct {
	Enabled     bool
	Address     string
	Environment string
}

func Validate(opt Options) error {
	if !opt.Enabled {
		return nil
	}
	if opt.Environment != "local" {
		return fmt.Errorf("THEIA_METRICS_ENABLED requires APP_ENV=local")
	}
	host, port, err := net.SplitHostPort(opt.Address)
	portNumber, portErr := strconv.ParseUint(port, 10, 16)
	if err != nil || portErr != nil || portNumber == 0 {
		return fmt.Errorf("THEIA_METRICS_ADDR must be a loopback host:port")
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("THEIA_METRICS_ADDR must be a numeric loopback address")
	}
	return nil
}

var routePattern = regexp.MustCompile(`^/[A-Za-z0-9_/.:*-]{1,159}$`)

// RouteLabel accepts only Gin's registered route template, NEVER Request.URL.
// The fallback is a constant, preventing attacker-controlled cardinality.
func RouteLabel(route string) string {
	if route == "" || !routePattern.MatchString(route) {
		return "unmatched"
	}
	return route
}

func methodLabel(method string) string {
	switch method {
	case "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS":
		return method
	default:
		return "OTHER"
	}
}

func statusClass(status int) string {
	switch {
	case status >= 200 && status < 300:
		return "2xx"
	case status >= 300 && status < 400:
		return "3xx"
	case status >= 400 && status < 500:
		return "4xx"
	case status >= 500 && status < 600:
		return "5xx"
	default:
		return "other"
	}
}

func fixedLabel(value string, allowed ...string) string {
	for _, v := range allowed {
		if value == v {
			return v
		}
	}
	return "unknown"
}

// Runtime owns a private registry, preventing third-party global registrations
// and avoiding accidental exporter collisions in tests or multiple API instances.
type Runtime struct {
	registry       *prometheus.Registry
	httpTotal      *prometheus.CounterVec
	httpDuration   *prometheus.HistogramVec
	httpInflight   prometheus.Gauge
	llmAttempts    *prometheus.CounterVec
	llmDuration    *prometheus.HistogramVec
	llmTokens      *prometheus.CounterVec
	llmCost        prometheus.Counter
	workerUp       *prometheus.GaugeVec
	workerExpected *prometheus.GaugeVec
	heartbeatAt    *prometheus.GaugeVec
}

func New() *Runtime {
	r := prometheus.NewRegistry()
	m := &Runtime{registry: r,
		httpTotal:      prometheus.NewCounterVec(prometheus.CounterOpts{Name: "theia_http_requests_total", Help: "HTTP requests excluding health probes, labeled only by route template, method and status class."}, []string{"route", "method", "status_class"}),
		httpDuration:   prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "theia_http_request_duration_seconds", Help: "Completed HTTP handler duration by route template and method.", Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10, 30, 60}}, []string{"route", "method"}),
		httpInflight:   prometheus.NewGauge(prometheus.GaugeOpts{Name: "theia_http_inflight_requests", Help: "Number of in-flight application HTTP requests."}),
		llmAttempts:    prometheus.NewCounterVec(prometheus.CounterOpts{Name: "theia_llm_attempts_total", Help: "Finalized physical provider attempts; best-effort operational counts, not billing."}, []string{"status", "cost_status"}),
		llmDuration:    prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "theia_llm_attempt_duration_seconds", Help: "Finalized provider attempt durations.", Buckets: []float64{.05, .1, .25, .5, 1, 2, 5, 10, 20, 40, 90, 180}}, []string{"status"}),
		llmTokens:      prometheus.NewCounterVec(prometheus.CounterOpts{Name: "theia_llm_reported_tokens_total", Help: "Reported token counts from finalized attempts; absent reports are not zero tokens."}, []string{"direction"}),
		llmCost:        prometheus.NewCounter(prometheus.CounterOpts{Name: "theia_llm_estimated_cost_micro_usd_total", Help: "Partial configured-price cost estimate in micro-USD; not invoiced usage."}),
		workerUp:       prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "theia_worker_running", Help: "Last reported worker state from successful SQL heartbeat; not proof of job progress."}, []string{"kind"}),
		workerExpected: prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "theia_worker_expected", Help: "AI workers expected when an AI Core is configured; not a liveness claim."}, []string{"kind"}),
		heartbeatAt:    prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "theia_worker_heartbeat_success_timestamp_seconds", Help: "Unix time of the latest successful SQL worker heartbeat."}, []string{"kind"}),
	}
	enabled := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "theia_metrics_enabled",
		Help: "Whether the private Theia metrics listener is enabled.",
	})
	enabled.Set(1)
	r.MustRegister(collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		enabled, m.httpTotal, m.httpDuration, m.httpInflight, m.llmAttempts, m.llmDuration, m.llmTokens, m.llmCost, m.workerUp, m.workerExpected, m.heartbeatAt)
	return m
}

func (m *Runtime) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{ErrorHandling: promhttp.HTTPErrorOnError})
}

// Middleware observes only application routes. It never captures request/response
// bodies, headers, query strings, client IPs or cookies.
func (m *Runtime) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.FullPath() == "/health/live" || c.FullPath() == "/health/ready" {
			c.Next()
			return
		}
		started := time.Now()
		m.httpInflight.Inc()
		defer m.httpInflight.Dec()
		c.Next()
		route := RouteLabel(c.FullPath())
		// Exclude health probes even if routing resolves after Next().
		if strings.HasPrefix(route, "/health/") {
			return
		}
		method := methodLabel(c.Request.Method)
		m.httpTotal.WithLabelValues(route, method, statusClass(c.Writer.Status())).Inc()
		m.httpDuration.WithLabelValues(route, method).Observe(time.Since(started).Seconds())
	}
}

func (m *Runtime) ObserveLLM(status, costStatus string, elapsed time.Duration, known bool, input, output, cached int, estimatedMicroUSD int64) {
	status = fixedLabel(status, "succeeded", "failed", "incomplete")
	costStatus = fixedLabel(costStatus, "estimated", "unpriced", "unknown")
	m.llmAttempts.WithLabelValues(status, costStatus).Inc()
	if elapsed >= 0 {
		m.llmDuration.WithLabelValues(status).Observe(elapsed.Seconds())
	}
	if known && input >= 0 && output >= 0 && cached >= 0 && cached <= input {
		m.llmTokens.WithLabelValues("input").Add(float64(input))
		m.llmTokens.WithLabelValues("output").Add(float64(output))
		m.llmTokens.WithLabelValues("cached").Add(float64(cached))
	}
	if known && costStatus == "estimated" && estimatedMicroUSD > 0 {
		m.llmCost.Add(float64(estimatedMicroUSD))
	}
}

// ExpectWorkers declares the configured worker kinds independently of their heartbeats.
func (m *Runtime) ExpectWorkers() {
	for _, kind := range []string{"memory", "reflection", "scheduler"} {
		m.workerExpected.WithLabelValues(kind).Set(1)
	}
}

func (m *Runtime) WorkerHeartbeat(kind, status string, at time.Time) {
	kind = fixedLabel(kind, "memory", "reflection", "scheduler")
	if kind == "unknown" {
		return
	}
	status = fixedLabel(status, "running", "stopped", "failed")
	if status == "unknown" {
		return
	}
	if status == "running" {
		m.workerUp.WithLabelValues(kind).Set(1)
	} else {
		m.workerUp.WithLabelValues(kind).Set(0)
	}
	m.heartbeatAt.WithLabelValues(kind).Set(float64(at.Unix()))
}
