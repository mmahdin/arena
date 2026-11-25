-- Rebuildable hybrid search index. Original/derived data and evidence are retained.
-- Source rowversions fence updates, visibility changes and late embedding jobs.
ALTER TABLE dbo.ai_memory_items ADD search_version rowversion;
ALTER TABLE dbo.messages ADD search_version rowversion;
ALTER TABLE dbo.ai_sessions ADD search_version rowversion;
ALTER TABLE dbo.ai_threads ADD search_version rowversion;
ALTER TABLE dbo.ai_formulations ADD search_version rowversion;
ALTER TABLE dbo.ai_protocol_instances ADD search_version rowversion;
ALTER TABLE dbo.ai_open_loops ADD search_version rowversion;
ALTER TABLE dbo.ai_embeddings ADD source_version binary(8) NULL;
DECLARE @constraint sysname;
SELECT @constraint=name FROM sys.check_constraints WHERE parent_object_id=OBJECT_ID(N'dbo.ai_embeddings') AND definition LIKE N'%ref_kind%';
DECLARE @sql nvarchar(max)=N'ALTER TABLE dbo.ai_embeddings DROP CONSTRAINT '+QUOTENAME(@constraint);
EXEC sys.sp_executesql @sql;
ALTER TABLE dbo.ai_embeddings ADD CONSTRAINT CK_ai_embedding_source CHECK(ref_kind IN(N'memory',N'message',N'session',N'thread',N'formulation',N'protocol',N'open_loop'));
EXEC(N'CREATE VIEW dbo.ai_search_sources AS
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
WHERE s.deleted_at IS NULL AND NOT EXISTS(SELECT 1 FROM dbo.ai_formulations newer WHERE newer.thread_id=s.thread_id AND newer.user_id=s.user_id AND newer.version>s.version AND newer.deleted_at IS NULL)
UNION ALL
SELECT N''protocol'' source_kind,s.id,s.user_id user_id,N''episode'' memory_kind,CONCAT(s.protocol_id,NCHAR(10),s.last_user_summary,NCHAR(10),s.inputs_json) content,CONVERT(varbinary(max),NULL) content_enc,N''system_observed'' epistemic,N''active'' status,N''normal'' sensitivity,s.created_at created_at,s.created_at updated_at,CONVERT(binary(8),s.search_version) source_version,CONVERT(uniqueidentifier,NULL) conversation_id,CONVERT(bigint,NULL) message_order,1 evidence_n,CONVERT(datetime2(3),NULL) valid_from
FROM dbo.ai_protocol_instances s
WHERE 1=1
UNION ALL
SELECT N''open_loop'' source_kind,s.id,s.user_id user_id,N''episode'' memory_kind,s.text content,CONVERT(varbinary(max),NULL) content_enc,N''system_observed'' epistemic,N''active'' status,N''normal'' sensitivity,s.created_at created_at,s.created_at updated_at,CONVERT(binary(8),s.search_version) source_version,CONVERT(uniqueidentifier,NULL) conversation_id,CONVERT(bigint,NULL) message_order,1 evidence_n,CONVERT(datetime2(3),NULL) valid_from
FROM dbo.ai_open_loops s
WHERE s.status=N''open'';');
EXEC(N'ALTER TRIGGER dbo.ai_embedding_owner ON dbo.ai_embeddings AFTER INSERT,UPDATE AS BEGIN SET NOCOUNT ON; IF EXISTS(SELECT 1 FROM inserted i WHERE (i.ref_kind=N''memory'' AND NOT EXISTS(SELECT 1 FROM dbo.ai_memory_items s WHERE s.id=i.ref_id AND s.user_id=i.user_id)) OR (i.ref_kind=N''message'' AND NOT EXISTS(SELECT 1 FROM dbo.messages s JOIN dbo.conversations c ON c.id=s.conversation_id WHERE s.id=i.ref_id AND c.user_id=i.user_id)) OR (i.ref_kind=N''session'' AND NOT EXISTS(SELECT 1 FROM dbo.ai_sessions s WHERE s.id=i.ref_id AND s.user_id=i.user_id)) OR (i.ref_kind=N''thread'' AND NOT EXISTS(SELECT 1 FROM dbo.ai_threads s WHERE s.id=i.ref_id AND s.user_id=i.user_id)) OR (i.ref_kind=N''formulation'' AND NOT EXISTS(SELECT 1 FROM dbo.ai_formulations s WHERE s.id=i.ref_id AND s.user_id=i.user_id)) OR (i.ref_kind=N''protocol'' AND NOT EXISTS(SELECT 1 FROM dbo.ai_protocol_instances s WHERE s.id=i.ref_id AND s.user_id=i.user_id)) OR (i.ref_kind=N''open_loop'' AND NOT EXISTS(SELECT 1 FROM dbo.ai_open_loops s WHERE s.id=i.ref_id AND s.user_id=i.user_id))) THROW 51001,N''AI reference owner mismatch'',1; END');
EXEC(N'CREATE TRIGGER dbo.ai_memory_items_search_enqueue ON dbo.ai_memory_items AFTER INSERT,UPDATE AS BEGIN
SET NOCOUNT ON;
INSERT dbo.ai_jobs(id,user_id,kind,dedupe_key,payload_json)
SELECT NEWID(),i.user_id,N''memory_embed'',CONCAT(N''memory:'',CONVERT(nvarchar(36),i.id),N'':'',CONVERT(varchar(16),CONVERT(binary(8),i.search_version),2)),
(SELECT N''embedding_job.v2'' [schema],N''memory'' source_kind,CONVERT(varchar(36),i.id) source_id,CONVERT(varchar(16),CONVERT(binary(8),i.search_version),2) source_version FOR JSON PATH,WITHOUT_ARRAY_WRAPPER)
FROM inserted i; END');
EXEC(N'CREATE TRIGGER dbo.messages_search_enqueue ON dbo.messages AFTER INSERT,UPDATE AS BEGIN
SET NOCOUNT ON;
INSERT dbo.ai_jobs(id,user_id,kind,dedupe_key,payload_json)
SELECT NEWID(),c.user_id,N''memory_embed'',CONCAT(N''message:'',CONVERT(nvarchar(36),i.id),N'':'',CONVERT(varchar(16),CONVERT(binary(8),i.search_version),2)),
(SELECT N''embedding_job.v2'' [schema],N''message'' source_kind,CONVERT(varchar(36),i.id) source_id,CONVERT(varchar(16),CONVERT(binary(8),i.search_version),2) source_version FOR JSON PATH,WITHOUT_ARRAY_WRAPPER)
FROM inserted i JOIN dbo.conversations c ON c.id=i.conversation_id; END');
EXEC(N'CREATE TRIGGER dbo.ai_sessions_search_enqueue ON dbo.ai_sessions AFTER INSERT,UPDATE AS BEGIN
SET NOCOUNT ON;
INSERT dbo.ai_jobs(id,user_id,kind,dedupe_key,payload_json)
SELECT NEWID(),i.user_id,N''memory_embed'',CONCAT(N''session:'',CONVERT(nvarchar(36),i.id),N'':'',CONVERT(varchar(16),CONVERT(binary(8),i.search_version),2)),
(SELECT N''embedding_job.v2'' [schema],N''session'' source_kind,CONVERT(varchar(36),i.id) source_id,CONVERT(varchar(16),CONVERT(binary(8),i.search_version),2) source_version FOR JSON PATH,WITHOUT_ARRAY_WRAPPER)
FROM inserted i; END');
EXEC(N'CREATE TRIGGER dbo.ai_threads_search_enqueue ON dbo.ai_threads AFTER INSERT,UPDATE AS BEGIN
SET NOCOUNT ON;
INSERT dbo.ai_jobs(id,user_id,kind,dedupe_key,payload_json)
SELECT NEWID(),i.user_id,N''memory_embed'',CONCAT(N''thread:'',CONVERT(nvarchar(36),i.id),N'':'',CONVERT(varchar(16),CONVERT(binary(8),i.search_version),2)),
(SELECT N''embedding_job.v2'' [schema],N''thread'' source_kind,CONVERT(varchar(36),i.id) source_id,CONVERT(varchar(16),CONVERT(binary(8),i.search_version),2) source_version FOR JSON PATH,WITHOUT_ARRAY_WRAPPER)
FROM inserted i; END');
EXEC(N'CREATE TRIGGER dbo.ai_formulations_search_enqueue ON dbo.ai_formulations AFTER INSERT,UPDATE AS BEGIN
SET NOCOUNT ON;
INSERT dbo.ai_jobs(id,user_id,kind,dedupe_key,payload_json)
SELECT NEWID(),i.user_id,N''memory_embed'',CONCAT(N''formulation:'',CONVERT(nvarchar(36),i.id),N'':'',CONVERT(varchar(16),CONVERT(binary(8),i.search_version),2)),
(SELECT N''embedding_job.v2'' [schema],N''formulation'' source_kind,CONVERT(varchar(36),i.id) source_id,CONVERT(varchar(16),CONVERT(binary(8),i.search_version),2) source_version FOR JSON PATH,WITHOUT_ARRAY_WRAPPER)
FROM inserted i; END');
EXEC(N'CREATE TRIGGER dbo.ai_protocol_instances_search_enqueue ON dbo.ai_protocol_instances AFTER INSERT,UPDATE AS BEGIN
SET NOCOUNT ON;
INSERT dbo.ai_jobs(id,user_id,kind,dedupe_key,payload_json)
SELECT NEWID(),i.user_id,N''memory_embed'',CONCAT(N''protocol:'',CONVERT(nvarchar(36),i.id),N'':'',CONVERT(varchar(16),CONVERT(binary(8),i.search_version),2)),
(SELECT N''embedding_job.v2'' [schema],N''protocol'' source_kind,CONVERT(varchar(36),i.id) source_id,CONVERT(varchar(16),CONVERT(binary(8),i.search_version),2) source_version FOR JSON PATH,WITHOUT_ARRAY_WRAPPER)
FROM inserted i; END');
EXEC(N'CREATE TRIGGER dbo.ai_open_loops_search_enqueue ON dbo.ai_open_loops AFTER INSERT,UPDATE AS BEGIN
SET NOCOUNT ON;
INSERT dbo.ai_jobs(id,user_id,kind,dedupe_key,payload_json)
SELECT NEWID(),i.user_id,N''memory_embed'',CONCAT(N''open_loop:'',CONVERT(nvarchar(36),i.id),N'':'',CONVERT(varchar(16),CONVERT(binary(8),i.search_version),2)),
(SELECT N''embedding_job.v2'' [schema],N''open_loop'' source_kind,CONVERT(varchar(36),i.id) source_id,CONVERT(varchar(16),CONVERT(binary(8),i.search_version),2) source_version FOR JSON PATH,WITHOUT_ARRAY_WRAPPER)
FROM inserted i; END');
-- Legacy jobs contain vectors computed before persistence. Preserve them for
-- audit; the new worker backfills current source versions using current models.
UPDATE dbo.ai_jobs SET status=N'cancelled',last_error_code=N'index_rebuild_v2'
WHERE kind=N'memory_embed' AND status IN(N'queued',N'running') AND COALESCE(JSON_VALUE(payload_json,'$.schema'),N'')<>N'embedding_job.v2';

-- Source triggers add jobs before finalization checks for existing work. Keep
-- owner-scoped idempotency lookups off unrelated users' uncommitted job rows.
CREATE INDEX IX_ai_jobs_owner_kind_key ON dbo.ai_jobs(user_id,kind,dedupe_key) INCLUDE(status,payload_json);
