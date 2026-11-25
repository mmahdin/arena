-- Internal synthetic-data prototype only. Letters require explicit per-request consent.
-- Each letter has one owner-checked source conversation and is deleted with it.
-- Migration runner owns the transaction; do not edit previously applied migrations.
CREATE TABLE dbo.personalized_letters (
    id uniqueidentifier NOT NULL PRIMARY KEY,
    user_id uniqueidentifier NOT NULL,
    source_conversation_id uniqueidentifier NOT NULL,
    title nvarchar(160) NOT NULL,
    salutation nvarchar(120) NULL,
    paragraphs_json nvarchar(max) NOT NULL,
    question nvarchar(600) NULL,
    created_at datetime2(3) NOT NULL CONSTRAINT personalized_letters_created_default DEFAULT SYSUTCDATETIME(),
    CONSTRAINT personalized_letters_source_unique UNIQUE (source_conversation_id),
    CONSTRAINT personalized_letters_source_owner_fk FOREIGN KEY (source_conversation_id, user_id)
      REFERENCES dbo.conversations(id, user_id) ON DELETE CASCADE,
    CONSTRAINT personalized_letters_title_nonempty CHECK (LEN(LTRIM(RTRIM(title))) > 0),
    CONSTRAINT personalized_letters_paragraphs_json_check CHECK (ISJSON(paragraphs_json) = 1)
);
CREATE INDEX personalized_letters_user_recent_idx ON dbo.personalized_letters(user_id, created_at DESC, id DESC);
