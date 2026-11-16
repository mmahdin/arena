#!/usr/bin/env python3
from pathlib import Path
import re,sys
if len(sys.argv)!=2 or not re.fullmatch(r'[a-z][a-z0-9_]*',sys.argv[1]):
 raise SystemExit('usage: python scripts/new_migration.py short_snake_case_description')
root=Path(__file__).resolve().parent.parent/'apps/api/db/migrations'
nums=[int(p.name[:4]) for p in root.glob('[0-9][0-9][0-9][0-9]_*.sql')]
name=f'{max(nums,default=0)+1:04d}_{sys.argv[1]}.sql'
p=root/name
p.write_text('-- SQL Server 2022 forward-only migration. No GO or BEGIN/COMMIT; runner owns transaction.\n')
print(p)
