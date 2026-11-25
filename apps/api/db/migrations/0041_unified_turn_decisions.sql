-- SQL Server 2022 additive migration. The runner owns transaction management.
CREATE TABLE dbo.ai_turn_decisions (
 turn_id uniqueidentifier NOT NULL PRIMARY KEY,
 user_id uniqueidentifier NOT NULL,
 carry_json nvarchar(max) NOT NULL CHECK(ISJSON(carry_json)=1),
 resolution_json nvarchar(max) NOT NULL CHECK(ISJSON(resolution_json)=1),
 created_at datetime2(3) NOT NULL DEFAULT SYSUTCDATETIME(),
 CONSTRAINT FK_ai_turn_decisions_owner FOREIGN KEY(turn_id,user_id) REFERENCES dbo.turns(id,user_id)
);
CREATE INDEX IX_ai_turn_decisions_owner ON dbo.ai_turn_decisions(user_id,created_at DESC);
EXEC(N'CREATE TRIGGER dbo.ai_turn_decisions_immutable ON dbo.ai_turn_decisions AFTER UPDATE,DELETE AS BEGIN IF EXISTS(SELECT 1 FROM deleted) THROW 51020,N''Immutable turn decisions'',1; END');
