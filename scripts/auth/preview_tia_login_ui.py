#!/usr/bin/env python3
"""Generate offline, synthetic previews of the Tia Keycloak SMS theme.

This is a deliberately narrow FreeMarker fixture renderer, NOT a Keycloak or
FreeMarker implementation. It never contacts Keycloak, reads SMS, or performs
an OAuth login. The optional Chromium smoke checks the rendered layout only.
"""
from __future__ import annotations

import argparse
import base64
from html import escape
import json
from pathlib import Path
import re
import sys

ROOT = Path(__file__).resolve().parents[2]
THEME = ROOT / "infra/keycloak/themes/theia-sms/login"
VIEWPORTS = ((320, 640), (360, 740), (390, 844), (430, 932), (1280, 800))


def condition(source: str, name: str, enabled: bool) -> str:
    pattern = re.compile(r"<#if " + re.escape(name) + r"\?\? && " + re.escape(name) + r"\?has_content>(.*?)</#if>", re.S)
    return pattern.sub(lambda match: match.group(1) if enabled else "", source)


def render_fixture(step: str, error: str = "") -> str:
    if step not in ("phone", "code"):
        raise ValueError("Unknown synthetic page")
    raw = (THEME / f"theia-sms-{step}.ftl").read_text(encoding="utf-8")
    content = re.search(r'<@layout\.page\b[^>]*>(.*?)</@layout\.page>', raw, re.S)
    if not content:
        raise ValueError("Missing themed page macro invocation")
    content = content.group(1)
    content = condition(content, "smsError", bool(error))
    content = condition(content, "smsEnteredPhone", step == "phone" and bool(error))
    content = content.replace("${smsError}", escape(error, quote=True))
    content = content.replace("${smsEnteredPhone}", "09120000000")
    content = content.replace("${(smsMaskedPhone!'')}", "\u2022\u2022\u2022\u20220000")
    content = content.replace("${(smsResendRemaining!0)?c}", "41")
    content = content.replace("${url.loginAction}", "#preview-only")
    content = content.replace('<#list 1..6 as i><span class="theia-otp__cell"></span></#list>',
                              '<span class="theia-otp__cell"></span>' * 6)

    layout = (THEME / "theia-layout.ftl").read_text(encoding="utf-8")
    layout_match = re.search(r'<#macro page title step>(.*?)</#macro>', layout, re.S)
    if not layout_match:
        raise ValueError("Missing themed layout macro")
    html = layout_match.group(1)
    html = html.replace("${title}", escape("\u0648\u0631\u0648\u062f \u0628\u0647 \u062d\u0633\u0627\u0628" if step == "phone" else "\u062a\u0623\u06cc\u06cc\u062f \u0634\u0645\u0627\u0631\u0647 \u0645\u0648\u0628\u0627\u06cc\u0644"))
    html = html.replace("${step}", step)
    html = html.replace("<#nested>", content)
    if re.search(r'<#|\$\{(?!url\.resourcesPath\})', html):
        raise ValueError("Fixture renderer missed FreeMarker markup")
    css = (THEME / "resources/css/theia-login.css").read_text(encoding="utf-8")
    js = (THEME / "resources/js/theia-login.js").read_text(encoding="utf-8")
    logo = (THEME / "resources/img/theia-mark.jpg").read_bytes()
    html = html.replace('<link rel="stylesheet" href="${url.resourcesPath}/css/theia-login.css" />',
                        '<style>\n' + css + '\n</style>')
    html = html.replace('<script defer src="${url.resourcesPath}/js/theia-login.js"></script>', '')
    html = html.replace("</body>", "<script>\n" + js + "\n</script>\n</body>")
    html = html.replace('${url.resourcesPath}/img/theia-mark.jpg',
                        'data:image/jpeg;base64,' + base64.b64encode(logo).decode('ascii'))
    return html


def screenshot_report(cases: dict[str, str], destination: Path, chromium: str | None) -> list[dict]:
    try:
        from playwright.sync_api import sync_playwright
    except ImportError as exc:
        raise RuntimeError("Install Python playwright for --screenshots") from exc
    results = []
    with sync_playwright() as p:
        browser = p.chromium.launch(executable_path=chromium, headless=True,
                                    args=["--no-sandbox", "--disable-dev-shm-usage"])
        try:
            for case, source in cases.items():
                for width, height in VIEWPORTS:
                    page = browser.new_page(viewport={"width": width, "height": height}, device_scale_factor=1)
                    errors = []
                    page.on("pageerror", lambda error: errors.append(str(error)))
                    try:
                        # In-memory HTML avoids file:// and localhost restrictions.
                        page.set_content(source, wait_until="load")
                        page.locator(".theia-primary").wait_for()
                        state = page.evaluate("""() => {
                          const doc = document.documentElement;
                          const cta = document.querySelector('.theia-primary');
                          const phone = document.querySelector('#theia-phone');
                          const code = document.querySelector('#theia-code');
                          const node = phone || code;
                          const box = cta.getBoundingClientRect();
                          const inputBox = node.getBoundingClientRect();
                          const image = document.querySelector('.theia-brand__mark');
                          return {
                            documentWidth: doc.scrollWidth,
                            viewportWidth: window.innerWidth,
                            buttonWidth: Math.round(box.width),
                            inputWidth: Math.round(inputBox.width),
                            inputHasName: Boolean(node.getAttribute('aria-labelledby') || document.querySelector('label[for="' + node.id + '"]')),
                            logoLoaded: image.complete && image.naturalWidth > 0,
                            otpEnhanced: Boolean(document.querySelector('#theia-otp.is-enhanced'))
                          };
                        }""")
                        page.screenshot(path=str(destination / f"{case}-{width}.png"), full_page=True)
                        state.update({"case": case, "width": width, "height": height, "jsErrors": errors})
                        if state["documentWidth"] > width + 1 or not state["inputHasName"] or not state["logoLoaded"] or errors:
                            raise RuntimeError(f"Visual fixture failed: {state}")
                        if "code" in case and not state["otpEnhanced"]:
                            raise RuntimeError(f"OTP enhancement failed: {state}")
                        results.append(state)
                    finally:
                        page.close()
        finally:
            browser.close()
    return results


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--out", type=Path, default=Path("/tmp/tia-login-theme-preview"))
    parser.add_argument("--screenshots", action="store_true", help="Requires Playwright Chromium")
    parser.add_argument("--chromium", default=None, help="Chromium executable path (optional)")
    args = parser.parse_args()
    cases = {
        "phone": render_fixture("phone"),
        "phone-error": render_fixture("phone", "\u0634\u0645\u0627\u0631\u0647 \u0645\u0648\u0628\u0627\u06cc\u0644 \u0645\u0639\u062a\u0628\u0631 \u0646\u06cc\u0633\u062a."),
        "code": render_fixture("code"),
        "code-error": render_fixture("code", "\u06a9\u062f \u0645\u0639\u062a\u0628\u0631 \u0646\u06cc\u0633\u062a."),
    }
    args.out.mkdir(parents=True, exist_ok=True)
    for name, markup in cases.items():
        (args.out / f"{name}.html").write_text(markup, encoding="utf-8")
    if args.screenshots:
        results = screenshot_report(cases, args.out, args.chromium)
        (args.out / "layout-report.json").write_text(json.dumps(results, ensure_ascii=False, indent=2), encoding="utf-8")
        print(f"AUTH-UI02: {len(results)} viewport screenshots and layout checks passed: {args.out}")
    else:
        print(f"AUTH-UI02: {len(cases)} offline HTML fixtures generated: {args.out}")
    print("Offline fixtures only; not Keycloak login or a FreeMarker integration test.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
