-- WP-12 additive memory revisions, receipts and leased job checkpoints.
-- Existing data, provenance, consent and visibility remain intact.
ALTER TABLE dbo.ai_memory_items ADD version bigint NOT NULL CONSTRAINT ai_memory_version_default DEFAULT(1), pinned bit NOT NULL CONSTRAINT ai_memory_pinned_default DEFAULT(0);
EXEC(N'ALTER TABLE dbo.ai_memory_items ADD CONSTRAINT ai_memory_version_positive CHECK(version>0)');
ALTER TABLE dbo.ai_user_state ADD memory_revision bigint NOT NULL CONSTRAINT ai_memory_revision_default DEFAULT(1);
-- Explicit memory controls can occur outside a conversational turn. User
-- ownership remains mandatory; non-null turn references retain the owner FK.
ALTER TABLE dbo.ai_turn_events ALTER COLUMN turn_id uniqueidentifier NULL;
CREATE UNIQUE INDEX ai_memory_event_identity ON dbo.ai_turn_events(id,user_id);
ALTER TABLE dbo.ai_jobs ADD memory_lease_token uniqueidentifier NULL, memory_lease_until datetime2(3) NULL, memory_result_enc varbinary(max) NULL;
CREATE INDEX ai_memory_jobs_ready ON dbo.ai_jobs(kind,status,run_after,memory_lease_until);

CREATE TABLE dbo.ai_memory_versions (
 user_id uniqueidentifier NOT NULL,
 item_id uniqueidentifier NOT NULL,
 version bigint NOT NULL,
 snapshot_json nvarchar(max) NOT NULL CHECK(ISJSON(snapshot_json)=1),
 content_enc varbinary(max) NULL,
 vector_model nvarchar(120) NULL,
 vector_bytes varbinary(max) NULL,
 created_at datetime2(3) NOT NULL DEFAULT SYSUTCDATETIME(),
 CONSTRAINT ai_memory_versions_pk PRIMARY KEY(item_id,version),
 CONSTRAINT ai_memory_versions_owner FOREIGN KEY(item_id,user_id) REFERENCES dbo.ai_memory_items(id,user_id) ON DELETE NO ACTION
);
CREATE TABLE dbo.ai_memory_batches (
 user_id uniqueidentifier NOT NULL REFERENCES dbo.users(id) ON DELETE NO ACTION,
 operation_id uniqueidentifier NOT NULL,
 request_hash binary(32) NOT NULL,
 result_json nvarchar(max) NOT NULL CHECK(ISJSON(result_json)=1),
 created_at datetime2(3) NOT NULL DEFAULT SYSUTCDATETIME(),
 CONSTRAINT ai_memory_batches_pk PRIMARY KEY(user_id,operation_id)
);
CREATE TABLE dbo.ai_memory_ops (
 user_id uniqueidentifier NOT NULL REFERENCES dbo.users(id) ON DELETE NO ACTION,
 scope_id uniqueidentifier NOT NULL,
 op_hash binary(32) NOT NULL,
 result_json nvarchar(max) NOT NULL CHECK(ISJSON(result_json)=1),
 created_at datetime2(3) NOT NULL DEFAULT SYSUTCDATETIME(),
 CONSTRAINT ai_memory_ops_pk PRIMARY KEY(user_id,scope_id,op_hash)
);
EXEC(N'CREATE TRIGGER dbo.ai_memory_versions_preserve_delete_guard ON dbo.ai_memory_versions AFTER DELETE AS BEGIN IF EXISTS(SELECT 1 FROM deleted) THROW 51020, N''Soft deletion only'', 1; END');
CREATE TABLE dbo.ai_memory_feedback (
 user_id uniqueidentifier NOT NULL,
 turn_id uniqueidentifier NOT NULL,
 item_id uniqueidentifier NOT NULL,
 referenced bit NOT NULL,
 created_at datetime2(3) NOT NULL DEFAULT SYSUTCDATETIME(),
 CONSTRAINT ai_memory_feedback_pk PRIMARY KEY(turn_id,item_id),
 CONSTRAINT ai_memory_feedback_turn_owner FOREIGN KEY(turn_id,user_id) REFERENCES dbo.turns(id,user_id) ON DELETE NO ACTION,
 CONSTRAINT ai_memory_feedback_item_owner FOREIGN KEY(item_id,user_id) REFERENCES dbo.ai_memory_items(id,user_id) ON DELETE NO ACTION
);
CREATE INDEX ai_memory_feedback_item ON dbo.ai_memory_feedback(user_id,item_id,created_at);
EXEC(N'CREATE TRIGGER dbo.ai_memory_feedback_preserve_delete_guard ON dbo.ai_memory_feedback AFTER DELETE AS BEGIN IF EXISTS(SELECT 1 FROM deleted) THROW 51020, N''Soft deletion only'', 1; END');
EXEC(N'CREATE TRIGGER dbo.ai_memory_batches_preserve_delete_guard ON dbo.ai_memory_batches AFTER DELETE AS BEGIN IF EXISTS(SELECT 1 FROM deleted) THROW 51020, N''Soft deletion only'', 1; END');
EXEC(N'CREATE TRIGGER dbo.ai_memory_ops_preserve_delete_guard ON dbo.ai_memory_ops AFTER DELETE AS BEGIN IF EXISTS(SELECT 1 FROM deleted) THROW 51020, N''Soft deletion only'', 1; END');
