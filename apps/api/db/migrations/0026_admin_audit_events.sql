-- BO-03: bounded, append-only metadata audit of privileged administrative reads.
-- No request bodies, raw prompts, tokens, email, phone numbers, IPs or secrets.
-- Staff authentication/session events remain in dbo.auth_audit_events.
CREATE TABLE dbo.admin_audit_events (
    id uniqueidentifier NOT NULL CONSTRAINT admin_audit_events_pk PRIMARY KEY,
    actor_staff_id uniqueidentifier NOT NULL,
    action_code varchar(48) NOT NULL,
    resource_kind varchar(24) NOT NULL,
    resource_id uniqueidentifier NULL,
    outcome_code varchar(16) NOT NULL,
    request_id uniqueidentifier NOT NULL,
    recorded_at datetime2(3) NOT NULL CONSTRAINT admin_audit_events_recorded_default DEFAULT SYSUTCDATETIME(),
    CONSTRAINT admin_audit_events_actor_fk FOREIGN KEY (actor_staff_id) REFERENCES dbo.staff_accounts(id),
    CONSTRAINT admin_audit_events_outcome_check CHECK (outcome_code IN ('authorized','success','denied','failed')),
    CONSTRAINT admin_audit_events_kind_check CHECK (resource_kind IN (
        'admin','audit','turn','job','worker','evaluation','bundle',
        'test_user','release','review','privacy','system'
    ))
);
CREATE INDEX admin_audit_events_time_idx ON dbo.admin_audit_events(recorded_at DESC,id DESC)
    INCLUDE (actor_staff_id,action_code,resource_kind,outcome_code,request_id);
CREATE INDEX admin_audit_events_actor_time_idx ON dbo.admin_audit_events(actor_staff_id,recorded_at DESC,id DESC);

-- This is DML tamper resistance, not cryptographic immutability: database
-- owners/sysadmins can bypass triggers. Apply least-privilege DB credentials,
-- protected backups and audit retention controls during production hardening.
-- CREATE TRIGGER must be the first statement in its batch: EXEC avoids GO.
EXEC(N'CREATE TRIGGER dbo.admin_audit_events_block_mutation
ON dbo.admin_audit_events AFTER UPDATE, DELETE
AS BEGIN
    SET NOCOUNT ON;
    THROW 51026, ''admin audit events are append only'', 1;
END');
