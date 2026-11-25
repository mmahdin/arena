-- WP-16 account visibility. Original and derived records remain retained.
ALTER TABLE dbo.users ADD deleted_at datetime2(3) NULL,
 deleted_by uniqueidentifier NULL,
 deletion_reason nvarchar(80) NULL;
EXEC(N'ALTER TABLE dbo.users ADD CONSTRAINT users_deletion_metadata CHECK
 ((deleted_at IS NULL AND deleted_by IS NULL AND deletion_reason IS NULL)
 OR (deleted_at IS NOT NULL AND deleted_by=id AND deletion_reason=N''user_request''));');
