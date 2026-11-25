-- AUTH-04: auditable local staff allowlist lifecycle.
-- Does not store Keycloak passwords, email, sessions, or credentials.
-- A CLI operator is not an authenticated BFF principal: actor_id remains NULL
-- for bootstrap operations. target_staff_id identifies the affected account.
ALTER TABLE dbo.auth_audit_events
    ADD target_staff_id uniqueidentifier NULL;

ALTER TABLE dbo.auth_audit_events
    ADD CONSTRAINT auth_audit_target_staff_fk FOREIGN KEY (target_staff_id)
    REFERENCES dbo.staff_accounts(id);

ALTER TABLE dbo.auth_audit_events DROP CONSTRAINT auth_audit_action;
ALTER TABLE dbo.auth_audit_events ADD CONSTRAINT auth_audit_action CHECK (
    action_code IN (
        'login','logout','session_rejected','staff_provisioned',
        'session_revoked','staff_deactivated','staff_reactivated'
    )
);
