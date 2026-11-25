"""MON-02: offline safety, topology, rule thresholds, and smoke-tool checks."""
import importlib.util
import json
from pathlib import Path
import stat
import tempfile
import unittest

import yaml

ROOT = Path(__file__).resolve().parents[2]


class AlertingContract(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.base = yaml.safe_load((ROOT / 'compose.yaml').read_text())
        cls.mon = yaml.safe_load((ROOT / 'compose.monitoring.dev.yaml').read_text())
        cls.alert = yaml.safe_load((ROOT / 'compose.alerting.dev.yaml').read_text())
        cls.prom_base = yaml.safe_load((ROOT / 'infra/monitoring/prometheus.yml').read_text())
        cls.prom = yaml.safe_load((ROOT / 'infra/monitoring/prometheus.alerting.yml').read_text())
        cls.rules = yaml.safe_load((ROOT / 'infra/monitoring/alerts/theia.rules.yml').read_text())
        cls.am = yaml.safe_load((ROOT / 'infra/monitoring/alertmanager/config.yml').read_text())
        cls.ds = yaml.safe_load((ROOT / 'infra/monitoring/grafana/provisioning/datasources/alertmanager.yml').read_text())
        cls.dashboard = json.loads((ROOT / 'infra/monitoring/grafana/dashboards/theia-alerts.json').read_text())
        cls.flat_rules = {r['alert']: r for group in cls.rules['groups'] for r in group['rules']}

    def test_monitoring_standalone_remains_usable(self):
        self.assertEqual(self.prom_base['scrape_configs'][0]['static_configs'][0]['targets'], ['127.0.0.1:9100'])
        # The MON-01A config is NOT changed to reference a missing alert service.
        self.assertNotIn('alerting', self.prom_base)
        self.assertNotIn('rule_files', self.prom_base)
        self.assertEqual(self.prom['scrape_configs'][0]['static_configs'][0]['targets'], ['127.0.0.1:9100'])
        self.assertEqual(self.prom['alerting']['alertmanagers'][0]['static_configs'][0]['targets'], ['alertmanager:9093'])
        self.assertEqual(self.prom['rule_files'], ['/etc/prometheus/rules/*.yml'])
        mon_prom_mounts = self.mon['services']['prometheus']['volumes']
        alert_mounts = self.alert['services']['prometheus']['volumes']
        self.assertIn('./infra/monitoring/prometheus.yml:/etc/prometheus/prometheus.yml:ro', mon_prom_mounts)
        self.assertIn('./infra/monitoring/prometheus.alerting.yml:/etc/prometheus/prometheus.yml:ro', alert_mounts)
        self.assertIn('./infra/monitoring/alerts:/etc/prometheus/rules:ro', alert_mounts)
        self.assertEqual(self.mon['services']['prometheus']['network_mode'], 'service:api')
        self.assertNotIn('network_mode', self.alert['services']['prometheus'])
        self.assertNotIn('ports', self.alert['services']['prometheus'])

    def test_single_private_alert_router_and_local_delivery_only(self):
        services = self.alert['services']
        for key in ('alertmanager', 'alerting-init', 'mailpit'):
            self.assertEqual(services[key]['profiles'], ['monitoring'])
            self.assertIn('no-new-privileges:true', services[key]['security_opt'])
        for key in ('alertmanager', 'alerting-init', 'alert-smoke'):
            self.assertNotIn('ports', services[key])
        self.assertEqual(services['mailpit']['ports'], ['127.0.0.1:8025:8025'])
        self.assertEqual(services['alert-smoke']['profiles'], ['alert-test'])
        self.assertNotIn('1025:1025', repr(services))
        self.assertNotIn('docker.sock', repr(services))
        self.assertIn('theia_alertmanager_data:/alertmanager', services['alertmanager']['volumes'])
        self.assertEqual(services['alertmanager']['user'], '65534:65534')
        self.assertEqual(self.am['global']['smtp_smarthost'], 'mailpit:1025')
        self.assertEqual(self.am['route']['receiver'], 'theia-local-inbox')
        self.assertEqual(len(self.am['receivers']), 1)
        self.assertTrue(self.am['receivers'][0]['email_configs'][0]['to'].endswith('.invalid'))
        self.assertEqual(self.am['route']['group_by'], ['alertname', 'severity'])
        self.assertLessEqual(self.am['route']['repeat_interval'].count('h'), 1)
        self.assertIn('MP_ALLOWED_HOSTS', services['mailpit']['environment'])
        self.assertEqual(services['mailpit']['environment']['MP_MAX_AGE'], '2d')

    def test_one_grafana_alertmanager_with_no_duplicate_evaluation(self):
        ds = self.ds['datasources'][0]
        self.assertEqual(ds['uid'], 'theia-alertmanager')
        self.assertEqual(ds['url'], 'http://alertmanager:9093')
        self.assertEqual(ds['type'], 'alertmanager')
        self.assertFalse(ds['editable'])
        self.assertFalse(ds['jsonData']['handleGrafanaManagedAlerts'])
        self.assertEqual(ds['jsonData']['implementation'], 'prometheus')
        self.assertEqual(self.mon['services']['grafana']['ports'], ['127.0.0.1:3000:3000'])
        self.assertGreaterEqual(len(self.dashboard['panels']), 4)
        for panel in self.dashboard['panels']:
            self.assertEqual(panel['datasource']['uid'], 'theia-prometheus')

    def test_alerts_cover_real_metrics_and_avoid_small_sample_noise(self):
        expected = {
            'TheiaAPIMetricsTargetDown', 'TheiaAPIMetricsTargetMissing',
            'TheiaAlertmanagerTargetDown', 'TheiaAPIHighServerErrorRate',
            'TheiaAPIHighP95Latency', 'TheiaLLMHighFailureRatio',
            'TheiaWorkerNeverReportedHeartbeat', 'TheiaWorkerHeartbeatStale',
            'TheiaWorkerNotRunning',
        }
        self.assertEqual(set(self.flat_rules), expected)
        for name, rule in self.flat_rules.items():
            self.assertIn(rule['labels']['severity'], ('warning', 'critical'))
            self.assertIn(rule['labels']['component'], ('api', 'alerting', 'llm', 'workers'))
            self.assertIn('for', rule)
            self.assertIn('#', rule['annotations']['runbook'])
            strings = json.dumps(rule).lower()
            for private in ('turn_id', 'conversation_id', 'user_id', 'prompt_body', 'response_body', 'cookie', 'api_key', 'authorization', 'phone'):
                self.assertNotIn(private, strings, f'{name} includes private term {private}')
        self.assertIn('>= 20', self.flat_rules['TheiaAPIHighServerErrorRate']['expr'])
        self.assertIn('>= 20', self.flat_rules['TheiaAPIHighP95Latency']['expr'])
        self.assertIn('>= 10', self.flat_rules['TheiaLLMHighFailureRatio']['expr'])
        self.assertIn('unless on (kind)', self.flat_rules['TheiaWorkerNeverReportedHeartbeat']['expr'])
        self.assertIn('theia_worker_running == 1', self.flat_rules['TheiaWorkerHeartbeatStale']['expr'])
        self.assertIn('job="theia-api"', self.flat_rules['TheiaAPIMetricsTargetDown']['expr'])

    def test_expected_worker_gauge_is_only_set_for_configured_core(self):
        source = (ROOT / 'apps/api/internal/platform/metrics/metrics.go').read_text()
        composition = (ROOT / 'apps/api/internal/app/composition/api.go').read_text()
        self.assertIn('theia_worker_expected', source)
        self.assertIn('func (m *Runtime) ExpectWorkers()', source)
        self.assertIn('for _, kind := range []string{"memory", "reflection", "scheduler"}', source)
        self.assertIn('cfg.MetricsRuntime.ExpectWorkers()', composition)
        self.assertIn('if opt.Core != nil {', composition)
        self.assertIn('if cfg.MetricsRuntime != nil {', composition)
        self.assertNotIn('theia.user_id', source)
        self.assertNotIn('theia.turn_id', source)

    def test_fixed_synthetic_probe_is_not_an_external_webhook(self):
        source = (ROOT / 'scripts/monitoring/emit_synthetic_alert.py').read_text()
        self.assertIn('http://alertmanager:9093/api/v2/alerts', source)
        self.assertIn('TheiaMON02Smoke', source)
        self.assertIn('ProxyHandler({})', source)
        self.assertIn('NoRedirect()', source)
        self.assertNotIn('input(', source)
        self.assertNotIn('argparse', source)
        self.assertNotIn('user_id', source)
        self.assertNotIn('prompt', source)
        self.assertIn('./scripts/monitoring/emit_synthetic_alert.py:/app/emit_synthetic_alert.py:ro',
                      self.alert['services']['alert-smoke']['volumes'])

    def test_readback_does_not_expose_credentials_or_bodies(self):
        spec = importlib.util.spec_from_file_location('smoke_alerts', ROOT / 'scripts/monitoring/smoke_alerting.py')
        mod = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(mod)
        with tempfile.TemporaryDirectory() as td:
            path = Path(td) / '.env.monitoring'
            path.write_text('THEIA_GRAFANA_ADMIN_PASSWORD=' + 'a' * 32 + '\n')
            path.chmod(0o600)
            self.assertEqual(mod.grafana_password(path), 'a' * 32)
            path.chmod(0o644)
            with self.assertRaises(ValueError): mod.grafana_password(path)
        self.assertIn('NoRedirect()', (ROOT / 'scripts/monitoring/smoke_alerting.py').read_text())
        for invalid in ('http://example.org/api/v1/messages', 'http://127.0.0.1:8025/api/v1/messages?token=x'):
            with self.assertRaises(ValueError): mod.fetch_json(invalid)
        fake = {'status': 'success', 'data': {'groups': [
            {'rules': [{'type': 'alerting', 'name': r} for r in mod.REQUIRED]}
        ]}}
        self.assertEqual(mod.check_rules(fake), len(mod.REQUIRED))
        self.assertEqual(mod.check_mailpit({'messages': [{'Subject': '[Theia] TheiaMON02Smoke'}]}, True), 1)
        with self.assertRaises(ValueError): mod.check_mailpit({'messages': []}, True)
        with self.assertRaises(ValueError): mod.check_alertmanager({'foo': {}})
        self.assertTrue(mod.check_alertmanager({'config': {'original': 'safe'}}))

if __name__ == '__main__':
    unittest.main()
