-- Explicit questionnaire selections for the signed-in synthetic tester only.
-- Existing users retain NULL; never backfill inferred answers from chats.
ALTER TABLE dbo.user_profiles ADD onboarding_answers_json nvarchar(max) NULL
    CONSTRAINT user_profiles_onboarding_answers_json_check
    CHECK (onboarding_answers_json IS NULL OR
        (ISJSON(onboarding_answers_json) = 1 AND DATALENGTH(onboarding_answers_json) <= 16000));
