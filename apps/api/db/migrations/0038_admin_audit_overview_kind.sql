-- The Overview endpoint records admin.overview.read with resource_kind='overview'.
ALTER TABLE dbo.admin_audit_events DROP CONSTRAINT admin_audit_events_kind_check;
ALTER TABLE dbo.admin_audit_events WITH CHECK
    ADD CONSTRAINT admin_audit_events_kind_check CHECK (resource_kind IN (
        'admin','audit','turn','job','worker','evaluation','bundle',
        'test_user','release','review','privacy','system','overview'
    ));
