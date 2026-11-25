-- SQL Server 2022. Additive background review accounting; the runner owns atomicity.
ALTER TABLE dbo.ai_user_state ADD memory_review_checked_at datetime2(3) NULL;
CREATE TABLE dbo.ai_memory_review_messages (
 user_id uniqueidentifier NOT NULL REFERENCES dbo.users(id),
 message_id uniqueidentifier NOT NULL REFERENCES dbo.messages(id),
 job_id uniqueidentifier NOT NULL,
 source_version binary(8) NOT NULL,
 reviewed_by_job_id uniqueidentifier NULL,
 reviewed_at datetime2(3) NULL,
 created_at datetime2(3) NOT NULL DEFAULT SYSUTCDATETIME(),
 CONSTRAINT PK_ai_memory_review_messages PRIMARY KEY(user_id,message_id),
 CONSTRAINT FK_ai_memory_review_job FOREIGN KEY(job_id,user_id) REFERENCES dbo.ai_jobs(id,user_id),
 CONSTRAINT FK_ai_memory_review_result FOREIGN KEY(reviewed_by_job_id,user_id) REFERENCES dbo.ai_jobs(id,user_id)
);
CREATE INDEX IX_ai_memory_review_job ON dbo.ai_memory_review_messages(job_id,user_id,reviewed_at);
EXEC(N'CREATE TRIGGER dbo.ai_memory_review_owner ON dbo.ai_memory_review_messages AFTER INSERT,UPDATE AS BEGIN
 IF EXISTS(SELECT 1 FROM inserted i JOIN dbo.messages m ON m.id=i.message_id JOIN dbo.conversations v ON v.id=m.conversation_id WHERE v.user_id<>i.user_id OR m.role<>N''user'') THROW 51020,N''Review ownership invalid'',1;
 END');
EXEC(N'CREATE TRIGGER dbo.ai_memory_review_preserve ON dbo.ai_memory_review_messages AFTER DELETE AS BEGIN IF EXISTS(SELECT 1 FROM deleted) THROW 51020,N''Soft deletion only'',1; END');
CREATE TABLE dbo.ai_background_hints (
 id uniqueidentifier NOT NULL PRIMARY KEY,
 user_id uniqueidentifier NOT NULL,
 job_id uniqueidentifier NOT NULL,
 thread_id uniqueidentifier NULL,
 hint_enc varbinary(max) NOT NULL,
 sources_json nvarchar(max) NOT NULL CHECK(ISJSON(sources_json)=1),
 ttl_turns int NOT NULL CHECK(ttl_turns BETWEEN 1 AND 3),
 expires_at datetime2(3) NOT NULL,
 invalidated_at datetime2(3) NULL,
 created_at datetime2(3) NOT NULL DEFAULT SYSUTCDATETIME(),
 CONSTRAINT FK_ai_background_hint_job FOREIGN KEY(job_id,user_id) REFERENCES dbo.ai_jobs(id,user_id),
 CONSTRAINT FK_ai_background_hint_thread FOREIGN KEY(thread_id,user_id) REFERENCES dbo.ai_threads(id,user_id)
);
CREATE INDEX IX_ai_background_hints_owner ON dbo.ai_background_hints(user_id,expires_at,created_at);
EXEC(N'CREATE TRIGGER dbo.ai_background_hints_preserve ON dbo.ai_background_hints AFTER DELETE AS BEGIN IF EXISTS(SELECT 1 FROM deleted) THROW 51020,N''Soft deletion only'',1; END');
-- Reuse the existing formulation and reflection records for explicitly scoped analysis.
ALTER TABLE dbo.ai_formulations DROP CONSTRAINT UQ_ai_formulations_thread_version;
ALTER TABLE dbo.ai_formulations ALTER COLUMN thread_id uniqueidentifier NULL;
CREATE UNIQUE INDEX UQ_ai_formulations_thread_version ON dbo.ai_formulations(thread_id,version) WHERE thread_id IS NOT NULL;
CREATE UNIQUE INDEX UQ_ai_formulations_user_version ON dbo.ai_formulations(user_id,version) WHERE thread_id IS NULL;
ALTER TABLE dbo.ai_formulations ADD analysis_status nvarchar(16) NOT NULL DEFAULT N'active', scope_json nvarchar(max) NULL, sources_json nvarchar(max) NULL;
ALTER TABLE dbo.ai_reflection_runs ALTER COLUMN thread_id uniqueidentifier NULL;
ALTER TABLE dbo.ai_reflection_runs ALTER COLUMN session_id uniqueidentifier NULL;

EXEC(N'ALTER VIEW dbo.ai_search_sources AS
SELECT N''memory'' source_kind,s.id,s.user_id user_id,s.kind memory_kind,s.content content,s.content_enc content_enc,s.epistemic epistemic,s.status status,s.sensitivity sensitivity,s.created_at created_at,s.updated_at updated_at,CONVERT(binary(8),s.search_version) source_version,CONVERT(uniqueidentifier,NULL) conversation_id,CONVERT(bigint,NULL) message_order,(SELECT COUNT(*) FROM dbo.ai_memory_evidence e WHERE e.user_id=s.user_id AND e.item_id=s.id) evidence_n,s.valid_from valid_from
FROM dbo.ai_memory_items s
WHERE s.deleted_at IS NULL AND s.status IN(N''active'',N''proposed'',N''supported'')
UNION ALL
SELECT N''message'' source_kind,s.id,v.user_id user_id,N''episode'' memory_kind,s.content content,CONVERT(varbinary(max),NULL) content_enc,CASE WHEN s.role=N''user'' THEN N''reported'' ELSE N''system_observed'' END epistemic,N''active'' status,N''normal'' sensitivity,s.created_at created_at,s.created_at updated_at,CONVERT(binary(8),s.search_version) source_version,CONVERT(uniqueidentifier,s.conversation_id) conversation_id,CONVERT(bigint,s.message_order) message_order,1 evidence_n,CONVERT(datetime2(3),NULL) valid_from
FROM dbo.messages s JOIN dbo.conversations v ON v.id=s.conversation_id
WHERE s.deleted_at IS NULL AND v.deleted_at IS NULL AND s.role IN(N''user'',N''assistant'')
UNION ALL
SELECT N''session'' source_kind,s.id,s.user_id user_id,N''episode'' memory_kind,s.summary content,CONVERT(varbinary(max),NULL) content_enc,N''system_observed'' epistemic,N''active'' status,N''normal'' sensitivity,s.started_at created_at,s.last_activity_at updated_at,CONVERT(binary(8),s.search_version) source_version,CONVERT(uniqueidentifier,s.conversation_id) conversation_id,CONVERT(bigint,NULL) message_order,1 evidence_n,CONVERT(datetime2(3),NULL) valid_from
FROM dbo.ai_sessions s JOIN dbo.conversations v ON v.id=s.conversation_id AND v.user_id=s.user_id
WHERE v.deleted_at IS NULL AND LEN(s.summary)>0
UNION ALL
SELECT N''thread'' source_kind,s.id,s.user_id user_id,N''episode'' memory_kind,CONCAT(s.topic,NCHAR(10),s.goal,NCHAR(10),s.slots_json) content,CONVERT(varbinary(max),NULL) content_enc,N''system_observed'' epistemic,N''active'' status,N''normal'' sensitivity,s.created_at created_at,s.last_active_at updated_at,CONVERT(binary(8),s.search_version) source_version,CONVERT(uniqueidentifier,NULL) conversation_id,CONVERT(bigint,NULL) message_order,1 evidence_n,CONVERT(datetime2(3),NULL) valid_from
FROM dbo.ai_threads s
WHERE 1=1
UNION ALL
SELECT N''formulation'' source_kind,s.id,s.user_id user_id,N''episode'' memory_kind,CONCAT(s.paragraph,NCHAR(10),s.fields_json) content,CONVERT(varbinary(max),NULL) content_enc,N''inferred'' epistemic,N''active'' status,N''normal'' sensitivity,s.created_at created_at,s.created_at updated_at,CONVERT(binary(8),s.search_version) source_version,CONVERT(uniqueidentifier,NULL) conversation_id,CONVERT(bigint,NULL) message_order,1 evidence_n,CONVERT(datetime2(3),NULL) valid_from
FROM dbo.ai_formulations s
WHERE s.deleted_at IS NULL AND s.analysis_status=N''active'' AND NOT EXISTS(SELECT 1 FROM dbo.ai_formulations newer WHERE (newer.thread_id=s.thread_id OR newer.thread_id IS NULL AND s.thread_id IS NULL) AND newer.user_id=s.user_id AND (s.scope_json IS NULL OR newer.framework=s.framework) AND newer.version>s.version AND newer.deleted_at IS NULL)
UNION ALL
SELECT N''protocol'' source_kind,s.id,s.user_id user_id,N''episode'' memory_kind,CONCAT(s.protocol_id,NCHAR(10),s.last_user_summary,NCHAR(10),s.inputs_json) content,CONVERT(varbinary(max),NULL) content_enc,N''system_observed'' epistemic,N''active'' status,N''normal'' sensitivity,s.created_at created_at,s.created_at updated_at,CONVERT(binary(8),s.search_version) source_version,CONVERT(uniqueidentifier,NULL) conversation_id,CONVERT(bigint,NULL) message_order,1 evidence_n,CONVERT(datetime2(3),NULL) valid_from
FROM dbo.ai_protocol_instances s
WHERE 1=1
UNION ALL
SELECT N''open_loop'' source_kind,s.id,s.user_id user_id,N''episode'' memory_kind,s.text content,CONVERT(varbinary(max),NULL) content_enc,N''system_observed'' epistemic,N''active'' status,N''normal'' sensitivity,s.created_at created_at,s.created_at updated_at,CONVERT(binary(8),s.search_version) source_version,CONVERT(uniqueidentifier,NULL) conversation_id,CONVERT(bigint,NULL) message_order,1 evidence_n,CONVERT(datetime2(3),NULL) valid_from
FROM dbo.ai_open_loops s
WHERE s.status=N''open'';');
