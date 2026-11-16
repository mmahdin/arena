-- SQL Server 2022 forward-only migration; runner owns the transaction.
-- UTC in datetime2(3), identifiers are uniqueidentifier, no GO batches.
CREATE TABLE dbo.users (
    id uniqueidentifier NOT NULL PRIMARY KEY,
    test_key nvarchar(100) NOT NULL UNIQUE,
    is_test bit NOT NULL CONSTRAINT users_is_test_default DEFAULT (1),
    created_at datetime2(3) NOT NULL CONSTRAINT users_created_at_default DEFAULT SYSUTCDATETIME(),
    CONSTRAINT users_test_key_nonblank CHECK (LEN(test_key)>0)
);
CREATE TABLE dbo.user_profiles (
    user_id uniqueidentifier NOT NULL PRIMARY KEY,
    display_name nvarchar(240) NULL,
    conversation_style nvarchar(20) NOT NULL CONSTRAINT user_profiles_style_default DEFAULT (N'balanced'),
    onboarding_completed_at datetime2(3) NULL,
    updated_at datetime2(3) NOT NULL CONSTRAINT user_profiles_updated_default DEFAULT SYSUTCDATETIME(),
    CONSTRAINT user_profiles_user_fk FOREIGN KEY (user_id) REFERENCES dbo.users(id) ON DELETE CASCADE,
    CONSTRAINT user_profiles_style_check CHECK (conversation_style IN (N'listening',N'guidance',N'balanced'))
);
CREATE TABLE dbo.conversations (
    id uniqueidentifier NOT NULL PRIMARY KEY,
    user_id uniqueidentifier NOT NULL,
    title nvarchar(150) NOT NULL CONSTRAINT conversations_title_default DEFAULT (N'گفت‌وگوی جدید'),
    created_at datetime2(3) NOT NULL CONSTRAINT conversations_created_default DEFAULT SYSUTCDATETIME(),
    updated_at datetime2(3) NOT NULL CONSTRAINT conversations_updated_default DEFAULT SYSUTCDATETIME(),
    CONSTRAINT conversations_owner_fk FOREIGN KEY (user_id) REFERENCES dbo.users(id) ON DELETE CASCADE,
    CONSTRAINT conversations_id_owner_unique UNIQUE (id,user_id)
);
CREATE INDEX conversations_user_updated_idx ON dbo.conversations(user_id,updated_at DESC,id DESC);
CREATE TABLE dbo.turns (
    id uniqueidentifier NOT NULL PRIMARY KEY,
    conversation_id uniqueidentifier NOT NULL,
    user_id uniqueidentifier NOT NULL,
    input_hash varbinary(32) NOT NULL,
    status nvarchar(20) NOT NULL CONSTRAINT turns_status_default DEFAULT (N'processing'),
    user_message_id uniqueidentifier NULL,
    assistant_message_id uniqueidentifier NULL,
    prompt_version nvarchar(100) NULL,
    completed_at datetime2(3) NULL,
    created_at datetime2(3) NOT NULL CONSTRAINT turns_created_default DEFAULT SYSUTCDATETIME(),
    CONSTRAINT turns_status_check CHECK (status IN (N'processing',N'completed')),
    CONSTRAINT turns_id_conversation_unique UNIQUE (id,conversation_id),
    CONSTRAINT turns_conversation_owner_fk FOREIGN KEY (conversation_id,user_id)
       REFERENCES dbo.conversations(id,user_id) ON DELETE CASCADE
);
CREATE INDEX turns_conversation_status_idx ON dbo.turns(conversation_id,status);
CREATE TABLE dbo.messages (
    id uniqueidentifier NOT NULL PRIMARY KEY,
    conversation_id uniqueidentifier NOT NULL,
    turn_id uniqueidentifier NOT NULL,
    role nvarchar(20) NOT NULL,
    content nvarchar(max) NOT NULL,
    created_at datetime2(3) NOT NULL CONSTRAINT messages_created_default DEFAULT SYSUTCDATETIME(),
    CONSTRAINT messages_role_check CHECK (role IN (N'user',N'assistant')),
    CONSTRAINT messages_content_check CHECK (LEN(content) BETWEEN 1 AND 20000),
    CONSTRAINT messages_turn_conversation_fk FOREIGN KEY (turn_id,conversation_id)
      REFERENCES dbo.turns(id,conversation_id) ON DELETE CASCADE
);
CREATE INDEX messages_conversation_order_idx ON dbo.messages(conversation_id,created_at,id);
