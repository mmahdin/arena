# ADR 0011 — AI Core HTTP product cutover

Status: implemented for private synthetic testing, 2026-11-18.

Contract Pack v1.2 and the explicit WP-16 request authorize replacing prototype AI
and retiring Daily Reflection. The platform remains Go/Gin, in-process composition,
SQL Server 2022 and one UUID conversation/message store.

Gin maps OpenAPI 0.3.0 JSON to Gateway/Orchestrator. Only validated final replies
are exposed after atomic finalization. Queued/status/replay/retry/recover are
explicit; there is no SSE or compatibility fallback to experimental AI.
Memory controls retain the WP-12 Committer's authority, evidence and receipts.
An externally reviewed deployment mapping supplies explicit consent/use grants;
there is no default policy inferred from retained data or UI visibility.

The API process owns Memory, Reflection and maintenance-only Scheduler worker
lifetime, joins on cancellation and fails readiness/stops on persistent worker
failure. **Proactive execution and dispatch remain disabled.** Synthetic turns
and notification delivery are not implemented.

Migration 0023 adds account visibility metadata. Hiding revokes account access and
fences AI work while retaining all content, derived records and consent. All
previous migrations and the legacy Daily Reflection table/rows are preserved.
Historical SQL fixture builders exist only in tests to validate upgrades; no old
AI generator/safety/prompt code is linked into runtime. Letters' experimental
provider dependency is retired; new letter inference is unavailable until reviewed.

The API change is intentionally breaking for old internal clients. See the
WP-16 operation guide for payloads, errors, consent, client changes and rollback.
Frontend/browser migration and reviewed release assets are separate activation
requirements; neither is implied by backend unit/SQL tests. Full runtime replay,
Reflection merge/split, production authentication and clinical/privacy approval
remain explicit gates. Reverting to the prototype is not an approved fallback.
