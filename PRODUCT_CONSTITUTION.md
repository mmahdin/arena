# Product Constitution — Internal Test Only

> **Next-stage decision (2026-11-19):** The original prototype exclusions
> below are historical for **Authentication and private Back-Office only**.
> ADR 0012 approves Keycloak + SMS OTP user login and staff login, and ADR
> 0013 approves a private full-access single-admin operational console.
> Read `docs/plans/README.md` for source-of-truth plans. These systems are
> **not implemented yet**. All synthetic-only, privacy, non-clinical and
> non-public-release restrictions below remain fully in force.

- **Primary job:** let a tester start a chat, receive a supportive AI response, return to a previous conversation, and see only their own test data.
- **Historical MVP surface:** Dev user switcher, tiny onboarding/profile, conversation list/history, JSON chat turns, simple AI response. Its former exclusions (including memory engine, Back-Office and OIDC) do not override later explicit ADR 0011/0012/0013 workstream decisions.
- **Persona decision [OPEN]:** companion vs professional growth guide. Use a warm, respectful, clearly non-human neutral tone for now.
- **No clinical claims:** The system is not a psychologist, psychiatrist, therapy, diagnosis engine, or crisis line. Do not present keyword safety filtering as validated clinical risk assessment.
- **No dependence optimization:** Do not increase time-on-chat or encourage replacing human relationships with AI.
- **Privacy:** Seed users and all texts must be synthetic/non-sensitive. User content is never logged, advertised or used to train models from our application code.
- **Internal-only switcher is not auth:** a tester can select any synthetic identity. It will be retired at cross-repo Auth cutover. Verified Auth alone does **not** authorize real-user access; encryption, retention/deletion policy, data processor review, rights handling and clinical safety evaluation remain separate gates.
- **Patient-facing boundaries:** No user-facing public release or treatment claims without expert review and appropriate local support resource policy.
- **Prioritization:** deliver working loop and correctness before any orchestration framework or complex distributed architecture.
