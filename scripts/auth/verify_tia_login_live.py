#!/usr/bin/env python3
"""Opt-in local smoke of the REAL Keycloak SMS pages and Theia User BFF.

Requires the local browser edge, Go API, Keycloak and SQL Server to be running.
Uses only a manually entered synthetic test phone and OTP. Never prints or
saves those values. No mocks, Keycloak Admin credentials, or SMS outbox reads.
"""
from __future__ import annotations

import argparse
from pathlib import Path
import sys
from urllib.parse import urlsplit


def checked_local_origin(value: str) -> str:
    parsed = urlsplit(value)
    if parsed.scheme != "http" or parsed.hostname not in ("app.localhost", "localhost", "127.0.0.1"):
        raise ValueError("Live smoke is restricted to local HTTP browser origins")
    if parsed.username or parsed.password or parsed.fragment or parsed.path not in ("", "/") or parsed.query:
        raise ValueError("Pass only the local user-app origin")
    if not parsed.port:
        raise ValueError("Supply the explicit local browser-edge port")
    return f"{parsed.scheme}://{parsed.netloc}"


def check_form(page, selector: str) -> None:
    if not page.locator(selector).is_visible():
        raise RuntimeError(f"Expected Keycloak form not visible: {selector}")
    page.wait_for_function("""() => {
      const logo = document.querySelector('.theia-brand__mark');
      return logo?.complete && logo.naturalWidth > 0;
    }""")
    healthy = page.evaluate("""() => {
      const root = document.documentElement;
      const logo = document.querySelector('.theia-brand__mark');
      const form = document.querySelector('form.theia-form');
      const label = form?.querySelector('label');
      const input = form?.querySelector('input:not([type=hidden])');
      return {
        lang: root.lang,
        dir: root.dir,
        width: root.scrollWidth,
        viewport: innerWidth,
        logo: Boolean(logo?.complete && logo.naturalWidth > 0),
        label: Boolean(input && label?.htmlFor === input.id),
        css: getComputedStyle(document.querySelector('.theia-frame')).backgroundColor,
        action: form?.action || ''
      };
    }""")
    if healthy["lang"] != "fa" or healthy["dir"] != "rtl" or healthy["width"] > healthy["viewport"] + 1:
        raise RuntimeError(f"Theme direction or viewport check failed: {healthy['lang']}, {healthy['dir']}, {healthy['width']}")
    if not healthy["logo"] or not healthy["label"] or healthy["css"] in ("rgba(0, 0, 0, 0)", ""):
        raise RuntimeError("Theme logo, input label or CSS missing")
    action = urlsplit(healthy["action"])
    current = urlsplit(page.url)
    if action.scheme != current.scheme or action.netloc != current.netloc or not action.path:
        raise RuntimeError("Login form action must remain on the Keycloak origin")


def run(origin: str, screenshot_dir: Path | None, timeout_sec: int, chromium: str | None = None) -> None:
    try:
        from playwright.sync_api import sync_playwright
    except ImportError as exc:
        raise RuntimeError("Install Python playwright and its Chromium browser") from exc
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=False, executable_path=chromium)
        try:
            context = browser.new_context(viewport={"width": 390, "height": 844}, locale="fa-IR")
            page = context.new_page()
            page.goto(origin + "/auth/user/login", wait_until="domcontentloaded", timeout=30_000)
            page.locator("#theia-phone-form").wait_for(timeout=30_000)
            check_form(page, "#theia-phone-form")
            if screenshot_dir is not None:
                screenshot_dir.mkdir(parents=True, exist_ok=True)
                # Captured before typing anything; no personal data or OTP.
                page.screenshot(path=str(screenshot_dir / "phone-empty.png"))
            print("PASS: real Keycloak phone page has Tia theme, RTL, safe form action and no mobile overflow")
            print("Use a fictional/test-only phone and its OTP in the open browser. No secrets are recorded.")
            page.locator("#theia-code-form").wait_for(timeout=timeout_sec * 1000)
            check_form(page, "#theia-code-form")
            if not page.locator("#theia-code[autocomplete='one-time-code']").count():
                raise RuntimeError("OTP autofill field not present")
            print("PASS: real Keycloak OTP page loaded with accessible Tia theme")
            # The human completes OTP. Do not capture screenshots after input.
            print("Finish OTP verification in the browser to test the BFF callback and SQL session.")
            page.wait_for_url(lambda url: url.startswith(origin + "/") and not url.startswith(origin + "/auth/"),
                              timeout=timeout_sec * 1000, wait_until="domcontentloaded")
            response = context.request.get(origin + "/auth/user/session", headers={"Cache-Control": "no-store"})
            if response.status != 200:
                raise RuntimeError(f"User BFF session returned HTTP {response.status} instead of 200")
            data = response.json()
            if data.get("authenticated") is not True or data.get("principalType") != "user" or not data.get("csrfToken"):
                raise RuntimeError("Missing verified User BFF identity or CSRF token")
            cookies = context.cookies([origin])
            valid = [c for c in cookies if c["name"] in ("theia_user_session_local", "__Host-theia_user_session")]
            if len(valid) != 1 or not valid[0]["httpOnly"] or valid[0]["domain"] != urlsplit(origin).hostname:
                raise RuntimeError("Missing host-only HttpOnly BFF session cookie")
            print("PASS: BFF callback created a verified user session with CSRF and HttpOnly cookie")
            print("Full product/onboarding/logout matrix remains in frontend FE03 tests.")
        finally:
            browser.close()


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--origin", default="http://app.localhost:8080", help="Theia User app browser origin")
    parser.add_argument("--timeout-sec", type=int, default=180, help="Time for each manual step")
    parser.add_argument("--screenshots", type=Path, default=None, help="Save empty phone page only")
    parser.add_argument("--chromium", help="Use an installed Chromium or Chrome executable")
    args = parser.parse_args()
    if args.timeout_sec < 30 or args.timeout_sec > 900:
        parser.error("--timeout-sec must be 30..900")
    try:
        origin = checked_local_origin(args.origin)
        run(origin, args.screenshots, args.timeout_sec, args.chromium)
    except Exception as exc:
        print(f"AUTH-UI02 live smoke FAILED/UNVERIFIED: {exc}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
