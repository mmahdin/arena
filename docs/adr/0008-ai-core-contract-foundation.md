# ADR 0008: AI Core WP-01 contract foundation

Date: 2026-11-16

Status: accepted for the requested WP-01 implementation only.

## Authority and scope

The Final AI Core Architecture Document and Contract Pack v1.2 in `docs/ai-core/`
are authoritative for this work. v1.2 supersedes earlier packs where they conflict:
one Go/Gin modular monolith, UUIDs, SQL Server 2022, direct Go ports, user-first
admission, per-user leases and atomic validated assistant/state finalization.

WP-01 implements §§3–6 and the record types in Appendix A: typed contracts,
consumer-owned interfaces, four embedded draft 2020-12 LLM schemas, strict decoding,
synthetic fixtures and tests. New code does not import, adapt, or fall back to the
prototype AI. `contract/schema` uses `santhosh-tekuri/jsonschema/v6` v6.0.2 (Go 1.21
minimum, compatible with the repository's Go 1.23 baseline) for actual schema
validation; formats are asserted and external schema loading is disabled.

Only contract primitives and structural validation execute. The main API remains
unwired to these new packages until a later work package supplies the implementation.
There is no new service, deployment, migration, prompt, production bundle content,
database/LLM adapter, worker, rule engine or OpenAPI change in this patch.

## Inspected prerequisite and bounded retirement inventory

WP-00's full runtime retirement is not claimed complete. The requested WP-01 scope
permits contract scaffolding, while Contract Pack §2.3(6) permits explicitly bounded
transitional code. Removing the running chat/provider flow in a contracts-only patch
would require unrequested runtime/API work or leave nonfunctional replacements.
The following dependencies were inspected and are explicitly targeted for removal:

| Existing code | Dependency and assigned transition |
| --- | --- |
| `ai/application/service.go`, `ai/policy/*`, `ai/module.go`, old AI tests | One-shot reply, recent history and `supportive-v5` placeholder safety. Delete with new Gateway/Orchestrator wiring in WP-05; no compatibility fallback in the new pipeline. |
| `ai/adapters/openai`, `ai/adapters/mock` | Replace with the new provider-neutral interfaces in WP-04; remove remaining consumers during WP-05 integration. |
| `conversation/application/service.go`, `conversation/module.go` | `ReplyGenerator` and two-message-at-end orchestration. Replace workflow in WP-05 using WP-02's single-store persistence redesign. Keep owner-scoped chat operations through appropriate ports. |
| `app/composition/api.go` and `/me/ai-status` | Old AI construction, prompt-version coupling and HTTP surfaces. Rewire in WP-05; remove obsolete status/public contracts in WP-16. |
| `letters/module.go`, `letters/adapters/generator/*` | Direct imports of old AI application, provider and policy. Migrate this consumer when deleting the provider/application in WP-04/05; do not retain old AI just for letters. |
| Top-level `modules/reflection`, its routes, tests and docs | User-authored Daily Reflection CRUD. Remove API/code surfaces in WP-16; this is unrelated to the new psychological Reflection contracts. |
| `0012_daily_reflections.sql` / `dbo.daily_reflections` | Applied migration stays immutable. The WP-02 soft-delete correction (ADR 0010) preserves the table and every row. Only code/route retirement is planned; no table disposal is authorized. |
| `dbo.messages`, `dbo.turns`, existing migration runner | One physical chat store retained. WP-02 changes lifecycle/status/leases and verifies actual SQL ownership, rollback, migration and recovery semantics. |

No user database or secrets were inspected or modified for this contracts-only work.
Existing untracked specification files are inputs, not generated patch content.

## Decisions that complete the sketches

See `modules/ai/contract/README.md` for exact schema/type reconciliation. Undefined
interface support types are minimal data contracts, not stub implementations.
Ownership travels on admission; generator input contains the current turn; protocol
patches can carry consent message provenance. `contract.LLMUsage` has one canonical
definition. Closed enums have typed constants and `Valid` methods. Required arrays,
snake_case tags, optional zeros and nullable identifiers have explicit JSON behavior.

The independent fixture/schema/mirror tests cover structural BR-001/002/003/007/012/
020/021/024 restrictions where expressible by JSON Schema. They do not claim the
runtime context-dependent repairs, word-count rules, MC/RF support checks or safety
algorithms. Those remain assigned to WP-03 and WP-06–14. SQL/provider contract suites
must be run against their real adapters when those work packages are implemented.

## Compatibility and rollback

The patch is additive except for documentation and the schema-validator dependency.
It changes no current HTTP behavior or database schema and needs no data rollback.
Removing the new unreferenced packages and dependency reverts WP-01. No old AI
behavior is approved as a correctness baseline or permanent fallback by this decision.
