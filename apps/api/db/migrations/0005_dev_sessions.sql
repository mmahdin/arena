-- Local-only revocable sessions for synthetic dev users.
-- Existing signed cookies from before this migration require re-login.
CREATE TABLE dbo.dev_sessions (
    id uniqueidentifier NOT NULL PRIMARY KEY,
    user_id uniqueidentifier NOT NULL,
    expires_at datetime2(3) NOT NULL,
    revoked_at datetime2(3) NULL,
    created_at datetime2(3) NOT NULL CONSTRAINT dev_sessions_created_default DEFAULT SYSUTCDATETIME(),
    CONSTRAINT dev_sessions_user_fk FOREIGN KEY (user_id) REFERENCES dbo.users(id) ON DELETE CASCADE
);
CREATE INDEX dev_sessions_user_active_idx ON dbo.dev_sessions(user_id,expires_at) WHERE revoked_at IS NULL;
