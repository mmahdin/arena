#!/usr/bin/env python3
"""Create local Keycloak secrets once; never print or overwrite credentials.

This is only for the opt-in Docker development overlay (compose.auth.yaml).
Production secrets must be managed separately.
"""

import os
from pathlib import Path
import secrets
import base64

ROOT = Path(__file__).resolve().parents[2]


def create_env(path: Path) -> bool:
    """Write a new owner-only .env.auth, or leave an existing one unchanged."""
    lines = [
        "# Generated locally. Never commit, share, or use in production.\n",
        "KEYCLOAK_DB_NAME=theia_keycloak\n",
        "KEYCLOAK_DB_USER=theia_keycloak\n",
        f"KEYCLOAK_DB_PASSWORD={secrets.token_urlsafe(32)}\n",
        "KEYCLOAK_BOOTSTRAP_ADMIN_USERNAME=theia-bootstrap\n",
        f"KEYCLOAK_BOOTSTRAP_ADMIN_PASSWORD={secrets.token_urlsafe(32)}\n",
        f"THEIA_USERS_BFF_CLIENT_SECRET={secrets.token_urlsafe(32)}\n",
        f"THEIA_STAFF_BFF_CLIENT_SECRET={secrets.token_urlsafe(32)}\n",
        f"AUTH_BFF_KEY_BASE64={base64.b64encode(secrets.token_bytes(32)).decode()}\n",
    ]
    try:
        fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    except FileExistsError:
        return False
    with os.fdopen(fd, "w", encoding="utf-8") as output:
        output.writelines(lines)
    return True


if __name__ == "__main__":
    destination = ROOT / ".env.auth"
    if create_env(destination):
        print("Created .env.auth with random credentials (mode 0600); values not displayed.")
    else:
        print(".env.auth already exists; left unchanged. Check required fields manually.")
