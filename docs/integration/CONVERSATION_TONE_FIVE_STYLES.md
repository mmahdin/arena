# Five conversational styles — Tone Refinement

The original five choices in the designer's Approach screen now round-trip as independent strings in `/api/v1/me` and `PATCH /api/v1/me/profile`. UI components, pages, names and artwork are unchanged.

| UI label | API/SQL value | Conversation behavior |
| --- | --- | --- |
| گفت‌وگوی آزاد | `balanced` | flexible listening and optional guidance |
| همدلی و شنیدن | `listening` | reflections before advice |
| به من یاد بده | `teaching` | clear general explanations, no clinical treatment |
| راهنمایی آرام | `gentle_guidance` | gentle optional practical steps |
| حل مسئله با هم | `collaborative` | exploring options and tradeoffs together |

Old profiles with `guidance` load as «راهنمایی آرام» and remain valid. Saving the choice writes `gentle_guidance`. Profile updates are immediately reflected in the **next new generated turn** by server profile lookup; existing messages and already completed idempotent turns are unchanged. The backend prompt policy is versioned `supportive-v4` and the shared safety boundaries are not relaxable by profile style. The mock model does not demonstrate response-style variation, so validate actual constructed system prompts via Go tests or an explicitly approved synthetic-only local OpenAI-compatible provider.

Apply the matching Backend patch (including SQL Server migration 0011) before Frontend. No onboarding UI or voice changes. Test with two synthetic accounts, refresh/re-entry, and five-way selection/reloading at mobile width. Current local-only non-production boundaries remain.
