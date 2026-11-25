#!/usr/bin/env python3
"""Validate the Android Digital Asset Links file before Web/TWA handoff.

This never generates a certificate fingerprint or modifies the production
static site. The actual HTTPS response and device verification remain FE-03.
"""
import argparse
import json
import re
from pathlib import Path
from urllib.parse import urlsplit

RELATION = "delegate_permission/common.handle_all_urls"
PACKAGE = re.compile(r"^[A-Za-z_][A-Za-z_0-9]*(?:\.[A-Za-z_][A-Za-z_0-9]*)+$")
FINGERPRINT = re.compile(r"^(?:[0-9A-Fa-f]{2}:){31}[0-9A-Fa-f]{2}$")


def canonical_fingerprint(value):
    if not isinstance(value, str) or not FINGERPRINT.fullmatch(value):
        raise ValueError("fingerprint must be a 32-byte colon-separated SHA-256 hex value")
    return value.upper()


def validate(document, package, fingerprint):
    if not PACKAGE.fullmatch(package):
        raise ValueError("expected Android package name is invalid")
    expected = canonical_fingerprint(fingerprint)
    if not isinstance(document, list) or not 1 <= len(document) <= 20:
        raise ValueError("assetlinks must contain 1 to 20 statements")
    matched = False
    for item in document:
        if not isinstance(item, dict) or not isinstance(item.get("target"), dict):
            raise ValueError("assetlinks statement is malformed")
        relations = item.get("relation")
        target = item["target"]
        if not isinstance(relations, list) or any(not isinstance(r, str) for r in relations):
            raise ValueError("assetlinks relations are malformed")
        if target.get("namespace") != "android_app":
            continue
        if not PACKAGE.fullmatch(str(target.get("package_name", ""))):
            raise ValueError("assetlinks Android package name is invalid")
        fingerprints = target.get("sha256_cert_fingerprints")
        if not isinstance(fingerprints, list) or not fingerprints:
            raise ValueError("assetlinks statement missing SHA-256 fingerprints")
        fingerprints = [canonical_fingerprint(f) for f in fingerprints]
        if target["package_name"] == package and expected in fingerprints and RELATION in relations:
            matched = True
    if not matched:
        raise ValueError("expected Android app/package/relation/fingerprint statement absent")
    return True


def validate_https_origin(origin):
    parsed = urlsplit(origin)
    if (parsed.scheme != "https" or not parsed.hostname or parsed.username or parsed.password or
            parsed.path or parsed.query or parsed.fragment or parsed.hostname.endswith(".localhost") or
            parsed.hostname == "localhost"):
        raise ValueError("TWA origin must be an external HTTPS origin without path or credentials")
    return origin


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--file", required=True, type=Path)
    parser.add_argument("--package", required=True)
    parser.add_argument("--fingerprint", required=True)
    parser.add_argument("--origin", required=True, help="Intended public HTTPS User app origin")
    args = parser.parse_args()
    validate_https_origin(args.origin)
    if args.file.stat().st_size > 16 * 1024:
        parser.error("assetlinks file exceeds 16 KiB")
    document = json.loads(args.file.read_text(encoding="utf-8"))
    validate(document, args.package, args.fingerprint)
    print("Asset Links statement valid. Still verify HTTPS serving and signed Android TWA on-device.")


if __name__ == "__main__":
    main()
