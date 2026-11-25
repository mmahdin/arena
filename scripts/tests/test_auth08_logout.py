"""Offline AUTH-08 logout configuration / API contract regression checks."""
import copy
import json
import sys
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "scripts" / "auth"))
from configure_keycloak_logout import CLIENTS, configure


class FakeRealmAdmin:
    def __init__(self, realm):
        self.realm = realm
        self.client = {
            "id": "opaque-id", "clientId": CLIENTS[realm][0], "protocol": "openid-connect",
            "publicClient": False, "standardFlowEnabled": True,
            "directAccessGrantsEnabled": False, "implicitFlowEnabled": False,
            "serviceAccountsEnabled": False, "frontchannelLogout": False,
            "redirectUris": ["http://app.localhost:8080/auth/user/callback" if realm == "theia-users"
                             else "http://admin.localhost:8080/auth/staff/callback"],
            "attributes": {"pkce.code.challenge.method": "S256"},
        }
        self.puts = 0

    def call(self, method, path, payload=None):
        if method == "GET" and path.startswith("/clients?clientId="):
            return [copy.deepcopy(self.client)]
        if method == "GET" and path == "/clients/opaque-id":
            return copy.deepcopy(self.client)
        if method == "PUT" and path == "/clients/opaque-id":
            self.client = copy.deepcopy(payload)
            self.puts += 1
            return None
        raise AssertionError("Unexpected admin request")


class LogoutContractTests(unittest.TestCase):
    def test_imported_clients_are_private_backend_webhook_only(self):
        for kind, (client, redirect, callback) in CLIENTS.items():
            with self.subTest(realm=kind):
                data = json.loads((ROOT / "infra/keycloak/realms" / (kind + "-realm.json")).read_text())
                self.assertEqual(data["realm"], kind)
                c = data["clients"][0]
                self.assertEqual(client, c["clientId"])
                self.assertFalse(c["frontchannelLogout"])
                self.assertEqual(redirect, c["attributes"]["post.logout.redirect.uris"])
                self.assertEqual(callback, c["attributes"]["backchannel.logout.url"])
                self.assertEqual("true", c["attributes"]["backchannel.logout.session.required"])
                self.assertEqual("false", c["attributes"]["backchannel.logout.revoke.offline.tokens"])
                self.assertTrue(callback.startswith("http://api:8080/auth/"))
                self.assertNotIn("*", redirect + callback)

    def test_local_config_script_is_explicit_and_idempotent(self):
        for realm in CLIENTS:
            with self.subTest(realm=realm):
                admin = FakeRealmAdmin(realm)
                self.assertFalse(configure(admin, realm, apply=False))
                self.assertEqual(0, admin.puts)
                self.assertTrue(configure(admin, realm, apply=True))
                self.assertEqual(1, admin.puts)
                self.assertTrue(configure(admin, realm, apply=True))
                self.assertEqual(1, admin.puts)
                self.assertEqual("S256", admin.client["attributes"]["pkce.code.challenge.method"])
                self.assertEqual(realm == "theia-staff", admin.client.get("enabled", realm == "theia-staff"))

    def test_refuse_insecure_client_before_writing(self):
        for attr, value in (("publicClient", True), ("directAccessGrantsEnabled", True),
                            ("authenticationFlowBindingOverrides", {"browser": "password"})):
            admin = FakeRealmAdmin("theia-users")
            admin.client[attr] = value
            with self.assertRaises(RuntimeError):
                configure(admin, "theia-users", apply=True)
            self.assertEqual(0, admin.puts)

    def test_contract_and_session_revival_fences(self):
        spec = (ROOT / "contracts/openapi/openapi.yaml").read_text()
        routing = (ROOT / "apps/api/internal/app/composition/api.go").read_text()
        sql = (ROOT / "apps/api/db/migrations/0031_auth_backchannel_logout.sql").read_text()
        repo = (ROOT / "apps/api/internal/modules/auth/adapters/mssql/repository.go").read_text()
        for kind in ("user", "staff"):
            for suffix in ("logout/sso", "logout-all", "backchannel-logout"):
                self.assertIn(f"/auth/{kind}/{suffix}", spec)
                self.assertIn(f"/auth/{kind}/{suffix}", routing)
        self.assertIn("auth_backchannel_events", sql)
        self.assertIn("auth_logout_fences", sql)
        self.assertIn("auth_principal_logout_watermarks", sql)
        self.assertIn("NOT EXISTS", repo)
        self.assertNotIn("access_token", sql)
        self.assertNotIn("refresh_token", sql)


if __name__ == "__main__":
    unittest.main()
