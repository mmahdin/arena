-- AI-owned persistence foundation; no workers or clinical algorithms.
CREATE TABLE dbo.ai_sessions (
    id uniqueidentifier NOT NULL PRIMARY KEY,
    user_id uniqueidentifier NOT NULL REFERENCES dbo.users(id) ON DELETE CASCADE,
    conversation_id uniqueidentifier NOT NULL,
    started_at datetime2(3) NOT NULL DEFAULT SYSUTCDATETIME(),
    last_activity_at datetime2(3) NOT NULL DEFAULT SYSUTCDATETIME(),
    closed_at datetime2(3) NULL,
    close_reason nvarchar(40) NULL,
    mode nvarchar(40) NOT NULL DEFAULT N'exploring',
    focus_thread_id uniqueidentifier NULL,
    turn_count int NOT NULL DEFAULT (0),
    last_objective nvarchar(40) NULL,
    consecutive_questions int NOT NULL DEFAULT (0),
    pending_question_json nvarchar(max) NULL CHECK (pending_question_json IS NULL OR ISJSON(pending_question_json)=1),
    summary nvarchar(max) NOT NULL DEFAULT N'',
    emotion_trajectory_json nvarchar(max) NOT NULL DEFAULT N'[]' CHECK (ISJSON(emotion_trajectory_json)=1),
    version bigint NOT NULL DEFAULT (1),
    CONSTRAINT FK_ai_sessions_owner FOREIGN KEY (conversation_id,user_id)
        REFERENCES dbo.conversations(id,user_id)
);
CREATE UNIQUE INDEX UX_ai_sessions_open ON dbo.ai_sessions(user_id, conversation_id) WHERE closed_at IS NULL;
CREATE INDEX IX_ai_sessions_recent ON dbo.ai_sessions(user_id,last_activity_at DESC);

CREATE TABLE dbo.ai_threads (
    id uniqueidentifier NOT NULL PRIMARY KEY,
    user_id uniqueidentifier NOT NULL REFERENCES dbo.users(id) ON DELETE CASCADE,
    topic nvarchar(400) NOT NULL,
    phase nvarchar(40) NOT NULL DEFAULT N'exploring' CHECK (phase IN (N'exploring',N'understanding_sufficient',N'working',N'maintaining',N'resolved',N'paused',N'dormant')),
    prev_phase nvarchar(40) NULL,
    priority real NOT NULL DEFAULT (0.5),
    slots_json nvarchar(max) NOT NULL DEFAULT N'{}' CHECK (ISJSON(slots_json)=1),
    goal nvarchar(max) NULL,
    hypothesis_ids_json nvarchar(max) NOT NULL DEFAULT N'[]' CHECK (ISJSON(hypothesis_ids_json)=1),
    related_thread_ids_json nvarchar(max) NOT NULL DEFAULT N'[]' CHECK (ISJSON(related_thread_ids_json)=1),
    formulation_version int NOT NULL DEFAULT (0),
    problem_tags_json nvarchar(max) NOT NULL DEFAULT N'[]' CHECK (ISJSON(problem_tags_json)=1),
    created_at datetime2(3) NOT NULL DEFAULT SYSUTCDATETIME(),
    last_active_at datetime2(3) NOT NULL DEFAULT SYSUTCDATETIME(),
    version bigint NOT NULL DEFAULT (1)
);
CREATE INDEX IX_ai_threads_user ON dbo.ai_threads(user_id,last_active_at DESC);

CREATE TABLE dbo.ai_user_state (
    user_id uniqueidentifier NOT NULL PRIMARY KEY REFERENCES dbo.users(id) ON DELETE CASCADE,
    tz nvarchar(100) NOT NULL DEFAULT N'UTC',
    locale nvarchar(24) NOT NULL DEFAULT N'fa', -- provisional, not evidence of full Persian safety coverage
    prefs_json nvarchar(max) NOT NULL DEFAULT N'{}' CHECK (ISJSON(prefs_json)=1),
    risk_level nvarchar(20) NOT NULL DEFAULT N'none'
        CHECK (risk_level IN (N'none',N'low',N'elevated',N'high',N'imminent')),
    risk_since datetime2(3) NULL,
    risk_follow_up_until datetime2(3) NULL,
    risk_history_json nvarchar(max) NOT NULL DEFAULT N'[]' CHECK (ISJSON(risk_history_json)=1),
    engagement_json nvarchar(max) NOT NULL DEFAULT N'{}' CHECK (ISJSON(engagement_json)=1),
    dependency_watch bit NOT NULL DEFAULT (0),
    protocol_history_json nvarchar(max) NOT NULL DEFAULT N'[]' CHECK (ISJSON(protocol_history_json)=1),
    proactive_opt_in bit NOT NULL DEFAULT (0),
    quiet_hours_json nvarchar(max) NULL CHECK (quiet_hours_json IS NULL OR ISJSON(quiet_hours_json)=1),
    memory_paused bit NOT NULL DEFAULT (1), -- synthetic-only: AI memory must be explicitly enabled
    consent_json nvarchar(max) NOT NULL DEFAULT N'{}' CHECK (ISJSON(consent_json)=1),
    version bigint NOT NULL DEFAULT (1)
);

CREATE TABLE dbo.ai_turn_context (
    turn_id uniqueidentifier NOT NULL PRIMARY KEY,
    user_id uniqueidentifier NOT NULL REFERENCES dbo.users(id) ON DELETE CASCADE,
    session_id uniqueidentifier NOT NULL,
    route nvarchar(20) NULL, objective nvarchar(50) NULL,
    bundle_version nvarchar(80) NULL,
    state_committed bit NOT NULL DEFAULT (1), -- record is inserted only after atomic turn/state persistence
    created_at datetime2(3) NOT NULL DEFAULT SYSUTCDATETIME()
);
CREATE INDEX IX_ai_turn_context_user ON dbo.ai_turn_context(user_id,created_at DESC);

CREATE TABLE dbo.ai_formulations (
    id uniqueidentifier NOT NULL PRIMARY KEY,
    user_id uniqueidentifier NOT NULL REFERENCES dbo.users(id) ON DELETE CASCADE,
    thread_id uniqueidentifier NOT NULL,
    version int NOT NULL,
    framework nvarchar(80) NOT NULL,
    fields_json nvarchar(max) NOT NULL CHECK (ISJSON(fields_json)=1),
    paragraph nvarchar(max) NOT NULL,
    gaps_json nvarchar(max) NOT NULL DEFAULT N'[]' CHECK (ISJSON(gaps_json)=1),
    created_at datetime2(3) NOT NULL DEFAULT SYSUTCDATETIME(),
    CONSTRAINT UQ_ai_formulations_thread_version UNIQUE(thread_id,version)
);

CREATE TABLE dbo.ai_protocol_instances (
    id uniqueidentifier NOT NULL PRIMARY KEY,
    user_id uniqueidentifier NOT NULL REFERENCES dbo.users(id) ON DELETE CASCADE,
    thread_id uniqueidentifier NULL,
    protocol_id nvarchar(100) NOT NULL, protocol_version nvarchar(40) NOT NULL,
    status nvarchar(20) NOT NULL CHECK (status IN (N'offered',N'active',N'paused',N'completed',N'stopped',N'declined')),
    step_id nvarchar(100) NULL,
    inputs_json nvarchar(max) NOT NULL DEFAULT N'{}' CHECK (ISJSON(inputs_json)=1),
    attempts int NOT NULL DEFAULT (0), stuck_count int NOT NULL DEFAULT (0),
    offered_at datetime2(3) NULL, consent_at datetime2(3) NULL,
    consent_message_id uniqueidentifier NULL,
    completed_at datetime2(3) NULL, follow_up_due_at datetime2(3) NULL,
    outcome nvarchar(100) NULL, last_user_summary nvarchar(max) NULL,
    created_at datetime2(3) NOT NULL DEFAULT SYSUTCDATETIME(),
    version bigint NOT NULL DEFAULT (1),
    CONSTRAINT CK_ai_protocol_consent CHECK (status <> N'active' OR (consent_at IS NOT NULL AND consent_message_id IS NOT NULL))
);
CREATE UNIQUE INDEX UX_ai_protocol_live ON dbo.ai_protocol_instances(user_id)
    WHERE status IN (N'offered',N'active');

CREATE TABLE dbo.ai_memory_items (
    id uniqueidentifier NOT NULL PRIMARY KEY,
    user_id uniqueidentifier NOT NULL REFERENCES dbo.users(id) ON DELETE CASCADE,
    kind nvarchar(24) NOT NULL CHECK (kind IN (N'fact',N'preference',N'episode',N'hypothesis',N'goal',N'event',N'safety_event')),
    content nvarchar(max) NULL,
    content_enc varbinary(max) NULL,
    lang nvarchar(24) NULL,
    epistemic nvarchar(24) NOT NULL CHECK (epistemic IN (N'reported',N'inferred',N'system_observed')),
    status nvarchar(24) NOT NULL DEFAULT N'active' CHECK (status IN (N'active',N'contested',N'superseded',N'rejected',N'stale',N'proposed',N'supported')),
    confidence nvarchar(12) NOT NULL DEFAULT N'low' CHECK (confidence IN (N'low',N'medium',N'high')),
    support_n int NOT NULL DEFAULT (0), contradict_n int NOT NULL DEFAULT (0), sessions_n int NOT NULL DEFAULT (0),
    thread_ids_json nvarchar(max) NOT NULL DEFAULT N'[]' CHECK (ISJSON(thread_ids_json)=1),
    entities_json nvarchar(max) NOT NULL DEFAULT N'[]' CHECK (ISJSON(entities_json)=1),
    tags_json nvarchar(max) NOT NULL DEFAULT N'[]' CHECK (ISJSON(tags_json)=1),
    sensitivity nvarchar(24) NOT NULL DEFAULT N'normal' CHECK (sensitivity IN (N'normal',N'sensitive',N'highly_sensitive')),
    importance real NOT NULL DEFAULT (0.3) CHECK (importance BETWEEN 0 AND 1),
    valid_from datetime2(3) NULL, valid_to datetime2(3) NULL,
    supersedes_id uniqueidentifier NULL,
    superseded_by_id uniqueidentifier NULL,
    cluster_id uniqueidentifier NULL,
    source_json nvarchar(max) NOT NULL CHECK (ISJSON(source_json)=1),
    created_at datetime2(3) NOT NULL DEFAULT SYSUTCDATETIME(),
    updated_at datetime2(3) NOT NULL DEFAULT SYSUTCDATETIME(),
    last_confirmed_at datetime2(3) NULL, last_used_at datetime2(3) NULL,
    CONSTRAINT CK_ai_mem_hyp CHECK (kind <> N'hypothesis' OR epistemic = N'inferred'),
    CONSTRAINT CK_ai_mem_hyp_status CHECK (kind = N'hypothesis' OR status NOT IN (N'proposed',N'supported')),
    CONSTRAINT CK_ai_mem_encrypted CHECK (content IS NOT NULL OR content_enc IS NOT NULL)
);
CREATE INDEX IX_ai_memory_user_kind ON dbo.ai_memory_items(user_id,kind,status);
CREATE INDEX IX_ai_memory_user_recent ON dbo.ai_memory_items(user_id,updated_at DESC);

CREATE TABLE dbo.ai_memory_evidence (
    user_id uniqueidentifier NOT NULL REFERENCES dbo.users(id) ON DELETE CASCADE,
    item_id uniqueidentifier NOT NULL,
    ref_kind nvarchar(12) NOT NULL CHECK (ref_kind IN (N'message',N'item',N'event')),
    ref_id uniqueidentifier NOT NULL,
    role nvarchar(12) NOT NULL CHECK (role IN (N'supports',N'contradicts',N'context')),
    session_id uniqueidentifier NULL,
    evidence_at datetime2(3) NOT NULL,
    CONSTRAINT PK_ai_memory_evidence PRIMARY KEY(item_id,ref_kind,ref_id,role)
);
CREATE INDEX IX_ai_memory_evidence_ref ON dbo.ai_memory_evidence(ref_kind,ref_id);

CREATE TABLE dbo.ai_memory_tombstones (
    id uniqueidentifier NOT NULL PRIMARY KEY,
    user_id uniqueidentifier NOT NULL REFERENCES dbo.users(id) ON DELETE CASCADE,
    deleted_at datetime2(3) NOT NULL DEFAULT SYSUTCDATETIME(),
    reason nvarchar(100) NOT NULL,
    claim_hash varbinary(32) NULL,
    suppress_until datetime2(3) NULL
);
CREATE INDEX IX_ai_memory_tombstone_user ON dbo.ai_memory_tombstones(user_id,claim_hash);

CREATE TABLE dbo.ai_hypothesis_cooldowns (
    user_id uniqueidentifier NOT NULL REFERENCES dbo.users(id) ON DELETE CASCADE,
    tag nvarchar(100) NOT NULL,
    thread_id uniqueidentifier NULL,
    until_at datetime2(3) NOT NULL,
    rejected_item_id uniqueidentifier NULL,
    CONSTRAINT UQ_ai_hyp_cooldown UNIQUE(user_id,tag,thread_id)
);

CREATE TABLE dbo.ai_open_loops (
    id uniqueidentifier NOT NULL PRIMARY KEY,
    user_id uniqueidentifier NOT NULL REFERENCES dbo.users(id) ON DELETE CASCADE,
    kind nvarchar(40) NOT NULL, text nvarchar(max) NOT NULL,
    ref_id uniqueidentifier NULL, due_at datetime2(3) NULL,
    status nvarchar(20) NOT NULL DEFAULT N'open',
    created_at datetime2(3) NOT NULL DEFAULT SYSUTCDATETIME()
);
CREATE INDEX IX_ai_loops_open ON dbo.ai_open_loops(user_id,status,due_at);

CREATE TABLE dbo.ai_follow_ups (
    id uniqueidentifier NOT NULL PRIMARY KEY,
    user_id uniqueidentifier NOT NULL REFERENCES dbo.users(id) ON DELETE CASCADE,
    reason nvarchar(40) NOT NULL, due_at datetime2(3) NOT NULL,
    ref_id uniqueidentifier NULL, status nvarchar(20) NOT NULL DEFAULT N'scheduled',
    skip_reason nvarchar(50) NULL,
    created_at datetime2(3) NOT NULL DEFAULT SYSUTCDATETIME()
);
CREATE INDEX IX_ai_followups_due ON dbo.ai_follow_ups(status,due_at);

CREATE TABLE dbo.ai_turn_events (
    seq bigint IDENTITY(1,1) NOT NULL PRIMARY KEY,
    id uniqueidentifier NOT NULL DEFAULT NEWID(),
    turn_id uniqueidentifier NOT NULL,
    user_id uniqueidentifier NOT NULL REFERENCES dbo.users(id) ON DELETE CASCADE,
    type nvarchar(80) NOT NULL, actor nvarchar(40) NOT NULL,
    delta_json nvarchar(max) NOT NULL CHECK (ISJSON(delta_json)=1),
    created_at datetime2(3) NOT NULL DEFAULT SYSUTCDATETIME()
);
CREATE INDEX IX_ai_turn_events_user ON dbo.ai_turn_events(user_id,seq DESC);

CREATE TABLE dbo.ai_jobs (
    id uniqueidentifier NOT NULL PRIMARY KEY,
    user_id uniqueidentifier NOT NULL REFERENCES dbo.users(id) ON DELETE CASCADE,
    turn_id uniqueidentifier NULL,
    kind nvarchar(50) NOT NULL, status nvarchar(20) NOT NULL DEFAULT N'queued',
    dedupe_key nvarchar(240) NULL,
    payload_json nvarchar(max) NOT NULL DEFAULT N'{}' CHECK (ISJSON(payload_json)=1),
    attempts int NOT NULL DEFAULT (0), max_attempts int NOT NULL DEFAULT (6),
    run_after datetime2(3) NOT NULL DEFAULT SYSUTCDATETIME(),
    locked_by nvarchar(100) NULL, locked_at datetime2(3) NULL,
    last_error_code nvarchar(80) NULL,
    created_at datetime2(3) NOT NULL DEFAULT SYSUTCDATETIME()
);
CREATE INDEX IX_ai_jobs_ready ON dbo.ai_jobs(status,run_after);
CREATE UNIQUE INDEX UX_ai_jobs_dedupe ON dbo.ai_jobs(user_id,kind,dedupe_key)
    WHERE dedupe_key IS NOT NULL AND status IN (N'queued',N'running');

CREATE TABLE dbo.ai_outbox (
    id uniqueidentifier NOT NULL PRIMARY KEY,
    user_id uniqueidentifier NOT NULL REFERENCES dbo.users(id) ON DELETE CASCADE,
    type nvarchar(80) NOT NULL,
    payload_json nvarchar(max) NOT NULL CHECK (ISJSON(payload_json)=1),
    created_at datetime2(3) NOT NULL DEFAULT SYSUTCDATETIME(),
    published_at datetime2(3) NULL
);

CREATE TABLE dbo.ai_turn_traces (
    turn_id uniqueidentifier NOT NULL PRIMARY KEY,
    user_id uniqueidentifier NOT NULL REFERENCES dbo.users(id) ON DELETE CASCADE,
    route nvarchar(20) NULL,
    versions_json nvarchar(max) NOT NULL CHECK (ISJSON(versions_json)=1),
    payload_enc varbinary(max) NULL,
    summary_json nvarchar(max) NOT NULL CHECK (ISJSON(summary_json)=1),
    review_flags_json nvarchar(max) NOT NULL DEFAULT N'[]' CHECK (ISJSON(review_flags_json)=1),
    sampled_for_review bit NOT NULL DEFAULT (0),
    created_at datetime2(3) NOT NULL DEFAULT SYSUTCDATETIME(),
    expires_at datetime2(3) NOT NULL
);
CREATE INDEX IX_ai_traces_expiry ON dbo.ai_turn_traces(expires_at);

CREATE TABLE dbo.ai_embeddings (
    id uniqueidentifier NOT NULL PRIMARY KEY,
    user_id uniqueidentifier NOT NULL REFERENCES dbo.users(id) ON DELETE CASCADE,
    ref_kind nvarchar(20) NOT NULL CHECK (ref_kind IN (N'message',N'memory')),
    ref_id uniqueidentifier NOT NULL,
    codec_version tinyint NOT NULL DEFAULT (1) CHECK(codec_version=1),
    model_id nvarchar(120) NOT NULL,
    dim int NOT NULL CHECK (dim BETWEEN 1 AND 8192),
    vector_bytes varbinary(max) NOT NULL,
    created_at datetime2(3) NOT NULL DEFAULT SYSUTCDATETIME(),
    CONSTRAINT CK_ai_embedding_bytes CHECK (DATALENGTH(vector_bytes) = dim * 4),
    CONSTRAINT UQ_ai_embeddings_ref_model UNIQUE(user_id,ref_kind,ref_id,model_id)
);
CREATE INDEX IX_ai_embeddings_owner ON dbo.ai_embeddings(user_id,ref_kind,ref_id);
