-- Local, synthetic phone-shaped tester identifiers only. No raw phone persisted.
-- HMAC lookup key is derived using the development session secret. Preserve that
-- secret across restarts or identities will no longer be discoverable by phone.
CREATE TABLE dbo.dev_phone_identities (
    lookup_key varbinary(32) NOT NULL PRIMARY KEY,
    user_id uniqueidentifier NOT NULL UNIQUE,
    created_at datetime2(3) NOT NULL CONSTRAINT dev_phone_identities_created_default DEFAULT SYSUTCDATETIME(),
    CONSTRAINT dev_phone_identities_user_fk FOREIGN KEY (user_id) REFERENCES dbo.users(id) ON DELETE CASCADE
);
