-- SQL Server 2022 forward-only migration; the runner owns the transaction.
EXEC(N'CREATE TRIGGER dbo.product_events_preserve_delete_guard
ON dbo.product_events AFTER DELETE
AS
BEGIN
    SET NOCOUNT ON;
    IF EXISTS (SELECT 1 FROM deleted)
        THROW 51020, ''Permanent deletion is disabled for product events.'', 1;
END');
