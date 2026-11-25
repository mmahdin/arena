# Onboarding and AI Core

Explicit allowlisted onboarding answers remain owner-scoped profile data saved on
final Ready. WP-16 removes the prototype prompt bridge that injected these answers
and experimental conversation styles into every model call. No historical answer
backfill or inferred profile is created.

The new core uses its approved WorkingContext and explicit AI preferences/consent
contracts. Onboarding completion still gates conversation creation and user turns.
Additional onboarding-to-AI mappings require the reviewed knowledge/use policy;
profile storage or conversation sharing alone never authorizes model disclosure.
See the WP-16 HTTP/deployment guide for client migration and clinical blockers.
