-- Persist insertion order independently of datetime2(3) ties and random UUIDs.
ALTER TABLE dbo.messages ADD message_order bigint NULL;
-- Compile after the new column exists within the runner's transaction.
EXEC(N'WITH ordered AS (
    SELECT m.message_order, ROW_NUMBER() OVER (
        ORDER BY t.created_at, t.id, CASE m.role WHEN N''user'' THEN 0 ELSE 1 END, m.id
    ) AS ordinal
    FROM dbo.messages m JOIN dbo.turns t ON t.id=m.turn_id
)
UPDATE ordered SET message_order=ordinal;
ALTER TABLE dbo.messages ALTER COLUMN message_order bigint NOT NULL;
DECLARE @next bigint = (SELECT ISNULL(MAX(message_order),0)+1 FROM dbo.messages);
DECLARE @sql nvarchar(max) = N''CREATE SEQUENCE dbo.message_order_sequence AS bigint START WITH ''
    + CONVERT(nvarchar(30),@next) + N'' INCREMENT BY 1'';
EXEC sys.sp_executesql @sql;
ALTER TABLE dbo.messages ADD CONSTRAINT messages_order_default
    DEFAULT (NEXT VALUE FOR dbo.message_order_sequence) FOR message_order;
CREATE UNIQUE INDEX messages_conversation_sequence_idx ON dbo.messages(conversation_id,message_order);
');
