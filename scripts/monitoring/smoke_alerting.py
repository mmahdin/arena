#!/usr/bin/env python3
"""MON-02 read-only smoke of Grafana/Prometheus/Alertmanager/Mailpit.

This script never prints notifications, query values, credentials, or private
messages. The only remote destinations are fixed numeric IPv4 loopback URLs.
"""
import argparse
import base64
import json
from pathlib import Path
import stat
from urllib.request import Request, build_opener, ProxyHandler, HTTPRedirectHandler

GRAFANA = 'http://127.0.0.1:3000'
MAILPIT = 'http://127.0.0.1:8025'
class NoRedirect(HTTPRedirectHandler):
    # Prevent a local server from sending the Grafana Basic Auth header to an
    # unexpected destination, even if the server were misconfigured.
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


REQUIRED = {
    'TheiaAPIMetricsTargetDown',
    'TheiaAPIMetricsTargetMissing',
    'TheiaAlertmanagerTargetDown',
    'TheiaAPIHighServerErrorRate',
    'TheiaAPIHighP95Latency',
    'TheiaLLMHighFailureRatio',
    'TheiaWorkerNeverReportedHeartbeat',
    'TheiaWorkerHeartbeatStale',
    'TheiaWorkerNotRunning',
}


def grafana_password(path):
    path = Path(path)
    if not path.is_file() or stat.S_IMODE(path.stat().st_mode) != 0o600:
        raise ValueError('monitoring credential file must be owner-only (0600)')
    # Never print this value.
    lines = path.read_text(encoding='utf-8').splitlines()
    matches = [s.split('=', 1)[1] for s in lines if s.startswith('THEIA_GRAFANA_ADMIN_PASSWORD=')]
    if len(matches) != 1 or len(matches[0]) < 16:
        raise ValueError('missing generated Grafana credential')
    return matches[0]


def fetch_json(url, password=None):
    # Reject external endpoints and query strings. Only fixed routes below are
    # allowed; caller-supplied URLs are never accepted by CLI arguments.
    approved = {
        GRAFANA + '/api/datasources/proxy/uid/theia-prometheus/api/v1/rules',
        GRAFANA + '/api/datasources/proxy/uid/theia-alertmanager/api/v2/status',
        MAILPIT + '/api/v1/messages',
    }
    if url not in approved:
        raise ValueError('unapproved local URL')
    headers = {'Accept': 'application/json'}
    if password is not None:
        cred = base64.b64encode(('admin:' + password).encode()).decode('ascii')
        headers['Authorization'] = 'Basic ' + cred
    request = Request(url, headers=headers)
    with build_opener(ProxyHandler({}), NoRedirect()).open(request, timeout=5) as res:
        if res.status != 200:
            raise RuntimeError('unexpected local service status')
        raw = res.read(512_001)
    if len(raw) > 512_000:
        raise ValueError('response too large')
    return json.loads(raw)


def check_rules(payload):
    if payload.get('status') != 'success':
        raise ValueError('Prometheus rules unavailable')
    groups = payload.get('data', {}).get('groups', [])
    found = {r.get('name') for g in groups for r in g.get('rules', []) if r.get('type') == 'alerting'}
    if not REQUIRED.issubset(found):
        raise ValueError('missing one or more expected alert rules')
    return len(found)


def check_mailpit(payload, expect_smoke=False):
    messages = payload.get('messages')
    if not isinstance(messages, list):
        raise ValueError('Mailpit messages unavailable')
    if expect_smoke and not any('TheiaMON02Smoke' in (m.get('Subject') or '') for m in messages):
        raise ValueError('synthetic alert not delivered to local inbox')
    return len(messages)


def check_alertmanager(payload):
    if not isinstance(payload, dict) or not isinstance(payload.get('config'), dict):
        raise ValueError('Alertmanager status unavailable')
    return True


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--env', default='.env.monitoring')
    parser.add_argument('--expect-smoke', action='store_true')
    args = parser.parse_args()
    pw = grafana_password(Path(args.env))
    count = check_rules(fetch_json(GRAFANA + '/api/datasources/proxy/uid/theia-prometheus/api/v1/rules', pw))
    check_alertmanager(fetch_json(GRAFANA + '/api/datasources/proxy/uid/theia-alertmanager/api/v2/status', pw))
    sent = check_mailpit(fetch_json(MAILPIT + '/api/v1/messages'), args.expect_smoke)
    print(f'Local alerting preflight passed: {count} rules available, {sent} local inbox messages.')
    if not args.expect_smoke:
        print('Notification delivery NOT yet proven; run the fixed synthetic alert probe then --expect-smoke.')


if __name__ == '__main__':
    try:
        main()
    except (OSError, ValueError, RuntimeError, KeyError, json.JSONDecodeError) as exc:
        raise SystemExit('Local alerting smoke failed (' + type(exc).__name__ + ').') from None
