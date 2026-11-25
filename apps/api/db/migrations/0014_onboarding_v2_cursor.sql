-- Extend the persisted onboarding cursor for Hamrah v2 (steps 0..25).
-- The migration runner provides the transaction and records this migration.
-- Do not rewrite saved cursors here: a user may already have advanced through
-- v2 screens 0..23 while the v1 database CHECK constraint was still in place.
-- v1 and v2 saved ordinals cannot be distinguished reliably from the column.
IF OBJECT_ID(N'dbo.user_profiles', N'U') IS NULL
    THROW 51614, 'User profiles must exist before onboarding v2.', 1;
IF COL_LENGTH(N'dbo.user_profiles', N'onboarding_step') IS NULL
    THROW 51615, 'Onboarding cursor column is missing.', 1;
IF EXISTS (SELECT 1 FROM dbo.user_profiles WHERE onboarding_step NOT BETWEEN 0 AND 25)
    THROW 51616, 'Unexpected onboarding cursor; investigate before widening.', 1;

IF EXISTS (
    SELECT 1 FROM sys.check_constraints
    WHERE parent_object_id = OBJECT_ID(N'dbo.user_profiles', N'U')
      AND name = N'user_profiles_onboarding_step_range'
)
    ALTER TABLE dbo.user_profiles DROP CONSTRAINT user_profiles_onboarding_step_range;

ALTER TABLE dbo.user_profiles WITH CHECK ADD CONSTRAINT user_profiles_onboarding_step_range
    CHECK (onboarding_step BETWEEN 0 AND 25);
