-- Correlation only: source payloads, revisions, eligibility and scheduling stay intact.
-- Session state is accepted ONLY in the transaction that installed it, for its
-- owning user. A pooled connection, aborted transaction or later transaction cannot reuse it.
EXEC(N'CREATE FUNCTION dbo.ai_trace_origin_turn(@user uniqueidentifier)
RETURNS uniqueidentifier AS BEGIN
 DECLARE @context nvarchar(4000)=TRY_CONVERT(nvarchar(4000),SESSION_CONTEXT(N''theia_trace_origin''));
 IF COALESCE(ISJSON(@context),0)<>1 RETURN NULL;
 IF COALESCE(TRY_CONVERT(bigint,JSON_VALUE(@context,N''$.transaction_id'')),-1)<>CURRENT_TRANSACTION_ID() RETURN NULL;
 IF COALESCE(TRY_CONVERT(uniqueidentifier,JSON_VALUE(@context,N''$.user_id'')),CONVERT(uniqueidentifier,N''00000000-0000-0000-0000-000000000000''))<>@user RETURN NULL;
 RETURN (SELECT id FROM dbo.turns WHERE id=TRY_CONVERT(uniqueidentifier,JSON_VALUE(@context,N''$.turn_id'')) AND user_id=@user);
END');
EXEC(N'ALTER TRIGGER dbo.ai_memory_items_search_enqueue ON dbo.ai_memory_items AFTER INSERT,UPDATE AS BEGIN
SET NOCOUNT ON;
INSERT dbo.ai_jobs(id,user_id,turn_id,kind,dedupe_key,payload_json)
SELECT NEWID(),i.user_id,dbo.ai_trace_origin_turn(i.user_id),N''memory_embed'',CONCAT(N''memory:'',CONVERT(nvarchar(36),i.id),N'':'',CONVERT(varchar(16),CONVERT(binary(8),i.search_version),2)),
(SELECT N''embedding_job.v2'' [schema],N''memory'' source_kind,CONVERT(varchar(36),i.id) source_id,CONVERT(varchar(16),CONVERT(binary(8),i.search_version),2) source_version FOR JSON PATH,WITHOUT_ARRAY_WRAPPER)
FROM inserted i; END');
EXEC(N'ALTER TRIGGER dbo.messages_search_enqueue ON dbo.messages AFTER INSERT,UPDATE AS BEGIN
SET NOCOUNT ON;
INSERT dbo.ai_jobs(id,user_id,turn_id,kind,dedupe_key,payload_json)
SELECT NEWID(),c.user_id,i.turn_id,N''memory_embed'',CONCAT(N''message:'',CONVERT(nvarchar(36),i.id),N'':'',CONVERT(varchar(16),CONVERT(binary(8),i.search_version),2)),
(SELECT N''embedding_job.v2'' [schema],N''message'' source_kind,CONVERT(varchar(36),i.id) source_id,CONVERT(varchar(16),CONVERT(binary(8),i.search_version),2) source_version FOR JSON PATH,WITHOUT_ARRAY_WRAPPER)
FROM inserted i JOIN dbo.conversations c ON c.id=i.conversation_id; END');
EXEC(N'ALTER TRIGGER dbo.ai_sessions_search_enqueue ON dbo.ai_sessions AFTER INSERT,UPDATE AS BEGIN
SET NOCOUNT ON;
INSERT dbo.ai_jobs(id,user_id,turn_id,kind,dedupe_key,payload_json)
SELECT NEWID(),i.user_id,dbo.ai_trace_origin_turn(i.user_id),N''memory_embed'',CONCAT(N''session:'',CONVERT(nvarchar(36),i.id),N'':'',CONVERT(varchar(16),CONVERT(binary(8),i.search_version),2)),
(SELECT N''embedding_job.v2'' [schema],N''session'' source_kind,CONVERT(varchar(36),i.id) source_id,CONVERT(varchar(16),CONVERT(binary(8),i.search_version),2) source_version FOR JSON PATH,WITHOUT_ARRAY_WRAPPER)
FROM inserted i; END');
EXEC(N'ALTER TRIGGER dbo.ai_threads_search_enqueue ON dbo.ai_threads AFTER INSERT,UPDATE AS BEGIN
SET NOCOUNT ON;
INSERT dbo.ai_jobs(id,user_id,turn_id,kind,dedupe_key,payload_json)
SELECT NEWID(),i.user_id,dbo.ai_trace_origin_turn(i.user_id),N''memory_embed'',CONCAT(N''thread:'',CONVERT(nvarchar(36),i.id),N'':'',CONVERT(varchar(16),CONVERT(binary(8),i.search_version),2)),
(SELECT N''embedding_job.v2'' [schema],N''thread'' source_kind,CONVERT(varchar(36),i.id) source_id,CONVERT(varchar(16),CONVERT(binary(8),i.search_version),2) source_version FOR JSON PATH,WITHOUT_ARRAY_WRAPPER)
FROM inserted i; END');
EXEC(N'ALTER TRIGGER dbo.ai_formulations_search_enqueue ON dbo.ai_formulations AFTER INSERT,UPDATE AS BEGIN
SET NOCOUNT ON;
INSERT dbo.ai_jobs(id,user_id,turn_id,kind,dedupe_key,payload_json)
SELECT NEWID(),i.user_id,dbo.ai_trace_origin_turn(i.user_id),N''memory_embed'',CONCAT(N''formulation:'',CONVERT(nvarchar(36),i.id),N'':'',CONVERT(varchar(16),CONVERT(binary(8),i.search_version),2)),
(SELECT N''embedding_job.v2'' [schema],N''formulation'' source_kind,CONVERT(varchar(36),i.id) source_id,CONVERT(varchar(16),CONVERT(binary(8),i.search_version),2) source_version FOR JSON PATH,WITHOUT_ARRAY_WRAPPER)
FROM inserted i; END');
EXEC(N'ALTER TRIGGER dbo.ai_protocol_instances_search_enqueue ON dbo.ai_protocol_instances AFTER INSERT,UPDATE AS BEGIN
SET NOCOUNT ON;
INSERT dbo.ai_jobs(id,user_id,turn_id,kind,dedupe_key,payload_json)
SELECT NEWID(),i.user_id,dbo.ai_trace_origin_turn(i.user_id),N''memory_embed'',CONCAT(N''protocol:'',CONVERT(nvarchar(36),i.id),N'':'',CONVERT(varchar(16),CONVERT(binary(8),i.search_version),2)),
(SELECT N''embedding_job.v2'' [schema],N''protocol'' source_kind,CONVERT(varchar(36),i.id) source_id,CONVERT(varchar(16),CONVERT(binary(8),i.search_version),2) source_version FOR JSON PATH,WITHOUT_ARRAY_WRAPPER)
FROM inserted i; END');
EXEC(N'ALTER TRIGGER dbo.ai_open_loops_search_enqueue ON dbo.ai_open_loops AFTER INSERT,UPDATE AS BEGIN
SET NOCOUNT ON;
INSERT dbo.ai_jobs(id,user_id,turn_id,kind,dedupe_key,payload_json)
SELECT NEWID(),i.user_id,dbo.ai_trace_origin_turn(i.user_id),N''memory_embed'',CONCAT(N''open_loop:'',CONVERT(nvarchar(36),i.id),N'':'',CONVERT(varchar(16),CONVERT(binary(8),i.search_version),2)),
(SELECT N''embedding_job.v2'' [schema],N''open_loop'' source_kind,CONVERT(varchar(36),i.id) source_id,CONVERT(varchar(16),CONVERT(binary(8),i.search_version),2) source_version FOR JSON PATH,WITHOUT_ARRAY_WRAPPER)
FROM inserted i; END');

-- Recover provable historical origins only. Never guess from the latest user turn.
UPDATE j SET turn_id=m.turn_id
FROM dbo.ai_jobs j JOIN dbo.messages m
 ON JSON_VALUE(j.payload_json,'$.source_kind')=N'message'
 AND m.id=TRY_CONVERT(uniqueidentifier,JSON_VALUE(j.payload_json,'$.source_id'))
JOIN dbo.turns t ON t.id=m.turn_id AND t.user_id=j.user_id
WHERE j.kind=N'memory_embed' AND j.turn_id IS NULL;
UPDATE dbo.ai_jobs SET turn_id=t.id
FROM dbo.ai_jobs j JOIN dbo.turns t
 ON t.id=TRY_CONVERT(uniqueidentifier,JSON_VALUE(j.payload_json,'$.turn_id')) AND t.user_id=j.user_id
WHERE j.turn_id IS NULL AND j.kind=N'reflection';
CREATE INDEX IX_ai_jobs_turn_lifecycle ON dbo.ai_jobs(turn_id) INCLUDE(status,attempts,run_after) WHERE turn_id IS NOT NULL;
