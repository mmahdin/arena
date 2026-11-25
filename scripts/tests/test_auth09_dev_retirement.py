"""AUTH-09: explicit irreversible retirement of synthetic browser authentication."""
from pathlib import Path
import re
import unittest
import yaml

ROOT = Path(__file__).resolve().parents[2]

class LegacySessionRetirement(unittest.TestCase):
    def test_old_runtime_modules_gone(self):
        self.assertFalse((ROOT / 'apps/api/internal/modules/devsession').exists())
        self.assertFalse((ROOT / 'apps/api/internal/platform/devsecurity').exists())

    def test_product_routes_are_bff_only(self):
        code = (ROOT / 'apps/api/internal/app/composition/api.go').read_text()
        self.assertNotIn('devsession', code)
        self.assertNotIn('devsecurity', code)
        self.assertIn('bff.RequireUser()', code)
        self.assertIn('AUTH_NOT_CONFIGURED', code)
        self.assertNotIn('group.POST("/dev/session"', code)

    def test_contract_has_no_legacy_security_scheme(self):
        spec = yaml.safe_load((ROOT / 'contracts/openapi/openapi.yaml').read_text())
        self.assertFalse(any(p.startswith('/api/v1/dev/') for p in spec['paths']))
        self.assertNotIn('devSession', spec['components']['securitySchemes'])
        self.assertIn('userSession', spec['components']['securitySchemes'])
        for path, methods in spec['paths'].items():
            if path.startswith('/api/v1/me') or path.startswith('/api/v1/conversations'):
                for method, operation in methods.items():
                    if isinstance(operation, dict) and 'security' in operation:
                        self.assertNotIn({'devSession': []}, operation['security'])

    def test_database_forward_migration_removes_only_retired_tables(self):
        migration = (ROOT / 'apps/api/db/migrations/0032_retire_dev_identity.sql').read_text()
        self.assertIn('DROP TABLE dbo.dev_sessions', migration)
        self.assertIn('DROP TABLE dbo.dev_phone_identities', migration)
        self.assertNotIn('dbo.auth_sessions', migration)
        self.assertIn('UPDATE dbo.auth_sessions', (ROOT / 'apps/api/internal/modules/user/adapters/mssql/ai_privacy.go').read_text())

    def test_deprecated_flag_cannot_reenable_dev_login(self):
        config = (ROOT / 'apps/api/internal/platform/config/config.go').read_text()
        self.assertIn('ALLOW_DEV_LOGIN was retired in AUTH-09', config)
        for p in ('compose.yaml', 'compose.auth-bff.yaml'):
            self.assertNotIn('ALLOW_DEV_LOGIN:', (ROOT / p).read_text())

if __name__ == '__main__': unittest.main()
