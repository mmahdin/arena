#!/usr/bin/env python3
"""Safely run local Go SQL login provisioning without secrets on argv."""
import os
from pathlib import Path
import subprocess
import sys
from urllib.parse import quote

from init_local_analytics_env import ROOT, read_local_env


def build_database_url(sa_password):
    if not sa_password:
        raise ValueError('Missing local MSSQL SA password')
    return 'sqlserver://sa:' + quote(sa_password, safe='') + '@127.0.0.1:14333?database=wellbeing&encrypt=disable'


def local_sa_password(path):
    if path.is_symlink() or not path.is_file():
        raise ValueError('Local .env missing or symlinked')
    values = {}
    for raw in path.read_text(encoding='utf-8').splitlines():
        if not raw or raw.lstrip().startswith('#') or '=' not in raw:
            continue
        key, value = raw.split('=', 1)
        values[key.strip()] = value.strip().strip('"').strip("'")
    return values.get('MSSQL_SA_PASSWORD', '')


def main():
    credentials = read_local_env()
    db_url = build_database_url(local_sa_password(ROOT / '.env'))
    env = dict(os.environ, APP_ENV='local', DATABASE_URL=db_url,
               THEIA_METABASE_READER_PASSWORD=credentials['THEIA_METABASE_READER_PASSWORD'])
    # Explicit split DATABASE_* environment values take precedence in config:
    # remove them to guarantee this script can only target the local container.
    for key in tuple(env):
        if key.startswith('DATABASE_') and key != 'DATABASE_URL':
            env.pop(key)
    subprocess.run(['go', 'run', './cmd/analytics-reader'], cwd=ROOT / 'apps/api', env=env, check=True)


if __name__ == '__main__':
    try:
        main()
    except (ValueError, OSError, subprocess.CalledProcessError):
        sys.exit('Local SQL reader bootstrap failed (secret details suppressed)')
