"""Static protection for BO-01/02 composition and OpenAPI route boundary.

The browser/session integration itself is tested by the Go Admin handler
suite with an in-memory Auth Store; this check catches composition drift.
"""
from pathlib import Path
import unittest
import yaml

ROOT = Path(__file__).resolve().parents[2]


class AdminBoundaryTests(unittest.TestCase):
    def test_admin_me_contract_uses_staff_cookie_only(self):
        spec = yaml.safe_load((ROOT / 'contracts/openapi/openapi.yaml').read_text())
        entry = spec['paths']['/api/v1/admin/me']['get']
        self.assertEqual(entry['security'], [{'staffSession': []}])
        self.assertEqual(entry['responses']['200']['content']['application/json']['schema']['$ref'],
                         '#/components/schemas/AdminMe')
        self.assertEqual(set(spec['components']['schemas']['AdminMe']['required']),
                         {'staffId', 'role', 'active', 'expiresAt'})
        self.assertEqual(spec['components']['schemas']['AdminMe']['properties']['role']['const'], 'admin')
        self.assertNotIn('devSession', str(entry))
        self.assertNotIn('userSession', str(entry))

    def test_admin_group_always_guarded_and_independent_of_dev_login(self):
        composition = (ROOT / 'apps/api/internal/app/composition/api.go').read_text()
        self.assertIn('adminAPI := group.Group("/admin")', composition)
        self.assertIn('adminAPI.Use(bff.RequireStaff())', composition)
        self.assertIn('adminAPI.GET("/me", adminModule.Me)', composition)
        self.assertIn('http.StatusServiceUnavailable, "AUTH_NOT_CONFIGURED"', composition)
        self.assertIn('userGuard = bff.RequireUser()', composition)
        self.assertNotIn('cfg.AllowDevLogin', composition)
        self.assertNotIn('adminAPI.Use(userGuard)', composition)
        self.assertNotIn('adminAPI.Use(devsecurity.Guard', composition)

    def test_admin_does_not_reimplement_staff_credentials_or_role_grants(self):
        root = ROOT / 'apps/api/internal/modules/admin'
        admin_sources = '\n'.join(p.read_text() for p in root.rglob('*.go'))
        for prohibited in ('admin_role_grants', 'admin_staff_identities',
                           'oauth2', 'client_secret', 'password_hash', 'jwks'):
            self.assertNotIn(prohibited, admin_sources.lower())


if __name__ == '__main__':
    unittest.main()
