# Conversation

Conversation owns one SQL Server conversation/message/turn store. Its application
service provides owner-scoped CRUD and completed-onboarding gating for creation.
AI processing belongs exclusively to Gateway/Orchestrator, wired by composition.
There is no ReplyGenerator or prototype processing workflow.

`adapters/mssql/TurnStore` persists user input first, serializes per user across
conversations with expiring fenced leases, and atomically publishes only validated
assistant text with state/jobs/trace. Provider calls run outside SQL transactions.
Completed retries replay; terminal failures require explicit retry then FIFO
recovery. New HTTP output is OpenAPI 0.3.0 TurnResult, not the prototype response.

All lists/status/replay/admission respect owner and resource visibility. Conversation
hiding expires its lease and preserves messages, evidence, derived memory/state,
consent and other chats. Internal session close does not hide a conversation.
User-only failed/pending turns remain in message history. Message listings retain
the existing latest-100 bound; the paginated AI message export supports older data.

The removed legacy processing algorithm survives only as historical SQL fixture
construction in integration tests, to prove upgrade and data preservation. It is
not linked into the application. See ADR 0011 and the WP-16 operation guide.
