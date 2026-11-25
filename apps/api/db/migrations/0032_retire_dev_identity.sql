-- AUTH-09: retire synthetic browser sessions and phone-shaped identity lookup.
-- Forward-only: historic 0005/0007 remain unchanged for migration checksum stability.
-- OIDC identity mapping, user profiles, conversations and Staff sessions remain intact.
IF OBJECT_ID(N'dbo.dev_phone_identities', N'U') IS NOT NULL
    DROP TABLE dbo.dev_phone_identities;
IF OBJECT_ID(N'dbo.dev_sessions', N'U') IS NOT NULL
    DROP TABLE dbo.dev_sessions;
