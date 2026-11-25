# AUTH-02: protected routes and AI runtime are no longer tied to Dev Login

**Scope/status:** intermediate backend refactor only. No OIDC code flow, SMS OTP,
SQL application sessions, staff identity, or Admin API is implemented here.
AUTH-03/04/06 will replace the intentionally unavailable authentication entry
points and the fail-closed user guard. Do not use this milestone to claim live
user authentication or release real mental-health content.

## HTTP route behavior

| Mode | `/api/v1/dev/*` | Existing `/api/v1/me`, chat, memory | `/auth/{user,staff}/*` | `/api/v1/admin/*` |
| --- | --- | --- | --- | --- |
| `ALLOW_DEV_LOGIN=true`, local/staging only | Legacy synthetic session routes | Existing signed/revocable **dev** guard | 503 `AUTH_NOT_CONFIGURED` | Not mounted (404) |
| `ALLOW_DEV_LOGIN=false` in any environment | Not mounted (404) | **Mounted but always 401 `SESSION_REQUIRED`** | 503 `AUTH_NOT_CONFIGURED` | Not mounted (404) |

The 401 guard is deliberately independent of fake phone numbers, old dev cookies,
caller-supplied `userId` fields, and Keycloak (which is not yet wired). Never
replace it with a pass-through middleware or default synthetic user. The
reserved OIDC routes create **no cookies, redirects, login transactions or
sessions**; their current OpenAPI contract only documents the 503 response.

## AI Core and readiness

* `AI_CORE_CONFIG_FILE` (if set) now loads the configured/pinned AI Core with
  `ALLOW_DEV_LOGIN=false`. If loading fails, startup still fails closed with
  sanitized `RuntimeSetupError` information. This change does not enable a
  prototype/mock reply path, unsigned production bundle, or proactive dispatch.
* `GET /health/ready` requires a successful SQL ping **and** a configured,
  running/healthy AI worker, whether Dev Login is enabled or not. Without the
  AI runtime, `/health/live` stays available while readiness returns 503. A
  deployed proxy/health-check system must not treat liveness as readiness.
* `AI_CORE_LOCAL_DRAFT=true` requires **all** of `APP_ENV=local`, a loopback
  `HTTP_ADDR`, and `AI_CORE_CONFIG_FILE`. `bundle.Read` must also verify the
  loaded draft is marked **synthetic_only**. The local draft is independent of
  `ALLOW_DEV_LOGIN`; this does **not** permit unreviewed drafts on public,
  staging, or production listeners.
* `AI_LOCAL_INSPECTOR=true` remains local + loopback + private-directory only.
  It collects highly sensitive local execution data; it is **not** a remote
  Back-Office feature and there is no HTTP route to inspect its files. A local
  inspector cannot grant authentication. Sensitive synthetic test execution
  still needs an authenticated test path; AUTH-05/06 will provide mock-SMS OIDC.

## BFF hosts and reverse-proxy contract (not wired yet)

The current local realm templates pin user callback
`http://app.localhost:8080/auth/user/callback` and staff callback
`http://admin.localhost:8080/auth/staff/callback`. Do not enable the user
client or provision staff until the corresponding secure implementation is
ready. Future browser/TWA deployments use **HTTPS** and independently allowlisted
origins. Their public endpoints should be same-origin with their respective
frontends (`/auth/*` and `/api/*` through a reviewed reverse proxy). No
wildcard callback URIs, cross-subdomain session cookies, or client-controlled
`X-Forwarded-Host`/`X-Forwarded-Proto` are acceptable.

AUTH-03 must implement verified host/origin allowlists, trusted-proxy
configuration, code+PKCE/state/nonce transaction checks, encrypted or omitted
server-side tokens, hashed opaque session verifiers, cookie separation, and
CSRF defenses. **This PR does not yet trust proxy headers or set Auth cookies.**

## Repeatable checks

From repository root:

```sh
python3 scripts/check_structure.py
python3 scripts/check_contract.py
python3 -m unittest discover -s scripts/tests -v
(cd apps/api && go build ./...)
(cd apps/api && go vet ./...)
(cd apps/api && go test ./...)
```

Go tests require the dependencies in `apps/api/go.mod` to be cached or
reachable. The new tests cover Dev Login on/off, a non-dev cookie failing closed,
reserved endpoints with no redirect/cookie, AI load on/off independently of
Dev Login, readiness as a function of DB+running core, and local-draft and
inspector environment guards. Real SQL Server / Keycloak integration tests
follow in AUTH-03 onward.
