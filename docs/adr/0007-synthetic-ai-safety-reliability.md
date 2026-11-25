# ADR 0007 — Minimal synthetic-only AI safety and cancellation policy

Status: Accepted for internal synthetic MVP only (Task 06).

## Context

Task 05 introduced a bounded, optional external provider. The original single-form spelling checks missed obvious Persian/Arabic variants, and a model gateway ignoring a canceled context could hand back a successful reply. Neither prompt text nor small keyword checks can certify a mental-health service.

## Decision

- Version system instructions as `supportive-v3` with explicit non-clinical, non-dependency, no-medication-change and non-fabricated-emergency boundaries. Preserve the version recorded on existing turns.
- Normalize common Persian letter forms and zero-width spacing before evaluating limited **synthetic** crisis and output-risk phrases. Keep deterministic server-owned fallbacks; log neither the triggering text nor the rejected model output.
- Refuse model output containing a few high-confidence prohibited claim/instruction patterns. Keep the generic HTTP contract (`503 AI_UNAVAILABLE` for model failure) and the existing JSON response shape; no schema changes.
- Honor the request context **before** invoking the gateway and **after** it returns, including badly behaved providers. Refuse invalid UTF-8, unexpected control code points and oversized output. Existing fenced SQL Server turn release/retry semantics remain unchanged; do not automatically retry an upstream completion.
- Keep mock provider as default. Add synthetic bilingual unit cases with negative controls and a separate matrix documenting **known false negatives**.

## Constraints and follow-up

- Keyword/phrase matching **is not validated clinical safety assessment**. It may overmatch ordinary discussion and miss urgent indirect speech. Prompt injection and output advice cannot be fully controlled by pattern lists. Do not ship this to real users or public networks.
- Real-user access requires qualified clinical reviewers, privacy/security and consent design, validated crisis routing for supported locales, formal threat modeling, escalation response and real-world evaluation.
- Model quotas, usage caps, distributed rate limits and provider-specific safeguards remain out of scope. No new SQL Server migration or frontend protocol changes for Task 06.
