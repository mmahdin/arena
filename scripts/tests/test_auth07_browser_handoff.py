"""AUTH-07 browser/TWA contract guardrails (not a live-browser test)."""
import importlib.util
import json
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[2]
SCRIPT = ROOT / "scripts/auth/validate_twa_assetlinks.py"
spec = importlib.util.spec_from_file_location("theia_assetlinks", SCRIPT)
assetlinks = importlib.util.module_from_spec(spec)
spec.loader.exec_module(assetlinks)


class Auth07BrowserContract(unittest.TestCase):
    def test_routes_are_opt_in_local_only(self):
        config = (ROOT / "apps/api/internal/platform/config/auth.go").read_text()
        composition = (ROOT / "apps/api/internal/app/composition/api.go").read_text()
        contract = (ROOT / "contracts/openapi/openapi.yaml").read_text()
        self.assertIn('AUTH_BROWSER_SMOKE_ENABLED', config)
        self.assertIn('environment != "local"', config)
        self.assertIn('config.BrowserSmokeAllowed', composition)
        for kind in ("user", "staff"):
            self.assertIn(f'/__auth-smoke/{kind}', contract)
        self.assertIn("name: return_to", contract)
        self.assertIn('AUTH_RETURN_PATH_INVALID', (ROOT / 'apps/api/internal/modules/auth/adapters/http/handler.go').read_text())

    def test_test_page_never_exposes_browser_credentials(self):
        page = (ROOT / 'apps/api/internal/modules/auth/adapters/http/smoke_page.html').read_text()
        js = (ROOT / 'apps/api/internal/modules/auth/adapters/http/smoke.js').read_text()
        self.assertNotIn('localStorage', js)
        self.assertNotIn('sessionStorage', js)
        self.assertNotIn('document.cookie', js)
        self.assertNotIn('innerHTML', js)
        self.assertIn("credentials: 'same-origin'", js)
        self.assertIn("method: 'POST'", js)
        self.assertIn('return_to=%2F__auth-smoke%2F', page)
        self.assertIn('csrfToken = session.csrfToken', js)

    def test_assetlinks_requires_package_fingerprint_and_relation(self):
        fp = ':'.join(f'{i:02X}' for i in range(32))
        manifest = [{"relation": [assetlinks.RELATION], "target": {
            "namespace": "android_app", "package_name": "ir.theia.synthetic", "sha256_cert_fingerprints": [fp]
        }}]
        self.assertTrue(assetlinks.validate(manifest, 'ir.theia.synthetic', fp))
        for package, cert in [('ir.theia.other', fp), ('ir.theia.synthetic', ':'.join(['FF']*32))]:
            with self.assertRaises(ValueError):
                assetlinks.validate(manifest, package, cert)
        manifest[0]['relation'] = ['delegate_permission/common.get_login_creds']
        with self.assertRaises(ValueError):
            assetlinks.validate(manifest, 'ir.theia.synthetic', fp)

    def test_assetlinks_rejects_placeholder_and_non_https_origins(self):
        for value in ('fake', 'AA:BB', 'not a fingerprint'):
            with self.assertRaises(ValueError):
                assetlinks.canonical_fingerprint(value)
        for origin in ('http://theia.example', 'https://app.localhost', 'https://app.example/path',
                       'https://user:pass@app.example', 'https://app.example?x=1'):
            with self.assertRaises(ValueError):
                assetlinks.validate_https_origin(origin)
        self.assertEqual(assetlinks.validate_https_origin('https://app.example'), 'https://app.example')


if __name__ == '__main__':
    unittest.main()
