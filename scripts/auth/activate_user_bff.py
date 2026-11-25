#!/usr/bin/env python3
"""AUTH-06: validate the local SMS-only Keycloak flow and opt in User BFF.

This script is intentionally restricted to the trusted loopback Keycloak
bootstrap connection used by AUTH-05. It never creates users or sends SMS.
"""
import argparse
from configure_user_sms_flow import Admin, BASE, FLOW, PROVIDER, ROOT, secrets_from_env

CLIENT_ID = "theia-users-bff"
REDIRECT = "http://app.localhost:8080/auth/user/callback"
MAPPER_NAME = "theia-sms-verification"
MAPPER_ID = "oidc-usersessionmodel-note-mapper"
OPTIONAL_USER_PROFILE_FIELDS = {"email", "firstName", "lastName"}
MAPPER_CONFIG = {
    "user.session.note": "theia_sms_verified",
    "claim.name": "theia_sms_verified",
    "jsonType.label": "String",
    "id.token.claim": "true",
    "access.token.claim": "false",
    "userinfo.token.claim": "false",
    "introspection.token.claim": "false",
}


def verify_realm(admin):
    realm = admin.call("GET", "")
    if (realm.get("browserFlow") != FLOW or realm.get("loginTheme") != "theia-sms"
            or realm.get("registrationAllowed") is not False
            or realm.get("resetPasswordAllowed") is not False
            or realm.get("bruteForceProtected") is not True):
        raise RuntimeError("User realm must use hardened SMS-only browser flow")
    executions = admin.call("GET", f"/authentication/flows/{FLOW}/executions")
    if (len(executions) != 1 or executions[0].get("providerId") != PROVIDER
            or executions[0].get("requirement") != "REQUIRED"
            or executions[0].get("authenticationFlow")):
        raise RuntimeError("SMS-only browser flow has unexpected executions")
    providers = admin.call("GET", "/authentication/authenticator-providers")
    if not any(item.get("id") == PROVIDER for item in providers):
        raise RuntimeError("Required SMS SPI is not installed")


def get_client(admin):
    matches = admin.call("GET", f"/clients?clientId={CLIENT_ID}")
    if len(matches) != 1 or matches[0].get("clientId") != CLIENT_ID:
        raise RuntimeError("Expected exactly one User BFF client")
    client_id = matches[0].get("id")
    if not isinstance(client_id, str) or not client_id:
        raise RuntimeError("User BFF client identifier unavailable")
    client = admin.call("GET", f"/clients/{client_id}")
    if (client.get("protocol") != "openid-connect" or client.get("publicClient") is not False
            or client.get("standardFlowEnabled") is not True
            or client.get("directAccessGrantsEnabled") is not False
            or client.get("implicitFlowEnabled") is not False
            or client.get("serviceAccountsEnabled") is not False
            or bool(client.get("authenticationFlowBindingOverrides"))
            or client.get("redirectUris") != [REDIRECT]
            or client.get("attributes", {}).get("pkce.code.challenge.method") != "S256"):
        raise RuntimeError("User BFF OIDC client is not confidential PKCE-only")
    return client_id, client


def verify_mapper(mappers):
    # Refuse collisions, extra mappers exposing or forging the same claim,
    # and static-claim mappers which could turn a password login into proof.
    for item in mappers:
        config = item.get("config") or {}
        if item.get("name") == MAPPER_NAME or config.get("claim.name") == "theia_sms_verified":
            if (item.get("name") != MAPPER_NAME or item.get("protocol") != "openid-connect"
                    or item.get("protocolMapper") != MAPPER_ID
                    or any(config.get(k) != v for k, v in MAPPER_CONFIG.items())):
                raise RuntimeError("SMS proof mapper mismatch or collision")
    matching = [m for m in mappers if m.get("name") == MAPPER_NAME]
    if len(matching) > 1:
        raise RuntimeError("Duplicate SMS proof protocol mapper")
    return bool(matching)


def verify_phone_only_profile(admin, apply=False):
    path = "/users/profile"
    profile = admin.call("GET", path)
    required = [
        field for field in profile["attributes"]
        if field["name"] in OPTIONAL_USER_PROFILE_FIELDS and field.get("required")
    ]
    if not required:
        return
    if not apply:
        raise RuntimeError("User realm still requires email or name; run with --apply")
    if any(field["required"] != {"roles": ["user"]} for field in required):
        raise RuntimeError("User realm has unexpected email or name requirements")
    for field in required:
        del field["required"]
    admin.call("PUT", path, profile)
    verify_phone_only_profile(admin)


def activate(admin, apply=False):
    verify_realm(admin)
    client_id, client = get_client(admin)
    path = f"/clients/{client_id}/protocol-mappers/models"
    mappers = admin.call("GET", path)
    present = verify_mapper(mappers)
    if not present and not apply:
        raise RuntimeError("SMS proof mapper missing; run with --apply")
    verify_phone_only_profile(admin, apply)
    if not present:
        admin.call("POST", path, {
            "name": MAPPER_NAME, "protocol": "openid-connect",
            "protocolMapper": MAPPER_ID, "consentRequired": False,
            "config": MAPPER_CONFIG,
        })
    mappers = admin.call("GET", path)
    if not verify_mapper(mappers):
        raise RuntimeError("SMS proof mapper installation not verified")
    if apply and client.get("enabled") is not True:
        client["enabled"] = True
        admin.call("PUT", f"/clients/{client_id}", client)
    _, final = get_client(admin)
    if apply and final.get("enabled") is not True:
        raise RuntimeError("User BFF client not enabled")
    # The only evidence asserted here is configuration: live Mock SMS and
    # OIDC/cookie/SQL tests are independent prerequisites to enabling Go.
    return bool(final.get("enabled"))


def main():
    parser = argparse.ArgumentParser(description="Inspect/activate the LOCAL SMS-only User BFF client")
    parser.add_argument("--apply", action="store_true", help="Install mapper and enable User client")
    args = parser.parse_args()
    env = secrets_from_env(ROOT / ".env.auth")
    admin = Admin(BASE, env["KEYCLOAK_BOOTSTRAP_ADMIN_USERNAME"], env["KEYCLOAK_BOOTSTRAP_ADMIN_PASSWORD"])
    enabled = activate(admin, args.apply)
    print("SMS User realm and signed proof mapper verified; client enabled:", enabled)
    print("Go User login remains OFF until AUTH_USER_SMS_FLOW_VERIFIED and AUTH_USER_LOGIN_ENABLED are set.")


if __name__ == "__main__":
    try:
        main()
    except (OSError, KeyError, ValueError, RuntimeError) as exc:
        raise SystemExit(f"User BFF activation refused: {type(exc).__name__}: {str(exc)[:120]}")
