-- AUTH-03: durable OIDC identity mapping and opaque browser sessions.
-- SQL Server 2022 forward-only; the migration runner owns the transaction.
-- No raw phone, passwords, OTP, ID/access/refresh tokens or PKCE verifier plaintext.
CREATE TABLE dbo.staff_accounts (
    id uniqueidentifier NOT NULL PRIMARY KEY,
    active bit NOT NULL CONSTRAINT staff_accounts_active_default DEFAULT(0),
    created_at datetime2(3) NOT NULL CONSTRAINT staff_accounts_created_default DEFAULT SYSUTCDATETIME(),
    deactivated_at datetime2(3) NULL,
    CONSTRAINT staff_accounts_activity CHECK ((active=1 AND deactivated_at IS NULL) OR active=0)
);
CREATE TABLE dbo.auth_identities (
    id uniqueidentifier NOT NULL PRIMARY KEY,
    issuer nvarchar(512) NOT NULL,
    subject nvarchar(200) NOT NULL,
    principal_kind varchar(10) NOT NULL,
    user_id uniqueidentifier NULL,
    staff_id uniqueidentifier NULL,
    created_at datetime2(3) NOT NULL CONSTRAINT auth_identities_created_default DEFAULT SYSUTCDATETIME(),
    CONSTRAINT auth_identities_issuer_subject UNIQUE (issuer,subject),
    CONSTRAINT auth_identities_kind CHECK (principal_kind IN ('user','staff')),
    CONSTRAINT auth_identities_owner CHECK (
      (principal_kind='user' AND user_id IS NOT NULL AND staff_id IS NULL) OR
      (principal_kind='staff' AND staff_id IS NOT NULL AND user_id IS NULL)),
    CONSTRAINT auth_identities_user_fk FOREIGN KEY (user_id) REFERENCES dbo.users(id),
    CONSTRAINT auth_identities_staff_fk FOREIGN KEY (staff_id) REFERENCES dbo.staff_accounts(id)
);
CREATE UNIQUE INDEX auth_identities_user_provider ON dbo.auth_identities(issuer,user_id) WHERE user_id IS NOT NULL;
CREATE UNIQUE INDEX auth_identities_staff_provider ON dbo.auth_identities(issuer,staff_id) WHERE staff_id IS NOT NULL;

CREATE TABLE dbo.auth_login_transactions (
    state_hash binary(32) NOT NULL PRIMARY KEY,
    principal_kind varchar(10) NOT NULL,
    expected_origin nvarchar(255) NOT NULL,
    redirect_uri nvarchar(512) NOT NULL,
    sealed_payload varbinary(1024) NOT NULL,
    expires_at datetime2(3) NOT NULL,
    created_at datetime2(3) NOT NULL CONSTRAINT auth_login_created_default DEFAULT SYSUTCDATETIME(),
    CONSTRAINT auth_login_kind CHECK (principal_kind IN ('user','staff'))
);
CREATE INDEX auth_login_expires_idx ON dbo.auth_login_transactions(expires_at);

CREATE TABLE dbo.auth_sessions (
    token_hash binary(32) NOT NULL PRIMARY KEY,
    principal_kind varchar(10) NOT NULL,
    user_id uniqueidentifier NULL,
    staff_id uniqueidentifier NULL,
    issuer nvarchar(512) NOT NULL,
    subject nvarchar(200) NOT NULL,
    keycloak_sid nvarchar(200) NULL,
    created_at datetime2(3) NOT NULL,
    last_seen_at datetime2(3) NOT NULL,
    idle_ttl_seconds int NOT NULL,
    idle_expires_at datetime2(3) NOT NULL,
    expires_at datetime2(3) NOT NULL,
    revoked_at datetime2(3) NULL,
    CONSTRAINT auth_sessions_kind CHECK (principal_kind IN ('user','staff')),
    CONSTRAINT auth_sessions_owner CHECK (
      (principal_kind='user' AND user_id IS NOT NULL AND staff_id IS NULL) OR
      (principal_kind='staff' AND staff_id IS NOT NULL AND user_id IS NULL)),
    CONSTRAINT auth_sessions_ttl CHECK (idle_ttl_seconds BETWEEN 60 AND 2592000 AND idle_expires_at<=expires_at),
    CONSTRAINT auth_sessions_user_fk FOREIGN KEY (user_id) REFERENCES dbo.users(id),
    CONSTRAINT auth_sessions_staff_fk FOREIGN KEY (staff_id) REFERENCES dbo.staff_accounts(id)
);
CREATE INDEX auth_sessions_user_active_idx ON dbo.auth_sessions(user_id,expires_at) WHERE revoked_at IS NULL AND user_id IS NOT NULL;
CREATE INDEX auth_sessions_staff_active_idx ON dbo.auth_sessions(staff_id,expires_at) WHERE revoked_at IS NULL AND staff_id IS NOT NULL;
CREATE INDEX auth_sessions_keycloak_sid_idx ON dbo.auth_sessions(issuer,keycloak_sid) WHERE revoked_at IS NULL AND keycloak_sid IS NOT NULL;

CREATE TABLE dbo.auth_audit_events (
    id uniqueidentifier NOT NULL PRIMARY KEY,
    principal_kind varchar(10) NOT NULL,
    actor_id uniqueidentifier NULL,
    action_code varchar(40) NOT NULL,
    outcome_code varchar(20) NOT NULL,
    request_id varchar(64) NULL,
    recorded_at datetime2(3) NOT NULL CONSTRAINT auth_audit_time_default DEFAULT SYSUTCDATETIME(),
    CONSTRAINT auth_audit_kind CHECK (principal_kind IN ('user','staff')),
    CONSTRAINT auth_audit_action CHECK (action_code IN ('login','logout','session_rejected','staff_provisioned','session_revoked')),
    CONSTRAINT auth_audit_outcome CHECK (outcome_code IN ('success','denied','failed'))
);
CREATE INDEX auth_audit_events_recorded_idx ON dbo.auth_audit_events(recorded_at,id);
