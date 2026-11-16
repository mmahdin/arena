# Explicit onboarding context for chat — 2026-11-11

The user requested using onboarding answers throughout chat, and exactly three
identity options: زن، مرد، تمایل ندارم بگم. This decision supersedes earlier
ephemeral-only intake restrictions for the internal synthetic prototype.

On final Ready, the frontend sends an explicit allowlist of selected age, identity,
therapy experience, onboarding mood, concerns, goals, desired feelings, faith,
self-reported neurodivergence, relationships, interests, strengths, barriers and
restorative activities. Missing answers are omitted. Referral, notification and
voice selections are not relevant text-chat context and are excluded. No local or
session storage is added. Privacy and Ready explain server storage and AI processing;
the separate conversation-sharing preference does not gate answering the user's chat.

`PATCH /me/profile` accepts optional `onboardingAnswers`. Omission preserves the
current stored value; `{}` clears it. Unknown keys, invalid types, identity/mood,
more than three focus selections, duplicate/list/field/aggregate overflow, control
and direction-override characters are rejected before writing. The API's existing
16 KiB body cap applies. No client-supplied user ID is accepted.

Migration `0016_onboarding_answers.sql` adds a nullable JSON column to
`dbo.user_profiles` with a valid-JSON/16,000-byte SQL constraint. It leaves old rows
NULL, adds no backfill and changes no previously applied migration. The runner
owns the transaction. Deploy migration before API, then frontend. Roll back
application binaries while retaining the nullable column; do not edit the migration
or drop stored answers as part of routine rollback. Local integration tests create
and delete only their own randomly named disposable database.

Conversation reads current answers through its profile port on each new turn.
Composition maps the validated user-domain data into a consumer-owned profile type
and removes declined disclosures. `supportive-v5` frames JSON as untrusted data,
asks the model to use relevant background and avoid repeating answered questions,
prioritizes current corrections, and treats onboarding mood as a historical snapshot.
No cross-conversation content search, inferred memory, automatic summaries or raw
payload logging is introduced. Completed idempotent replays keep the original reply
and prompt version. Existing profile edits preserve answers and completion time.

Selections discarded by an older frontend cannot be recovered. Draft answers still
disappear on refresh before Ready. Personalization guides model behavior; automated
tests of the payload do not prove real-model answer quality. The provider is never
silently replaced by Mock when an external call fails.

OpenAI request compatibility follows [official Chat Completions documentation](https://developers.openai.com/api/reference/resources/chat/subresources/completions/methods/create)
and the [GPT-5.6 Sol model](https://developers.openai.com/api/docs/models/gpt-5.6-sol).

## Validation

`go test ./...`, `python3 scripts/check_structure.py` and
`python3 scripts/check_contract.py` passed. The authored contract has 30 registered
method/route pairs. `TestOnboardingAnswersSQLServer` passed against SQL Server 2022
in a newly created isolated container: upgrade from migrations 0001–0015, repeat
migration, Unicode/mood-zero round trip, ordinary edit preservation, user isolation,
invalid-JSON constraint and explicit context clearing. The disposable database and
container were removed. Tests never accessed credentials from the existing local
SQL Server. No real OpenAI call, staging migration or deployment was executed.
