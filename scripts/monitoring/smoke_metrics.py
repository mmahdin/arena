#!/usr/bin/env python3
"""Offline-safe read-only metric endpoint smoke. Intended for native Go localhost."""
import argparse
import ipaddress
from urllib.parse import urlsplit
from urllib.request import urlopen

EXPECTED = (
    'theia_metrics_enabled',
    'go_goroutines',
)
# CounterVec/HistogramVec series appear only after their first observation.
# Generate an HTTP request before checking these as part of full acceptance.


def check(url):
    parsed = urlsplit(url)
    if parsed.scheme != 'http' or parsed.username or parsed.password or parsed.query or parsed.fragment or parsed.path != '/metrics':
        raise ValueError('Only a plain HTTP loopback /metrics URL is permitted')
    try:
        ip = ipaddress.ip_address(parsed.hostname or '')
        if not ip.is_loopback:
            raise ValueError('Only a loopback URL is permitted')
    except ValueError as exc:
        raise ValueError('Only IP loopback addresses are permitted') from exc
    with urlopen(url, timeout=3) as result:
        text = result.read(2_000_000).decode('utf-8')
    for name in EXPECTED:
        if f'# HELP {name} ' not in text:
            raise ValueError('Missing expected metric family: ' + name)
    if any(s in text.lower() for s in ('csrf_token', 'conversation_id', 'user_id', 'turn_id', 'api_key', 'password')):
        raise ValueError('Unexpected sensitive metric label')
    return True


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--url', default='http://127.0.0.1:9100/metrics')
    args = parser.parse_args()
    check(args.url)
    print('Local metrics scrape passed (metric families present, no sensitive labels).')
