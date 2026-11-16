# Product Constitution — Internal Test Only

- **Primary job:** let a tester start a chat, receive a supportive AI response, return to a previous conversation, and see only their own test data.
- **MVP surface:** Dev user switcher, tiny onboarding/profile, conversation list/history, JSON chat turns, simple AI response. No memory engine, reports, back office, OIDC, or diagnostic tools.
- **Persona decision [OPEN]:** companion vs professional growth guide. Use a warm, respectful, clearly non-human neutral tone for now.
- **No clinical claims:** The system is not a psychologist, psychiatrist, therapy, diagnosis engine, or crisis line. Do not present keyword safety filtering as validated clinical risk assessment.
- **No dependence optimization:** Do not increase time-on-chat or encourage replacing human relationships with AI.
- **Privacy:** Seed users and all texts must be synthetic/non-sensitive. User content is never logged, advertised or used to train models from our application code.
- **Internal-only switcher is not auth:** a tester can select any seeded identity. Before any real-user access, require OIDC/real authorization, encryption, retention/deletion policy, data processor review and clinical safety evaluation.
- **Patient-facing boundaries:** No user-facing public release or treatment claims without expert review and appropriate local support resource policy.
- **Prioritization:** deliver working loop and correctness before any orchestration framework or complex distributed architecture.
