-- Enable five distinct conversational preferences; preserve existing legacy guidance rows.
-- No user profile rows or their onboarding timestamps are changed.
ALTER TABLE dbo.user_profiles DROP CONSTRAINT user_profiles_style_check;
ALTER TABLE dbo.user_profiles WITH CHECK ADD CONSTRAINT user_profiles_style_check
    CHECK (conversation_style IN (
        N'balanced', N'listening', N'teaching', N'gentle_guidance',
        N'collaborative', N'guidance' -- legacy profile value retained for compatibility
    ));
