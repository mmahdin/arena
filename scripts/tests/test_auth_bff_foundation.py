"""Static AUTH-03 guardrails; SQL/Keycloak runtime tested separately."""
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[2]


class AuthBffFoundationTests(unittest.TestCase):
    def test_compose_overlay_only_injects_settings_into_existing_api(self):
        contents = (ROOT / 'compose.auth-bff.yaml').read_text()
        import yaml
        services = yaml.safe_load(contents)['services']
        self.assertEqual(set(services), {'api'})
        env = services['api']['environment']
        self.assertEqual(env['AUTH_BFF_ENABLED'], '${AUTH_BFF_ENABLED:-false}')
        self.assertEqual(env['AUTH_USER_LOGIN_ENABLED'], '${AUTH_USER_LOGIN_ENABLED:-false}')
        self.assertIn('${AUTH_BFF_KEY_BASE64:', env['AUTH_BFF_KEY_BASE64'])
        self.assertIn('${THEIA_STAFF_BFF_CLIENT_SECRET:', env['THEIA_STAFF_BFF_CLIENT_SECRET'])

    def test_auth_tables_and_no_plain_oauth_secrets(self):
        sql = (ROOT / 'apps/api/db/migrations/0024_auth_identity_sessions.sql').read_text()
        for table in ('staff_accounts', 'auth_identities', 'auth_login_transactions', 'auth_sessions', 'auth_audit_events'):
            self.assertIn('CREATE TABLE dbo.' + table, sql)
        self.assertIn('sealed_payload varbinary(1024)', sql)
        self.assertIn('token_hash binary(32)', sql)
        self.assertNotIn('refresh_token nvarchar', sql)
        self.assertNotIn('phone nvarchar', sql)

    def test_login_gates_and_no_legacy_cookie_reuse(self):
        handler = (ROOT / 'apps/api/internal/modules/auth/adapters/http/handler.go').read_text()
        self.assertIn('cfg.UserSMSFlowVerified', handler)
        self.assertIn('cfg.UserIssuer.RequireSMSProof', handler)
        self.assertIn('theia_staff_session', handler)
        self.assertIn('theia_user_session', handler)
        self.assertNotIn('wellbeing_dev_session', handler)


if __name__ == '__main__':
    unittest.main()
