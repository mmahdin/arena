"""AUTH-01 local infrastructure guardrails; no Docker daemon required.

A real Compose boot + realm inspection is still required on a Docker host.
"""

import importlib.util
import json
from pathlib import Path
import stat
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
REALMS = ROOT / "infra" / "keycloak" / "realms"
GENERATOR = ROOT / "scripts" / "auth" / "init_local_keycloak_env.py"
spec = importlib.util.spec_from_file_location("init_local_keycloak_env", GENERATOR)
bootstrap = importlib.util.module_from_spec(spec)
spec.loader.exec_module(bootstrap)

try:
    import yaml  # type: ignore
except ImportError:
    yaml = None


class KeycloakRealmTests(unittest.TestCase):
    def test_no_registration_or_imported_staff_or_users(self):
        for realm in ("theia-users", "theia-staff"):
            path = REALMS / (realm + "-realm.json")
            doc = json.loads(path.read_text(encoding="utf-8"))
            with self.subTest(realm=realm):
                self.assertEqual(doc["realm"], realm)
                self.assertTrue(doc["enabled"])
                self.assertFalse(doc["registrationAllowed"])
                self.assertTrue(doc["bruteForceProtected"])
                self.assertEqual(doc["sslRequired"].lower(), "external")
                self.assertFalse(doc.get("users"))
                self.assertFalse(doc.get("identityProviders"))
                self.assertFalse(doc.get("groups"))

    def test_confidential_bff_clients_and_exact_callback(self):
        for kind, host in (("users", "app"), ("staff", "admin")):
            realm = json.loads((REALMS / f"theia-{kind}-realm.json").read_text())
            self.assertEqual(len(realm["clients"]), 1)
            client = realm["clients"][0]
            self.assertEqual(client["clientId"], f"theia-{kind}-bff")
            self.assertFalse(client["publicClient"])
            self.assertEqual(client["clientAuthenticatorType"], "client-secret")
            self.assertEqual(client["secret"], "${THEIA_" + kind.upper() + "_BFF_CLIENT_SECRET}")
            self.assertEqual(client["redirectUris"], [f"http://{host}.localhost:8080/auth/{'user' if kind == 'users' else 'staff'}/callback"])
            self.assertEqual(client["webOrigins"], [])
            self.assertEqual(client["attributes"]["pkce.code.challenge.method"], "S256")
            self.assertTrue(client["standardFlowEnabled"])
            for not_allowed in ("implicitFlowEnabled", "directAccessGrantsEnabled", "serviceAccountsEnabled"):
                self.assertFalse(client[not_allowed])
            self.assertEqual(client["enabled"], kind == "staff")

    def test_staff_password_policy_and_no_public_reset(self):
        realm = json.loads((REALMS / "theia-staff-realm.json").read_text())
        self.assertIn("length(14)", realm["passwordPolicy"])
        self.assertFalse(realm["resetPasswordAllowed"])


class GeneratedCredentialsTests(unittest.TestCase):
    def test_generated_file_is_private_random_and_never_overwritten(self):
        with tempfile.TemporaryDirectory() as directory:
            destination = Path(directory) / ".env.auth"
            self.assertTrue(bootstrap.create_env(destination))
            before = destination.read_bytes()
            self.assertFalse(bootstrap.create_env(destination))
            self.assertEqual(destination.read_bytes(), before)
            if hasattr(stat, "S_IMODE") and __import__("os").name == "posix":
                self.assertEqual(stat.S_IMODE(destination.stat().st_mode), 0o600)
            values = dict(line.split("=", 1) for line in before.decode().splitlines() if line and not line.startswith("#"))
            for key in (
                "KEYCLOAK_DB_PASSWORD",
                "KEYCLOAK_BOOTSTRAP_ADMIN_PASSWORD",
                "THEIA_USERS_BFF_CLIENT_SECRET",
                "THEIA_STAFF_BFF_CLIENT_SECRET",
            ):
                self.assertGreaterEqual(len(values[key]), 40)
            self.assertEqual(len(set(values[key] for key in (
                "KEYCLOAK_DB_PASSWORD",
                "KEYCLOAK_BOOTSTRAP_ADMIN_PASSWORD",
                "THEIA_USERS_BFF_CLIENT_SECRET",
                "THEIA_STAFF_BFF_CLIENT_SECRET",
            ))), 4)

    def test_secret_template_is_blank(self):
        env = (ROOT / ".env.auth.example").read_text(encoding="utf-8")
        for name in (
            "KEYCLOAK_DB_PASSWORD", "KEYCLOAK_BOOTSTRAP_ADMIN_PASSWORD",
            "THEIA_USERS_BFF_CLIENT_SECRET", "THEIA_STAFF_BFF_CLIENT_SECRET",
        ):
            self.assertIn(f"{name}=\n", env)


@unittest.skipIf(yaml is None, "PyYAML not installed; use docker compose config --quiet")
class ComposeOverlayTests(unittest.TestCase):
    def test_opt_in_loopback_profile_and_pinned_versions(self):
        compose = yaml.safe_load((ROOT / "compose.auth.yaml").read_text(encoding="utf-8"))
        services = compose["services"]
        self.assertEqual(set(services), {"keycloak", "keycloak-postgres"})
        for service in services.values():
            self.assertEqual(service["profiles"], ["auth"])
        kc = services["keycloak"]
        self.assertEqual(kc["image"], "quay.io/keycloak/keycloak:26.8.2")
        self.assertEqual(services["keycloak-postgres"]["image"], "postgres:17.12")
        self.assertEqual(kc["ports"], ["127.0.0.1:8180:8180"])
        self.assertNotIn("ports", services["keycloak-postgres"])
        self.assertEqual(kc["depends_on"]["keycloak-postgres"]["condition"], "service_healthy")
        self.assertEqual(kc["networks"]["default"]["aliases"], ["auth.localhost"])
        self.assertTrue(kc["environment"]["KC_HOSTNAME"].startswith("http://auth.localhost:"))
        self.assertEqual(kc["command"], ["start-dev", "--import-realm"])
        self.assertIn("healthcheck", kc)


if __name__ == "__main__":
    unittest.main()
