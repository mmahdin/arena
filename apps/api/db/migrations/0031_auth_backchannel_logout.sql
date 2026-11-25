-- AUTH-08: replay-protected Keycloak back-channel logout and callback fences.
-- Forward-only migration. Keycloak's signed JWT, jti, sub and sid are NEVER
-- stored. Application logout/session IDs remain hashed as before.
CREATE TABLE dbo.auth_backchannel_events (
    event_hash binary(32) NOT NULL PRIMARY KEY,
    received_at datetime2(3) NOT NULL CONSTRAINT auth_backchannel_received_default DEFAULT SYSUTCDATETIME(),
    expires_at datetime2(3) NOT NULL
);
CREATE INDEX auth_backchannel_events_expiry_idx ON dbo.auth_backchannel_events(expires_at);

CREATE TABLE dbo.auth_logout_fences (
    fence_hash binary(32) NOT NULL PRIMARY KEY,
    created_at datetime2(3) NOT NULL CONSTRAINT auth_logout_fence_created_default DEFAULT SYSUTCDATETIME(),
    expires_at datetime2(3) NOT NULL
);
CREATE INDEX auth_logout_fences_expiry_idx ON dbo.auth_logout_fences(expires_at);
-- Cleanup of expired replay records/fences belongs to a separately approved,
-- audited maintenance process. No DML trigger or background job is installed.

-- A logout-all watermark blocks callbacks begun *before* all-device revocation
-- from creating replacement sessions after the transaction commits. A new
-- login transaction begun after logout-all may still authenticate at Keycloak.
CREATE TABLE dbo.auth_principal_logout_watermarks (
    principal_kind varchar(10) NOT NULL,
    principal_id uniqueidentifier NOT NULL,
    revoked_at datetime2(3) NOT NULL,
    CONSTRAINT auth_principal_logout_watermarks_pk PRIMARY KEY (principal_kind,principal_id),
    CONSTRAINT auth_principal_logout_watermarks_kind CHECK (principal_kind IN ('user','staff'))
);
