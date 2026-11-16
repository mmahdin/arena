# Release Gate — Never Skip for Real Users

Current version is internal-only with synthetic text. Moving to a real-user beta requires a separate decision and at least:

1. OIDC or a verified authentication strategy; disable/remove public Dev Login routes, introduce production authorized actor resolvers and tested session rotation.
2. Per-user and admin privacy threat model, encryption of sensitive data, secure data residence and approved legal basis/consent.
3. History/retention and export/erasure implementation, including backups and provider retention; independent tests.
4. Professional evaluation of mental-health safety (self-harm, delusions, medication, dependency, adverse responses), verified region-specific resources and incident plan.
5. Rate limits and model spend limits, prompt-injection/privacy evaluation, monitoring and rollback.
6. Automated integration/security/load/E2E tests; actual public deployment network, TLS and secrets controls.

The current System Prompt and keyword heuristics are neither medical advice nor adequate clinical safeguards.
