# Admin SQL adapters

BO-03 adds `admin_audit_events` only. Staff identity and sessions remain
owned by the shared Auth module. BO-04 onward adds bounded, explicit
administrative read models. Audit storage exposes insert and keyset-paginated
select only; table mutation is forbidden by migration trigger and restricted
SQL permissions. Never query private AI Core payloads through a generic
Admin SQL browser.

BO-08 adds `admin_eval_runs` (`0030`), fixed SQL projections, idempotent
SHA-256 + baseline imports and an append-only trigger. It stores strictly
validated sanitized reports, not text, trace payloads, raw fixtures or a
clinical approval state. Use only a disposable DB for integration tests.
