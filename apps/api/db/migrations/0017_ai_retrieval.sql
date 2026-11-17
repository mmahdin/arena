-- WP-09 Retrieval support: additive SQL Server 2022 schema for the AI Core
-- retrieval read model (Contract Pack §7.2/§7.3).
--
-- Scope discipline:
--   * This migration is additive only. It creates the ai_* tables the retrieval
--     strategies read and the soft-delete / provenance columns they enforce.
--   * It never drops or rewrites data. Deletion is a flag (deleted_at), never a
--     row removal: the product rule is "never hard-delete".
--   * Existing applied migrations 0001–0016 are untouched.
--   * No PostgreSQL dialect: uniqueidentifier, nvarchar, datetime2(3),
--     SYSUTCDATETIME(), ISJSON(), filtered indexes only.
CREATE TABLE dbo.ai_memory_items (
    id uniqueidentifier NOT NULL PRIMARY KEY,
    user_id uniqueidentifier NOT NULL,
    kind nvarchar(24) NOT NULL,
    content nvarchar(max) NULL,
    lang nvarchar(24) NULL,
    epistemic nvarchar(24) NOT NULL,
    status nvarchar(24) NOT NULL CONSTRAINT ai_memory_items_status_default DEFAULT (N'active'),
    confidence nvarchar(12) NOT NULL CONSTRAINT ai_memory_items_confidence_default DEFAULT (N'low'),
    support_n int NOT NULL CONSTRAINT ai_memory_items_support_default DEFAULT (0),
    contradict_n int NOT NULL CONSTRAINT ai_memory_items_contradict_default DEFAULT (0),
    sessions_n int NOT NULL CONSTRAINT ai_memory_items_sessions_default DEFAULT (0),
    thread_ids_json nvarchar(max) NOT NULL CONSTRAINT ai_memory_items_threads_default DEFAULT (N'[]'),
    entities_json nvarchar(max) NOT NULL CONSTRAINT ai_memory_items_entities_default DEFAULT (N'[]'),
    tags_json nvarchar(max) NOT NULL CONSTRAINT ai_memory_items_tags_default DEFAULT (N'[]'),
    sensitivity nvarchar(24) NOT NULL CONSTRAINT ai_memory_items_sensitivity_default DEFAULT (N'normal'),
    importance real NOT NULL CONSTRAINT ai_memory_items_importance_default DEFAULT (0.3),
    valid_from datetime2(3) NULL,
    valid_to datetime2(3) NULL,
    supersedes_id uniqueidentifier NULL,
    superseded_by_id uniqueidentifier NULL,
    cluster_id uniqueidentifier NULL,
    source_json nvarchar(max) NOT NULL CONSTRAINT ai_memory_items_source_default DEFAULT (N'{}'),
    created_at datetime2(3) NOT NULL CONSTRAINT ai_memory_items_created_default DEFAULT SYSUTCDATETIME(),
    updated_at datetime2(3) NOT NULL CONSTRAINT ai_memory_items_updated_default DEFAULT SYSUTCDATETIME(),
    last_confirmed_at datetime2(3) NULL,
    last_used_at datetime2(3) NULL,
    -- Soft-delete policy: rows are flagged, never removed. Retrieval excludes them.
    deleted_at datetime2(3) NULL,
    deleted_reason nvarchar(100) NULL,
    CONSTRAINT ai_memory_items_kind_check CHECK (kind IN (N'fact',N'preference',N'episode',N'hypothesis',N'goal',N'event',N'safety_event')),
    CONSTRAINT ai_memory_items_epistemic_check CHECK (epistemic IN (N'reported',N'inferred',N'system_observed')),
    CONSTRAINT ai_memory_items_status_check CHECK (status IN (N'active',N'contested',N'superseded',N'rejected',N'stale',N'proposed',N'supported')),
    CONSTRAINT ai_memory_items_sensitivity_check CHECK (sensitivity IN (N'normal',N'sensitive',N'highly_sensitive')),
    CONSTRAINT ai_memory_items_importance_check CHECK (importance >= 0 AND importance <= 1),
    CONSTRAINT ai_memory_items_hyp_epistemic_check CHECK (kind <> N'hypothesis' OR epistemic = N'inferred'),
    CONSTRAINT ai_memory_items_hyp_status_check CHECK (kind = N'hypothesis' OR status NOT IN (N'proposed',N'supported')),
    CONSTRAINT ai_memory_items_content_check CHECK (content IS NOT NULL OR deleted_at IS NOT NULL),
    CONSTRAINT ai_memory_items_owner_fk FOREIGN KEY (user_id) REFERENCES dbo.users(id) ON DELETE CASCADE,
    CONSTRAINT ai_memory_items_threads_json_check CHECK (ISJSON(thread_ids_json) = 1),
    CONSTRAINT ai_memory_items_entities_json_check CHECK (ISJSON(entities_json) = 1),
    CONSTRAINT ai_memory_items_tags_json_check CHECK (ISJSON(tags_json) = 1),
    CONSTRAINT ai_memory_items_source_json_check CHECK (ISJSON(source_json) = 1)
);
CREATE INDEX ai_memory_items_user_kind_status_idx ON dbo.ai_memory_items(user_id, kind, status);
CREATE INDEX ai_memory_items_user_updated_idx ON dbo.ai_memory_items(user_id, updated_at DESC);
CREATE INDEX ai_memory_items_user_live_idx ON dbo.ai_memory_items(user_id, updated_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX ai_memory_items_cluster_idx ON dbo.ai_memory_items(cluster_id) WHERE cluster_id IS NOT NULL;

CREATE TABLE dbo.ai_memory_evidence (
    item_id uniqueidentifier NOT NULL,
    ref_kind nvarchar(12) NOT NULL,
    ref_id uniqueidentifier NOT NULL,
    role nvarchar(12) NOT NULL,
    session_id uniqueidentifier NULL,
    evidence_at datetime2(3) NOT NULL CONSTRAINT ai_memory_evidence_at_default DEFAULT SYSUTCDATETIME(),
    CONSTRAINT ai_memory_evidence_pk PRIMARY KEY (item_id, ref_kind, ref_id, role),
    CONSTRAINT ai_memory_evidence_ref_kind_check CHECK (ref_kind IN (N'message',N'item',N'event')),
    CONSTRAINT ai_memory_evidence_role_check CHECK (role IN (N'supports',N'contradicts',N'context')),
    CONSTRAINT ai_memory_evidence_item_fk FOREIGN KEY (item_id) REFERENCES dbo.ai_memory_items(id) ON DELETE CASCADE
);
CREATE INDEX ai_memory_evidence_ref_idx ON dbo.ai_memory_evidence(ref_kind, ref_id);

CREATE TABLE dbo.ai_memory_tombstones (
    id uniqueidentifier NOT NULL PRIMARY KEY,
    user_id uniqueidentifier NOT NULL,
    deleted_at datetime2(3) NOT NULL CONSTRAINT ai_memory_tombstones_at_default DEFAULT SYSUTCDATETIME(),
    reason nvarchar(100) NOT NULL,
    claim_hash varbinary(32) NULL,
    suppress_until datetime2(3) NULL,
    CONSTRAINT ai_memory_tombstones_owner_fk FOREIGN KEY (user_id) REFERENCES dbo.users(id) ON DELETE CASCADE
);
CREATE INDEX ai_memory_tombstones_user_idx ON dbo.ai_memory_tombstones(user_id, claim_hash);

CREATE TABLE dbo.ai_embeddings (
    id uniqueidentifier NOT NULL PRIMARY KEY,
    user_id uniqueidentifier NOT NULL,
    ref_kind nvarchar(20) NOT NULL,
    ref_id uniqueidentifier NOT NULL,
    model_id nvarchar(120) NOT NULL,
    dim int NOT NULL,
    vector_bytes varbinary(max) NOT NULL,
    created_at datetime2(3) NOT NULL CONSTRAINT ai_embeddings_created_default DEFAULT SYSUTCDATETIME(),
    CONSTRAINT ai_embeddings_ref_kind_check CHECK (ref_kind IN (N'message',N'memory')),
    CONSTRAINT ai_embeddings_dim_check CHECK (dim BETWEEN 1 AND 8192),
    CONSTRAINT ai_embeddings_bytes_check CHECK (DATALENGTH(vector_bytes) = dim * 4),
    CONSTRAINT ai_embeddings_ref_model_unique UNIQUE (user_id, ref_kind, ref_id, model_id),
    CONSTRAINT ai_embeddings_owner_fk FOREIGN KEY (user_id) REFERENCES dbo.users(id) ON DELETE CASCADE
);
CREATE INDEX ai_embeddings_owner_idx ON dbo.ai_embeddings(user_id, ref_kind, ref_id);

CREATE TABLE dbo.ai_threads (
    id uniqueidentifier NOT NULL PRIMARY KEY,
    user_id uniqueidentifier NOT NULL,
    topic nvarchar(400) NOT NULL,
    phase nvarchar(40) NOT NULL CONSTRAINT ai_threads_phase_default DEFAULT (N'exploring'),
    prev_phase nvarchar(40) NULL,
    priority real NOT NULL CONSTRAINT ai_threads_priority_default DEFAULT (0.5),
    slots_json nvarchar(max) NOT NULL CONSTRAINT ai_threads_slots_default DEFAULT (N'{}'),
    goal nvarchar(max) NULL,
    hypothesis_ids_json nvarchar(max) NOT NULL CONSTRAINT ai_threads_hyp_default DEFAULT (N'[]'),
    related_thread_ids_json nvarchar(max) NOT NULL CONSTRAINT ai_threads_related_default DEFAULT (N'[]'),
    formulation_version int NOT NULL CONSTRAINT ai_threads_formulation_default DEFAULT (0),
    problem_tags_json nvarchar(max) NOT NULL CONSTRAINT ai_threads_tags_default DEFAULT (N'[]'),
    created_at datetime2(3) NOT NULL CONSTRAINT ai_threads_created_default DEFAULT SYSUTCDATETIME(),
    last_active_at datetime2(3) NOT NULL CONSTRAINT ai_threads_active_default DEFAULT SYSUTCDATETIME(),
    deleted_at datetime2(3) NULL,
    version bigint NOT NULL CONSTRAINT ai_threads_version_default DEFAULT (1),
    CONSTRAINT ai_threads_phase_check CHECK (phase IN (N'exploring',N'understanding_sufficient',N'working',N'maintaining',N'resolved',N'paused',N'dormant')),
    CONSTRAINT ai_threads_slots_json_check CHECK (ISJSON(slots_json) = 1),
    CONSTRAINT ai_threads_hyp_json_check CHECK (ISJSON(hypothesis_ids_json) = 1),
    CONSTRAINT ai_threads_related_json_check CHECK (ISJSON(related_thread_ids_json) = 1),
    CONSTRAINT ai_threads_tags_json_check CHECK (ISJSON(problem_tags_json) = 1),
    CONSTRAINT ai_threads_owner_fk FOREIGN KEY (user_id) REFERENCES dbo.users(id) ON DELETE CASCADE
);
CREATE INDEX ai_threads_user_active_idx ON dbo.ai_threads(user_id, last_active_at DESC);

-- Owner-scoped locator for soft-deleted or removed raw messages. dbo.messages has
-- no deleted_at column (0001 is applied and immutable), so a deletion is recorded
-- here instead of deleting the row: the message stays for provenance, retrieval
-- excludes it. No FK to dbo.messages so a removed conversation cannot cascade
-- away the suppression record.
CREATE TABLE dbo.ai_message_suppressions (
    message_id uniqueidentifier NOT NULL PRIMARY KEY,
    user_id uniqueidentifier NOT NULL,
    suppressed_at datetime2(3) NOT NULL CONSTRAINT ai_message_suppressions_at_default DEFAULT SYSUTCDATETIME(),
    reason nvarchar(100) NOT NULL,
    CONSTRAINT ai_message_suppressions_owner_fk FOREIGN KEY (user_id) REFERENCES dbo.users(id) ON DELETE CASCADE
);
CREATE INDEX ai_message_suppressions_user_idx ON dbo.ai_message_suppressions(user_id, suppressed_at DESC);
