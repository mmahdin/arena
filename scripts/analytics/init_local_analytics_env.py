#!/usr/bin/env python3
"""Create owner-only local Metabase/Postgres/SQL reader credentials.

Never print credentials, overwrite an existing file, or follow a symlink.
"""
import base64
import os
from pathlib import Path
import secrets
import stat
import sys

ROOT = Path(__file__).resolve().parents[2]
ENV_FILE = ROOT / '.env.analytics'
REQUIRED = (
    'THEIA_METABASE_POSTGRES_PASSWORD',
    'THEIA_METABASE_ENCRYPTION_SECRET_KEY',
    'THEIA_METABASE_READER_PASSWORD',
    'THEIA_METABASE_ADMIN_EMAIL',
    'THEIA_METABASE_ADMIN_PASSWORD',
)


def read_local_env(path=ENV_FILE):
    if path.is_symlink() or not path.is_file():
        raise ValueError('Local analytics credentials must be a regular file')
    if stat.S_IMODE(path.stat().st_mode) & 0o077:
        raise ValueError('Local analytics credentials must be owner-only (0600)')
    values = {}
    for raw in path.read_text(encoding='ascii').splitlines():
        if not raw or raw.startswith('#'):
            continue
        key, separator, value = raw.partition('=')
        if not separator or key not in REQUIRED or key in values:
            raise ValueError('Invalid or duplicate analytics environment entry')
        values[key] = value
    if set(values) != set(REQUIRED) or any(not value for value in values.values()):
        raise ValueError('Incomplete local analytics credentials')
    if not values['THEIA_METABASE_READER_PASSWORD'].startswith('Aa1!') or (
        len(values['THEIA_METABASE_READER_PASSWORD']) != 64 or any(
            c not in '0123456789abcdef' for c in values['THEIA_METABASE_READER_PASSWORD'][4:]
        )
    ):
        raise ValueError('Invalid local SQL reader secret')
    return values


def initialize(path=ENV_FILE):
    if path.exists() or path.is_symlink():
        read_local_env(path)
        return False
    values = {
        'THEIA_METABASE_POSTGRES_PASSWORD': secrets.token_hex(32),
        'THEIA_METABASE_ENCRYPTION_SECRET_KEY': base64.b64encode(secrets.token_bytes(48)).decode('ascii'),
        'THEIA_METABASE_READER_PASSWORD': 'Aa1!' + secrets.token_hex(30),
        'THEIA_METABASE_ADMIN_EMAIL': 'admin@theia.local',
        'THEIA_METABASE_ADMIN_PASSWORD': secrets.token_urlsafe(40),
    }
    previous_umask = os.umask(0o077)
    try:
        with path.open('x', encoding='ascii') as stream:
            os.fchmod(stream.fileno(), 0o600)
            for key, value in values.items():
                stream.write(f'{key}={value}\n')
    finally:
        os.umask(previous_umask)
    return True


if __name__ == '__main__':
    try:
        created = initialize()
    except (OSError, ValueError):
        sys.exit('Cannot safely initialize local analytics secrets (details suppressed)')
    print('Local analytics credentials ' + ('created' if created else 'verified') + ' at .env.analytics (not displayed).')
