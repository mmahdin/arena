-- Expand only: legacy rows remain readable during the WP-05 cutover.
ALTER TABLE dbo.turns DROP CONSTRAINT turns_status_check, turns_lease_state_check;
ALTER TABLE dbo.turns ADD client_turn_id uniqueidentifier NULL,
    lifecycle_version tinyint NOT NULL DEFAULT (0),
    attempt_count int NOT NULL DEFAULT (0), error_code nvarchar(80) NULL,
    result_meta_json nvarchar(max) NULL, request_meta_json nvarchar(max) NULL,
    admission_order bigint IDENTITY(1,1) NOT NULL;
ALTER TABLE dbo.turns ADD CONSTRAINT turns_owner_unique UNIQUE(id,user_id);
EXEC(N'UPDATE dbo.turns SET client_turn_id=id;
ALTER TABLE dbo.turns ALTER COLUMN client_turn_id uniqueidentifier NOT NULL;
ALTER TABLE dbo.turns ADD CONSTRAINT turns_client_default DEFAULT NEWID() FOR client_turn_id;
CREATE UNIQUE INDEX turns_client_owner_unique ON dbo.turns(user_id,client_turn_id);
CREATE INDEX turns_user_queue ON dbo.turns(user_id,lifecycle_version,status,admission_order);
ALTER TABLE dbo.turns ADD CONSTRAINT turns_status_check CHECK (
 (lifecycle_version=0 AND status IN (N''processing'',N''completed'')) OR
 (lifecycle_version=1 AND status IN (N''queued'',N''running'',N''completed'',N''failed'',N''cancelled'')));
ALTER TABLE dbo.turns ADD CONSTRAINT turns_lease_state_check CHECK (
 (lifecycle_version=0 AND ((status=N''processing'' AND lease_id IS NOT NULL AND lease_expires_at IS NOT NULL)
 OR (status=N''completed'' AND lease_id IS NULL AND lease_expires_at IS NULL))) OR
 (lifecycle_version=1 AND lease_id IS NULL AND lease_expires_at IS NULL));
ALTER TABLE dbo.turns ADD CONSTRAINT turns_result_json CHECK(result_meta_json IS NULL OR ISJSON(result_meta_json)=1),
 CONSTRAINT turns_request_json CHECK(request_meta_json IS NULL OR ISJSON(request_meta_json)=1),
 CONSTRAINT turns_attempts_check CHECK(attempt_count>=0),
 CONSTRAINT turns_v1_messages CHECK(lifecycle_version=0 OR
 (user_message_id IS NOT NULL AND ((status=N''completed'' AND assistant_message_id IS NOT NULL AND completed_at IS NOT NULL)
 OR (status<>N''completed'' AND assistant_message_id IS NULL AND completed_at IS NULL))));');
-- Prevent duplicate role rows, without deleting inconsistent historical data.
-- If validation fails, the runner rolls back and an operator must investigate.
CREATE UNIQUE INDEX messages_turn_role_unique ON dbo.messages(turn_id,role);
ALTER TABLE dbo.messages ADD lang nvarchar(24) NULL;
CREATE TABLE dbo.ai_user_turn_leases (
 user_id uniqueidentifier NOT NULL PRIMARY KEY REFERENCES dbo.users(id) ON DELETE CASCADE,
 turn_id uniqueidentifier NOT NULL, lease_token uniqueidentifier NOT NULL,
 lease_expires_at datetime2(3) NOT NULL,
 updated_at datetime2(3) NOT NULL DEFAULT SYSUTCDATETIME(),
 CONSTRAINT ai_lease_turn_owner FOREIGN KEY(turn_id,user_id) REFERENCES dbo.turns(id,user_id)
);
CREATE INDEX ai_lease_expiry ON dbo.ai_user_turn_leases(lease_expires_at);
