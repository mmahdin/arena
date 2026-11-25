package httpadapter

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// Ambiguous callback parameters must not consume an OIDC transaction or
// establish a browser session. In particular, Gin's c.Query silently takes
// the first value unless the handler checks cardinality explicitly.
func TestBO10OIDCCallbackRejectsDuplicateParameters(t *testing.T) {
	for _, extra := range []string{
		"&state=attacker", "&code=attacker", "&error=access_denied", "&error=",
	} {
		t.Run(url.QueryEscape(extra), func(t *testing.T) {
			router, _ := setupHandler(t)
			start := request(router, http.MethodGet, "/auth/user/login", "app.localhost:8080")
			if start.Code != http.StatusFound || len(start.Result().Cookies()) == 0 {
				t.Fatalf("login setup: %d", start.Code)
			}
			cookie := start.Result().Cookies()[0]
			callback := "/auth/user/callback?state=" + url.QueryEscape(cookie.Value) + "&code=valid" + extra
			got := request(router, http.MethodGet, callback, "app.localhost:8080", cookie)
			if got.Code != http.StatusUnauthorized || !strings.Contains(got.Body.String(), "AUTH_CALLBACK_INVALID") {
				t.Fatalf("ambiguous OIDC callback %q accepted: %d %s", extra, got.Code, got.Body.String())
			}
			for _, c := range got.Result().Cookies() {
				if strings.Contains(c.Name, "session") && c.Value != "" {
					t.Fatalf("ambiguous OIDC callback issued session cookie: %+v", c)
				}
			}
		})
	}
}
