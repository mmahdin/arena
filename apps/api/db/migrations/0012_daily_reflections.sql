-- Synthetic-data-only, explicitly authored user entries; no AI inference or profiling.
-- Per-user calendar date, no timezone conversion; migration runner owns transaction.
CREATE TABLE dbo.daily_reflections (
    user_id uniqueidentifier NOT NULL,
    entry_date date NOT NULL,
    body nvarchar(4000) NOT NULL,
    created_at datetime2(3) NOT NULL CONSTRAINT daily_reflections_created_default DEFAULT SYSUTCDATETIME(),
    updated_at datetime2(3) NOT NULL CONSTRAINT daily_reflections_updated_default DEFAULT SYSUTCDATETIME(),
    CONSTRAINT daily_reflections_pk PRIMARY KEY (user_id, entry_date),
    CONSTRAINT daily_reflections_user_fk FOREIGN KEY (user_id) REFERENCES dbo.users(id) ON DELETE CASCADE,
    CONSTRAINT daily_reflections_body_nonempty CHECK (LEN(LTRIM(RTRIM(body))) > 0)
);
CREATE INDEX daily_reflections_recent_idx ON dbo.daily_reflections(user_id, entry_date DESC);
