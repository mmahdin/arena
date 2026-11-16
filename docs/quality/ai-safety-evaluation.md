# Task 06 — Synthetic AI safety evaluation (not clinical validation)

**Scope:** internal-only synthetic tester messages, Persian-first chat with optional OpenAI-compatible provider. This evaluation assesses a few **known phrases**, not the safety of psychological advice or actual crisis detection. Do not feed real mental-health disclosures to this system.

## Deterministic safeguards

| Scenario | Expected behavior | Test location |
| --- | --- | --- |
| Explicit self-harm phrase (Persian/English) | Respond with a fixed supportive message; skip provider and avoid invented emergency numbers | `policy/safety_test.go`, `application/service_test.go` |
| Ordinary stress | Forward to configured provider with bounded recent context | `application/service_test.go` |
| Model impersonates clinician | Replace model answer with neutral safety fallback | `policy/safety_test.go` |
| Model directs stopping/increasing medication | Replace model answer; do not issue individualized instruction | `policy/safety_test.go`, `application/service_test.go` |
| Model promotes exclusive attachment | Replace model answer | `policy/safety_test.go`, `application/service_test.go` |
| Model asserts it has called emergency services | Replace model answer (this app cannot summon help) | `policy/safety_test.go` |
| Model discloses recognizable system prompt text | Replace model answer for the tested exact fragment | `policy/safety_test.go` |
| Model returns empty, malformed, huge or control-character reply | Fail as generic provider failure; don't save incomplete turn | `application/service_test.go` |
| Context canceled before/during generation | Fail as generic provider failure, even when gateway ignores cancellation | `application/service_test.go` |
| Provider times out, refuses, returns truncated/filtered text | Generic `503 AI_UNAVAILABLE`, no raw provider payload/secret in error | `adapters/openai/gateway_test.go` plus existing HTTP tests |
| Resend same `clientTurnId` after failure | Existing fenced-turn retry; never generate a second stored pair for a completed ID | Task 04 SQL Server integration/smoke tests |

## Known gaps — explicit NOT-PASSED/NOT-TESTED safety dimensions

- Phrase matching can **miss indirect, metaphorical, negated, euphemistic, culturally specific, or misspelled crisis statements** and can overmatch when discussing crises academically or quoting someone else. Detection accuracy is **not measured**; don't describe it as screening or classification.
- Output patterns miss paraphrased or multi-step unsafe advice, ungrounded medical suggestions, subtle coercion, delusion reinforcement, invented psychological traits, and novel prompt-injection techniques.
- The system prompt is guidance, not a deterministic enforcement mechanism. Human review and dedicated evaluation are mandatory before any real-user or externally accessible use.
- No policy for country-specific verified emergency resources or clinical escalation; do not invent a hotline number or promise contact with responders.
- Provider HTTP validation is not a real provider acceptance test. Live requests are strictly optional and **only with non-sensitive synthetic text** after provider data-handling review.
- No per-user rate limit/cost quota, structured multi-turn adversarial testing, or validated content moderation service in this slice.

## Developer verification

From the backend root:

```sh
python3 scripts/check_structure.py
python3 scripts/check_contract.py
cd apps/api && go test ./internal/modules/ai/... ./internal/platform/config
cd ../.. && make integration-test && make smoke
```

For a dependency-free smoke of *only* the deterministic phrase matcher, from `apps/api/internal/modules/ai/policy`:

```sh
GOPROXY=off GOWORK=off go test safety.go safety_test.go -count=1
```

Use a disposable local SQL Server with seeded synthetic users. Mark `PASSED / FAILED / BLOCKED / NOT RUN` separately. Do **not** include real chat payloads or secrets in test logs or CI artifacts. The prompt version is `supportive-v3`; older stored turns keep their previous `promptVersion`.
