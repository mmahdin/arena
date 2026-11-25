"""Offline structural/privacy checks for MON-01A. No Docker or Go needed."""
import importlib.util
import json
import os
from pathlib import Path
import stat
import tempfile
import unittest

import yaml

ROOT = Path(__file__).resolve().parents[2]

class MonitoringContract(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.overlay = yaml.safe_load((ROOT / 'compose.monitoring.dev.yaml').read_text())
        cls.scrape = yaml.safe_load((ROOT / 'infra/monitoring/prometheus.yml').read_text())
        cls.datasource = yaml.safe_load((ROOT / 'infra/monitoring/grafana/provisioning/datasources/prometheus.yml').read_text())
        cls.dashboard = json.loads((ROOT / 'infra/monitoring/grafana/dashboards/theia-engineering.json').read_text())

    def test_api_only_private_metrics_server(self):
        go = (ROOT/'apps/api/cmd/api/main.go').read_text()
        router = (ROOT/'apps/api/internal/app/composition/api.go').read_text()
        self.assertIn('net.Listen("tcp", cfg.MetricsOptions.Address)',go)
        self.assertNotIn('router.GET("/metrics"',router)
        self.assertNotIn('adminAPI.GET("/metrics"',router)
        options = (ROOT/'apps/api/internal/platform/metrics/metrics.go').read_text()
        self.assertIn('opt.Environment != "local"',options)
        self.assertIn('ip.IsLoopback()',options)
        self.assertIn('func RouteLabel(route string)',options)
        self.assertNotIn('Request.URL.Path',options)
        for sensitive in ('theia.user_id', 'theia.turn_id', 'theia.conversation_id'):
            self.assertNotIn(sensitive, options)

    def test_overlay_is_nonpublic(self):
        api=self.overlay['services']['api']
        prom=self.overlay['services']['prometheus']
        graf=self.overlay['services']['grafana']
        self.assertEqual(api['environment']['THEIA_METRICS_ADDR'],'127.0.0.1:9100')
        self.assertNotIn('ports',api)
        self.assertEqual(prom['network_mode'],'service:api')
        self.assertNotIn('ports',prom)
        self.assertEqual(graf['ports'],['127.0.0.1:3000:3000'])
        self.assertEqual(prom['profiles'],['monitoring'])
        self.assertEqual(graf['profiles'],['monitoring'])
        self.assertIn('${THEIA_GRAFANA_ADMIN_PASSWORD:?',graf['environment']['GF_SECURITY_ADMIN_PASSWORD'])
        self.assertEqual(graf['environment']['GF_AUTH_ANONYMOUS_ENABLED'],'false')
        targets=self.scrape['scrape_configs'][0]['static_configs'][0]['targets']
        self.assertEqual(targets,['127.0.0.1:9100'])
        self.assertEqual(self.datasource['datasources'][0]['url'],'http://api:9090')

    def test_dashboard_has_only_aggregated_metrics(self):
        self.assertGreaterEqual(len(self.dashboard['panels']),6)
        for panel in self.dashboard['panels']:
            self.assertEqual(panel['datasource']['uid'],'theia-prometheus')
            for q in panel['targets']:
                exp=q['expr'].lower()
                for private in ('turn_id','conversation_id','user_id','prompt','response','api_key','secret'):
                    self.assertNotIn(private,exp)

    def test_credential_initializer_is_owner_only_idempotent(self):
        path=ROOT/'scripts/monitoring/init_local_monitoring_env.py'
        spec=importlib.util.spec_from_file_location('local_monitoring_init',path)
        mod=importlib.util.module_from_spec(spec)
        spec.loader.exec_module(mod)
        with tempfile.TemporaryDirectory() as td:
            outfile=Path(td)/'.env.monitoring'
            self.assertTrue(mod.initialize(outfile))
            self.assertEqual(stat.S_IMODE(outfile.stat().st_mode),0o600)
            contents=outfile.read_text()
            self.assertTrue(contents.startswith('THEIA_GRAFANA_ADMIN_PASSWORD='))
            self.assertFalse(mod.initialize(outfile))
            self.assertEqual(outfile.read_text(),contents)
            outfile.chmod(0o644)
            with self.assertRaises(ValueError): mod.initialize(outfile)

    def test_readonly_smoke_rejects_remote_and_auth_urls(self):
        spec=importlib.util.spec_from_file_location('smoke',ROOT/'scripts/monitoring/smoke_metrics.py')
        mod=importlib.util.module_from_spec(spec);spec.loader.exec_module(mod)
        for bad in ('https://example.com/metrics','http://127.0.0.1:9100/private',
                    'http://localhost:9100/metrics','http://127.0.0.1:9100/metrics?token=abc',
                    'http://192.0.2.1:9100/metrics'):
            with self.assertRaises(ValueError):mod.check(bad)

if __name__ == '__main__': unittest.main()
