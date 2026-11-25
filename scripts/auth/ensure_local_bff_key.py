#!/usr/bin/env python3
"""Add AUTH-03 key to an existing AUTH-01 .env.auth, without replacing secrets."""
import base64
import os
from pathlib import Path
import secrets

path = Path(__file__).resolve().parents[2] / '.env.auth'
if not path.is_file() or path.is_symlink():
    raise SystemExit('.env.auth must already exist as an ordinary file; run init_local_keycloak_env.py')
if path.stat().st_mode & 0o077:
    raise SystemExit('chmod 600 .env.auth before continuing')
with path.open('r+', encoding='utf-8') as f:
    existing = f.read()
    if any(line.startswith('AUTH_BFF_KEY_BASE64=') for line in existing.splitlines()):
        print('AUTH_BFF_KEY_BASE64 already exists; left unchanged.')
    else:
        f.seek(0, os.SEEK_END)
        if existing and not existing.endswith('\n'):
            f.write('\n')
        f.write('AUTH_BFF_KEY_BASE64=' + base64.b64encode(secrets.token_bytes(32)).decode() + '\n')
        print('Added random AUTH_BFF_KEY_BASE64 to .env.auth; value not displayed.')
