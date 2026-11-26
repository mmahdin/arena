# ADR-003: real LLM tests require explicit gate

- **Status:** Proposed internal testing rule.
- Backend is previously validated for MSSQL + Mock AI. Connect frontend to Mock API first and verify full E2E.
- Before toggling real provider: server-only credentials, model selection, request-cost/rate caps, synthetic prompts, error/timeout behavior and crisis test matrix.
- Real-provider test success does not imply clinical safety; never test on real mental-health disclosures or public internet.
