#!/usr/bin/env python3
"""Create owner-only local Grafana credentials; never print credentials."""
import os
from pathlib import Path
import secrets
import sys

ROOT = Path(__file__).resolve().parents[2]
DEST = ROOT / '.env.monitoring'


def initialize(path=DEST):
    if path.exists():
        if path.is_symlink() or not path.is_file() or path.stat().st_mode & 0o077:
            raise ValueError('Existing monitoring credentials are not a regular owner-only file')
        return False
    old_umask = os.umask(0o077)
    try:
        with path.open('x', encoding='ascii') as stream:
            os.fchmod(stream.fileno(), 0o600)
            stream.write('THEIA_GRAFANA_ADMIN_PASSWORD=' + secrets.token_urlsafe(36) + '\n')
    finally:
        os.umask(old_umask)
    return True


if __name__ == '__main__':
    try:
        created = initialize()
    except (OSError, ValueError):
        sys.exit('Could not safely initialize monitoring credentials')
    print('Local monitoring credentials ' + ('created' if created else 'already exist') + ' at .env.monitoring (not displayed).')
