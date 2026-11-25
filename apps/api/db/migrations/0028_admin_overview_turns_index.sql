-- BO-05: bounded admin overview queries use the admission-time window.
-- Index contains no sensitive content. SQL Server 2022, additive migration.
CREATE INDEX turns_admin_created_idx ON dbo.turns(created_at DESC)
    INCLUDE (lifecycle_version,status,completed_at);
