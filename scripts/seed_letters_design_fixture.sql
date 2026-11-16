-- Local synthetic data only. Not a schema migration and no model is called.
-- Run against the loopback-bound wellbeing development database.
-- One identical, explicitly illustrative letter per EXISTING test account.
-- Deterministic owner-specific IDs make reruns idempotent; no profiles/messages
-- or existing conversations/letters are modified. The empty source conversation
-- exists only to satisfy the letter's composite ownership foreign key.
SET NOCOUNT ON;
SET XACT_ABORT ON;
IF DB_NAME() <> N'wellbeing'
    THROW 51000, 'This fixture is restricted to the local wellbeing database.', 1;
BEGIN TRANSACTION;
DECLARE @fixtures TABLE (user_id uniqueidentifier PRIMARY KEY, source_id uniqueidentifier, letter_id uniqueidentifier);
INSERT INTO @fixtures(user_id, source_id, letter_id)
SELECT id,
       CONVERT(uniqueidentifier, SUBSTRING(HASHBYTES('SHA2_256', N'hamrah-letters-design-source-v1:' + CONVERT(nvarchar(36), id)), 1, 16)),
       CONVERT(uniqueidentifier, SUBSTRING(HASHBYTES('SHA2_256', N'hamrah-letters-design-letter-v1:' + CONVERT(nvarchar(36), id)), 1, 16))
FROM dbo.users WHERE is_test = 1;

IF EXISTS (
    SELECT 1 FROM @fixtures f JOIN dbo.conversations c ON c.id=f.source_id
    WHERE c.user_id<>f.user_id OR c.title<>N'[آزمایشی] منبع نامهٔ مقایسهٔ طراحی'
) OR EXISTS (
    SELECT 1 FROM @fixtures f JOIN dbo.personalized_letters l ON l.id=f.letter_id
    WHERE l.user_id<>f.user_id OR l.source_conversation_id<>f.source_id
)
    THROW 51001, 'Fixture ID conflict: no data was changed.', 1;

INSERT INTO dbo.conversations(id, user_id, title)
SELECT f.source_id, f.user_id, N'[آزمایشی] منبع نامهٔ مقایسهٔ طراحی'
FROM @fixtures f
WHERE NOT EXISTS (SELECT 1 FROM dbo.conversations c WITH (UPDLOCK,HOLDLOCK) WHERE c.id=f.source_id);
DECLARE @newSources int=@@ROWCOUNT;

INSERT INTO dbo.personalized_letters(id, user_id, source_conversation_id, title, salutation, paragraphs_json, question)
SELECT f.letter_id, f.user_id, f.source_id,
       N'نامهٔ آزمایشی: کمی آرام‌تر، برای خودت',
       N'دوست من،',
       N'["گاهی روزها آن‌قدر شلوغ می‌شوند که یادمان می‌رود از خودمان بپرسیم: من امروز چطورم؟ لازم نیست همین حالا جواب روشنی داشته باشی. همین که لحظه‌ای برای شنیدن خودت کنار بگذاری، کافی است.", "قرار نیست همهٔ راه را امروز بروی. می‌توانی فقط یک قدم کوچک برداری؛ کاری را کمی آهسته‌تر انجام بدهی، کنار پنجره بایستی، یا به خودت اجازه بدهی چند دقیقه هیچ کاری نکنی.", "اگر امروز آن‌طور که می‌خواستی پیش نرفت، هنوز می‌شود با خودت مهربان بود. ارزش تو به تعداد کارهایی که تمام کرده‌ای وابسته نیست.", "این نامه یک دادهٔ آزمایشی ثابت برای مقایسهٔ طراحی‌های مختلف است و از گفت‌وگوهای تو ساخته نشده است."]',
       N'اگر امروز فقط یک لطف کوچک در حق خودت بکنی، آن لطف چه خواهد بود؟'
FROM @fixtures f
WHERE NOT EXISTS (SELECT 1 FROM dbo.personalized_letters l WITH (UPDLOCK,HOLDLOCK) WHERE l.id=f.letter_id);
DECLARE @newLetters int=@@ROWCOUNT;
COMMIT TRANSACTION;
SELECT @newSources AS created_source_conversations, @newLetters AS created_test_letters,
       (SELECT COUNT(*) FROM @fixtures) AS existing_test_accounts;
