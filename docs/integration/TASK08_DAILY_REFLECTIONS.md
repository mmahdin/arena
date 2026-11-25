# Daily Reflection retirement

WP-16 removes the unrelated calendar CRUD feature, its routes, OpenAPI operations,
source and obsolete feature tests. The legacy `dbo.daily_reflections` table and
all rows remain retained and protected against physical deletion. AI psychological
Reflection is independently implemented under `modules/ai/application/reflection`.
See ADR 0011 and `docs/quality/ai-core-wp16.md` for the active product boundary.
