-- Internal synthetic-only message bookmarks. The message FK is the only cascade path.
-- No separate users FK: ownership is enforced in all repository queries.
CREATE TABLE dbo.saved_messages (
    message_id uniqueidentifier NOT NULL PRIMARY KEY,
    user_id uniqueidentifier NOT NULL,
    saved_at datetime2(3) NOT NULL CONSTRAINT saved_messages_at_default DEFAULT SYSUTCDATETIME(),
    CONSTRAINT saved_messages_message_fk FOREIGN KEY (message_id)
        REFERENCES dbo.messages(id) ON DELETE CASCADE
);
CREATE INDEX saved_messages_user_recent_idx ON dbo.saved_messages(user_id, saved_at DESC, message_id);
