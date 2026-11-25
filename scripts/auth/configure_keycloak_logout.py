#!/usr/bin/env python3
"""AUTH-08: configure local two-realm BFF back-channel logout (idempotent).

Updates only the two known clients in a loopback Keycloak. This is mandatory
for existing Keycloak PostgreSQL volumes: realm import is NOT a migration.
Does not enable User Login or create/disable any user accounts.
"""
import argparse
import urllib.error
from configure_user_sms_flow import Admin, BASE, ROOT, secrets_from_env

CLIENTS = {
    "theia-users": ("theia-users-bff", "http://app.localhost:8080/", "http://api:8080/auth/user/backchannel-logout"),
    "theia-staff": ("theia-staff-bff", "http://admin.localhost:8080/", "http://api:8080/auth/staff/backchannel-logout"),
}


class RealmAdmin(Admin):
    def __init__(self, username, password, realm):
        if realm not in CLIENTS:
            raise ValueError("unsupported realm")
        super().__init__(BASE, username, password)
        self.realm = realm

    def call(self, method, path, payload=None):
        return self._request(method, "/admin/realms/" + self.realm + path, payload, self.headers)


def configure(admin, realm, apply=False):
    client_name, post_logout, backchannel_url = CLIENTS[realm]
    matches = admin.call("GET", "/clients?clientId=" + client_name)
    if len(matches) != 1 or matches[0].get("clientId") != client_name:
        raise RuntimeError("expected exactly one fixed BFF client")
    client_id = matches[0].get("id")
    if not isinstance(client_id, str) or not client_id:
        raise RuntimeError("missing Keycloak client ID")
    path = "/clients/" + client_id
    client = admin.call("GET", path)
    expected_callback = ("http://app.localhost:8080/auth/user/callback" if realm == "theia-users"
                         else "http://admin.localhost:8080/auth/staff/callback")
    if (client.get("protocol") != "openid-connect" or client.get("publicClient") is not False
            or client.get("standardFlowEnabled") is not True
            or client.get("directAccessGrantsEnabled") is not False
            or client.get("implicitFlowEnabled") is not False
            or client.get("serviceAccountsEnabled") is not False
            or client.get("redirectUris") != [expected_callback]
            or client.get("webOrigins", []) != []
            or (client.get("attributes") or {}).get("pkce.code.challenge.method") != "S256"
            or bool(client.get("authenticationFlowBindingOverrides"))):
        raise RuntimeError("client OIDC configuration changed; refuse mutation")
    desired = {
        "post.logout.redirect.uris": post_logout,
        "backchannel.logout.url": backchannel_url,
        "backchannel.logout.session.required": "true",
        "backchannel.logout.revoke.offline.tokens": "false",
    }
    changed = client.get("frontchannelLogout") is not False or any(
        (client.get("attributes") or {}).get(k) != v for k, v in desired.items()
    )
    if changed and apply:
        client["frontchannelLogout"] = False
        attrs = dict(client.get("attributes") or {})
        attrs.update(desired)
        client["attributes"] = attrs
        admin.call("PUT", path, client)
    current = admin.call("GET", path)
    verified = current.get("frontchannelLogout") is False and all(
        (current.get("attributes") or {}).get(k) == v for k, v in desired.items()
    )
    if apply and not verified:
        raise RuntimeError("Keycloak logout configuration did not persist")
    return verified


def main():
    parser = argparse.ArgumentParser(description="Validate local Theia Keycloak logout callbacks")
    parser.add_argument("--apply", action="store_true", help="Write fixed local logout client settings")
    args = parser.parse_args()
    env = secrets_from_env(ROOT / ".env.auth")
    for realm in CLIENTS:
        admin = RealmAdmin(env["KEYCLOAK_BOOTSTRAP_ADMIN_USERNAME"], env["KEYCLOAK_BOOTSTRAP_ADMIN_PASSWORD"], realm)
        valid = configure(admin, realm, args.apply)
        print(realm, "backchannel configured:", valid)
    if not args.apply:
        print("Dry check only; use --apply to configure existing Keycloak realms")


if __name__ == "__main__":
    try:
        main()
    except (OSError, ValueError, KeyError, RuntimeError, urllib.error.URLError) as exc:
        raise SystemExit(f"Local logout configuration failed: {type(exc).__name__}")
