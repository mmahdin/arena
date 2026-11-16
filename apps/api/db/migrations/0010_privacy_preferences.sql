-- Internal synthetic-data-only preference; no conversation sharing pipeline exists.
-- A future real-user sharing feature must require separate affirmative consent.
ALTER TABLE dbo.user_profiles ADD conversation_sharing_enabled bit NOT NULL
    CONSTRAINT user_profiles_conversation_sharing_default DEFAULT (1) WITH VALUES;
