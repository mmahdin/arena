#!/usr/bin/env python3
"""Read-only, loopback-only Grafana -> Loki smoke; never echo log content or keys."""
import base64
import json
from pathlib import Path
import re
import stat
import sys
import time
from urllib.error import HTTPError, URLError
from urllib.parse import urlencode
from urllib.request import build_opener, HTTPRedirectHandler, ProxyHandler, Request

ROOT = Path(__file__).resolve().parents[2]
CREDS = ROOT / '.env.monitoring'
BASE = 'http://127.0.0.1:3000'
PREFIX = '/api/datasources/proxy/uid/theia-loki'
ALLOWED = frozenset({
    'time','level','event','service','environment','schema',
    'request_id','turn_ref','trace_id','span_id','component','stage',
    'status','method','route','error_code','http_status',
    'duration_ms','count','attempt','retry_index','max_attempts',
    'provider_status','db_error_number','scheduler_failures',
    'failed','degraded','fallback_used','retryable',
})


class NoRedirect(HTTPRedirectHandler):
    def redirect_request(self, request, fp, code, msg, headers, newurl):
        return None


def read_password(path=CREDS):
    if path.is_symlink() or not path.is_file() or stat.S_IMODE(path.stat().st_mode) != 0o600:
        raise ValueError('Missing or insecure Grafana credentials')
    lines = path.read_text(encoding='ascii').splitlines()
    if len(lines) != 1 or not lines[0].startswith('THEIA_GRAFANA_ADMIN_PASSWORD='):
        raise ValueError('Invalid Grafana credentials format')
    password = lines[0].split('=',1)[1]
    if re.fullmatch(r'[A-Za-z0-9_-]{20,100}',password) is None:
        raise ValueError('Invalid Grafana credential value')
    return password


def request_json(path, password=None):
    # No proxy and no redirect: a Grafana redirect must never take Basic auth
    # to another origin. The caller supplies only fixed internal paths.
    if not path.startswith('/api/') or '//' in path:
        raise ValueError('Unapproved API path')
    opener = build_opener(ProxyHandler({}), NoRedirect())
    headers = {'Accept':'application/json'}
    if password:
        raw = 'admin:' + password
        headers['Authorization'] = 'Basic ' + base64.b64encode(raw.encode('ascii')).decode('ascii')
    try:
        with opener.open(Request(BASE+path, headers=headers), timeout=5) as response:
            if response.status != 200:
                return response.status, None
            data = response.read(1_000_001)
    except HTTPError as exc:
        return exc.code, None
    except (OSError, URLError) as exc:
        raise ValueError('Local Grafana unavailable') from None
    if len(data)>1_000_000:
        raise ValueError('Grafana response too large')
    try:
        return 200, json.loads(data)
    except (UnicodeError, json.JSONDecodeError):
        raise ValueError('Grafana response is not valid JSON') from None


def validate_log_response(body):
    if not isinstance(body,dict) or body.get('status')!='success':
        raise ValueError('Loki query failed')
    result = body.get('data',{}).get('result')
    if not isinstance(result,list):
        raise ValueError('Missing Loki result streams')
    count=0
    for stream in result:
        labels=stream.get('stream')
        if not isinstance(labels,dict) or labels.get('service_name')!='theia-api' or labels.get('environment')!='local':
            raise ValueError('Unapproved stream labels')
        if any(key in labels for key in ('request_id','turn_ref','trace_id','user_id','conversation_id','prompt','response')):
            raise ValueError('High-cardinality or private stream labels')
        for stamp,line in stream.get('values',[]):
            if not isinstance(stamp,str) or not isinstance(line,str) or len(line)>safe_line_max():
                raise ValueError('Unexpected Loki log entry')
            try: record=json.loads(line)
            except json.JSONDecodeError: raise ValueError('Malformed log JSON') from None
            if not isinstance(record,dict) or set(record)-ALLOWED or record.get('schema')!='theia.safe-log.v1':
                raise ValueError('Unapproved fields in Loki response')
            count+=1
    if count==0:
        raise ValueError('No logs found: generate synthetic HTTP traffic and retry')
    return count


def safe_line_max():
    return 32_768


def main():
    password=read_password()
    status,_=request_json('/api/datasources/uid/theia-loki')
    if status not in (302,401,403):
        raise ValueError('Grafana allows unauthenticated datasource access')
    status,source=request_json('/api/datasources/uid/theia-loki',password)
    if status!=200 or not isinstance(source,dict) or source.get('uid')!='theia-loki':
        raise ValueError('Authenticated Grafana Loki datasource unavailable')
    end=time.time_ns()
    args=urlencode({'query':'{service_name="theia-api",environment="local"}',
                    'start':end-15*60*1_000_000_000, 'end':end,'limit':100})
    status,data=request_json(PREFIX+'/loki/api/v1/query_range?'+args,password)
    if status!=200:
        raise ValueError('Grafana Loki proxy unavailable')
    print('Private Loki readback verified: %d sanitized records; no sensitive fields or indexed IDs.' % validate_log_response(data))


if __name__=='__main__':
    try: main()
    except (ValueError, OSError) as exc:
        sys.exit('MON-01B smoke failed: '+str(exc))
