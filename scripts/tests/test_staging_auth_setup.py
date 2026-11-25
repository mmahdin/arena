"""Credential lifecycle and realm invariants for dedicated Theia staging auth."""
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

import yaml

ROOT = Path(__file__).resolve().parents[2]
spec = importlib.util.spec_from_file_location("staging_auth", ROOT / "scripts/auth/prepare_staging_auth.py")
setup = importlib.util.module_from_spec(spec)
spec.loader.exec_module(setup)


class StagingAuthSetup(unittest.TestCase):
    def test_credentials_are_private_stable_and_scoped_to_theia(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            provider = root / "provider.env"
            provider.write_text("THEIA_SMS_KAVENEGAR_API_KEY=" + "A" * 32 + "\n")
            output = root / "private"
            setup.prepare(output, provider, "loginotp", "10.42.0.0/16")
            first = (output / "credentials.env").read_bytes()
            setup.prepare(output, provider, "loginotp", "10.43.0.0/16")
            self.assertEqual(first, (output / "credentials.env").read_bytes())
            self.assertEqual(output.stat().st_mode & 0o777, 0o700)
            for file in output.iterdir():
                self.assertEqual(file.stat().st_mode & 0o777, 0o600)
            config = yaml.safe_load((output / "auth-configmap.yaml").read_text())["data"]
            self.assertEqual(config["THEIA_SMS_TRANSPORT"], "kavenegar")
            self.assertNotIn("KC_PROXY_TRUSTED_ADDRESSES", config)
            self.assertNotIn("KC_PROXY_HEADERS", config)
            self.assertNotIn("THEIA_SMS_KAVENEGAR_API_KEY", config)
            self.assertNotIn("start-dev", str(config))
            backend = yaml.safe_load((output / "backend-configmap.patch.yaml").read_text())
            self.assertEqual(set(backend), {"data"})
            self.assertFalse(any(key.startswith(("DATABASE_", "AI_", "MODEL_")) for key in backend["data"]))
            self.assertEqual(backend["data"]["AUTH_TRUSTED_PROXY_CIDRS"], "10.43.0.0/16")
            auth_secret = yaml.safe_load((output / "auth-secret.yaml").read_text())["stringData"]
            backend_secret = yaml.safe_load((output / "backend-secret.patch.yaml").read_text())["data"]
            for key in ("THEIA_USERS_BFF_CLIENT_SECRET", "THEIA_STAFF_BFF_CLIENT_SECRET"):
                self.assertEqual(setup.base64.b64decode(backend_secret[key]).decode(), auth_secret[key])
            self.assertNotEqual(auth_secret["THEIA_USERS_BFF_CLIENT_SECRET"], auth_secret["THEIA_STAFF_BFF_CLIENT_SECRET"])

    def test_existing_partial_credentials_and_symlinks_are_not_overwritten(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp); provider = root / "provider.env"
            provider.write_text("THEIA_SMS_KAVENEGAR_API_KEY=" + "A" * 32 + "\n")
            output = root / "private"; output.mkdir(mode=0o700)
            existing = output / "credentials.env"; existing.write_text("KEYCLOAK_DB_PASSWORD=keep\n")
            with self.assertRaises(ValueError):
                setup.prepare(output, provider, "loginotp", "127.0.0.1/32")
            self.assertEqual(existing.read_text(), "KEYCLOAK_DB_PASSWORD=keep\n")
            link = output / "link"; link.symlink_to(existing)
            with self.assertRaises(ValueError):
                setup.private_write(link, "overwrite")

    def test_apply_refuses_rotation_of_existing_database_or_client_credentials(self):
        with tempfile.TemporaryDirectory() as tmp:
            output = Path(tmp)
            desired = {"POSTGRES_PASSWORD": "original"}
            (output / "postgres-secret.yaml").write_text(yaml.safe_dump({"stringData": desired}))
            existing = json.dumps({"data": {"POSTGRES_PASSWORD": setup.base64.b64encode(b"different").decode()}})
            with patch.object(setup, "kubectl", return_value=existing):
                with self.assertRaises(ValueError):
                    setup.verify_existing_credentials("explicit-stage", output)

    def test_rejects_unbounded_or_public_proxy_trust(self):
        for invalid in ("0.0.0.0/0", "::/0", "8.8.8.8/32", "", "not-a-network"):
            with self.assertRaises(ValueError):
                setup.proxy_networks(invalid)
        self.assertEqual(setup.proxy_networks("10.42.1.1/16,10.42.0.0/16"), "10.42.0.0/16")

    def test_default_proxy_discovery_uses_only_ready_frontend_peers(self):
        pods = {"items": [
            {"status": {"conditions": [{"type": "Ready", "status": "True"}], "podIPs": [{"ip": "10.84.5.53"}]}},
            {"status": {"conditions": [{"type": "Ready", "status": "False"}], "podIPs": [{"ip": "10.84.5.54"}]}},
            {"metadata": {"deletionTimestamp": "2026-11-25T00:00:00Z"}, "status": {"conditions": [{"type": "Ready", "status": "True"}], "podIPs": [{"ip": "10.84.5.55"}]}},
        ]}
        with patch.object(setup, "kubectl", return_value=json.dumps(pods)) as read:
            self.assertEqual(setup.discovered_proxies("stage"), "10.84.5.53/32")
            self.assertIn("app=ai-therapist-frontend", read.call_args.args)
            self.assertNotIn("nodes", read.call_args.args)

    def test_import_requires_sms_proof_pkce_and_exact_https_origins(self):
        for kind in ("users", "staff"):
            realm = json.loads((ROOT / f"infra/keycloak/realms/staging/theia-{kind}-realm.json").read_text())
            self.assertEqual(realm["sslRequired"], "all")
            self.assertFalse(realm["registrationAllowed"])
            self.assertNotIn("users", realm)
            client = realm["clients"][0]
            self.assertTrue(client["enabled"])
            self.assertFalse(client["publicClient"])
            self.assertFalse(client["directAccessGrantsEnabled"])
            self.assertEqual(client["attributes"]["pkce.code.challenge.method"], "S256")
            self.assertTrue(client["secret"].startswith("${THEIA_"))
            self.assertEqual(len(client["redirectUris"]), 1)
            self.assertNotIn("localhost", str(realm))
            if kind == "users":
                flow = realm["authenticationFlows"][0]
                self.assertEqual(len(flow["authenticationExecutions"]), 1)
                execution = flow["authenticationExecutions"][0]
                self.assertEqual(execution["authenticator"], "theia-sms-otp")
                self.assertEqual(execution["requirement"], "REQUIRED")
                mapper = client["protocolMappers"][0]
                self.assertEqual(mapper["protocolMapper"], "oidc-usersessionmodel-note-mapper")
                self.assertEqual(mapper["config"]["user.session.note"], "theia_sms_verified")
                self.assertEqual(mapper["config"]["access.token.claim"], "false")


if __name__ == "__main__":
    unittest.main()
