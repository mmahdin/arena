# WP-11 — plan-bound Response Generator

Owner: AI module. Authority: Final Architecture §3.7–3.8 and Contract Pack v1.2
§10.3, §10.7, §11.1/11.3. This package consumes a WP-10 gated objective/plan;
it does not perform reasoning, select techniques, or write state.

`Load` consumes an immutable release-verified WP-03 snapshot. It loads reviewed
persona/crisis prompts, communication/boundary policies, selected Skill Cards,
protocol text and budgets. Synthetic bundles require an explicit test-only
option. The shipped placeholder bundle remains unusable for production.

The deterministic renderer emits static blocks first, then the plan, primary
and at most two secondary full cards, optional resolved protocol specification,
only permitted context, preferences, recent history (at most six), current user
message and a code-only regeneration note. All dynamic content is XML escaped;
UUID handles are attributes. Memory epistemic/status/age and verbatim provenance
are retained. Only oldest history is trimmed to fit; an impossible budget rejects
the call rather than altering instructions, evidence, the current message or
the plan. Input-token counts use WP-08/10's conservative rune approximation.

`Service.Generate` is ONE streaming attempt through WP-04's strong tier. WP-04
owns pre-token model failover. Streams are closed on every exit; malformed,
empty, unfinished, oversized and partial responses never become accepted output.
Usage and provider finish metadata survive successful completion, and available
usage is retained on rejected attempts. Split sentinels are intercepted before
forwarding their prefix. Elevated/crisis drafts are locally buffered. Sinks are
internal only and must never be HTTP responses.

Use `composition.NewAIResponseGenerator` with an explicit WP-09 AI-use policy,
the same pinned snapshot and WP-04 client used by `NewAITurnBrain`. Supply the
result plus a real WP-07 Validator to `orchestrator.NewIntegrated`. That existing
controller supplies authenticated call metadata, runs WP-07 StreamGuard and
Finalize, permits at most one regeneration, validates every reviewed fallback,
and commits the accepted reply through the existing fenced atomic finalizer.
Never deliver `Generate`'s draft directly to a caller.

The WP-11 integration clamps total/stage deadlines and bundle/caller token and
regeneration limits. A shared request-local call allowance covers small/strong
provider preflights across Brain/hop/generation/failover/regeneration; classifier
and embedding calls retain their own budgets. An exhausted allowance goes to a
validated template. The optional allowance is inert for earlier callers.

Source verification is mandatory in production: the MSSQL adapter reuses WP-10's
owner/visibility/content/chronology checks and WP-09's explicit AI-use policy,
also checking derived-memory evidence counts and sensitivity. Assistant history
remains system-observed; it cannot be relabelled as a user report. Reads never
delete data, clear content, invalidate derived records or alter consent.

FB-003/004 use objective templates; FB-005 retains the approved crisis script.
FB-006 uses the bundle's existing generic `other` crisis family with resources
on an elevated/follow-up provider outage. Missing/stale/unapproved content fails
closed. This selects response content without fabricating a higher risk state.
Sentinels select the approved reflective fallback outside an existing crisis.

Resolved `ProtocolStepSpec` rendering checks instructions, adaptations, offers
and required text against the bundle. It does not resolve protocols, establish
consent or advance an instance. The Orchestrator's WP-13 execution blocker stays
in place. A future resolver must also bind WP-07 protocol requirements on the
turn context. Existing protocol definitions contain English instructions only;
unreviewed translations are rejected.

No HTTP/API/schema/migration changes, runtime cutover, WP-12/13/14 worker or
experimental AI response dependency is introduced. See the
[verification and release blockers](../../../../../../../docs/quality/ai-core-wp11.md).
