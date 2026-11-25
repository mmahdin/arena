#!/usr/bin/env python3
"""Create or update owner-only OBS-02 secrets; never echo the values.

For local synthetic development only. `--set-ingest-key` uses hidden stdin
(getpass) rather than placing a key in shell history or command arguments.
"""
from __future__ import annotations

import argparse
import getpass
import os
from pathlib import Path
import re
import secrets
import stat
import tempfile

ROOT = Path(__file__).resolve().parents[2]
DEFAULT_ENV = ROOT / ".env.observability"
INGEST_PATTERN = re.compile(r"[A-Za-z0-9._~-]{32,4096}\Z")


def _safe_file(path: Path) -> None:
    if path.is_symlink() or not path.is_file():
        raise ValueError("Observability env must be a regular file, not a symlink")
    info = path.stat()
    if info.st_uid != os.getuid() or stat.S_IMODE(info.st_mode) & 0o077:
        raise ValueError("Observability env must be owned by the current user and mode 0600")


def _phoenix_secret() -> str:
    # Current Phoenix releases require a long signing secret with a digit and
    # lowercase letter; avoid even a theoretical unlucky all-letter sample.
    while True:
        candidate = secrets.token_urlsafe(48)
        if re.search(r"[a-z]", candidate) and re.search(r"[0-9]", candidate):
            return candidate


def _admin_password() -> str:
    # Strong-password bootstrap: uppercase/lowercase/digit/special guaranteed.
    return "Aa1!" + secrets.token_urlsafe(32)


def create_env(path: Path = DEFAULT_ENV) -> bool:
    if path.is_symlink():
        raise ValueError("Refusing to follow a symbolic link")
    # urlsafe prevents Compose/shell interpolation and avoids quotes/newlines.
    content = (
        "# Generated locally for synthetic OBS-02. Never commit or share.\n"
        f"PHOENIX_SECRET={_phoenix_secret()}\n"
        f"PHOENIX_DEFAULT_ADMIN_INITIAL_PASSWORD={_admin_password()}\n"
        "# Fill this with --set-ingest-key after creating a Phoenix system key.\n"
        "THEIA_OTEL_CONTENT_API_KEY=\n"
    )
    try:
        fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    except FileExistsError:
        _safe_file(path)
        return False
    with os.fdopen(fd, "w", encoding="utf-8") as output:
        output.write(content)
        output.flush()
        os.fsync(output.fileno())
    return True


def read_env(path: Path = DEFAULT_ENV) -> dict[str, str]:
    _safe_file(path)
    values: dict[str, str] = {}
    for raw in path.read_text(encoding="utf-8").splitlines():
        line = raw.strip()
        if not line or line.startswith("#"):
            continue
        if "=" not in line:
            raise ValueError("Malformed observability env file")
        key, value = line.split("=", 1)
        if key not in {"PHOENIX_SECRET", "PHOENIX_DEFAULT_ADMIN_INITIAL_PASSWORD", "THEIA_OTEL_CONTENT_API_KEY"}:
            raise ValueError("Unexpected key in observability env file")
        if key in values or not value.isascii() or "\r" in value or "\n" in value:
            raise ValueError("Malformed or duplicate key in observability env file")
        values[key] = value
    if (len(values.get("PHOENIX_SECRET", "")) < 48 or
            len(values.get("PHOENIX_DEFAULT_ADMIN_INITIAL_PASSWORD", "")) < 32):
        raise ValueError("Observability credentials are missing or too short")
    key = values.get("THEIA_OTEL_CONTENT_API_KEY", "")
    if key and not INGEST_PATTERN.fullmatch(key):
        raise ValueError("Ingestion key does not match OBS-01B's accepted format")
    return values


def set_ingest_key(path: Path = DEFAULT_ENV, key: str | None = None) -> None:
    values = read_env(path)
    if key is None:
        key = getpass.getpass("Paste the Phoenix system ingestion API key (hidden): ")
    if not INGEST_PATTERN.fullmatch(key):
        raise ValueError("Invalid Phoenix key format; expected 32-4096 URL-safe characters")
    if values.get("THEIA_OTEL_CONTENT_API_KEY"):
        raise ValueError("An ingestion key is already set; rotate intentionally, do not overwrite")
    lines = path.read_text(encoding="utf-8").splitlines(keepends=True)
    if sum(line.startswith("THEIA_OTEL_CONTENT_API_KEY=") for line in lines) != 1:
        raise ValueError("Missing or duplicate key field")
    content = "".join((f"THEIA_OTEL_CONTENT_API_KEY={key}\n" if line.startswith("THEIA_OTEL_CONTENT_API_KEY=")
                       else line) for line in lines)
    # Atomic owner-only replacement; never leave a plaintext temp file.
    fd, name = tempfile.mkstemp(prefix=".env.observability.", dir=path.parent)
    try:
        os.fchmod(fd, 0o600)
        with os.fdopen(fd, "w", encoding="utf-8") as out:
            out.write(content)
            out.flush()
            os.fsync(out.fileno())
        _safe_file(path)
        os.replace(name, path)
    finally:
        if os.path.exists(name):
            os.unlink(name)


def main() -> None:
    parser = argparse.ArgumentParser(description="Initialize private local Phoenix credentials")
    parser.add_argument("--set-ingest-key", action="store_true", help="Store a Phoenix API key entered invisibly")
    args = parser.parse_args()
    if args.set_ingest_key:
        set_ingest_key()
        print("Stored Phoenix ingestion key in owner-only env file; value not printed.")
    elif create_env():
        print("Created .env.observability (mode 0600), with random credentials; values not printed.")
    else:
        print("Existing .env.observability retained without overwriting credentials.")


if __name__ == "__main__":
    try:
        main()
    except (ValueError, OSError, UnicodeError) as exc:
        raise SystemExit("Phoenix local setup refused: " + str(exc)) from None
