"""Offline security/semantic contract regression for PROD-02."""
import importlib
from pathlib import Path
import os
import stat
import sys
import tempfile
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0,str(ROOT/'scripts/analytics'))
import init_local_analytics_env as secrets_file
import provision_metabase as metabase


class ReportingContract(unittest.TestCase):
    def test_sql_migration_grants_only_safe_views(self):
        sql=(ROOT/'apps/api/db/migrations/0037_product_reporting_views.sql').read_text()
        for view in ('v_product_onboarding_cohorts_synthetic','v_product_daily_engagement_synthetic',
                     'v_product_chat_retention_synthetic','v_product_ai_usage_daily_synthetic'):
            self.assertIn('CREATE VIEW dbo.'+view,sql)
            self.assertIn('GRANT SELECT ON OBJECT::dbo.'+view+' TO theia_analytics_reader',sql)
        for table in ('users','product_events','messages','turns','ai_llm_attempts'):
            self.assertIn('DENY SELECT ON OBJECT::dbo.'+table+' TO theia_analytics_reader',sql)
        self.assertIn("u.is_test=1",sql)
        self.assertIn("DATEADD(day,-30,CONVERT(date,SYSUTCDATETIME()))",sql)
        self.assertIn('ELSE CAST(NULL AS bigint)',sql)
        self.assertNotIn('CREATE LOGIN',sql)
        self.assertNotIn('BEGIN TRAN',sql)
        self.assertNotIn('\nGO\n',sql)

    def test_compose_has_pinned_images_and_loopback_only(self):
        compose=(ROOT/'compose.analytics.dev.yaml').read_text()
        self.assertIn('metabase/metabase:v0.64.8.3',compose)
        self.assertIn('postgres:16.16-alpine',compose)
        self.assertIn('127.0.0.1:3100:3000',compose)
        self.assertNotIn('1433:1433',compose)
        self.assertNotIn('5432:5432',compose)
        self.assertIn('MB_ENABLE_PUBLIC_SHARING: "false"',compose)
        self.assertIn('MB_ANON_TRACKING_ENABLED: "false"',compose)
        self.assertNotIn('MSSQL_SA_PASSWORD',compose)
        self.assertIn('THEIA_METABASE_POSTGRES_PASSWORD',compose)
        self.assertIn('MB_ENCRYPTION_SECRET_KEY',compose)

    def test_generated_secrets_owner_only_and_idempotent(self):
        with tempfile.TemporaryDirectory() as tmp:
            path=Path(tmp)/'.env.analytics'
            self.assertTrue(secrets_file.initialize(path))
            self.assertFalse(secrets_file.initialize(path))
            self.assertEqual(stat.S_IMODE(path.stat().st_mode),0o600)
            values=secrets_file.read_local_env(path)
            self.assertEqual(len(values['THEIA_METABASE_READER_PASSWORD']),64)
            self.assertTrue(values['THEIA_METABASE_READER_PASSWORD'].startswith('Aa1!'))
            other=Path(tmp)/'other'
            other.write_text('secret')
            path.unlink()
            path.symlink_to(other)
            with self.assertRaises(ValueError):
                secrets_file.initialize(path)
            self.assertEqual(other.read_text(),'secret')

    def test_fixed_queries_only_read_aggregate_views(self):
        self.assertEqual(len(metabase.QUESTIONS),6)
        for name,display,sql,row,col in metabase.QUESTIONS:
            with self.subTest(name=name):
                self.assertIn('FROM dbo.v_product_',sql)
                for forbidden in ('FROM dbo.users','FROM dbo.messages','FROM dbo.product_events',
                                  'FROM dbo.turns','FROM dbo.ai_llm_attempts'):
                    self.assertNotIn(forbidden,sql)
                self.assertNotIn('user_id',sql)
                self.assertIn(display,('line','bar','table'))

    def test_reader_probe_fails_closed(self):
        class Fake:
            def request(self,method,path,payload=None):
                if payload and path == '/api/dataset':
                    text=payload['native']['query']
                    if 'dbo.v_product_' in text:
                        return {'status':'completed'}
                    return {'status':'failed'}
                raise AssertionError('unexpected API call')
        metabase.verify_reader(Fake(),1)
        class Unsafe(Fake):
            def request(self,method,path,payload=None):
                return {'status':'completed'}
        with self.assertRaises(metabase.APIError):
            metabase.verify_reader(Unsafe(),1)

    def test_api_does_not_follow_redirects(self):
        self.assertIsNone(metabase.NoRedirect().redirect_request(None,None,302,'redirect',{},'https://other.invalid/'))
        with self.assertRaises(ValueError):
            metabase.Metabase('http://anywhere.invalid')

    def test_existing_datasource_must_match_selected_local_sql_endpoint(self):
        class Fake:
            def request(self, method, path, payload=None):
                if path == '/api/database':
                    return [{'id': 1, 'name': metabase.DATABASE_NAME, 'engine': 'sqlserver'}]
                if path == '/api/database/1':
                    return {'details': {'host': '127.0.0.1', 'port': '14333', 'db': 'wellbeing',
                                        'user': 'theia_metabase_reader'}}
                raise AssertionError(path)

        self.assertEqual(metabase.report_database(Fake(), 'unused', native=True), 1)
        with self.assertRaises(metabase.APIError):
            metabase.report_database(Fake(), 'unused')

    def test_dashboard_provisioning_is_idempotent(self):
        class Fake:
            def __init__(self):
                self.items=[]
                self.next_id=10
                self.dashcards=[]
                self.calls=[]
            def request(self,method,path,payload=None):
                self.calls.append((method,path))
                if method=='GET' and '/items?' in path:
                    return {'data':self.items}
                if method=='POST' and path=='/api/card':
                    self.next_id+=1
                    self.items.append({'model':'card','id':self.next_id,'name':payload['name']})
                    return {'id':self.next_id}
                if method=='POST' and path=='/api/dashboard':
                    self.next_id+=1
                    self.items.append({'model':'dashboard','id':self.next_id,'name':payload['name']})
                    return {'id':self.next_id}
                if method=='GET' and path.startswith('/api/dashboard/'):
                    return {'dashcards':self.dashcards}
                if method=='PUT' and path.startswith('/api/dashboard/'):
                    self.dashcards=[{'card_id':x['card_id'],'id':i} for i,x in enumerate(payload['dashcards'])]
                    return {}
                raise AssertionError((method,path))
        f=Fake()
        first=metabase.create_dashboard(f,1,2)
        second=metabase.create_dashboard(f,1,2)
        self.assertEqual(first,second)
        self.assertEqual(len(f.items),7)
        self.assertEqual(len(f.dashcards),6)
        self.assertEqual(sum(m=='POST' for m,_ in f.calls),7)

    def test_go_reader_is_restricted_to_local_and_generated_secret(self):
        go=(ROOT/'apps/api/cmd/analytics-reader/main.go').read_text()
        self.assertIn('if env != "local"',go)
        self.assertIn('^Aa1![0-9a-f]{60}$',go)
        self.assertIn('migration 0037 must be applied first',go)
        self.assertIn('theia_analytics_reader',go)
        self.assertNotIn('log.Print(secret)',go)
        script=(ROOT/'scripts/analytics/bootstrap_reader.py').read_text()
        self.assertIn("'@127.0.0.1:14333?database=wellbeing&encrypt=disable'",script)
        self.assertIn("env.pop(key)",script)


if __name__=='__main__':
    unittest.main()
