-- Deterministic, explicitly synthetic, non-sensitive local demo fixtures.

INSERT INTO dbo.conversations(id,user_id,title) VALUES('a0000000-0000-4000-8000-000000000001','11111111-1111-4111-8111-111111111111',N'نمونه ساختگی: پیاده‌روی');
INSERT INTO dbo.turns(id,conversation_id,user_id,input_hash,status,user_message_id,assistant_message_id,prompt_version,completed_at)
VALUES('b0000000-0000-4000-8000-000000000001','a0000000-0000-4000-8000-000000000001','11111111-1111-4111-8111-111111111111',HASHBYTES('SHA2_256','Synthetic demo 1: plan a short walk.'),N'completed','c0000000-0000-4000-8000-000000000001','d0000000-0000-4000-8000-000000000001',N'supportive-v1',SYSUTCDATETIME());
INSERT INTO dbo.messages(id,conversation_id,turn_id,role,content)
VALUES('c0000000-0000-4000-8000-000000000001','a0000000-0000-4000-8000-000000000001','b0000000-0000-4000-8000-000000000001',N'user',N'Synthetic demo 1: plan a short walk.');
INSERT INTO dbo.messages(id,conversation_id,turn_id,role,content)
VALUES('d0000000-0000-4000-8000-000000000001','a0000000-0000-4000-8000-000000000001','b0000000-0000-4000-8000-000000000001',N'assistant',N'این یک پاسخ ساختگی برای نمایش تاریخچه است. می‌توانیم با یک قدم ساده شروع کنیم.');

INSERT INTO dbo.conversations(id,user_id,title) VALUES('a0000000-0000-4000-8000-000000000002','33333333-3333-4333-8333-333333333333',N'نمونه ساختگی: مطالعه');
INSERT INTO dbo.turns(id,conversation_id,user_id,input_hash,status,user_message_id,assistant_message_id,prompt_version,completed_at)
VALUES('b0000000-0000-4000-8000-000000000002','a0000000-0000-4000-8000-000000000002','33333333-3333-4333-8333-333333333333',HASHBYTES('SHA2_256','Synthetic demo 2: plan a short walk.'),N'completed','c0000000-0000-4000-8000-000000000002','d0000000-0000-4000-8000-000000000002',N'supportive-v1',SYSUTCDATETIME());
INSERT INTO dbo.messages(id,conversation_id,turn_id,role,content)
VALUES('c0000000-0000-4000-8000-000000000002','a0000000-0000-4000-8000-000000000002','b0000000-0000-4000-8000-000000000002',N'user',N'Synthetic demo 2: plan a short walk.');
INSERT INTO dbo.messages(id,conversation_id,turn_id,role,content)
VALUES('d0000000-0000-4000-8000-000000000002','a0000000-0000-4000-8000-000000000002','b0000000-0000-4000-8000-000000000002',N'assistant',N'این یک پاسخ ساختگی برای نمایش تاریخچه است. می‌توانیم با یک قدم ساده شروع کنیم.');

INSERT INTO dbo.conversations(id,user_id,title) VALUES('a0000000-0000-4000-8000-000000000003','33333333-3333-4333-8333-333333333333',N'نمونه ساختگی: برنامه روزانه');
INSERT INTO dbo.turns(id,conversation_id,user_id,input_hash,status,user_message_id,assistant_message_id,prompt_version,completed_at)
VALUES('b0000000-0000-4000-8000-000000000003','a0000000-0000-4000-8000-000000000003','33333333-3333-4333-8333-333333333333',HASHBYTES('SHA2_256','Synthetic demo 3: plan a short walk.'),N'completed','c0000000-0000-4000-8000-000000000003','d0000000-0000-4000-8000-000000000003',N'supportive-v1',SYSUTCDATETIME());
INSERT INTO dbo.messages(id,conversation_id,turn_id,role,content)
VALUES('c0000000-0000-4000-8000-000000000003','a0000000-0000-4000-8000-000000000003','b0000000-0000-4000-8000-000000000003',N'user',N'Synthetic demo 3: plan a short walk.');
INSERT INTO dbo.messages(id,conversation_id,turn_id,role,content)
VALUES('d0000000-0000-4000-8000-000000000003','a0000000-0000-4000-8000-000000000003','b0000000-0000-4000-8000-000000000003',N'assistant',N'این یک پاسخ ساختگی برای نمایش تاریخچه است. می‌توانیم با یک قدم ساده شروع کنیم.');
