-- Hamrah v1 onboarding cursor only. No answers, preferences, health or identity data.
-- Completed legacy profiles may retain zero: completion timestamp remains authoritative.
ALTER TABLE dbo.user_profiles ADD onboarding_step tinyint NOT NULL
    CONSTRAINT user_profiles_onboarding_step_default DEFAULT (0) WITH VALUES;
-- Compile the constraint only after SQL Server has added the column.
EXEC sys.sp_executesql N'ALTER TABLE dbo.user_profiles ADD CONSTRAINT user_profiles_onboarding_step_range
    CHECK (onboarding_step BETWEEN 0 AND 23)';
