"""Static BO-03 contract and SQL migration regression guard."""
from pathlib import Path
import unittest
import yaml

ROOT = Path(__file__).resolve().parents[2]


class AdminAuditTests(unittest.TestCase):
    def test_staff_only_audit_contract_and_bounded_page(self):
        spec = yaml.safe_load((ROOT / 'contracts/openapi/openapi.yaml').read_text())
        route = spec['paths']['/api/v1/admin/audit']['get']
        self.assertEqual(route['security'], [{'staffSession': []}])
        self.assertEqual(route['responses']['200']['content']['application/json']['schema']['$ref'],
                         '#/components/schemas/AdminAuditPage')
        self.assertEqual(spec['components']['schemas']['AdminAuditPage']['properties']['items']['maxItems'], 100)
        params = {item['name']: item for item in route['parameters']}
        self.assertEqual(params['pageSize']['schema']['maximum'], 100)
        self.assertEqual(params['cursor']['schema']['maxLength'], 180)
        for forbidden in ('devSession', 'userSession'):
            self.assertNotIn(forbidden, str(route))

    def test_audit_migration_is_append_only_and_metadata_only(self):
        sql = (ROOT / 'apps/api/db/migrations/0026_admin_audit_events.sql').read_text()
        self.assertIn('CREATE TABLE dbo.admin_audit_events', sql)
        self.assertIn('admin_audit_events_block_mutation', sql)
        self.assertIn('AFTER UPDATE, DELETE', sql)
        for forbidden in ('phone_number ', 'message_text ', 'prompt_text ', 'token_hash ', 'raw_payload '):
            self.assertNotIn(forbidden, sql.lower())
        app = (ROOT / 'apps/api/internal/modules/admin/adapters/mssql/audit.go').read_text()
        self.assertIn('INSERT INTO dbo.admin_audit_events', app)
        self.assertIn('SELECT TOP (@p1)', app)
        self.assertIn('ORDER BY recorded_at DESC,id DESC', app)
        for forbidden in ('UPDATE dbo.admin_audit_events', 'DELETE FROM dbo.admin_audit_events'):
            self.assertNotIn(forbidden, app)

    def test_every_admin_route_uses_existing_staff_guard(self):
        api = (ROOT / 'apps/api/internal/app/composition/api.go').read_text()
        self.assertIn('adminAPI.Use(bff.RequireStaff())', api)
        self.assertIn('adminAPI.GET("/audit", adminModule.Audit)', api)
        self.assertLess(api.index('adminAPI.Use(bff.RequireStaff())'),
                        api.index('adminAPI.GET("/audit", adminModule.Audit)'))


if __name__ == '__main__':
    unittest.main()
