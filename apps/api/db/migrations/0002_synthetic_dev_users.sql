-- SQL Server synthetic identities only. Do not seed clinical/private chat content.
INSERT INTO dbo.users(id,test_key,is_test) VALUES
 (CONVERT(uniqueidentifier,'11111111-1111-4111-8111-111111111111'),N'tester-1',1),
 (CONVERT(uniqueidentifier,'22222222-2222-4222-8222-222222222222'),N'tester-2',1),
 (CONVERT(uniqueidentifier,'33333333-3333-4333-8333-333333333333'),N'tester-3',1);
INSERT INTO dbo.user_profiles(user_id,display_name,conversation_style,onboarding_completed_at) VALUES
 (CONVERT(uniqueidentifier,'11111111-1111-4111-8111-111111111111'),N'نیلو',N'balanced',SYSUTCDATETIME()),
 (CONVERT(uniqueidentifier,'22222222-2222-4222-8222-222222222222'),NULL,N'balanced',NULL),
 (CONVERT(uniqueidentifier,'33333333-3333-4333-8333-333333333333'),N'آرمان',N'listening',SYSUTCDATETIME());
