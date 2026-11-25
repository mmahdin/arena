#!/usr/bin/env python3
"""Append only a new local OTP HMAC key to an existing owner-only .env.auth.

Does not rotate/replace prior Keycloak or SQL session credentials. Never prints keys.
"""
import base64
import os
from pathlib import Path
import secrets
import stat

ROOT = Path(__file__).resolve().parents[2]


def ensure_key(path: Path) -> bool:
    import fcntl
    if path.is_symlink():
        raise ValueError("Refusing symlinked auth environment")
    flags = os.O_RDWR | os.O_APPEND | getattr(os, "O_NOFOLLOW", 0)
    fd = os.open(path, flags)
    with os.fdopen(fd, "r+", encoding="ascii") as output:
        if not stat.S_ISREG(os.fstat(fd).st_mode):
            raise ValueError("Auth environment must be a regular file")
        if stat.S_IMODE(os.fstat(fd).st_mode) & 0o077:
            raise ValueError(".env.auth must be owner-only (chmod 600)")
        fcntl.flock(fd, fcntl.LOCK_EX)
        output.seek(0)
        content = output.read()
        lines = [line for line in content.splitlines() if line.startswith("THEIA_SMS_HMAC_KEY_BASE64=")]
        if len(lines) > 1:
            raise ValueError("Duplicate SMS HMAC configuration")
        if lines:
            if not lines[0].split("=", 1)[1]:
                raise ValueError("Existing empty SMS HMAC key; fix manually")
            return False
        secret = base64.b64encode(secrets.token_bytes(32)).decode("ascii")
        output.write(("\n" if not content.endswith("\n") else "") + f"THEIA_SMS_HMAC_KEY_BASE64={secret}\n")
        output.flush()
        os.fsync(fd)
        return True


if __name__ == "__main__":
    destination = ROOT / ".env.auth"
    try:
        changed = ensure_key(destination)
    except (OSError, ValueError) as exc:
        raise SystemExit(f"Cannot update .env.auth: {exc}")
    print("Added a private local SMS HMAC key." if changed else "Existing local SMS HMAC key left unchanged.")
