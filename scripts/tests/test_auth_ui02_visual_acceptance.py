"""Offline theme fixture and browser-acceptance contract checks."""
import importlib.util
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
SOURCE = ROOT / "scripts/auth/preview_tia_login_ui.py"
spec = importlib.util.spec_from_file_location("preview_tia", SOURCE)
preview = importlib.util.module_from_spec(spec)
spec.loader.exec_module(preview)
LIVE = ROOT / "scripts/auth/verify_tia_login_live.py"
spec_live = importlib.util.spec_from_file_location("verify_tia", LIVE)
verify = importlib.util.module_from_spec(spec_live)
spec_live.loader.exec_module(verify)
CSS = ROOT / "infra/keycloak/themes/theia-sms/login/resources/css/theia-login.css"
JS = ROOT / "infra/keycloak/themes/theia-sms/login/resources/js/theia-login.js"


class AuthUI02VisualAcceptance(unittest.TestCase):
    def test_preview_fixtures_are_real_template_source_and_offline(self):
        for step in ("phone", "code"):
            plain = preview.render_fixture(step)
            bad = preview.render_fixture(step, '<img src=x onerror=alert(1)>')
            self.assertIn('lang="fa" dir="rtl"', plain)
            self.assertIn('data:image/jpeg;base64,', plain)
            self.assertIn('<style>', plain)
            self.assertIn('<script>', plain)
            self.assertIn('action="#preview-only"', plain)
            self.assertNotIn('http://', plain)
            self.assertIn('&lt;img src=x onerror=alert(1)&gt;', bad)
            self.assertNotIn('<img src=x onerror=alert(1)>', bad)
            self.assertIn('id="theia-auth-error"', bad)
        self.assertEqual(preview.render_fixture("code").count('class="theia-otp__cell"'), 6)

    def test_accessible_countdown_and_readability(self):
        css = CSS.read_text()
        js = JS.read_text()
        code = (preview.THEME / 'theia-sms-code.ftl').read_text()
        self.assertIn('theia-resend-ready', code)
        self.assertIn('role="status"', code)
        self.assertIn('Date.now()', js)
        self.assertIn("document.addEventListener('visibilitychange', render)", js)
        self.assertIn("window.addEventListener('pageshow', render)", js)
        self.assertIn('color: var(--theia-muted)', css)
        self.assertIn('input::placeholder { color: var(--theia-muted)', css)
        self.assertIn('.theia-visually-hidden', css)
        self.assertNotIn('localStorage', js)
        self.assertNotIn('document.cookie', js)

    def test_warm_background_and_muted_copy_have_readable_contrast(self):
        import re
        css = CSS.read_text()
        tokens = {name: value for name, value in re.findall(r'--(theia-[\w-]+):\s*(#[0-9a-fA-F]{6});', css)}

        def luminance(value):
            components = [int(value[i:i + 2], 16) / 255 for i in (1, 3, 5)]
            linear = [c / 12.92 if c <= 0.04045 else ((c + 0.055) / 1.055) ** 2.4 for c in components]
            return sum(a * b for a, b in zip(linear, (0.2126, 0.7152, 0.0722)))

        bg = luminance(tokens['theia-bg'])
        for role in ('theia-ink', 'theia-muted'):
            fg = luminance(tokens[role])
            self.assertGreaterEqual((max(bg, fg) + 0.05) / (min(bg, fg) + 0.05), 4.5, role)
        self.assertIn('.theia-footer { margin-top: 32px; padding-top: 22px; text-align: center; color: var(--theia-muted)', css)
        self.assertIn('.theia-resend button:disabled { color: var(--theia-muted)', css)

    def test_live_verifier_restricts_origin_and_does_not_read_sms(self):
        self.assertEqual(verify.checked_local_origin('http://app.localhost:8080'), 'http://app.localhost:8080')
        for invalid in ('https://example.com', 'http://example.com',
                        'http://auth.localhost:8180', 'http://app.localhost:8080/#/chat',
                        'http://app.localhost:8080/?token=abc'):
            with self.assertRaises(ValueError):
                verify.checked_local_origin(invalid)
        code = LIVE.read_text()
        self.assertNotIn('theia-sms-outbox', code)
        self.assertNotIn('smsCode', code)

    @unittest.skipUnless(shutil.which('node'), 'Node unavailable')
    def test_node_countdown_background_tab_regression(self):
        proc = subprocess.run(['node', str(ROOT / 'scripts/tests/auth_ui01_theme_browser.mjs')],
                              capture_output=True, text=True, timeout=12)
        self.assertEqual(proc.returncode, 0, proc.stderr)
        self.assertIn('AUTH_UI01_DOM_INTERACTIONS_OK', proc.stdout)

    def test_preview_cli_produces_four_fixture_pages(self):
        with tempfile.TemporaryDirectory() as d:
            proc = subprocess.run(['python3', str(SOURCE), '--out', d],
                                  capture_output=True, text=True, timeout=10)
            self.assertEqual(proc.returncode, 0, proc.stderr)
            self.assertEqual(sorted(p.name for p in Path(d).glob('*.html')),
                             ['code-error.html', 'code.html', 'phone-error.html', 'phone.html'])


if __name__ == '__main__':
    unittest.main()
