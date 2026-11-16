#!/usr/bin/env python3
"""Integration smoke test for local-only mock AI + synthetic SQL Server data.

Requires: docker compose up --build -d, mock provider, local-only port.
No third-party Python dependencies and no real private content.
"""
import http.cookiejar
import json
import os
import urllib.error
import urllib.request
import uuid

BASE = os.environ.get('SMOKE_BASE_URL', 'http://127.0.0.1:8080').rstrip('/')
U1 = '11111111-1111-4111-8111-111111111111'
U2 = '22222222-2222-4222-8222-222222222222'

def client():
    jar = http.cookiejar.CookieJar()
    return urllib.request.build_opener(urllib.request.HTTPCookieProcessor(jar))

def query(opener, method, route, body=None, status=200, headers=None):
    payload = json.dumps(body).encode('utf-8') if body is not None else None
    req = urllib.request.Request(BASE + route, data=payload, method=method)
    if body is not None:
        req.add_header('Content-Type', 'application/json')
    for key, value in (headers or {}).items():
        req.add_header(key, value)
    try:
        with opener.open(req, timeout=15) as response:
            got = response.status
            data = response.read()
    except urllib.error.HTTPError as exc:
        got = exc.code
        data = exc.read()
    if got != status:
        raise AssertionError(f'{method} {route}: expected {status}, got {got}, body={data[:200]!r}')
    return json.loads(data) if data else None

def main():
    a, b = client(), client()
    query(a, 'GET', '/health/live')
    query(a, 'GET', '/health/ready')
    query(a, 'GET', '/api/v1/me', status=401)
    users = query(a, 'GET', '/api/v1/dev/users')['items']
    assert len(users) >= 3
    query(a, 'POST', '/api/v1/dev/session', {'userId': U1})
    session_cookie = next(c.value for handler in a.handlers if isinstance(handler, urllib.request.HTTPCookieProcessor) for c in handler.cookiejar if c.name == "wellbeing_dev_session")
    query(b, 'POST', '/api/v1/dev/session', {'userId': U2})
    profile_a = query(a, 'GET', '/api/v1/me')
    profile_b = query(b, 'GET', '/api/v1/me')
    assert profile_a['userId'] == U1 and profile_b['userId'] == U2
    # Seed tester 2 begins without onboarding; do not mutate demo state here.
    if profile_b['onboardingCompletedAt'] is None:
        query(b, 'POST', '/api/v1/conversations', {}, status=409)
    conv = query(a, 'POST', '/api/v1/conversations', {}, 201)
    cid = conv['id']
    try:
        query(b, 'GET', f'/api/v1/conversations/{cid}/messages', status=404)
        query(b, 'DELETE', f'/api/v1/conversations/{cid}', status=404)
        tid = str(uuid.uuid4())
        body = {'clientTurnId': tid, 'message': 'یک پیام کاملا ساختگی برای تست.'}
        first = query(a, 'POST', f'/api/v1/conversations/{cid}/turns', body)
        again = query(a, 'POST', f'/api/v1/conversations/{cid}/turns', body)
        assert first['reply'] == again['reply'] and first['assistantMessageId'] == again['assistantMessageId']
        wrong = query(a, 'POST', f'/api/v1/conversations/{cid}/turns', {'clientTurnId': tid, 'message': 'changed test text'}, status=400)
        assert wrong['code'] == 'VALIDATION_ERROR'
        messages = query(a, 'GET', f'/api/v1/conversations/{cid}/messages')['items']
        assert len(messages) == 2 and messages[0]['role'] == 'user' and messages[1]['role'] == 'assistant'
        second = query(a, 'POST', f'/api/v1/conversations/{cid}/turns', {'clientTurnId': str(uuid.uuid4()), 'message': 'پیام دوم ساختگی'})
        assert second['assistantMessageId'] != first['assistantMessageId']
        messages = query(a, 'GET', f'/api/v1/conversations/{cid}/messages')['items']
        assert len(messages) == 4 and [x['role'] for x in messages] == ['user', 'assistant', 'user', 'assistant']
        assert any(x['id'] == cid for x in query(a, 'GET', '/api/v1/conversations')['items'])
        query(b, 'POST', f'/api/v1/conversations/{cid}/turns', {'clientTurnId': str(uuid.uuid4()), 'message': 'not mine'}, status=404)
        query(a, 'POST', '/api/v1/dev/session', {'userId': U2})
        assert query(a, 'GET', '/api/v1/me')['userId'] == U2
        query(a, 'GET', '/api/v1/me', status=401, headers={'Cookie': 'wellbeing_dev_session=' + session_cookie})
        query(a, 'GET', f'/api/v1/conversations/{cid}/messages', status=404)
        query(a, 'DELETE', '/api/v1/dev/session', status=204)
        query(a, 'GET', '/api/v1/me', status=401)
        print('PASS: revoked sessions, switch/logout, isolated profiles, foreign 404, idempotent turns, ordered multi-turn history')
    finally:
        query(a, 'POST', '/api/v1/dev/session', {'userId': U1})
        query(a, 'DELETE', f'/api/v1/conversations/{cid}', status=204)

if __name__ == '__main__':
    main()
