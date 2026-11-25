-- Forward-only, SQL Server 2022. The migration runner supplies the transaction.
-- Existing in-flight reservations from an older binary become reclaimable; old
-- binaries must be stopped before applying this migration.
ALTER TABLE dbo.turns ADD lease_id uniqueidentifier NULL, lease_expires_at datetime2(3) NULL;
-- Compile statements referencing the new columns after ALTER TABLE has executed.
-- Dynamic batches still participate in the migration runner's transaction.
EXEC sys.sp_executesql N'UPDATE dbo.turns
    SET lease_id=NEWID(), lease_expires_at=DATEADD(second,-1,SYSUTCDATETIME())
    WHERE status=N''processing''';
EXEC sys.sp_executesql N'ALTER TABLE dbo.turns ADD CONSTRAINT turns_lease_state_check CHECK (
    (status=N''processing'' AND lease_id IS NOT NULL AND lease_expires_at IS NOT NULL)
    OR (status=N''completed'' AND lease_id IS NULL AND lease_expires_at IS NULL)
)';
