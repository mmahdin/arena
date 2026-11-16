-- Remove ONLY this deterministic design fixture. No existing user messages
-- or unrelated letters are deleted. If the fixture conversation acquired real
-- test turns/messages, keep that conversation and remove only its fixture letter.
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

DELETE l FROM dbo.personalized_letters l
JOIN @fixtures f ON l.id=f.letter_id AND l.user_id=f.user_id AND l.source_conversation_id=f.source_id;
DECLARE @deletedLetters int=@@ROWCOUNT;
DELETE c FROM dbo.conversations c
JOIN @fixtures f ON c.id=f.source_id AND c.user_id=f.user_id
WHERE c.title=N'[آزمایشی] منبع نامهٔ مقایسهٔ طراحی'
  AND NOT EXISTS (SELECT 1 FROM dbo.turns t WHERE t.conversation_id=c.id)
  AND NOT EXISTS (SELECT 1 FROM dbo.messages m WHERE m.conversation_id=c.id)
  AND NOT EXISTS (SELECT 1 FROM dbo.personalized_letters l WHERE l.source_conversation_id=c.id);
DECLARE @deletedSources int=@@ROWCOUNT;
COMMIT TRANSACTION;
SELECT @deletedLetters AS removed_fixture_letters, @deletedSources AS removed_empty_fixture_sources;
