-- BO-06: bounded admin job inspection, ordered by creation time and UUID.
-- No job payload, user data or lease token is included in this index.
CREATE INDEX ai_jobs_admin_created_idx ON dbo.ai_jobs(created_at DESC,id DESC)
    INCLUDE (status,kind,turn_id,attempts,max_attempts,run_after,last_error_code);
