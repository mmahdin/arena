#!/usr/bin/env python3
"""Idempotent local Metabase setup and synthetic product dashboards.

Only fixed SQL against approved aggregate views is shipped. Never passes
credentials on argv, prints API error response bodies or follows redirects.
"""
import argparse
import json
import sys
from urllib.error import HTTPError, URLError
from urllib.request import HTTPRedirectHandler, ProxyHandler, Request, build_opener

from init_local_analytics_env import read_local_env

BASE = 'http://127.0.0.1:3100'
DATABASE_NAME = 'Theia · Synthetic Reporting'
COLLECTION_NAME = 'Theia · Synthetic Product Analytics'
DASHBOARD_NAME = 'Theia · Synthetic Product Overview'

# The views contain no person-level identifiers; these SQL queries must NEVER
# select directly from event rows, users, messages, turns or AI attempts.
QUESTIONS = (
    ('Chat-active synthetic users / day', 'line',
     'SELECT activity_date_utc,chat_active_test_users FROM dbo.v_product_daily_engagement_synthetic ORDER BY activity_date_utc', 0, 0),
    ('Admitted and completed turns / day', 'line',
     'SELECT activity_date_utc,turns_admitted,turns_completed FROM dbo.v_product_daily_engagement_synthetic ORDER BY activity_date_utc', 0, 12),
    ('Registration-to-onboarding cohorts', 'bar',
     'SELECT cohort_date_utc,registered_test_users,started_test_users,completed_test_users FROM dbo.v_product_onboarding_cohorts_synthetic ORDER BY cohort_date_utc', 7, 0),
    ('First-chat cohorts: exact D1, D7 and D30', 'table',
     'SELECT cohort_date_utc,first_chat_test_users,d1_returning_test_users,d7_returning_test_users,d30_returning_test_users FROM dbo.v_product_chat_retention_synthetic ORDER BY cohort_date_utc DESC', 7, 12),
    ('Known configured-price cost / day (USD, partial)', 'line',
     'SELECT attempt_date_utc,CONVERT(decimal(18,6),known_estimated_micro_usd)/1000000.0 AS known_estimated_usd FROM dbo.v_product_ai_usage_daily_synthetic ORDER BY attempt_date_utc', 15, 0),
    ('LLM attempt health and pricing coverage', 'table',
     'SELECT attempt_date_utc,llm_attempts,succeeded_attempts,failed_attempts,incomplete_attempts,priced_attempts,unpriced_attempts,missing_usage_attempts FROM dbo.v_product_ai_usage_daily_synthetic ORDER BY attempt_date_utc DESC', 15, 12),
)


class NoRedirect(HTTPRedirectHandler):
    def redirect_request(self, request, file_pointer, code, message, headers, new_url):
        return None


class APIError(Exception):
    pass


class Metabase:
    def __init__(self, base=BASE):
        if base != BASE:
            raise ValueError('Metabase administration must target local loopback')
        self.base = base
        self.opener = build_opener(ProxyHandler({}), NoRedirect())
        self.session = None

    def request(self, method, path, payload=None):
        if not path.startswith('/api/') or '://' in path or '..' in path:
            raise ValueError('Invalid API path')
        headers = {'Accept': 'application/json', 'Content-Type': 'application/json'}
        if self.session:
            headers['X-Metabase-Session'] = self.session
        body = json.dumps(payload).encode('utf-8') if payload is not None else None
        request = Request(self.base + path, data=body, headers=headers, method=method)
        try:
            with self.opener.open(request, timeout=20) as response:
                raw = response.read(2_000_000)
                return json.loads(raw.decode('utf-8')) if raw else {}
        except HTTPError as exc:
            raise APIError(f'Metabase API {method} {path.split("?",1)[0]} returned HTTP {exc.code}') from None
        except (URLError, TimeoutError):
            raise APIError('Local Metabase unavailable (localhost:3100)') from None
        except (UnicodeError, ValueError):
            raise APIError('Local Metabase returned invalid JSON') from None

    def login_or_setup(self, env):
        props = self.request('GET', '/api/session/properties')
        if not props.get('has-user-setup', False):
            token = props.get('setup-token')
            if not token:
                raise APIError('Metabase initial setup token unavailable')
            result = self.request('POST', '/api/setup', {
                'token': token,
                'user': {'first_name':'Theia','last_name':'Admin',
                         'email':env['THEIA_METABASE_ADMIN_EMAIL'],
                         'password':env['THEIA_METABASE_ADMIN_PASSWORD']},
                'prefs': {'site_name': 'Theia Synthetic Analytics', 'site_locale':'en'},
            })
        else:
            result = self.request('POST','/api/session',{
                'username':env['THEIA_METABASE_ADMIN_EMAIL'],
                'password':env['THEIA_METABASE_ADMIN_PASSWORD'],
            })
        self.session = result.get('id')
        if not self.session:
            raise APIError('Metabase authentication did not create a session')


def as_list(result):
    if isinstance(result, list):
        return result
    if isinstance(result, dict) and isinstance(result.get('data'), list):
        return result['data']
    raise APIError('Unexpected Metabase collection response')


def report_database(client, secret, native=False):
    host, port = ('127.0.0.1', 14333) if native else ('mssql', 1433)
    for database in as_list(client.request('GET','/api/database')):
        if database.get('name') != DATABASE_NAME:
            continue
        if database.get('engine') != 'sqlserver':
            raise APIError('An incompatible data source has the reporting name')
        details = client.request('GET', f'/api/database/{database["id"]}').get('details') or {}
        if details.get('user') != 'theia_metabase_reader':
            raise APIError('Existing reporting datasource does not use the restricted reader')
        if details.get('host') != host or str(details.get('port')) != str(port) or details.get('db') != 'wellbeing':
            raise APIError('Existing reporting datasource points at a different SQL server or database')
        return database['id']
    created = client.request('POST','/api/database',{
        'name': DATABASE_NAME,
        'engine': 'sqlserver',
        'details': {'host':host,'port':port,'db':'wellbeing',
                    'user':'theia_metabase_reader','password':secret,
                    'additional-options':'encrypt=false'},
        'auto_run_queries':False,
    })
    if not isinstance(created.get('id'), int):
        raise APIError('Metabase did not return the new database ID')
    return created['id']


def report_collection(client):
    for item in as_list(client.request('GET','/api/collection')):
        if item.get('name') == COLLECTION_NAME:
            return item['id']
    created = client.request('POST','/api/collection',{
        'name': COLLECTION_NAME,
        'description': 'Synthetic-only aggregate views; no individual user records.',
        'color':'#276FBF',
    })
    return created['id']


def existing_items(client, collection_id):
    # Metabase 0.64 returns a paginated collection listing. Fail safely if its
    # API shape changes rather than silently creating duplicate dashboards.
    result = client.request('GET',f'/api/collection/{collection_id}/items?limit=100')
    return as_list(result)


def create_dashboard(client, database_id, collection_id):
    existing = existing_items(client, collection_id)
    cards = {}
    dashboard_id = None
    for item in existing:
        if item.get('name') == DASHBOARD_NAME and item.get('model') == 'dashboard':
            dashboard_id = item['id']
        for name, _, _, _, _ in QUESTIONS:
            if item.get('name') == name and item.get('model') == 'card':
                cards[name] = item['id']
    for name, display, sql, _, _ in QUESTIONS:
        if name in cards:
            continue
        created = client.request('POST','/api/card', {
            'name': name,'collection_id':collection_id,'display':display,
            'description': 'Synthetic-only, post-PROD-01 event data. Read docs/analytics/PROD02_METRICS.md.',
            'dataset_query': {
                'database': database_id,'type':'native',
                'native':{'query':sql,'template-tags':{}}
            },
            'visualization_settings':{},
        })
        cards[name] = created['id']
    if dashboard_id is None:
        created = client.request('POST','/api/dashboard', {
            'name':DASHBOARD_NAME,'collection_id':collection_id,
            'description':'Synthetic-only onboarding, chat engagement and partial AI usage; not real-user KPIs.',
        })
        dashboard_id = created['id']
    full = client.request('GET',f'/api/dashboard/{dashboard_id}')
    attached = {entry.get('card_id') for entry in full.get('dashcards', [])}
    if any(cards[name] not in attached for name,_,_,_,_ in QUESTIONS):
        # Preserve existing cards and their layout on repeat execution.
        placement = [
            {'id':-i-1,'card_id':cards[name],'row':row,'col':col,'size_x':12,'size_y':7}
            for i,(name,_,_,row,col) in enumerate(QUESTIONS)
            if cards[name] not in attached
        ]
        client.request('PUT',f'/api/dashboard/{dashboard_id}', {
            'dashcards':full.get('dashcards',[]) + placement,
        })
    return dashboard_id


def verify_reader(client, database_id):
    def query(sql):
        return client.request('POST','/api/dataset', {
            'database':database_id,'type':'native',
            'native':{'query':sql,'template-tags':{}},'parameters':[],
        })
    allowed = query('SELECT TOP (1) chat_active_test_users FROM dbo.v_product_daily_engagement_synthetic')
    if allowed.get('status') != 'completed':
        raise APIError('Reporting view query failed')
    for table in ('product_events','users','messages','ai_llm_attempts','turns'):
        try:
            forbidden = query('SELECT TOP (1) * FROM dbo.' + table)
        except APIError as error:
            if 'returned HTTP 400' in str(error) or 'returned HTTP 403' in str(error):
                continue
            raise
        if forbidden.get('status') == 'completed':
            raise APIError('SECURITY FAILURE: reporting reader could SELECT forbidden base table')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--native', action='store_true',
                        help='Use loopback SQL Server when the Metabase JAR runs on the host.')
    args = parser.parse_args()
    env = read_local_env()
    client = Metabase()
    client.login_or_setup(env)
    db_id = report_database(client, env['THEIA_METABASE_READER_PASSWORD'], native=args.native)
    verify_reader(client, db_id)
    collection_id = report_collection(client)
    dashboard_id = create_dashboard(client, db_id, collection_id)
    print(f'Synthetic product analytics ready: {BASE}/dashboard/{dashboard_id}')
    print('Data source permission checks passed (aggregate views allowed; base tables denied).')


if __name__ == '__main__':
    try:
        main()
    except APIError as error:
        sys.exit(f'Local Metabase provisioning failed: {error}')
    except (ValueError, OSError, KeyError, TypeError):
        sys.exit('Local Metabase provisioning failed (sensitive details suppressed); see PROD-02 runbook')
