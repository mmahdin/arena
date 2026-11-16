#!/usr/bin/env python3
"""Run local Go commands with literal .env values (never evaluate shell code)."""
import os
from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[1]
env = os.environ.copy()
for line in (ROOT / '.env').read_text().splitlines():
    line = line.strip()
    if not line or line.startswith('#'):
        continue
    key, value = line.split('=', 1)
    env.setdefault(key.strip(), value.strip())
if len(sys.argv) < 2:
    sys.exit('Usage: python3 scripts/run_local.py go run ./cmd/api')
os.chdir(ROOT / 'apps/api')
os.execvpe(sys.argv[1], sys.argv[1:], env)
