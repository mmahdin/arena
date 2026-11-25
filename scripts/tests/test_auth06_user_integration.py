"""AUTH-06 static guardrails, complementary to Go/Keycloak/SQL integration."""
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[2]


class UserIntegrationContract(unittest.TestCase):
    def test_signed_sms_session_note_is_mandatory(self):
        spi = (ROOT / "infra/keycloak/sms-spi/src/main/java/ir/theia/keycloak/sms/SmsOtpAuthenticator.java").read_text()
        verifier = (ROOT / "apps/api/internal/modules/auth/oidc/jwt.go").read_text()
        client = (ROOT / "apps/api/internal/modules/auth/oidc/client.go").read_text()
        settings = (ROOT / "apps/api/internal/platform/config/auth.go").read_text()
        self.assertIn('setUserSessionNote("theia_sms_verified", "sms-otp-v1")', spi)
        self.assertIn('c.SMSVerified != "sms-otp-v1"', verifier)
        self.assertIn('p.cfg.RequireSMSProof', client)
        self.assertIn('RequireSMSProof: true', settings)
        self.assertIn('AUTH_USER_SMS_FLOW_VERIFIED', settings)

    def test_product_handlers_no_longer_depend_on_devsession(self):
        modules = ROOT / 'apps/api/internal/modules'
        protected = ('user', 'conversation', 'ai', 'letters', 'bookmark')
        for name in protected:
            for handler in (modules / name / 'adapters/http').glob('*.go'):
                if handler.name.endswith('_test.go'):
                    continue
                with self.subTest(handler=handler.name, module=name):
                    self.assertNotIn('devsecurity.UserID(', handler.read_text())
        self.assertIn('principal.SetUserID(c, s.PrincipalID)', (modules / 'auth/adapters/http/handler.go').read_text())
        self.assertFalse((ROOT / 'apps/api/internal/platform/devsecurity').exists())

    def test_activation_does_not_implicitly_enable_login(self):
        code = (ROOT / 'scripts/auth/activate_user_bff.py').read_text()
        self.assertIn('oidc-usersessionmodel-note-mapper', code)
        self.assertIn('"directAccessGrantsEnabled"', code)
        self.assertIn('"requirement") != "REQUIRED"', code)
        bff = (ROOT / 'compose.auth-bff.yaml').read_text()
        self.assertIn('AUTH_USER_LOGIN_ENABLED: ${AUTH_USER_LOGIN_ENABLED:-false}', bff)
        self.assertIn('AUTH_USER_SMS_FLOW_VERIFIED: ${AUTH_USER_SMS_FLOW_VERIFIED:-false}', bff)


if __name__ == '__main__':
    unittest.main()
