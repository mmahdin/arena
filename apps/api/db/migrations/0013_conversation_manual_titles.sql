-- SQL Server 2022 forward-only: explicit title edits must never be replaced by
-- the first-turn automatic title. Existing titles and conversations are retained.
ALTER TABLE dbo.conversations
    ADD title_is_manual bit NOT NULL
        CONSTRAINT conversations_title_is_manual_default DEFAULT (0);
