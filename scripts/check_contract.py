#!/usr/bin/env python3
"""Check OpenAPI source parses and routes match handwritten Gin route registration.

This is not a full OpenAPI schema validator or replacement for generated-code drift tests.
"""
from pathlib import Path
import re
import sys
ROOT = Path(__file__).resolve().parent.parent
spec = ROOT / 'contracts/openapi/openapi.yaml'
source = (ROOT / 'apps/api/internal/app/composition/api.go').read_text()
try:
    import yaml
except ImportError:
    print('PyYAML unavailable: cannot validate authored OpenAPI', file=sys.stderr)
    sys.exit(2)
try:
    parsed = yaml.safe_load(spec.read_text(encoding='utf-8'))
    assert parsed['openapi'] == '3.1.0'
    declared = {(method.upper(), path)
                for path, methods in parsed['paths'].items()
                for method in methods if method.lower() in ('get', 'post', 'patch', 'delete', 'put')}
    observed = set()
    regex = r'(router|group|authorized)\.(GET|POST|PATCH|DELETE|PUT)\("([^"\n]+)"'
    for target, method, path in re.findall(regex, source):
        if target != 'router':
            path = '/api/v1' + path
        path = re.sub(r':([a-zA-Z_][a-zA-Z_0-9]*)', r'{\1}', path)
        observed.add((method, path))
    missing = observed - declared
    unused = declared - observed
    if missing or unused:
        print('Missing in contract:', sorted(missing))
        print('Declared but no route:', sorted(unused))
        sys.exit(1)
    print(f'OpenAPI baseline OK: {len(observed)} method+route pairs match Gin composition')
except (ValueError, KeyError, AssertionError, TypeError) as exc:
    print('Contract invalid:', exc, file=sys.stderr)
    sys.exit(1)
