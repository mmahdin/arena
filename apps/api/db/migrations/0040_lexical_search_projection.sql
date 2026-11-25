-- Rebuildable owner-scoped lexical metadata, versioned with canonical sources.
-- No original user records or historical indexes are removed.
CREATE TABLE dbo.ai_search_lexical (
 user_id uniqueidentifier NOT NULL REFERENCES dbo.users(id),
 source_kind nvarchar(24) NOT NULL,
 source_id uniqueidentifier NOT NULL,
 source_version binary(8) NOT NULL,
 doc_length int NOT NULL CHECK(doc_length>=0),
 content_key char(64) NOT NULL,
 signature binary(32) NOT NULL,
 terms_json nvarchar(max) NOT NULL CHECK(ISJSON(terms_json)=1),
 created_at datetime2(3) NOT NULL DEFAULT SYSUTCDATETIME(),
 CONSTRAINT PK_ai_search_lexical PRIMARY KEY(user_id,source_kind,source_id,source_version)
);

EXEC(N'CREATE TRIGGER dbo.ai_search_lexical_preserve_delete_guard ON dbo.ai_search_lexical AFTER DELETE AS BEGIN IF EXISTS(SELECT 1 FROM deleted) THROW 51020, N''Soft deletion only'', 1; END');

EXEC(N'CREATE TRIGGER dbo.ai_search_lexical_owner ON dbo.ai_search_lexical AFTER INSERT,UPDATE AS BEGIN SET NOCOUNT ON; IF EXISTS(SELECT 1 FROM inserted i WHERE NOT EXISTS(SELECT 1 FROM dbo.ai_search_sources s WHERE s.user_id=i.user_id AND s.source_kind=i.source_kind AND s.id=i.source_id AND s.source_version=i.source_version)) THROW 51001,N''AI reference owner mismatch'',1; END');
