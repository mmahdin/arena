-- Preserve every existing row. The runner owns the transaction.
-- Deletion metadata is visibility state, never AI consent or retention authority.
DECLARE @table sysname, @sql nvarchar(max);
DECLARE targets CURSOR LOCAL FAST_FORWARD FOR
 SELECT name FROM sys.tables WHERE schema_id=SCHEMA_ID(N'dbo') AND name IN
 (N'conversations',N'messages',N'saved_messages',N'daily_reflections',N'personalized_letters',N'ai_memory_items',N'ai_formulations');
OPEN targets;
FETCH NEXT FROM targets INTO @table;
WHILE @@FETCH_STATUS=0 BEGIN
 SET @sql=N'ALTER TABLE dbo.'+QUOTENAME(@table)+N' ADD deleted_at datetime2(3) NULL, deleted_by uniqueidentifier NULL, deletion_reason nvarchar(40) NULL;';
 EXEC sys.sp_executesql @sql;
 SET @sql=N'ALTER TABLE dbo.'+QUOTENAME(@table)+N' ADD CONSTRAINT '+QUOTENAME(@table+N'_deletion_metadata_check')+N' CHECK
 ((deleted_at IS NULL AND deleted_by IS NULL AND deletion_reason IS NULL) OR
 (deleted_at IS NOT NULL AND deleted_by IS NOT NULL AND deletion_reason IS NOT NULL AND deletion_reason=N''user_request''));';
 EXEC sys.sp_executesql @sql;
 FETCH NEXT FROM targets INTO @table;
END;
CLOSE targets;
DEALLOCATE targets;

-- Allow a new bookmark/diary entry/letter after deletion without overwriting the
-- preserved record. No other table references these primary/unique keys.
DECLARE @pk sysname;
SELECT @pk=name FROM sys.key_constraints WHERE parent_object_id=OBJECT_ID(N'dbo.saved_messages') AND type=N'PK';
SET @sql=N'ALTER TABLE dbo.saved_messages DROP CONSTRAINT '+QUOTENAME(@pk);
EXEC sys.sp_executesql @sql;
ALTER TABLE dbo.saved_messages ADD id uniqueidentifier NOT NULL CONSTRAINT saved_messages_id_default DEFAULT NEWID();
EXEC(N'ALTER TABLE dbo.saved_messages ADD CONSTRAINT saved_messages_pk PRIMARY KEY(id);
 CREATE UNIQUE INDEX saved_messages_active_unique ON dbo.saved_messages(message_id) WHERE deleted_at IS NULL;');
ALTER TABLE dbo.daily_reflections DROP CONSTRAINT daily_reflections_pk;
ALTER TABLE dbo.daily_reflections ADD id uniqueidentifier NOT NULL CONSTRAINT daily_reflections_id_default DEFAULT NEWID();
EXEC(N'ALTER TABLE dbo.daily_reflections ADD CONSTRAINT daily_reflections_pk PRIMARY KEY(id);
 CREATE UNIQUE INDEX daily_reflections_active_unique ON dbo.daily_reflections(user_id,entry_date) WHERE deleted_at IS NULL;');
ALTER TABLE dbo.personalized_letters DROP CONSTRAINT personalized_letters_source_unique;
EXEC(N'CREATE UNIQUE INDEX personalized_letters_active_source_unique ON dbo.personalized_letters(source_conversation_id) WHERE deleted_at IS NULL;
 CREATE INDEX conversations_active_owner_idx ON dbo.conversations(user_id,updated_at DESC,id) WHERE deleted_at IS NULL;');

-- Replace existing cascade actions with the same validated owner references,
-- using catalog order for composite keys (including generated constraint names).
DECLARE fks CURSOR LOCAL FAST_FORWARD FOR
 SELECT N'ALTER TABLE '+QUOTENAME(OBJECT_SCHEMA_NAME(f.parent_object_id))+N'.'+QUOTENAME(OBJECT_NAME(f.parent_object_id))+N' DROP CONSTRAINT '+QUOTENAME(f.name)+N'; ALTER TABLE '+
 QUOTENAME(OBJECT_SCHEMA_NAME(f.parent_object_id))+N'.'+QUOTENAME(OBJECT_NAME(f.parent_object_id))+N' WITH CHECK ADD CONSTRAINT '+QUOTENAME(f.name)+N' FOREIGN KEY ('+
 (SELECT STRING_AGG(CONVERT(nvarchar(max),QUOTENAME(COL_NAME(fc.parent_object_id,fc.parent_column_id))),N',') WITHIN GROUP(ORDER BY fc.constraint_column_id) FROM sys.foreign_key_columns fc WHERE fc.constraint_object_id=f.object_id)+N') REFERENCES '+
 QUOTENAME(OBJECT_SCHEMA_NAME(f.referenced_object_id))+N'.'+QUOTENAME(OBJECT_NAME(f.referenced_object_id))+N' ('+
 (SELECT STRING_AGG(CONVERT(nvarchar(max),QUOTENAME(COL_NAME(fc.referenced_object_id,fc.referenced_column_id))),N',') WITHIN GROUP(ORDER BY fc.constraint_column_id) FROM sys.foreign_key_columns fc WHERE fc.constraint_object_id=f.object_id)+N') ON DELETE NO ACTION ON UPDATE NO ACTION;'
 FROM sys.foreign_keys f WHERE f.delete_referential_action<>0 AND OBJECT_SCHEMA_NAME(f.parent_object_id)=N'dbo';
OPEN fks;
FETCH NEXT FROM fks INTO @sql;
WHILE @@FETCH_STATUS=0 BEGIN
 EXEC sys.sp_executesql @sql;
 FETCH NEXT FROM fks INTO @sql;
END;
CLOSE fks;
DEALLOCATE fks;
-- Historical expiry metadata is retained; new traces have no automatic expiry.
ALTER TABLE dbo.ai_turn_traces ALTER COLUMN expires_at datetime2(3) NULL;
DROP PROCEDURE dbo.ai_purge_user_derivatives;
DROP TRIGGER dbo.ai_raw_delete_guard;

-- Fail closed for old binaries, accidental raw deletes and future cleanup code.
-- Schema changes and disposable test database teardown remain operator actions;
-- this does not create an erasure API or an exemption for user data.
DECLARE guards CURSOR LOCAL FAST_FORWARD FOR
 SELECT name FROM sys.tables WHERE schema_id=SCHEMA_ID(N'dbo') AND name<>N'schema_migrations';
OPEN guards;
FETCH NEXT FROM guards INTO @table;
WHILE @@FETCH_STATUS=0 BEGIN
 SET @sql=N'CREATE TRIGGER dbo.'+QUOTENAME(@table+N'_preserve_delete_guard')+N' ON dbo.'+QUOTENAME(@table)+N' AFTER DELETE AS BEGIN SET NOCOUNT ON; IF EXISTS(SELECT 1 FROM deleted) THROW 51020, ''Permanent deletion is disabled; use the owning soft-delete operation.'', 1; END;';
 EXEC sys.sp_executesql @sql;
 FETCH NEXT FROM guards INTO @table;
END;
CLOSE guards;
DEALLOCATE guards;
