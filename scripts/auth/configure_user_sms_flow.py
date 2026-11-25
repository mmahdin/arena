#!/usr/bin/env python3
"""Opt-in, idempotent AUTH-05 Keycloak realm flow activation (NOT user client).

Only accepts loopback auth.localhost, uses existing .env.auth bootstrap admin,
refuses incompatible flow aliases and never enables the User BFF client.
"""
import argparse
import ipaddress
import socket
import json
from pathlib import Path
import urllib.error
import urllib.parse
import urllib.request

ROOT = Path(__file__).resolve().parents[2]
BASE = "http://auth.localhost:8180"
FLOW = "theia-users-sms"
PROVIDER = "theia-sms-otp"


def secrets_from_env(path: Path) -> dict[str, str]:
    values = {}
    for line in path.read_text(encoding="utf-8").splitlines():
        if not line or line.startswith("#") or "=" not in line:
            continue
        name, value = line.split("=", 1)
        values[name] = value
    for key in ("KEYCLOAK_BOOTSTRAP_ADMIN_USERNAME", "KEYCLOAK_BOOTSTRAP_ADMIN_PASSWORD"):
        if not values.get(key):
            raise ValueError("Bootstrap admin is not configured")
    return values


class Admin:
    def __init__(self, base: str, username: str, password: str):
        if base != BASE:
            raise ValueError("Only loopback local Keycloak is supported")
        addresses = socket.getaddrinfo("auth.localhost", 8180, type=socket.SOCK_STREAM)
        if not addresses or any(not ipaddress.ip_address(item[4][0]).is_loopback for item in addresses):
            raise ValueError("auth.localhost must resolve exclusively to loopback")
        payload = urllib.parse.urlencode({
            "grant_type": "password", "client_id": "admin-cli", "username": username, "password": password,
        }).encode("utf-8")
        token = self._request("POST", "/realms/master/protocol/openid-connect/token", payload, None)["access_token"]
        self.headers = {"Authorization": "Bearer " + token}
        self.base = base

    def _request(self, method: str, path: str, payload=None, headers=None):
        body = payload
        if payload is not None and isinstance(payload, dict):
            body = json.dumps(payload).encode("utf-8")
        req = urllib.request.Request(BASE + path, data=body, method=method, headers=headers or {})
        if payload is not None:
            req.add_header("Content-Type", "application/json" if isinstance(payload, dict) else "application/x-www-form-urlencoded")
        # Do not send local admin credentials through HTTP proxy environment
        # variables or follow unexpected redirects to another origin.
        class NoRedirect(urllib.request.HTTPRedirectHandler):
            def redirect_request(self, request, fp, code, msg, headers, newurl):
                return None
        opener = urllib.request.build_opener(urllib.request.ProxyHandler({}), NoRedirect())
        try:
            with opener.open(req, timeout=8) as result:
                raw = result.read(1048576)
                return json.loads(raw) if raw else None
        except urllib.error.HTTPError as exc:
            # Do not expose full URLs, payload, token, or potentially sensitive response text.
            raise RuntimeError(f"Keycloak Admin API failed (HTTP {exc.code})") from None

    def call(self, method: str, path: str, payload=None):
        return self._request(method, "/admin/realms/theia-users" + path, payload, self.headers)


def configure(admin: Admin):
    clients = admin.call("GET", "/clients?clientId=theia-users-bff")
    if len(clients) != 1 or clients[0].get("enabled") is not False:
        raise RuntimeError("User BFF must remain disabled during AUTH-05")
    providers = admin.call("GET", "/authentication/authenticator-providers")
    if not any(p.get("id") == PROVIDER for p in providers):
        raise RuntimeError("Theia SMS provider is not installed; rebuild Keycloak first")
    flows = admin.call("GET", "/authentication/flows")
    if not any(f.get("alias") == FLOW for f in flows):
        admin.call("POST", "/authentication/flows", {
            "alias": FLOW, "providerId": "basic-flow", "topLevel": True, "builtIn": False,
            "description": "Theia users: required SMS OTP only",
        })
    executions = admin.call("GET", "/authentication/flows/" + FLOW + "/executions")
    if not executions:
        admin.call("POST", "/authentication/flows/" + FLOW + "/executions/execution", {"provider": PROVIDER})
        executions = admin.call("GET", "/authentication/flows/" + FLOW + "/executions")
    if len(executions) != 1 or executions[0].get("providerId") != PROVIDER or executions[0].get("authenticationFlow"):
        raise RuntimeError("Unexpected authenticator in SMS flow: refusing to change browser binding")
    execution = executions[0]
    if execution.get("requirement") != "REQUIRED":
        admin.call("PUT", "/authentication/flows/" + FLOW + "/executions", {
            "id": execution["id"], "requirement": "REQUIRED",
        })
    executions = admin.call("GET", "/authentication/flows/" + FLOW + "/executions")
    if len(executions) != 1 or executions[0].get("providerId") != PROVIDER or executions[0].get("requirement") != "REQUIRED":
        raise RuntimeError("SMS flow verification failed")
    realm = admin.call("GET", "")
    if realm.get("browserFlow") != FLOW or realm.get("loginTheme") != "theia-sms":
        admin.call("PUT", "", {"browserFlow": FLOW, "loginTheme": "theia-sms"})
    final_realm = admin.call("GET", "")
    if final_realm.get("browserFlow") != FLOW or final_realm.get("loginTheme") != "theia-sms":
        raise RuntimeError("SMS realm browser binding not applied")
    clients = admin.call("GET", "/clients?clientId=theia-users-bff")
    if len(clients) != 1 or clients[0].get("enabled") is not False:
        raise RuntimeError("Unexpected client state: stop and investigate")


def main():
    parser = argparse.ArgumentParser(description="Set theia-users SMS-only browser flow (user BFF remains disabled)")
    parser.add_argument("--apply", action="store_true", help="Make changes to local Keycloak")
    args = parser.parse_args()
    if not args.apply:
        print("Dry run: no changes. Apply only after custom image is healthy: --apply")
        return
    values = secrets_from_env(ROOT / ".env.auth")
    configure(Admin(BASE, values["KEYCLOAK_BOOTSTRAP_ADMIN_USERNAME"], values["KEYCLOAK_BOOTSTRAP_ADMIN_PASSWORD"]))
    print("SMS-only user realm flow configured. User BFF client is still DISABLED (AUTH-06 gate).")


if __name__ == "__main__":
    try: main()
    except (OSError, KeyError, ValueError, RuntimeError, urllib.error.URLError) as exc:
        # Never print request, token, OTP, provider URL, or HTTP response body.
        raise SystemExit(f"SMS flow setup failed: {type(exc).__name__}: {str(exc)[:120]}")
