-- BO-04: durable, sanitized operational evidence. Never store prompts,
-- response bodies, phone numbers, provider error text, or user IDs here.
-- An uncompleted attempt means interrupted/unknown, not zero spending.
CREATE TABLE dbo.ai_llm_attempts (
    id uniqueidentifier NOT NULL CONSTRAINT ai_llm_attempts_pk PRIMARY KEY,
    turn_id uniqueidentifier NULL,
    kind varchar(16) NOT NULL,
    tier varchar(40) NOT NULL,
    provider varchar(80) NOT NULL,
    model varchar(80) NOT NULL,
    started_at datetime2(3) NOT NULL,
    finished_at datetime2(3) NULL,
    status varchar(16) NOT NULL CONSTRAINT ai_llm_attempts_status_default DEFAULT 'started',
    failure_kind varchar(32) NULL,
    duration_ms bigint NULL,
    input_tokens bigint NULL,
    output_tokens bigint NULL,
    cached_tokens bigint NULL,
    input_rate_micro_usd_per_m bigint NOT NULL,
    output_rate_micro_usd_per_m bigint NOT NULL,
    cached_rate_micro_usd_per_m bigint NOT NULL,
    cost_micro_usd bigint NULL,
    cost_status varchar(16) NOT NULL CONSTRAINT ai_llm_attempts_cost_default DEFAULT 'unknown',
    CONSTRAINT ai_llm_attempts_kind_check CHECK (kind IN ('structured','stream','embed')),
    CONSTRAINT ai_llm_attempts_status_check CHECK (status IN ('started','succeeded','failed','incomplete')),
    CONSTRAINT ai_llm_attempts_cost_check CHECK (cost_status IN ('unknown','unpriced','estimated')),
    CONSTRAINT ai_llm_attempts_usage_check CHECK (
        (input_tokens IS NULL AND output_tokens IS NULL AND cached_tokens IS NULL)
        OR (input_tokens>=0 AND output_tokens>=0 AND cached_tokens>=0 AND cached_tokens<=input_tokens)),
    CONSTRAINT ai_llm_attempts_cost_value_check CHECK (
        (cost_status='estimated' AND cost_micro_usd IS NOT NULL AND cost_micro_usd>=0)
        OR (cost_status<>'estimated' AND cost_micro_usd IS NULL)),
    CONSTRAINT ai_llm_attempts_rates_check CHECK (
        input_rate_micro_usd_per_m>=0 AND output_rate_micro_usd_per_m>=0 AND cached_rate_micro_usd_per_m>=0),
    CONSTRAINT ai_llm_attempts_times_check CHECK (finished_at IS NULL OR finished_at>=started_at),
    CONSTRAINT ai_llm_attempts_lifecycle_check CHECK (
        (status='started' AND finished_at IS NULL AND duration_ms IS NULL)
        OR (status<>'started' AND finished_at IS NOT NULL AND duration_ms>=0))
);
CREATE INDEX ai_llm_attempts_time_idx ON dbo.ai_llm_attempts(started_at DESC,id DESC)
    INCLUDE (status,kind,tier,provider,model,cost_status,cost_micro_usd);
CREATE INDEX ai_llm_attempts_turn_idx ON dbo.ai_llm_attempts(turn_id,started_at DESC)
    WHERE turn_id IS NOT NULL;

-- Individual stage observations are independent of the full-payload local
-- inspector. Duration may be NULL when unavailable; do not fabricate P95.
CREATE TABLE dbo.ai_stage_events (
    id uniqueidentifier NOT NULL CONSTRAINT ai_stage_events_pk PRIMARY KEY,
    turn_id uniqueidentifier NOT NULL,
    stage varchar(40) NOT NULL,
    status varchar(16) NOT NULL,
    duration_ms bigint NULL,
    attempt smallint NULL,
    event_count int NULL,
    fallback_used bit NOT NULL CONSTRAINT ai_stage_fallback_default DEFAULT 0,
    rule_id varchar(80) NULL,
    recorded_at datetime2(3) NOT NULL CONSTRAINT ai_stage_recorded_default DEFAULT SYSUTCDATETIME(),
    CONSTRAINT ai_stage_status_check CHECK (status IN ('success','failed','degraded','skipped')),
    CONSTRAINT ai_stage_duration_check CHECK (duration_ms IS NULL OR duration_ms>=0),
    CONSTRAINT ai_stage_attempt_check CHECK (attempt IS NULL OR attempt BETWEEN 1 AND 20),
    CONSTRAINT ai_stage_count_check CHECK (event_count IS NULL OR event_count>=0)
);
CREATE INDEX ai_stage_events_time_idx ON dbo.ai_stage_events(recorded_at DESC,id DESC)
    INCLUDE (stage,status,duration_ms);
CREATE INDEX ai_stage_events_turn_idx ON dbo.ai_stage_events(turn_id,recorded_at DESC);

-- Liveness only. A fresh heartbeat does NOT prove the worker is processing
-- jobs correctly; future BO jobs APIs report backlog, failures and stale data.
CREATE TABLE dbo.ai_worker_heartbeats (
    instance_id uniqueidentifier NOT NULL,
    worker_kind varchar(16) NOT NULL,
    status varchar(16) NOT NULL,
    last_seen_at datetime2(3) NOT NULL,
    CONSTRAINT ai_worker_heartbeats_pk PRIMARY KEY (instance_id,worker_kind),
    CONSTRAINT ai_worker_heartbeats_kind_check CHECK (worker_kind IN ('memory','reflection','scheduler')),
    CONSTRAINT ai_worker_heartbeats_status_check CHECK (status IN ('running','stopped','failed'))
);
CREATE INDEX ai_worker_heartbeats_freshness_idx ON dbo.ai_worker_heartbeats(last_seen_at DESC)
    INCLUDE (worker_kind,status);
