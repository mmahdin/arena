"""MON-01B: static local privacy/deployment checks, no Docker needed."""
import json
from pathlib import Path
import unittest

import yaml

ROOT = Path(__file__).resolve().parents[2]


class PrivateLogContract(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.base = yaml.safe_load((ROOT / "compose.yaml").read_text())
        cls.metrics = yaml.safe_load((ROOT / "compose.monitoring.dev.yaml").read_text())
        cls.overlay = yaml.safe_load((ROOT / "compose.logging.dev.yaml").read_text())
        cls.loki = yaml.safe_load((ROOT / "infra/monitoring/loki/config.yml").read_text())
        cls.alloy = (ROOT / "infra/monitoring/alloy/config.alloy").read_text()
        cls.source = (ROOT / "apps/api/internal/platform/observability/safe_loki.go").read_text()
        cls.dashboard = json.loads((ROOT / "infra/monitoring/grafana/dashboards/theia-logs.json").read_text())
        cls.datasource = yaml.safe_load((ROOT / "infra/monitoring/grafana/provisioning/datasources/loki.yml").read_text())

    def test_services_overlay_nonpublic_and_synthetic(self):
        s = self.overlay["services"]
        self.assertEqual(s["api"]["environment"]["THEIA_SAFE_LOG_EXPORT_ENABLED"], "true")
        self.assertEqual(s["api"]["environment"]["THEIA_SAFE_LOG_EXPORT_PATH"], "/var/log/theia/theia-safe.jsonl")
        self.assertNotIn("ports", s["api"])
        self.assertEqual(s["api"]["group_add"], ["2000"])
        self.assertEqual(s["alloy"]["user"], "473:473")
        self.assertEqual(s["loki"]["user"], "10001:10001")
        for name in ("loki", "alloy", "logging-init"):
            self.assertEqual(s[name]["profiles"], ["monitoring"])
            self.assertNotIn("ports", s[name])
            self.assertNotIn("network_mode", s[name])
        self.assertEqual(self.metrics["services"]["grafana"]["ports"], ["127.0.0.1:3000:3000"])
        self.assertEqual(self.metrics["services"]["grafana"]["environment"]["GF_AUTH_ANONYMOUS_ENABLED"], "false")

    def test_file_only_collection_and_readonly_spool(self):
        s = self.overlay["services"]
        alloy_mounts = s["alloy"]["volumes"]
        self.assertIn("theia_safe_log_data:/var/log/theia:ro", alloy_mounts)
        self.assertIn("theia_safe_log_data:/var/log/theia", s["api"]["volumes"])
        mounts = " ".join(str(v) for item in s.values() for v in item.get("volumes", []))
        for forbidden in ("docker.sock", "ai-inspector", "LOG_FILE", "theia-api.jsonl", "/var/lib/docker"):
            self.assertNotIn(forbidden, mounts)
        self.assertIn('loki.source.file "theia_safe_jsonl"', self.alloy)
        self.assertIn('"/var/log/theia/theia-safe.jsonl*"', self.alloy)
        self.assertNotIn("discovery.docker", self.alloy)
        self.assertNotIn("loki.source.docker", self.alloy)
        self.assertNotIn("otelcol.receiver", self.alloy)
        self.assertIn('url = "http://loki:3100/loki/api/v1/push"', self.alloy)
        self.assertNotIn('request_id = "request_id"', self.alloy)
        self.assertNotIn('turn_ref = "turn_ref"', self.alloy)

    def test_loki_persistence_and_compactor_retention(self):
        self.assertFalse(self.loki["auth_enabled"]) # private Compose network only
        self.assertFalse(self.loki["analytics"]["reporting_enabled"])
        self.assertEqual(self.loki["schema_config"]["configs"][0]["schema"], "v13")
        self.assertEqual(self.loki["schema_config"]["configs"][0]["index"]["period"], "24h")
        self.assertEqual(self.loki["common"]["storage"]["filesystem"]["chunks_directory"], "/loki/chunks")
        self.assertEqual(self.loki["limits_config"]["retention_period"], "72h")
        self.assertEqual(self.loki["limits_config"]["max_query_lookback"], "72h")
        self.assertTrue(self.loki["compactor"]["retention_enabled"])
        self.assertEqual(self.loki["compactor"]["delete_request_store"], "filesystem")
        self.assertIn("theia_loki_data:/loki", self.overlay["services"]["loki"]["volumes"])
        self.assertIn("theia_alloy_data:/var/lib/alloy/data", self.overlay["services"]["alloy"]["volumes"])

    def test_grafana_datasource_and_log_dashboard(self):
        ds = self.datasource["datasources"][0]
        self.assertEqual(ds["uid"], "theia-loki")
        self.assertEqual(ds["url"], "http://loki:3100")
        self.assertFalse(ds["editable"])
        self.assertGreaterEqual(len(self.dashboard["panels"]), 2)
        for panel in self.dashboard["panels"]:
            self.assertEqual(panel["datasource"]["uid"], "theia-loki")
            for q in panel["targets"]:
                self.assertIn('service_name=\\"theia-api\\"'.replace('\\"', '"'), q["expr"])
                self.assertNotIn("user_id", q["expr"])
                self.assertNotIn("prompt", q["expr"])


    def test_readback_script_rejects_private_fields_and_preserves_credentials(self):
        import importlib.util
        import stat
        import tempfile
        spec = importlib.util.spec_from_file_location('smoke_loki', ROOT / 'scripts/monitoring/smoke_loki.py')
        smoke = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(smoke)
        with tempfile.TemporaryDirectory() as td:
            path = Path(td) / '.env.monitoring'
            path.write_text('THEIA_GRAFANA_ADMIN_PASSWORD=' + 'a'*32 + '\n')
            path.chmod(0o600)
            self.assertEqual(smoke.read_password(path), 'a'*32)
            path.chmod(0o644)
            with self.assertRaises(ValueError): smoke.read_password(path)
        result={'status':'success', 'data':{'result':[{'stream':{'service_name':'theia-api','environment':'local','level':'INFO'},'values':[['1764000000000000000', json.dumps({'schema':'theia.safe-log.v1','event':'http.request','level':'INFO'})]]}]}}
        self.assertEqual(smoke.validate_log_response(result), 1)
        result['data']['result'][0]['values'][0][1]=json.dumps({'schema':'theia.safe-log.v1','event':'http.request','prompt':'SECRET'})
        with self.assertRaises(ValueError): smoke.validate_log_response(result)
        result['data']['result'][0]['values'][0][1]=json.dumps({'schema':'theia.safe-log.v1','event':'http.request'})
        result['data']['result'][0]['stream']['turn_ref']='secret'
        with self.assertRaises(ValueError): smoke.validate_log_response(result)
        for bad in ('https://example.com/private', '/loki/api/v1/query_range'):
            with self.assertRaises(ValueError): smoke.request_json(bad)

    def test_export_guard_and_payload_denial_boundary(self):
        main = (ROOT / "apps/api/cmd/api/main.go").read_text()
        cfg = (ROOT / "apps/api/internal/platform/config/config.go").read_text()
        self.assertIn("observability.NewSafeLogWriter", main)
        self.assertIn("cfg.Environment != \"local\"", cfg)
        self.assertIn('filepath.Base(path) != "theia-safe.jsonl"', cfg)
        self.assertIn("sanitizedLogRecord(p)", self.source)
        self.assertIn("exportedEvents[event]", self.source)
        self.assertIn("knownErrorCodes[value]", self.source)
        self.assertIn("safeLogMaxFile", self.source)
        self.assertIn("safeLogBackups", self.source)
        for forbidden in ('"prompt": value', '"response": value', 'record[field] = incoming[field]', '"user_id": value'):
            self.assertNotIn(forbidden, self.source)


if __name__ == "__main__":
    unittest.main()
