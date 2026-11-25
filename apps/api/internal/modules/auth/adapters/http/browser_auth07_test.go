package httpadapter

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	auth "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/auth"
)

func TestAUTH07CallbackReturnToIsSealedAndSameOrigin(t *testing.T) {
	router, _ := setupHandler(t)
	for _, raw := range []string{
		"?return_to=" + url.QueryEscape("//evil.invalid"),
		"?return_to=" + url.QueryEscape("https://evil.invalid"),
		"?return_to=%2Fvalid&return_to=%2Fother",
		"?return_to=%2F%2Fevil.invalid",
		"?return_to=%ZZ",
	} {
		w := request(router, "GET", "/auth/user/login"+raw, "app.localhost:8080")
		if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "AUTH_RETURN_PATH_INVALID") || w.Header().Get("Set-Cookie") != "" {
			t.Fatalf("unsafe return_to accepted %q: %d %s", raw, w.Code, w.Body.String())
		}
	}
	start := request(router, "GET", "/auth/user/login?return_to=%2Fconversations%3Ftab%3Dmessages", "app.localhost:8080")
	if start.Code != http.StatusFound {
		t.Fatalf("begin: %d %s", start.Code, start.Body.String())
	}
	cookie := start.Result().Cookies()[0]
	// A forged callback return_to must not replace the sealed destination.
	callback := request(router, "GET", "/auth/user/callback?state="+url.QueryEscape(cookie.Value)+"&code=valid&return_to=https%3A%2F%2Fevil.invalid", "app.localhost:8080", cookie)
	if callback.Code != http.StatusSeeOther || callback.Header().Get("Location") != "/conversations?tab=messages" {
		t.Fatalf("callback redirect not transaction-bound: %d %q", callback.Code, callback.Header().Get("Location"))
	}
	var sessionCookie *http.Cookie
	for _, candidate := range callback.Result().Cookies() {
		if strings.Contains(candidate.Name, "_session") {
			sessionCookie = candidate
		}
	}
	if sessionCookie == nil {
		t.Fatal("missing session cookie")
	}
	response := request(router, "GET", "/auth/user/session", "app.localhost:8080", sessionCookie)
	if response.Code != http.StatusOK || response.Header().Get("Cache-Control") != "no-store" || response.Header().Get("Vary") != "Cookie" {
		t.Fatalf("session response not cache-safe: %d %q %q", response.Code, response.Header().Get("Cache-Control"), response.Header().Get("Vary"))
	}
}

func TestAUTH07BrowserSmokeIsHostBoundAndOptional(t *testing.T) {
	router, h := setupHandler(t)
	router.GET("/__auth-smoke/user", h.SmokePage(auth.User))
	router.GET("/__auth-smoke/staff", h.SmokePage(auth.Staff))
	router.GET("/__auth-smoke/smoke.js", h.SmokeJavaScript())
	if got := request(router, "GET", "/__auth-smoke/user", "app.localhost:8080"); got.Code != http.StatusNotFound {
		t.Fatalf("smoke unexpectedly enabled: %d", got.Code)
	}
	h.settings.BrowserSmokeEnabled = true
	page := request(router, "GET", "/__auth-smoke/user", "app.localhost:8080")
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), "return_to=%2F__auth-smoke%2Fuser") ||
		!strings.Contains(page.Header().Get("Content-Security-Policy"), "default-src 'none'") ||
		page.Header().Get("Cache-Control") != "no-store" || page.Header().Get("Set-Cookie") != "" {
		t.Fatalf("unsafe user smoke response: %d %s", page.Code, page.Body.String())
	}
	if got := request(router, "GET", "/__auth-smoke/user", "admin.localhost:8080"); got.Code != http.StatusBadRequest {
		t.Fatalf("cross-host smoke page accessible: %d", got.Code)
	}
	if got := request(router, "GET", "/__auth-smoke/staff", "admin.localhost:8080"); got.Code != http.StatusOK {
		t.Fatalf("staff smoke page inaccessible: %d", got.Code)
	}
	if got := request(router, "GET", "/__auth-smoke/smoke.js", "evil.invalid"); got.Code != http.StatusBadRequest {
		t.Fatalf("smoke script served for arbitrary host: %d", got.Code)
	}
	js := request(router, "GET", "/__auth-smoke/smoke.js", "app.localhost:8080")
	if js.Code != http.StatusOK || !strings.Contains(js.Body.String(), "credentials: 'same-origin'") || !strings.Contains(js.Header().Get("Content-Type"), "text/javascript") {
		t.Fatalf("smoke JS invalid: %d", js.Code)
	}
}

func TestAUTH07ProxyAcceptsOnlyConfiguredHostAndTrustedScheme(t *testing.T) {
	_, handler := setupHandler(t)
	handler.settings.UserOrigin = "https://app.example.test"
	handler.settings.TrustedProxyCIDRs = []netip.Prefix{netip.MustParsePrefix("10.42.0.0/16")}
	newReq := func(peer, host string, headers map[string][]string) *gin.Context {
		r := httptest.NewRequest(http.MethodGet, "/auth/user/login", nil)
		r.Host = host
		r.RemoteAddr = peer
		for name, values := range headers {
			for _, value := range values {
				r.Header.Add(name, value)
			}
		}
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = r
		return c
	}
	for _, tc := range []struct {
		name, peer, host string
		headers          map[string][]string
		want             bool
	}{
		{"trusted edge", "10.42.1.2:12345", "app.example.test", map[string][]string{"X-Forwarded-Proto": {"https"}, "X-Forwarded-Host": {"app.example.test"}}, true},
		{"untrusted peer spoof", "203.0.113.3:12345", "app.example.test", map[string][]string{"X-Forwarded-Proto": {"https"}}, false},
		{"wrong Host", "10.42.1.2:12345", "admin.example.test", map[string][]string{"X-Forwarded-Proto": {"https"}, "X-Forwarded-Host": {"app.example.test"}}, false},
		{"wrong forwarded Host", "10.42.1.2:12345", "app.example.test", map[string][]string{"X-Forwarded-Proto": {"https"}, "X-Forwarded-Host": {"attacker.test"}}, false},
		{"multiple forwarded proto", "10.42.1.2:12345", "app.example.test", map[string][]string{"X-Forwarded-Proto": {"https", "http"}}, false},
		{"comma joined proto", "10.42.1.2:12345", "app.example.test", map[string][]string{"X-Forwarded-Proto": {"https,http"}}, false},
		{"untrusted HTTP", "203.0.113.3:12345", "app.example.test", nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := handler.expectedRequest(newReq(tc.peer, tc.host, tc.headers), auth.User); got != tc.want {
				t.Fatalf("expectedRequest=%v, want=%v", got, tc.want)
			}
		})
	}
}
