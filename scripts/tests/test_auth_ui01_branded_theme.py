"""AUTH-UI01 theme, input and challenge cancellation regressions without Keycloak startup."""
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
THEME = ROOT / 'infra/keycloak/themes/theia-sms/login'
SPI = ROOT / 'infra/keycloak/sms-spi/src/main/java/ir/theia/keycloak/sms'


class ThemeContract(unittest.TestCase):
    def test_user_only_branding_and_self_contained_template(self):
        layout = (THEME / 'theia-layout.ftl').read_text()
        self.assertTrue(layout.startswith('<#ftl output_format="HTML" auto_esc=true>'))
        self.assertIn('lang="fa" dir="rtl"', layout)
        self.assertIn('${url.resourcesPath}/css/theia-login.css', layout)
        self.assertIn('${url.resourcesPath}/js/theia-login.js', layout)
        self.assertIn('${url.resourcesPath}/img/theia-mark.jpg', layout)
        self.assertIn('theia-surface', layout)
        self.assertNotIn('http://', layout)
        self.assertNotIn('https://', layout)
        self.assertGreater((THEME / 'resources/img/theia-mark.jpg').stat().st_size, 10_000)
        self.assertIn('parent=keycloak', (THEME / 'theme.properties').read_text())

    def test_phone_and_code_post_only_to_keycloak_action(self):
        phone = (THEME / 'theia-sms-phone.ftl').read_text()
        code = (THEME / 'theia-sms-code.ftl').read_text()
        for template in (phone, code):
            self.assertIn('theia-layout.ftl', template)
            self.assertIn('${url.loginAction}', template)
            self.assertIn('method="post"', template)
            self.assertTrue(template.startswith('<#ftl output_format="HTML" auto_esc=true>'))
            self.assertIn('${smsError}', template)
            self.assertNotIn('?html', template)
            self.assertNotIn('Authorization:', template)
        self.assertIn('value="request"', phone)
        self.assertIn('autocomplete="tel-national"', phone)
        self.assertIn('value="verify"', code)
        self.assertIn('value="resend"', code)
        self.assertIn('value="change"', code)
        self.assertIn('autocomplete="one-time-code"', code)
        self.assertIn('data-wait-seconds=', code)
        self.assertIn('aria-hidden="true"', code)
        self.assertIn('role="alert"', code)
        self.assertNotIn('localStorage', code)

    def test_mobile_responsive_accessibility_and_reduced_motion(self):
        css = (THEME / 'resources/css/theia-login.css').read_text()
        js = (THEME / 'resources/js/theia-login.js').read_text()
        self.assertIn('100dvh', css)
        self.assertIn('max-width: 360px', css)
        self.assertIn('focus-visible', css)
        self.assertIn('prefers-reduced-motion', css)
        self.assertIn('.theia-otp.is-enhanced', css)
        self.assertIn('window.setInterval', js)
        self.assertIn('replace(/[^0-9]/g', js)
        self.assertIn('code.checkValidity()', js)
        self.assertIn('resend.disabled = remaining > 0', js)
        self.assertNotIn('fetch(', js)
        self.assertNotIn('document.cookie', js)

    def test_only_successful_verification_proves_sms(self):
        java = (SPI / 'SmsOtpAuthenticator.java').read_text()
        service = (SPI / 'ChallengeService.java').read_text()
        self.assertIn('"change".equals(command)', java)
        self.assertIn('service(ctx).cancel(auth.getAuthNote(CHALLENGE))', java)
        self.assertIn('session.removeAuthNote(SENT_AT)', java)
        self.assertIn('resendRemainingSeconds(auth.getAuthNote(SENT_AT)', java)
        self.assertEqual(java.count('setUserSessionNote("theia_sms_verified", "sms-otp-v1")'), 1)
        self.assertIn('public void cancel(String id)', service)
        self.assertIn('store.consume(challengeKey(id))', service)

    @unittest.skipUnless(shutil.which('node'), 'Node.js unavailable')
    def test_theme_javascript_syntax(self):
        result = subprocess.run(['node', '--check', str(THEME / 'resources/js/theia-login.js')],
                                capture_output=True, text=True, timeout=10)
        self.assertEqual(result.returncode, 0, result.stderr)

    @unittest.skipUnless(shutil.which('node'), 'Node.js unavailable')
    def test_js_otp_normalization_countdown_and_browser_resume(self):
        result = subprocess.run(['node', str(ROOT / 'scripts/tests/auth_ui01_theme_browser.mjs')],
                                capture_output=True, text=True, timeout=10)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn('AUTH_UI01_DOM_INTERACTIONS_OK', result.stdout)

    @unittest.skipUnless(shutil.which('javac') and shutil.which('java'), 'JDK unavailable')
    def test_cancelled_code_is_invalid_and_rate_limit_is_retained(self):
        names = ['PhoneNumber', 'SmsSender', 'OtpCrypto', 'AtomicStore', 'ChallengeService']
        source = '''
import ir.theia.keycloak.sms.*;
import java.time.*;
import java.util.*;
import java.util.concurrent.*;
public class AuthUI01Smoke {
  static class Store implements AtomicStore {
    ConcurrentHashMap<String, Map<String,String>> data = new ConcurrentHashMap<>();
    public boolean reserve(String key,long t){return data.putIfAbsent(key,Map.of())==null;}
    public void put(String key,long t,Map<String,String> value){data.put(key,Map.copyOf(value));}
    public Map<String,String> get(String key){return data.get(key);}
    public Map<String,String> consume(String key){return data.remove(key);}
  }
  static class Sender implements SmsSender {
    String code;
    public void send(String phone,String otp){ code=otp; }
  }
  public static void main(String[] args) throws Exception {
    byte[] secret = new byte[32]; new java.security.SecureRandom().nextBytes(secret);
    Store store = new Store(); Sender sender = new Sender();
    var svc = new ChallengeService(store, sender,
      new OtpCrypto(Base64.getEncoder().encodeToString(secret)),
      Clock.fixed(Instant.parse("2026-11-21T13:00:00Z"), ZoneOffset.UTC));
    var old = svc.send("09123456789", "127.0.0.1", null);
    String previousCode = sender.code;
    svc.cancel(old.id());
    if (svc.verify(old.id(), old.phone(), previousCode) != ChallengeService.Verified.INVALID)
       throw new AssertionError("cancelled code accepted");
    try {svc.send(old.phone(),"127.0.0.1",null);throw new AssertionError("cooldown bypass");}
    catch (ChallengeService.RateLimited expected) { }
    var changed = svc.send("09123456788", "127.0.0.1", null);
    if (svc.verify(changed.id(), changed.phone(), sender.code) != ChallengeService.Verified.SUCCESS)
      throw new AssertionError("new phone cannot verify");
    System.out.println("AUTH_UI01_CANCEL_OK");
  }
}
'''
        with tempfile.TemporaryDirectory() as d:
            main = Path(d) / 'AuthUI01Smoke.java'
            main.write_text(source)
            build = subprocess.run(['javac', '-encoding', 'UTF-8', '-d', d]
                                   + [str(SPI / (name + '.java')) for name in names] + [str(main)],
                                   capture_output=True, text=True, timeout=25)
            self.assertEqual(build.returncode, 0, build.stderr)
            result = subprocess.run(['java', '-cp', d, 'AuthUI01Smoke'],
                                    capture_output=True, text=True, timeout=10)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertIn('AUTH_UI01_CANCEL_OK', result.stdout)


if __name__ == '__main__':
    unittest.main()
