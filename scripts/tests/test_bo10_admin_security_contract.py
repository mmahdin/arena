"""BO-10: protect the existing Admin route/auth/PII contract during refactors.

Pure static checks are complementary to the Go guard/audit matrix and live
SQL/Keycloak testing; they do not prove runtime authorization by themselves.
"""
import re
import unittest
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parents[2]


class AdminBoundaryContract(unittest.TestCase):
    def test_every_admin_operation_declares_the_staff_session(self):
        spec = yaml.safe_load((ROOT / 'contracts/openapi/openapi.yaml').read_text())
        admin_ops = {
            (method.upper(), path): operation
            for path, methods in spec['paths'].items() if path.startswith('/api/v1/admin/')
            for method, operation in methods.items()
            if method.lower() in {'get', 'post', 'patch', 'put', 'delete'}
        }
        self.assertEqual(len(admin_ops), 12, 'Update BO-10 security matrix before adding Admin endpoints')
        for (method, path), operation in admin_ops.items():
            with self.subTest(method=method, path=path):
                self.assertEqual(operation.get('security'), [{'staffSession': []}])
                for status in ('401', '503'):
                    self.assertIn(status, operation.get('responses', {}))
                self.assertNotIn('userSession', repr(operation.get('security')))

    def test_admin_routes_are_registered_only_on_guarded_group(self):
        source = (ROOT / 'apps/api/internal/app/composition/api.go').read_text()
        begin = source.index('adminAPI := group.Group("/admin")')
        guard = source.index('adminAPI.Use(bff.RequireStaff())', begin)
        first_route = re.search(r'adminAPI\.(?:GET|POST|PATCH|PUT|DELETE)\(', source[begin:])
        self.assertIsNotNone(first_route)
        self.assertLess(guard, begin + first_route.start())
        self.assertIn('adminAPI.Use(func(c *gin.Context)', source[begin:first_route.start()+begin])
        self.assertIn('AUTH_NOT_CONFIGURED', source[begin:first_route.start()+begin])
        self.assertIsNone(re.search(r'(?:router|group|authorized)\.(?:GET|POST|PATCH|PUT|DELETE)\("/admin/', source))

    def test_operational_queries_do_not_select_private_payload_columns(self):
        adapter = ROOT / 'apps/api/internal/modules/admin/adapters/mssql'
        forbidden = re.compile(r'\b(?:user_id|payload_json|request_meta_json|result_meta_json|'
                               r'prompt_text|response_text|content_text|phone_number|email_address)\b', re.I)
        for name in ('inspector.go', 'overview.go'):
            source = (adapter / name).read_text()
            # Only check SELECT statements; comments explicitly mention omitted columns.
            statements = re.findall(r'`(SELECT\b.*?)`', source, flags=re.I | re.S)
            self.assertTrue(statements, name)
            for query in statements:
                with self.subTest(source=name, query=query[:55]):
                    self.assertNotRegex(query, forbidden)


if __name__ == '__main__':
    unittest.main()
