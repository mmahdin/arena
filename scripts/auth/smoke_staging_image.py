#!/usr/bin/env python3
"""Boot the staging Auth image on disposable PostgreSQL; never sends SMS.

Uses random test credentials, a fake provider key, loopback ports and its own
Compose project. Only containers/volumes created by this script are removed.
"""
import argparse
import json
from pathlib import Path
import secrets
import subprocess
import tempfile
import time
import urllib.error
import urllib.parse
import urllib.request

import yaml

from prepare_staging_auth import AUTH_ORIGIN, USER_ORIGIN, STAFF_ORIGIN, INTERNAL_ORIGIN


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--image", default="theia-keycloak-staging:auth-kavenegar")
    args = parser.parse_args()
    project = "theia-auth-smoke-" + secrets.token_hex(5)
    with tempfile.TemporaryDirectory(prefix=project) as tmp:
        directory = Path(tmp)
        password = secrets.token_urlsafe(32)
        import base64
        values = {
            "KC_DB_URL": "jdbc:postgresql://postgres:5432/theia_keycloak", "KC_DB_USERNAME": "theia",
            "KC_DB_PASSWORD": password, "KC_HTTP_ENABLED": "true", "KC_HOSTNAME": AUTH_ORIGIN,
            "KC_CACHE": "ispn", "KC_HOSTNAME_STRICT": "true",
            "KC_BOOTSTRAP_ADMIN_USERNAME": "theia-smoke-admin", "KC_BOOTSTRAP_ADMIN_PASSWORD": password,
            "THEIA_SMS_ENV": "stage", "THEIA_SMS_TRANSPORT": "kavenegar",
            "THEIA_SMS_KAVENEGAR_TEMPLATE": "loginotp", "THEIA_SMS_KAVENEGAR_API_KEY": "TESTONLY" * 8,
            "THEIA_SMS_HMAC_KEY_BASE64": base64.b64encode(secrets.token_bytes(32)).decode(),
            "THEIA_USERS_BFF_CLIENT_SECRET": secrets.token_urlsafe(32),
            "THEIA_STAFF_BFF_CLIENT_SECRET": secrets.token_urlsafe(32),
            "AUTH_USER_ORIGIN": USER_ORIGIN, "AUTH_STAFF_ORIGIN": STAFF_ORIGIN,
            "THEIA_BACKEND_INTERNAL_ORIGIN": INTERNAL_ORIGIN,
        }
        compose = {"services": {
            "postgres": {"image": "postgres:17.12", "environment": {
                "POSTGRES_DB": "theia_keycloak", "POSTGRES_USER": "theia", "POSTGRES_PASSWORD": password}},
            "keycloak": {"image": args.image, "env_file": ["auth.env"],
                         "ports": ["127.0.0.1::8080"]},
        }}
        file = directory / "compose.yaml"
        file.write_text(yaml.safe_dump(compose)); file.chmod(0o600)

        def docker(*params):
            return subprocess.check_output(["docker", *params], stderr=subprocess.DEVNULL, text=True).strip()

        def dc(*params):
            return docker("compose", "-p", project, "-f", str(file), *params)

        try:
            dc("up", "-d", "postgres")
            env_file = directory / "auth.env"
            env_file.write_text("".join(f"{key}={value}\n" for key, value in values.items()))
            env_file.chmod(0o600)
            dc("up", "-d", "keycloak")
            origin = "http://" + dc("port", "keycloak", "8080")
            opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))

            def request(path, form=None, token=None):
                headers = {"Host": urllib.parse.urlsplit(AUTH_ORIGIN).netloc,
                           "X-Forwarded-Proto": "https", "X-Forwarded-Port": "443"}
                if token:
                    headers["Authorization"] = "Bearer " + token
                payload = None if form is None else urllib.parse.urlencode(form).encode()
                req = urllib.request.Request(origin + path, data=payload, headers=headers)
                with opener.open(req, timeout=5) as response:
                    return json.load(response)

            deadline = time.monotonic() + 150
            while True:
                try:
                    discovery = request("/realms/theia-users/.well-known/openid-configuration")
                    break
                except (OSError, ValueError):
                    if time.monotonic() >= deadline:
                        raise RuntimeError("Staging image did not become ready") from None
                    time.sleep(1)
            assert discovery["issuer"] == AUTH_ORIGIN + "/realms/theia-users"
            import re, html, http.cookies
            query = urllib.parse.urlencode({"client_id":"theia-users-bff", "response_type":"code", "scope":"openid", "redirect_uri":USER_ORIGIN+"/auth/user/callback", "state":"theia-synthetic-state", "nonce":"theia-synthetic-nonce", "code_challenge":"a"*43, "code_challenge_method":"S256"})
            headers={"Host":urllib.parse.urlsplit(AUTH_ORIGIN).netloc,"Origin":AUTH_ORIGIN}
            with opener.open(urllib.request.Request(origin+"/realms/theia-users/protocol/openid-connect/auth?"+query,headers=headers),timeout=10) as response:
                page=response.read().decode()
                cookies=http.cookies.SimpleCookie()
                for item in response.headers.get_all("Set-Cookie",[]):cookies.load(item)
            assert 'id="theia-phone"' in page, "Phone form unavailable without proxy headers"
            action=html.unescape(re.search(r'<form[^>]*action="([^"]+)"',page).group(1))
            url=urllib.parse.urlsplit(action)
            headers["Cookie"]="; ".join(k+"="+v.value for k,v in cookies.items())
            data=urllib.parse.urlencode({"smsAction":"request","phone":"not-a-phone"}).encode()
            with opener.open(urllib.request.Request(origin+url.path+"?"+url.query,data=data,headers=headers),timeout=10) as response:
                assert response.status==200 and 'id="theia-phone"' in response.read().decode(), "Phone form origin check failed"
            print("PASS: canonical HTTPS hostname handles real phone form GET and invalid POST without proxy-header trust; no SMS sent.",flush=True)
            token = request("/realms/master/protocol/openid-connect/token", {
                "grant_type": "password", "client_id": "admin-cli",
                "username": values["KC_BOOTSTRAP_ADMIN_USERNAME"], "password": password})["access_token"]
            prefix = "/admin/realms/theia-users"
            realm = request(prefix, token=token)
            assert realm["browserFlow"] == "theia-users-sms" and realm["sslRequired"] == "all"
            executions = request(prefix + "/authentication/flows/theia-users-sms/executions", token=token)
            assert len(executions) == 1
            assert executions[0]["providerId"] == "theia-sms-otp" and executions[0]["requirement"] == "REQUIRED"
            client = request(prefix + "/clients?clientId=theia-users-bff", token=token)[0]
            assert client["enabled"] and client["redirectUris"] == [USER_ORIGIN + "/auth/user/callback"]
            mapper = next(item for item in client["protocolMappers"] if item["name"] == "theia-sms-verification")
            assert mapper["protocolMapper"] == "oidc-usersessionmodel-note-mapper"
            assert mapper["config"]["user.session.note"] == "theia_sms_verified"
            profile = request(prefix + "/users/profile", token=token)
            assert all(not field.get("required") for field in profile["attributes"]
                       if field["name"] in {"email", "firstName", "lastName"})
            assert request("/realms/theia-staff/.well-known/openid-configuration")["issuer"] == AUTH_ORIGIN + "/realms/theia-staff"
            print("PASS: optimized staging image, real PostgreSQL, HTTPS issuers, SMS-only flow, exact callback, signed proof mapper and optional email/name.")
            print("Only fake provider credentials used; no SMS sent.")
        finally:
            dc("down", "-v", "--remove-orphans")


if __name__ == "__main__":
    try:
        main()
    except Exception as error:
        raise SystemExit(f"Staging image smoke failed ({type(error).__name__}); no private error payload printed.")
