#!/usr/bin/env python3
"""Owner-only OBS-05A local collector/Langfuse bootstrap; never echo credentials.

The collector receiver uses an independent random Bearer key. The existing
Phoenix ingestion key stays in .env.observability. Langfuse credentials are
only sent from the Collector to the operator-approved OTLP destination.
"""
from __future__ import annotations

import argparse
import base64
import getpass
import os
from pathlib import Path
import re
import secrets
import stat
import tempfile
from urllib.parse import urlsplit

ROOT = Path(__file__).resolve().parents[2]
DEFAULT_PATH = ROOT / ".env.dual-otlp"
KEYS = {"THEIA_COLLECTOR_INGEST_KEY", "THEIA_LANGFUSE_AUTH_BASE64", "THEIA_LANGFUSE_OTLP_ENDPOINT"}
TOKEN = re.compile(r"[A-Za-z0-9._~-]{32,4096}\Z")
LANGFUSE_KEY = re.compile(r"(?:pk-lf|sk-lf)-[A-Za-z0-9_-]{8,256}\Z")


def validate_endpoint(raw: str) -> str:
    """No query, secrets, plaintext remote endpoint or ambiguous hostname."""
    if not raw or raw != raw.strip() or not raw.isascii() or any(c.isspace() for c in raw) or "\\" in raw:
        raise ValueError("Langfuse endpoint must be a plain URL with no whitespace or escapes")
    try:
        u = urlsplit(raw)
        port = u.port  # Reject malformed or out-of-range port numbers.
        if (u.username is not None or u.password is not None or u.query or u.fragment or
            u.path != "/api/public/otel" or not u.hostname or (port is not None and port < 1)):
            raise ValueError("Invalid OTLP endpoint")
        # Permit only HTTPS to an operator-approved host, or the explicit
        # local Docker-network service name used by a separately deployed Langfuse.
        if u.scheme != "https" and not (u.scheme == "http" and u.netloc == "langfuse-web:3000"):
            raise ValueError("Langfuse endpoint requires HTTPS except private Docker langfuse-web:3000")
        if u.scheme == "https" and u.hostname.lower() in ("localhost", "127.0.0.1", "0.0.0.0"):
            raise ValueError("Container localhost is not the host Langfuse service")
        if "@" in u.netloc or "%" in u.netloc or u.hostname.endswith("."):
            raise ValueError("Invalid Langfuse host")
        return raw
    except (ValueError, TypeError):
        raise ValueError("Invalid Langfuse OTLP origin; use https://HOST/api/public/otel") from None


def secure_file(path: Path) -> None:
    if path.is_symlink() or not path.is_file():
        raise ValueError("Dual OTel secrets must be a regular file")
    info = path.stat()
    if info.st_uid != os.getuid() or stat.S_IMODE(info.st_mode) != 0o600:
        raise ValueError("Dual OTel secrets must be owned by the user and mode 0600")


def create_env(path: Path = DEFAULT_PATH) -> bool:
    if path.is_symlink():
        raise ValueError("Refusing a symlink for dual OTel secrets")
    content = ("# Local synthetic OBS-05A only. Never commit or share.\n"
               + "THEIA_COLLECTOR_INGEST_KEY=" + secrets.token_urlsafe(48) + "\n"
               + "THEIA_LANGFUSE_AUTH_BASE64=\n"
               + "THEIA_LANGFUSE_OTLP_ENDPOINT=\n")
    try:
        fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    except FileExistsError:
        secure_file(path)
        return False
    with os.fdopen(fd, "w", encoding="utf-8") as output:
        output.write(content)
        output.flush()
        os.fsync(output.fileno())
    return True


def read_env(path: Path = DEFAULT_PATH, *, require_complete: bool = True) -> dict[str, str]:
    secure_file(path)
    result: dict[str, str] = {}
    for line in path.read_text(encoding="utf-8").splitlines():
        if not line.strip() or line.startswith("#"):
            continue
        if "=" not in line:
            raise ValueError("Malformed dual OTel secret entry")
        key, value = line.split("=", 1)
        if key not in KEYS or key in result or not value.isascii() or any(c.isspace() for c in value):
            raise ValueError("Unknown, duplicated or unsafe dual OTel secret")
        result[key] = value
    if set(result) != KEYS or not TOKEN.fullmatch(result["THEIA_COLLECTOR_INGEST_KEY"]):
        raise ValueError("Missing or unsafe dual Collector ingestion key")
    if require_complete:
        validate_endpoint(result["THEIA_LANGFUSE_OTLP_ENDPOINT"])
        encoded = result["THEIA_LANGFUSE_AUTH_BASE64"]
        try:
            raw = base64.b64decode(encoded, validate=True).decode("ascii")
            public, secret = raw.split(":", 1)
            if not LANGFUSE_KEY.fullmatch(public) or not LANGFUSE_KEY.fullmatch(secret) or not encoded:
                raise ValueError("Unconfigured Langfuse keys")
        except (ValueError, UnicodeError):
            raise ValueError("Missing or invalid Langfuse project credentials") from None
    return result


def configure(path: Path, endpoint: str, public: str, secret: str) -> None:
    current = read_env(path, require_complete=False)
    if current["THEIA_LANGFUSE_AUTH_BASE64"] or current["THEIA_LANGFUSE_OTLP_ENDPOINT"]:
        raise ValueError("Already configured; rotate credentials intentionally, do not overwrite")
    url = validate_endpoint(endpoint)
    if not LANGFUSE_KEY.fullmatch(public) or not LANGFUSE_KEY.fullmatch(secret):
        raise ValueError("Invalid Langfuse public or secret key format")
    encoded = base64.b64encode(f"{public}:{secret}".encode("ascii")).decode("ascii")
    text = path.read_text(encoding="utf-8")
    text = text.replace("THEIA_LANGFUSE_AUTH_BASE64=\n", f"THEIA_LANGFUSE_AUTH_BASE64={encoded}\n")
    text = text.replace("THEIA_LANGFUSE_OTLP_ENDPOINT=\n", f"THEIA_LANGFUSE_OTLP_ENDPOINT={url}\n")
    fd, name = tempfile.mkstemp(prefix=".env.dual-otlp.", dir=path.parent)
    try:
        os.fchmod(fd, 0o600)
        with os.fdopen(fd, "w", encoding="utf-8") as f:
            f.write(text)
            f.flush()
            os.fsync(f.fileno())
        secure_file(path)
        os.replace(name, path)
    finally:
        if os.path.exists(name):
            os.unlink(name)
    read_env(path)


def main() -> None:
    parser = argparse.ArgumentParser(description="Initialize local OBS-05A dual OTLP credentials")
    parser.add_argument("--configure", action="store_true", help="Enter Langfuse OTLP endpoint and API keys privately")
    args = parser.parse_args()
    if not args.configure:
        if create_env():
            print("Created .env.dual-otlp (0600); Collector key generated. Run --configure next.")
        else:
            print("Existing .env.dual-otlp retained; no credentials overwritten.")
        return
    endpoint = input("Langfuse OTLP base URL (no /v1/traces): ").strip()
    public = getpass.getpass("Langfuse project public key (hidden): ")
    secret = getpass.getpass("Langfuse project secret key (hidden): ")
    configure(DEFAULT_PATH, endpoint, public, secret)
    print("Stored Langfuse destination and credentials in owner-only .env.dual-otlp.")


if __name__ == "__main__":
    try:
        main()
    except (ValueError, OSError) as exc:
        # Do not interpolate malformed input/endpoint or user-supplied secrets.
        raise SystemExit(f"OBS-05A bootstrap failed: {type(exc).__name__}. Check endpoint/key format and 0600 file mode.") from None
