#!/usr/bin/env python3
"""MON-02: emit ONE fixed synthetic test alert from an internal Docker network.

Never expose the Alertmanager API on the host or accept operator-provided
alert labels, annotations, payloads, or URLs. This is a local test, NOT an
alternative source of real operational alerts (Prometheus owns those).
"""
import json
from datetime import datetime, timedelta, timezone
from urllib.request import Request, build_opener, ProxyHandler, HTTPRedirectHandler

ENDPOINT = 'http://alertmanager:9093/api/v2/alerts'


class NoRedirect(HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


def payload(now):
    return [{
        'labels': {
            'alertname': 'TheiaMON02Smoke',
            'severity': 'warning',
            'component': 'smoke',
            'environment': 'local',
        },
        'annotations': {
            'summary': 'Theia local synthetic notification test',
            'description': 'This is a fixed, deliberately generated synthetic alert.',
        },
        'startsAt': now.isoformat().replace('+00:00', 'Z'),
        'endsAt': (now + timedelta(minutes=2)).isoformat().replace('+00:00', 'Z'),
    }]


def emit():
    body = json.dumps(payload(datetime.now(timezone.utc)), separators=(',', ':')).encode()
    req = Request(ENDPOINT, data=body, headers={'Content-Type': 'application/json'}, method='POST')
    # No outbound proxy or redirects; only the internal alertmanager DNS name.
    opener = build_opener(ProxyHandler({}), NoRedirect())
    with opener.open(req, timeout=6) as response:
        if response.status != 200:
            raise RuntimeError('synthetic alert rejected')
    return True


if __name__ == '__main__':
    emit()
    print('One fixed synthetic local alert submitted. Check Mailpit after group_wait.')
