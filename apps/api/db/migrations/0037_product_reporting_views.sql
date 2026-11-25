-- PROD-02: aggregate-only, synthetic-only business intelligence contract.
-- SQL Server 2022. The migration runner owns the transaction and checksum.
-- Never expose user/subject IDs, text, answers, phones, prompts or content.
-- No backfill: metrics start with the first PROD-01 event.
CREATE ROLE theia_analytics_reader AUTHORIZATION dbo;

-- Funnel by the DATE OF REGISTRATION, not by the date of conversion.
-- A user created before event tracking has no registration cohort and is
-- intentionally absent; lifetime conversions may change recent cohort totals.
EXEC(N'CREATE VIEW dbo.v_product_onboarding_cohorts_synthetic AS
WITH registered AS (
    SELECT e.user_id, CONVERT(date,e.occurred_at) AS cohort_date_utc
    FROM dbo.product_events AS e
    INNER JOIN dbo.users AS u ON u.id=e.user_id AND u.is_test=1
    WHERE e.event_name=''user_registered''
), member_state AS (
    SELECT r.cohort_date_utc, r.user_id,
           MAX(CASE WHEN e.event_name=''onboarding_started'' THEN 1 ELSE 0 END) AS started,
           MAX(CASE WHEN e.event_name=''onboarding_completed'' THEN 1 ELSE 0 END) AS completed
    FROM registered AS r
    LEFT JOIN dbo.product_events AS e
      ON e.user_id=r.user_id
      AND e.event_name IN (''onboarding_started'',''onboarding_completed'')
    GROUP BY r.cohort_date_utc,r.user_id
)
SELECT cohort_date_utc, COUNT_BIG(*) AS registered_test_users,
       SUM(CONVERT(bigint,started)) AS started_test_users,
       SUM(CONVERT(bigint,completed)) AS completed_test_users
FROM member_state
GROUP BY cohort_date_utc');

-- Chat activity: a day counts as active only when an AI Turn was ADMITTED.
-- This is NOT login DAU, MAU, app-session DAU or clinical engagement.
-- Completion/failure are event-date facts, not a terminal failure-rate ratio.
EXEC(N'CREATE VIEW dbo.v_product_daily_engagement_synthetic AS
SELECT CONVERT(date,e.occurred_at) AS activity_date_utc,
       COUNT(DISTINCT CASE WHEN e.event_name=''ai_turn_admitted'' THEN e.user_id END) AS chat_active_test_users,
       SUM(CASE WHEN e.event_name=''ai_turn_admitted'' THEN CONVERT(bigint,1) ELSE CONVERT(bigint,0) END) AS turns_admitted,
       SUM(CASE WHEN e.event_name=''conversation_started'' THEN CONVERT(bigint,1) ELSE CONVERT(bigint,0) END) AS conversations_started,
       SUM(CASE WHEN e.event_name=''bookmark_saved'' THEN CONVERT(bigint,1) ELSE CONVERT(bigint,0) END) AS bookmarks_saved,
       SUM(CASE WHEN e.event_name=''ai_turn_completed'' THEN CONVERT(bigint,1) ELSE CONVERT(bigint,0) END) AS turns_completed,
       SUM(CASE WHEN e.event_name=''ai_turn_failed'' THEN CONVERT(bigint,1) ELSE CONVERT(bigint,0) END) AS turns_first_failed
FROM dbo.product_events AS e
INNER JOIN dbo.users AS u ON u.id=e.user_id AND u.is_test=1
WHERE e.event_name IN (''ai_turn_admitted'',''ai_turn_completed'',''ai_turn_failed'',
                       ''conversation_started'',''bookmark_saved'')
GROUP BY CONVERT(date,e.occurred_at)');

-- First chat cohort: exact UTC calendar-day 1 / 7 / 30 returns.
-- NULL retention until the ENTIRE target day has elapsed; never treat an
-- unfinished day as zero. A chat on the first day defines cohort membership.
EXEC(N'CREATE VIEW dbo.v_product_chat_retention_synthetic AS
WITH chat_days AS (
    SELECT DISTINCT e.user_id, CONVERT(date,e.occurred_at) AS chat_date_utc
    FROM dbo.product_events AS e
    INNER JOIN dbo.users AS u ON u.id=e.user_id AND u.is_test=1
    WHERE e.event_name=''ai_turn_admitted''
), first_chats AS (
    SELECT user_id, MIN(chat_date_utc) AS cohort_date_utc
    FROM chat_days GROUP BY user_id
), outcomes AS (
    SELECT c.cohort_date_utc, c.user_id,
           MAX(CASE WHEN d.chat_date_utc=DATEADD(day,1,c.cohort_date_utc) THEN 1 ELSE 0 END) AS d1,
           MAX(CASE WHEN d.chat_date_utc=DATEADD(day,7,c.cohort_date_utc) THEN 1 ELSE 0 END) AS d7,
           MAX(CASE WHEN d.chat_date_utc=DATEADD(day,30,c.cohort_date_utc) THEN 1 ELSE 0 END) AS d30
    FROM first_chats AS c
    LEFT JOIN chat_days AS d ON d.user_id=c.user_id
    GROUP BY c.cohort_date_utc,c.user_id
)
SELECT cohort_date_utc, COUNT_BIG(*) AS first_chat_test_users,
       CASE WHEN cohort_date_utc<DATEADD(day,-1,CONVERT(date,SYSUTCDATETIME()))
            THEN SUM(CONVERT(bigint,d1)) ELSE CAST(NULL AS bigint) END AS d1_returning_test_users,
       CASE WHEN cohort_date_utc<DATEADD(day,-7,CONVERT(date,SYSUTCDATETIME()))
            THEN SUM(CONVERT(bigint,d7)) ELSE CAST(NULL AS bigint) END AS d7_returning_test_users,
       CASE WHEN cohort_date_utc<DATEADD(day,-30,CONVERT(date,SYSUTCDATETIME()))
            THEN SUM(CONVERT(bigint,d30)) ELSE CAST(NULL AS bigint) END AS d30_returning_test_users
FROM outcomes GROUP BY cohort_date_utc');

-- Configured-price ESTIMATES of turn-linked physical LLM attempts only.
-- Orphan/background attempts with no turn are excluded, not counted as zero.
-- Unknown or unpriced costs remain explicit; this is not provider billing.
EXEC(N'CREATE VIEW dbo.v_product_ai_usage_daily_synthetic AS
SELECT CONVERT(date,a.started_at) AS attempt_date_utc,
       COUNT_BIG(*) AS llm_attempts,
       SUM(CASE WHEN a.status=''succeeded'' THEN CONVERT(bigint,1) ELSE CONVERT(bigint,0) END) AS succeeded_attempts,
       SUM(CASE WHEN a.status=''failed'' THEN CONVERT(bigint,1) ELSE CONVERT(bigint,0) END) AS failed_attempts,
       SUM(CASE WHEN a.status=''incomplete'' OR a.status=''started'' THEN CONVERT(bigint,1) ELSE CONVERT(bigint,0) END) AS incomplete_attempts,
       SUM(CASE WHEN a.cost_status=''estimated'' THEN CONVERT(bigint,1) ELSE CONVERT(bigint,0) END) AS priced_attempts,
       SUM(CASE WHEN a.cost_status<>''estimated'' THEN CONVERT(bigint,1) ELSE CONVERT(bigint,0) END) AS unpriced_attempts,
       SUM(CASE WHEN a.cost_status=''estimated'' THEN a.cost_micro_usd ELSE CONVERT(bigint,0) END) AS known_estimated_micro_usd,
       SUM(CASE WHEN a.input_tokens IS NULL THEN CONVERT(bigint,1) ELSE CONVERT(bigint,0) END) AS missing_usage_attempts,
       SUM(COALESCE(a.input_tokens,CONVERT(bigint,0))) AS reported_input_tokens,
       SUM(COALESCE(a.output_tokens,CONVERT(bigint,0))) AS reported_output_tokens
FROM dbo.ai_llm_attempts AS a
INNER JOIN dbo.turns AS t ON t.id=a.turn_id
INNER JOIN dbo.users AS u ON u.id=t.user_id AND u.is_test=1
GROUP BY CONVERT(date,a.started_at)');

-- Dedicated role: SELECT on views only. The login/user is provisioned outside
-- migrations, with a generated local secret, after migration 0037 is applied.
GRANT SELECT ON OBJECT::dbo.v_product_daily_events_synthetic TO theia_analytics_reader;
GRANT SELECT ON OBJECT::dbo.v_product_onboarding_cohorts_synthetic TO theia_analytics_reader;
GRANT SELECT ON OBJECT::dbo.v_product_daily_engagement_synthetic TO theia_analytics_reader;
GRANT SELECT ON OBJECT::dbo.v_product_chat_retention_synthetic TO theia_analytics_reader;
GRANT SELECT ON OBJECT::dbo.v_product_ai_usage_daily_synthetic TO theia_analytics_reader;
-- Defense in depth if the login accidentally receives wider schema access.
DENY SELECT ON OBJECT::dbo.product_events TO theia_analytics_reader;
DENY SELECT ON OBJECT::dbo.users TO theia_analytics_reader;
DENY SELECT ON OBJECT::dbo.user_profiles TO theia_analytics_reader;
DENY SELECT ON OBJECT::dbo.conversations TO theia_analytics_reader;
DENY SELECT ON OBJECT::dbo.messages TO theia_analytics_reader;
DENY SELECT ON OBJECT::dbo.turns TO theia_analytics_reader;
DENY SELECT ON OBJECT::dbo.ai_llm_attempts TO theia_analytics_reader;
DENY SELECT ON OBJECT::dbo.ai_trace_refs TO theia_analytics_reader;
