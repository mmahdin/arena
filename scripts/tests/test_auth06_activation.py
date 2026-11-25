"""Safe, offline AUTH-06 configuration tests; no Keycloak network access."""
import copy
import sys
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "scripts" / "auth"))
from activate_user_bff import (MAPPER_CONFIG, MAPPER_ID, MAPPER_NAME, activate)


class FakeAdmin:
    def __init__(self):
        self.realm = {"browserFlow": "theia-users-sms", "loginTheme": "theia-sms",
                      "registrationAllowed": False, "resetPasswordAllowed": False,
                      "bruteForceProtected": True}
        self.executions = [{"providerId": "theia-sms-otp", "requirement": "REQUIRED", "authenticationFlow": False}]
        self.providers = [{"id": "theia-sms-otp"}]
        self.client = {"id": "one-uuid", "clientId": "theia-users-bff", "enabled": False,
                       "protocol": "openid-connect", "publicClient": False,
                       "standardFlowEnabled": True, "directAccessGrantsEnabled": False,
                       "implicitFlowEnabled": False, "serviceAccountsEnabled": False,
                       "redirectUris": ["http://app.localhost:8080/auth/user/callback"],
                       "attributes": {"pkce.code.challenge.method": "S256"}}
        self.mappers = []
        self.profile = {"attributes": [
            {"name": "username", "validations": {"length": {"min": 3}}},
            {"name": "email", "validations": {"email": {}}, "required": {"roles": ["user"]}},
            {"name": "firstName", "required": {"roles": ["user"]}},
            {"name": "lastName", "required": {"roles": ["user"]}},
        ]}
        self.calls = []

    def call(self, method, path, payload=None):
        self.calls.append((method, path))
        if method == "GET":
            if path == "": return copy.deepcopy(self.realm)
            if path.endswith("/executions"): return copy.deepcopy(self.executions)
            if path == "/authentication/authenticator-providers": return copy.deepcopy(self.providers)
            if path == "/users/profile": return copy.deepcopy(self.profile)
            if path.startswith("/clients?clientId="): return [copy.deepcopy(self.client)]
            if path == "/clients/one-uuid": return copy.deepcopy(self.client)
            if path.endswith("/protocol-mappers/models"): return copy.deepcopy(self.mappers)
        if method == "POST" and path.endswith("/protocol-mappers/models"):
            self.mappers.append(copy.deepcopy(payload)); return None
        if method == "PUT" and path == "/clients/one-uuid":
            self.client = copy.deepcopy(payload); return None
        if method == "PUT" and path == "/users/profile":
            self.profile = copy.deepcopy(payload); return None
        raise AssertionError(f"unknown admin action {method} {path}")


class ActivationTests(unittest.TestCase):
    def test_explicit_activation_and_idempotency(self):
        admin = FakeAdmin()
        with self.assertRaises(RuntimeError): activate(admin)
        self.assertFalse(admin.client["enabled"])
        self.assertTrue(activate(admin, apply=True))
        self.assertTrue(activate(admin))
        self.assertTrue(activate(admin, apply=True))
        self.assertEqual(1, len(admin.mappers))
        self.assertEqual("oidc-usersessionmodel-note-mapper", admin.mappers[0]["protocolMapper"])
        self.assertEqual(
            [{"name": "username", "validations": {"length": {"min": 3}}},
             {"name": "email", "validations": {"email": {}}},
             {"name": "firstName"}, {"name": "lastName"}],
            admin.profile["attributes"],
        )
        self.assertEqual(1, admin.calls.count(("PUT", "/users/profile")))

    def test_profile_drift_requires_explicit_activation(self):
        admin = FakeAdmin()
        self.assertTrue(activate(admin, apply=True))
        email = admin.profile["attributes"][1]
        email["required"] = {"roles": ["user"], "scopes": ["email"]}
        with self.assertRaisesRegex(RuntimeError, "requires email or name"):
            activate(admin)
        with self.assertRaisesRegex(RuntimeError, "unexpected email or name requirements"):
            activate(admin, apply=True)
        self.assertEqual({"roles": ["user"], "scopes": ["email"]}, email["required"])
        self.assertEqual(1, admin.calls.count(("PUT", "/users/profile")))

    def test_refuse_unapproved_paths_or_proof_mappers(self):
        for target, value in [
            ("realm.browserFlow", "browser"), ("client.publicClient", True),
            ("client.directAccessGrantsEnabled", True),
            ("client.authenticationFlowBindingOverrides", {"browser": "other-flow"}),
            ("client.redirectUris", ["*"]), ("client.attributes", {}),
            ("executions", [{"providerId": "auth-username-password-form", "requirement": "REQUIRED"}]),
        ]:
            with self.subTest(target=target):
                admin = FakeAdmin()
                if target == "executions": admin.executions = value
                else:
                    group, key = target.split(".", 1)
                    getattr(admin, group)[key] = value
                with self.assertRaises(RuntimeError): activate(admin, apply=True)
                self.assertFalse(admin.client.get("enabled", False))
        for mapper in [
            {"name": MAPPER_NAME, "protocolMapper": "oidc-hardcoded-claim-mapper", "config": MAPPER_CONFIG},
            {"name": "evil-static", "protocolMapper": "oidc-hardcoded-claim-mapper", "config": {"claim.name": "theia_sms_verified"}},
        ]:
            admin = FakeAdmin(); admin.mappers.append(mapper)
            with self.assertRaises(RuntimeError): activate(admin, apply=True)
            self.assertFalse(admin.client["enabled"])

    def test_invalid_proof_configuration_denied(self):
        admin = FakeAdmin()
        admin.mappers.append({"name": MAPPER_NAME, "protocol": "openid-connect",
                              "protocolMapper": MAPPER_ID,
                              "config": {**MAPPER_CONFIG, "id.token.claim": "false"}})
        with self.assertRaises(RuntimeError): activate(admin, apply=True)
        self.assertFalse(admin.client["enabled"])


if __name__ == "__main__": unittest.main()
