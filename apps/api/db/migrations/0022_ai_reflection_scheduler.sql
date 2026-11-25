-- WP-14: retained reflection receipts and timer delivery metadata.
-- Existing memories, formulations, sources and jobs remain authoritative.
ALTER TABLE dbo.ai_jobs ADD CONSTRAINT ai_jobs_owner_unique UNIQUE(id,user_id);
CREATE TABLE dbo.ai_reflection_runs (
 id uniqueidentifier NOT NULL PRIMARY KEY,
 user_id uniqueidentifier NOT NULL,
 thread_id uniqueidentifier NOT NULL,
 session_id uniqueidentifier NOT NULL,
 job_id uniqueidentifier NOT NULL,
 trigger_kind nvarchar(40) NOT NULL,
 thread_version bigint NOT NULL,
 formulation_version int NULL,
 result_enc varbinary(max) NOT NULL,
 provenance_json nvarchar(max) NOT NULL CHECK(ISJSON(provenance_json)=1),
 created_at datetime2(3) NOT NULL DEFAULT SYSUTCDATETIME(),
 deleted_at datetime2(3) NULL,
 deleted_by uniqueidentifier NULL,
 deletion_reason nvarchar(100) NULL,
 CONSTRAINT FK_ai_reflection_runs_user FOREIGN KEY(user_id) REFERENCES dbo.users(id) ON DELETE NO ACTION,
 CONSTRAINT FK_ai_reflection_runs_thread FOREIGN KEY(thread_id,user_id) REFERENCES dbo.ai_threads(id,user_id) ON DELETE NO ACTION,
 CONSTRAINT FK_ai_reflection_runs_session FOREIGN KEY(session_id,user_id) REFERENCES dbo.ai_sessions(id,user_id) ON DELETE NO ACTION,
 CONSTRAINT FK_ai_reflection_runs_job FOREIGN KEY(job_id,user_id) REFERENCES dbo.ai_jobs(id,user_id) ON DELETE NO ACTION,
 CONSTRAINT UQ_ai_reflection_runs_job UNIQUE(job_id,user_id)
);
CREATE INDEX IX_ai_reflection_runs_thread ON dbo.ai_reflection_runs(user_id,thread_id,created_at DESC);
EXEC(N'CREATE TRIGGER dbo.ai_reflection_runs_preserve_delete_guard ON dbo.ai_reflection_runs INSTEAD OF DELETE AS THROW 51000, ''Physical deletion is not permitted'', 1;');
ALTER TABLE dbo.ai_follow_ups ADD dispatched_at datetime2(3) NULL;
CREATE INDEX IX_ai_follow_ups_dispatch ON dbo.ai_follow_ups(user_id,dispatched_at);
