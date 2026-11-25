"""OBS-02 offline security/config and mock-HTTP tests.

These do NOT claim Docker startup, Phoenix OTLP compatibility or live Go tests.
"""
from __future__ import annotations

from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import io
import os
from pathlib import Path
import socket
import stat
import sys
import tempfile
import threading
import unittest
from contextlib import redirect_stdout

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "scripts/observability"))
import init_local_phoenix_env as env_tool
import run_native_with_phoenix as native_tool
import smoke_phoenix as smoke


class ComposeContractTest(unittest.TestCase):
    def test_isolated_loopback_authenticated_pinned_deployment(self):
        import yaml
        path = ROOT / "compose.observability.dev.yaml"
        doc = yaml.safe_load(path.read_text())
        self.assertEqual(set(doc["services"]), {"phoenix"})
        svc = doc["services"]["phoenix"]
        self.assertEqual(svc["image"], "arizephoenix/phoenix:version-22.0.1")
        self.assertEqual(svc["ports"], ["127.0.0.1:6006:6006"])
        self.assertEqual(svc["profiles"], ["observability"])
        self.assertEqual(svc["environment"]["PHOENIX_ENABLE_AUTH"], "true")
        self.assertEqual(svc["environment"]["PHOENIX_HOST"], "0.0.0.0")
        self.assertEqual(svc["environment"]["PHOENIX_ALLOW_EXTERNAL_RESOURCES"], "false")
        self.assertEqual(svc["environment"]["PHOENIX_TELEMETRY_ENABLED"], "false")
        self.assertIn("PHOENIX_SECRET", svc["environment"])
        self.assertIn("PHOENIX_DEFAULT_ADMIN_INITIAL_PASSWORD", svc["environment"])
        self.assertNotIn("THEIA_OTEL_CONTENT_API_KEY", svc["environment"])
        self.assertEqual(svc["security_opt"], ["no-new-privileges:true"])
        self.assertEqual(svc["cap_drop"], ["ALL"])
        self.assertEqual(len(svc["volumes"]), 1)
        self.assertIn("phoenix_data", doc["volumes"])

    def test_observability_secrets_ignored_and_empty_examples(self):
        self.assertIn(".env.observability\n", (ROOT / ".gitignore").read_text())
        values = (ROOT / ".env.observability.example").read_text()
        for name in ("PHOENIX_SECRET", "PHOENIX_DEFAULT_ADMIN_INITIAL_PASSWORD", "THEIA_OTEL_CONTENT_API_KEY"):
            self.assertIn(name + "=\n", values)
        self.assertNotIn("api_key", (ROOT / "compose.observability.dev.yaml").read_text().lower())


class SecretInitTest(unittest.TestCase):
    def setUp(self):
        self.dir = tempfile.TemporaryDirectory()
        self.addCleanup(self.dir.cleanup)
        self.path = Path(self.dir.name) / ".env.observability"

    def test_creates_owner_only_secret_and_never_overwrites(self):
        with redirect_stdout(io.StringIO()):
            self.assertTrue(env_tool.create_env(self.path))
            self.assertFalse(env_tool.create_env(self.path))
        self.assertEqual(stat.S_IMODE(self.path.stat().st_mode), 0o600)
        values = env_tool.read_env(self.path)
        self.assertGreaterEqual(len(values["PHOENIX_SECRET"]), 48)
        self.assertGreaterEqual(len(values["PHOENIX_DEFAULT_ADMIN_INITIAL_PASSWORD"]), 32)
        self.assertEqual(values["THEIA_OTEL_CONTENT_API_KEY"], "")
        self.assertNotEqual(values["PHOENIX_SECRET"], values["PHOENIX_DEFAULT_ADMIN_INITIAL_PASSWORD"])

    def test_rejects_symlink_and_group_readability(self):
        self.assertTrue(env_tool.create_env(self.path))
        self.path.chmod(0o644)
        with self.assertRaises(ValueError):
            env_tool.read_env(self.path)
        self.path.chmod(0o600)
        link = Path(self.dir.name) / "symlink"
        link.symlink_to(self.path)
        with self.assertRaises(ValueError):
            env_tool.create_env(link)
        with self.assertRaises(ValueError):
            env_tool.read_env(link)

    def test_ingestion_key_set_once_no_command_history(self):
        self.assertTrue(env_tool.create_env(self.path))
        with self.assertRaises(ValueError):
            env_tool.set_ingest_key(self.path, "unsafe-key")
        key = "phoenix_" + "Z" * 40
        env_tool.set_ingest_key(self.path, key)
        self.assertEqual(env_tool.read_env(self.path)["THEIA_OTEL_CONTENT_API_KEY"], key)
        self.assertEqual(stat.S_IMODE(self.path.stat().st_mode), 0o600)
        with self.assertRaises(ValueError):
            env_tool.set_ingest_key(self.path, "a" * 50)

    def test_native_wrapper_preserves_bff_and_supports_explicit_full_capture(self):
        with self.assertRaises(ValueError):
            native_tool.local_go_environment({}, {})
        env = native_tool.local_go_environment({
            "APP_ENV": "production", "ALLOW_DEV_LOGIN": "false", "AUTH_BFF_ENABLED": "true",
            "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT": "https://other-host/v1/traces",
            "MODEL_API_KEY": "model_secret", "THEIA_STAFF_BFF_CLIENT_SECRET": "bff_secret",
        }, {"THEIA_OTEL_CONTENT_API_KEY": "X" * 48, "PHOENIX_SECRET": "should-not-pass"})
        self.assertEqual(env["APP_ENV"], "local")
        self.assertEqual(env["HTTP_ADDR"], "127.0.0.1:8080")
        self.assertEqual(env["AUTH_BFF_ENABLED"], "true")
        self.assertEqual(env["ALLOW_DEV_LOGIN"], "false")
        self.assertEqual(env["AI_LOCAL_INSPECTOR"], "false")
        self.assertEqual(env["THEIA_OTEL_CAPTURE_CONTENT"], "false")
        self.assertEqual(env["OTEL_EXPORTER_OTLP_TRACES_ENDPOINT"], "http://127.0.0.1:6006/v1/traces")
        self.assertEqual(env["THEIA_OTEL_CONTENT_API_KEY"], "X" * 48)
        self.assertEqual(env["MODEL_API_KEY"], "model_secret")
        self.assertNotIn("PHOENIX_SECRET", env)
        self.assertEqual(env["THEIA_STAFF_BFF_CLIENT_SECRET"], "bff_secret")
        full = native_tool.local_go_environment({}, {"THEIA_OTEL_CONTENT_API_KEY": "X" * 48}, content=True)
        self.assertEqual(full["THEIA_OTEL_CAPTURE_CONTENT"], "true")
        self.assertEqual(full["ALLOW_DEV_LOGIN"], "false")
        self.assertEqual(full["AI_LOCAL_INSPECTOR"], "false")
        self.assertEqual(full["HTTP_ADDR"], "127.0.0.1:8080")


class FakePhoenixHandler(BaseHTTPRequestHandler):
    span_seen = b""
    unauth_write_seen = False
    headers_from_ingest = None

    def log_message(self, *args):
        pass  # Never echo authorization headers or private capture into tests.

    def do_GET(self):
        if self.path == "/healthz":
            return self._respond(200, b"ok")
        auth = self.headers.get("Authorization", "")
        if auth != "Bearer " + "X" * 48:
            return self._respond(401)
        if self.path == "/v1/projects":
            return self._respond(200, b'{"data":[]}')
        if self.path.startswith(f"/v1/projects/{smoke.PROJECT}/traces"):
            # Actual trace value is controlled in the test via class state.
            return self._respond(200, b'{"data":[{"trace_id":"' + self.span_seen.hex().encode() + b'"}]}')
        return self._respond(404)

    def do_POST(self):
        raw = self.rfile.read(int(self.headers.get("Content-Length", "0")))
        if self.headers.get("Authorization") != "Bearer " + "X" * 48:
            type(self).unauth_write_seen = True
            return self._respond(401)
        if self.path != "/v1/traces" or self.headers.get("Content-Type") != "application/x-protobuf":
            return self._respond(415)
        # Synthetic trace ID appears in the OTLP Span's trace_id bytes.
        # A full OTLP decoder is provided by a separate Go/Phoenix live check.
        needle = self._find_otel_trace_id(raw)
        if not needle:
            return self._respond(400)
        type(self).span_seen = needle
        type(self).headers_from_ingest = self.headers.get("x-project-name")
        return self._respond(200, b"{}")

    @staticmethod
    def _find_otel_trace_id(raw: bytes) -> bytes:
        # Handcrafted protobuf has the root Span's trace_id after its field-1 tag.
        prefix = bytes([0x0a, 0x10])
        at = raw.find(prefix)
        if at < 0 or at + 18 > len(raw):
            return b""
        return raw[at + 2:at + 18]

    def _respond(self, status: int, payload: bytes = b""):
        self.send_response(status)
        self.send_header("Content-Length", str(len(payload)))
        self.end_headers()
        self.wfile.write(payload)


class PhoenixSmokeTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.server = ThreadingHTTPServer(("127.0.0.1", 0), FakePhoenixHandler)
        cls.thread = threading.Thread(target=cls.server.serve_forever, daemon=True)
        cls.thread.start()
        cls.base = f"http://127.0.0.1:{cls.server.server_address[1]}"

    @classmethod
    def tearDownClass(cls):
        cls.server.shutdown()
        cls.server.server_close()
        cls.thread.join(timeout=5)

    def test_otlp_is_correctly_encoded_and_bounded(self):
        trace_id, span_id = "0123456789abcdef" * 2, "12345678abcdef00"
        payload = smoke.make_otlp_span(trace_id, span_id, now_ns=1730000000000000000)
        self.assertIn(bytes.fromhex(trace_id), payload)
        self.assertIn(bytes.fromhex(span_id), payload)
        self.assertIn(b"openinference.project.name", payload)
        self.assertIn(smoke.PROJECT.encode(), payload)
        self.assertLess(len(payload), 1000)
        self.assertRaises(ValueError, smoke.make_otlp_span, "123", span_id)

    def test_authenticated_ingest_and_unauthenticated_denial(self):
        smoke.preflight(self.base)
        trace_id = smoke.ingest(self.base, "X" * 48)
        self.assertEqual(len(trace_id), 32)
        self.assertTrue(FakePhoenixHandler.unauth_write_seen)
        self.assertEqual(FakePhoenixHandler.headers_from_ingest, smoke.PROJECT)
        smoke.verify_after_restart(self.base, "X" * 48, trace_id)

    def test_rejects_redirect_target_and_remote_hostname(self):
        with self.assertRaises(ValueError):
            smoke._call("https://evil.example:6006", "/healthz")
        with self.assertRaises(ValueError):
            smoke._call("http://localhost:6006", "/healthz")
        with self.assertRaises(ValueError):
            smoke._call("http://127.0.0.1:6006/path", "/healthz")


if __name__ == "__main__":
    unittest.main()
