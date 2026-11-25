-- PROD-01: server-authored, deliberately content-free product events.
-- Synthetic users only. No raw chat, questionnaire answers, phones, prompt text,
-- IPs, user agents, identity-provider subjects, arbitrary properties or URLs.
-- Forward-only SQL Server 2022 migration; the runner owns the transaction.
CREATE TABLE dbo.product_events (
    id uniqueidentifier NOT NULL CONSTRAINT product_events_pk PRIMARY KEY,
    user_id uniqueidentifier NOT NULL,
    event_name varchar(40) COLLATE Latin1_General_100_BIN2 NOT NULL,
    subject_id uniqueidentifier NOT NULL,
    schema_version tinyint NOT NULL CONSTRAINT product_events_version_default DEFAULT (1),
    occurred_at datetime2(3) NOT NULL CONSTRAINT product_events_occurred_default DEFAULT SYSUTCDATETIME(),
    CONSTRAINT product_events_user_fk FOREIGN KEY (user_id) REFERENCES dbo.users(id),
    CONSTRAINT product_events_schema_check CHECK (schema_version=1),
    CONSTRAINT product_events_catalog_check CHECK (event_name IN (
        'user_registered','onboarding_started','onboarding_completed',
        'conversation_started','ai_turn_admitted','ai_turn_completed',
        'ai_turn_failed','bookmark_saved')),
    CONSTRAINT product_events_user_subject_check CHECK (
        event_name NOT IN ('user_registered','onboarding_started','onboarding_completed')
        OR subject_id=user_id),
    CONSTRAINT product_events_once_per_subject UNIQUE(event_name,subject_id)
);
-- A subject can be a user, conversation, turn or saved message. The referenced
-- business write and the event are in the same application SQL transaction.
-- All event writes must use the server-side synthetic eligibility predicate.
CREATE INDEX product_events_daily_idx
    ON dbo.product_events(occurred_at,event_name) INCLUDE (user_id);
CREATE INDEX product_events_owner_idx
    ON dbo.product_events(user_id,occurred_at) INCLUDE (event_name);
-- An aggregate-only analytics surface for later read-only Metabase access.
-- Counts cover NEW events recorded after PROD-01 deployment; not a backfill.
EXEC(N'CREATE VIEW dbo.v_product_daily_events_synthetic
AS
SELECT CONVERT(date,e.occurred_at) AS activity_date_utc,
       e.event_name,
       COUNT_BIG(*) AS event_count,
       COUNT(DISTINCT e.user_id) AS distinct_test_users
FROM dbo.product_events AS e
JOIN dbo.users AS u ON u.id=e.user_id AND u.is_test=1
GROUP BY CONVERT(date,e.occurred_at),e.event_name');
-- Defense in depth: direct SQL writers must not circumvent the application gate.
EXEC(N'CREATE TRIGGER dbo.product_events_reject_real_users
ON dbo.product_events AFTER INSERT, UPDATE
AS
BEGIN
    SET NOCOUNT ON;
    IF EXISTS (
        SELECT 1 FROM inserted AS e
        JOIN dbo.users AS u ON u.id=e.user_id
        WHERE u.is_test<>1 OR u.deleted_at IS NOT NULL
    )
        THROW 51041, ''product events require an active synthetic user'', 1;
END');
