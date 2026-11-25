"""OBS-05A local contract tests. Do NOT require Docker, cloud or real secrets."""
from __future__ import annotations

import base64
import importlib.util
import json
import os
from pathlib import Path
import stat
import sys
import tempfile
import unittest
from unittest.mock import patch
import uuid

import yaml

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / 'scripts' / 'observability'))
import init_dual_otlp_env as env_tool
import run_native_with_dual_otlp as native
import smoke_dual_otlp as smoke


class CollectorConfigTest(unittest.TestCase):
    def test_opt_in_private_collector_with_two_authenticated_destinations(self):
        base = yaml.safe_load((ROOT / 'compose.observability.dev.yaml').read_text())
        overlay = yaml.safe_load((ROOT / 'compose.observability.dual.dev.yaml').read_text())
        self.assertEqual(set(base['services']), {'phoenix'})
        self.assertEqual(set(overlay['services']), {'otel-collector-dual'})
        svc = overlay['services']['otel-collector-dual']
        self.assertEqual(svc['profiles'], ['dual-observability'])
        self.assertEqual(svc['ports'], ['127.0.0.1:4318:4318'])
        self.assertIn('contrib:0.', svc['image'])
        self.assertEqual(svc['depends_on']['phoenix']['condition'], 'service_healthy')
        self.assertEqual(svc['security_opt'], ['no-new-privileges:true'])
        self.assertEqual(svc['cap_drop'], ['ALL'])
        self.assertNotIn('latest', svc['image'])
        self.assertEqual(set(svc['environment']), {
            'THEIA_COLLECTOR_INGEST_KEY', 'THEIA_PHOENIX_API_KEY',
            'THEIA_LANGFUSE_AUTH_BASE64', 'THEIA_LANGFUSE_OTLP_ENDPOINT'})
        self.assertTrue(all(':' in k for k in svc['environment'].values()))
        cfg = yaml.safe_load((ROOT / 'infra/observability/otelcol-dual.yaml').read_text())
        self.assertEqual(cfg['receivers']['otlp/theia']['protocols']['http']['auth']['authenticator'], 'bearertokenauth/theia')
        self.assertEqual(set(cfg['service']['pipelines']['traces/theia_dual']['exporters']), {
            'otlphttp/phoenix','otlphttp/langfuse'})
        self.assertEqual(cfg['exporters']['otlphttp/langfuse']['headers']['x-langfuse-ingestion-version'], '4')
        self.assertEqual(cfg['exporters']['otlphttp/phoenix']['headers']['x-project-name'], smoke.PROJECT)
        for item in cfg['exporters'].values():
            self.assertTrue(item['sending_queue']['enabled'])
            self.assertTrue(item['retry_on_failure']['enabled'])
        self.assertNotIn('logging', cfg['exporters'])
        self.assertIn('.env.dual-otlp\n', (ROOT / '.gitignore').read_text())
        self.assertNotIn('secret', (ROOT / '.env.dual-otlp.example').read_text().lower())


class SecretsTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.path = Path(self.tmp.name) / '.env.dual-otlp'

    def test_secret_bootstrap_0600_no_clobber_and_hidden_ingest_key(self):
        self.assertTrue(env_tool.create_env(self.path))
        self.assertFalse(env_tool.create_env(self.path))
        self.assertEqual(stat.S_IMODE(self.path.stat().st_mode), 0o600)
        env = env_tool.read_env(self.path, require_complete=False)
        self.assertGreaterEqual(len(env['THEIA_COLLECTOR_INGEST_KEY']), 48)
        self.assertFalse(env['THEIA_LANGFUSE_AUTH_BASE64'])
        self.assertNotEqual(env['THEIA_COLLECTOR_INGEST_KEY'], env.get('THEIA_LANGFUSE_AUTH_BASE64'))
        with self.assertRaises(ValueError):
            env_tool.read_env(self.path)
        self.path.chmod(0o644)
        with self.assertRaises(ValueError):
            env_tool.read_env(self.path)
        self.path.chmod(0o600)
        symlink = Path(self.tmp.name) / 'link'
        symlink.symlink_to(self.path)
        with self.assertRaises(ValueError):
            env_tool.create_env(symlink)

    def test_configure_stores_only_base64_credentials_and_refuses_overwrite(self):
        env_tool.create_env(self.path)
        public = 'pk-lf-' + 'a'*24
        secret = 'sk-lf-' + 'b'*24
        env_tool.configure(self.path, 'https://cloud.langfuse.com/api/public/otel', public, secret)
        decoded = env_tool.read_env(self.path)
        self.assertEqual(base64.b64decode(decoded['THEIA_LANGFUSE_AUTH_BASE64']), f'{public}:{secret}'.encode())
        self.assertNotIn(public, self.path.read_text())
        self.assertNotIn(secret, self.path.read_text())
        with self.assertRaises(ValueError):
            env_tool.configure(self.path, 'https://cloud.langfuse.com/api/public/otel', public, secret)

    def test_endpoint_validation(self):
        for url in ['https://cloud.langfuse.com/api/public/otel',
                    'https://private.example.org:443/api/public/otel',
                    'http://langfuse-web:3000/api/public/otel']:
            self.assertEqual(env_tool.validate_endpoint(url), url)
        for url in ['', 'https://cloud.langfuse.com/api/public/otel/v1/traces',
                    'http://cloud.langfuse.com/api/public/otel',
                    'https://a@evil.test/api/public/otel',
                    'https://example.com/api/public/otel?token=secret',
                    'https://example.com/api/public/otel#secret',
                    'http://127.0.0.1:3000/api/public/otel',
                    'https://example.com.evil/path',
                    'https://example.com/api/public/otel\\@evil.com',
                    'https://example.com:bad/api/public/otel',
                    'https://example.com:65536/api/public/otel',
                    'https://exa\tample.com/api/public/otel']:
            with self.subTest(url=url), self.assertRaises(ValueError):
                env_tool.validate_endpoint(url)

    def test_native_go_environment_retains_bff_and_separates_all_sink_keys(self):
        self.assertTrue(env_tool.create_env(self.path))
        secrets = env_tool.read_env(self.path, require_complete=False)
        env = native.go_environment({
            'AUTH_BFF_ENABLED': 'true', 'THEIA_PHOENIX_API_KEY': 'server-secret',
            'THEIA_LANGFUSE_AUTH_BASE64': 'secret-basic',
            'MODEL_API_KEY': 'model-secret', 'ALLOW_DEV_LOGIN': 'true',
            'THEIA_OTEL_CONTENT_API_KEY': 'previous-phoenix-key',
        }, secrets, content=True)
        self.assertEqual(env['THEIA_OTEL_CAPTURE_CONTENT'], 'true')
        self.assertEqual(env['OTEL_EXPORTER_OTLP_TRACES_ENDPOINT'], 'http://127.0.0.1:4318/v1/traces')
        self.assertEqual(env['THEIA_OTEL_CONTENT_API_KEY'], secrets['THEIA_COLLECTOR_INGEST_KEY'])
        self.assertEqual(env['ALLOW_DEV_LOGIN'], 'false')
        self.assertEqual(env['AUTH_BFF_ENABLED'], 'true')
        self.assertEqual(env['MODEL_API_KEY'], 'model-secret')
        for key in ('THEIA_LANGFUSE_AUTH_BASE64', 'THEIA_PHOENIX_API_KEY', 'THEIA_COLLECTOR_INGEST_KEY'):
            self.assertNotIn(key, env)
        self.assertEqual(native.go_environment({}, secrets)['THEIA_OTEL_CAPTURE_CONTENT'], 'false')


class SmokeTest(unittest.TestCase):
    def test_probe_contains_one_synthetic_span_no_keys_and_same_trace_id(self):
        trace = uuid.uuid4().hex
        payload = smoke.make_synthetic_span(trace, os.urandom(8).hex())
        self.assertIn(bytes.fromhex(trace), payload)
        self.assertIn(b'theia.obs05a.dual-smoke', payload)
        self.assertNotIn(b'Authorization', payload)
        self.assertNotIn(b'health', payload.lower())
        self.assertNotIn(b'patient', payload.lower())

    def test_preflight_rejects_unauthed_and_bad_auth(self):
        with patch.object(smoke, 'ingest_collector', side_effect=[401, 403, 200]) as calls:
            smoke.preflight('collector-secret')
            self.assertEqual(calls.call_count, 3)
            self.assertIsNone(calls.call_args_list[0].args[1])
            self.assertEqual(calls.call_args_list[-1].args[1], 'collector-secret')
        with patch.object(smoke, 'ingest_collector', return_value=200):
            with self.assertRaisesRegex(RuntimeError, 'SECURITY FAILURE'):
                smoke.preflight('collector-secret')

    def test_readback_same_trace_id_from_both_services_without_reingest(self):
        trace = uuid.uuid4().hex
        creds = {'THEIA_LANGFUSE_OTLP_ENDPOINT': 'https://private.example/api/public/otel',
                 'THEIA_LANGFUSE_AUTH_BASE64': base64.b64encode(b'pk-lf-a:sk-lf-b').decode()}
        with patch.object(smoke, 'phoenix_has_trace', return_value=True) as phoenix, \
             patch.object(smoke, 'langfuse_has_trace', return_value=True) as langfuse:
            self.assertEqual(smoke.inspect(trace, creds, 'phoenix-key'), (True, True))
            phoenix.assert_called_once_with(trace, 'phoenix-key')
            langfuse.assert_called_once_with(trace, creds)
        with patch.object(smoke, 'request', return_value=(200, json.dumps({'data':[{'traceId':trace}]}).encode())) as read:
            self.assertTrue(smoke.langfuse_has_trace(trace, creds))
            self.assertIn('/api/public/v2/observations?', read.call_args.args[0])
            self.assertIn('traceId='+trace, read.call_args.args[0])
            self.assertIn('fromStartTime=', read.call_args.args[0])
            self.assertIn('toStartTime=', read.call_args.args[0])


if __name__ == '__main__':
    unittest.main()
