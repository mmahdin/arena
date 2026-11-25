#!/usr/bin/env python3
"""Structural and SQL Server dialect guard; not a runtime database test."""
from pathlib import Path
import re
import sys

ROOT = Path(__file__).resolve().parent.parent
SKILLS = [
    'wellbeing-backend-module', 'wellbeing-api-contract',
    'wellbeing-database-migration', 'wellbeing-permission-review',
    'wellbeing-ai-prompt-safety',
]
NEEDED = [
    'AGENTS.md', 'README.md', 'README_FA.md', 'TECHNICAL_CONSTITUTION.md',
    'PRODUCT_CONSTITUTION.md', 'REPO_MAP.md', 'MVP_SCOPE.md',
    'contracts/openapi/openapi.yaml', 'apps/api/go.mod',
    'apps/api/cmd/api/main.go', 'apps/api/cmd/migrate/main.go',
    'apps/api/internal/app/composition/api.go',
    'apps/api/internal/platform/database/database.go',
    'apps/api/internal/platform/migrations/runner.go',
    'apps/api/db/migrations/0001_initial_schema.sql',
    'apps/api/db/migrations/0002_synthetic_dev_users.sql',
]
for module in ('user', 'conversation', 'admin'):
    for child in ('module.go', 'domain', 'application', 'adapters/http', 'adapters/mssql', 'README.md'):
        NEEDED.append(f'apps/api/internal/modules/{module}/{child}')
NEEDED.append('apps/api/internal/modules/auth/adapters/http')
for skill in SKILLS:
    NEEDED.extend((f'.agents/skills/{skill}/SKILL.md', f'.agents/skills/{skill}/agents/openai.yaml'))
missing = [path for path in NEEDED if not (ROOT / path).exists()]
if missing:
    sys.exit('Missing required paths: ' + ', '.join(missing))
for file in (ROOT / 'apps/api/internal').rglob('*.go'):
    content = file.read_text()
    if any(s in content for s in ('github.com/jackc/pgx', '/adapters/postgres', 'lib/pq')):
        sys.exit(f'PostgreSQL Go implementation leaked: {file}')
for module in ('user', 'conversation', 'admin'):
    mroot = ROOT / 'apps/api/internal/modules' / module
    for file in (mroot / 'domain').rglob('*.go'):
        if any(s in file.read_text() for s in ('github.com/gin-gonic/gin', 'database/sql', '/adapters/', '/platform/database', 'go-mssqldb')):
            sys.exit(f'Domain layer violation: {file}')
    for file in (mroot / 'application').rglob('*.go'):
        if any(s in file.read_text() for s in ('github.com/gin-gonic/gin', 'database/sql', '/adapters/', 'go-mssqldb')):
            sys.exit(f'Application layer violation: {file}')
for file in (ROOT / 'apps/api/db/migrations').glob('*.sql'):
    sql = file.read_text()
    if re.search(r'(?im)^\s*GO\s*$', sql):
        sys.exit(f'GO separator in migration {file}')
    if re.search(r'(?i)\b(timestamptz|on\s+conflict|pg_advisory_lock|returning)\b', sql):
        sys.exit(f'Non-MSSQL migration dialect: {file}')
    if not re.match(r'^\d{4}_[a-z][a-z0-9_]*\.sql$', file.name):
        sys.exit(f'Non-sequential migration name: {file}')
mod = (ROOT / 'apps/api/go.mod').read_text()
if 'github.com/microsoft/go-mssqldb' not in mod or 'jackc/pgx' in mod:
    sys.exit('go.mod must use SQL Server driver, not pgx')
for path in (ROOT / '.agents/skills').iterdir():
    if path.is_dir() and path.name not in SKILLS:
        sys.exit(f'Unknown/stale skill folder: {path.name}')
print(f'Structure and SQL Server dialect OK: {len(NEEDED)} required paths; {len(SKILLS)} Wellbeing skills')
