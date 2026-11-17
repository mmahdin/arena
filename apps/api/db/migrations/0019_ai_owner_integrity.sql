-- Composite references enforce ownership; only user FKs cascade.
ALTER TABLE dbo.ai_sessions ADD CONSTRAINT ai_sessions_owner_unique UNIQUE(id,user_id);
ALTER TABLE dbo.ai_threads ADD CONSTRAINT ai_threads_owner_unique UNIQUE(id,user_id);
ALTER TABLE dbo.ai_protocol_instances ADD CONSTRAINT ai_protocol_instances_owner_unique UNIQUE(id,user_id);
ALTER TABLE dbo.ai_memory_items ADD CONSTRAINT ai_memory_items_owner_unique UNIQUE(id,user_id);
ALTER TABLE dbo.ai_turn_events ADD CONSTRAINT ai_turn_events_owner_unique UNIQUE(id,user_id);
ALTER TABLE dbo.ai_sessions ADD CONSTRAINT ai_sessions_focus_thread_id_owner FOREIGN KEY(focus_thread_id,user_id) REFERENCES dbo.ai_threads(id,user_id);
ALTER TABLE dbo.ai_turn_context ADD CONSTRAINT ai_turn_context_turn_id_owner FOREIGN KEY(turn_id,user_id) REFERENCES dbo.turns(id,user_id);
ALTER TABLE dbo.ai_turn_context ADD CONSTRAINT ai_turn_context_session_id_owner FOREIGN KEY(session_id,user_id) REFERENCES dbo.ai_sessions(id,user_id);
ALTER TABLE dbo.ai_formulations ADD CONSTRAINT ai_formulations_thread_id_owner FOREIGN KEY(thread_id,user_id) REFERENCES dbo.ai_threads(id,user_id);
ALTER TABLE dbo.ai_protocol_instances ADD CONSTRAINT ai_protocol_instances_thread_id_owner FOREIGN KEY(thread_id,user_id) REFERENCES dbo.ai_threads(id,user_id);
ALTER TABLE dbo.ai_memory_items ADD CONSTRAINT ai_memory_items_supersedes_id_owner FOREIGN KEY(supersedes_id,user_id) REFERENCES dbo.ai_memory_items(id,user_id);
ALTER TABLE dbo.ai_memory_items ADD CONSTRAINT ai_memory_items_superseded_by_id_owner FOREIGN KEY(superseded_by_id,user_id) REFERENCES dbo.ai_memory_items(id,user_id);
ALTER TABLE dbo.ai_memory_evidence ADD CONSTRAINT ai_memory_evidence_item_id_owner FOREIGN KEY(item_id,user_id) REFERENCES dbo.ai_memory_items(id,user_id);
ALTER TABLE dbo.ai_memory_evidence ADD CONSTRAINT ai_memory_evidence_session_id_owner FOREIGN KEY(session_id,user_id) REFERENCES dbo.ai_sessions(id,user_id);
ALTER TABLE dbo.ai_hypothesis_cooldowns ADD CONSTRAINT ai_hypothesis_cooldowns_thread_id_owner FOREIGN KEY(thread_id,user_id) REFERENCES dbo.ai_threads(id,user_id);
ALTER TABLE dbo.ai_hypothesis_cooldowns ADD CONSTRAINT ai_hypothesis_cooldowns_rejected_item_id_owner FOREIGN KEY(rejected_item_id,user_id) REFERENCES dbo.ai_memory_items(id,user_id);
ALTER TABLE dbo.ai_turn_events ADD CONSTRAINT ai_turn_events_turn_id_owner FOREIGN KEY(turn_id,user_id) REFERENCES dbo.turns(id,user_id);
ALTER TABLE dbo.ai_jobs ADD CONSTRAINT ai_jobs_turn_id_owner FOREIGN KEY(turn_id,user_id) REFERENCES dbo.turns(id,user_id);
ALTER TABLE dbo.ai_turn_traces ADD CONSTRAINT ai_turn_traces_turn_id_owner FOREIGN KEY(turn_id,user_id) REFERENCES dbo.turns(id,user_id);
ALTER TABLE dbo.ai_jobs ADD CONSTRAINT ai_jobs_status CHECK(status IN(N'queued',N'running',N'completed',N'failed',N'cancelled')),
 CONSTRAINT ai_jobs_attempts CHECK(attempts>=0 AND max_attempts>0);
ALTER TABLE dbo.ai_sessions ADD CONSTRAINT ai_sessions_counts CHECK(turn_count>=0 AND consecutive_questions>=0 AND version>0),
 CONSTRAINT ai_sessions_mode CHECK(mode IN(N'venting',N'exploring',N'problem_solving',N'exercise',N'checking_in',N'safety',N'small_talk'));
ALTER TABLE dbo.ai_protocol_instances ADD CONSTRAINT ai_protocol_counts CHECK(attempts>=0 AND stuck_count>=0 AND version>0);
ALTER TABLE dbo.ai_open_loops ADD CONSTRAINT ai_loop_status CHECK(status IN(N'open',N'closed',N'expired'));
ALTER TABLE dbo.ai_follow_ups ADD CONSTRAINT ai_follow_status CHECK(status IN(N'scheduled',N'cancelled',N'sent',N'skipped'));
EXEC(N'CREATE TRIGGER dbo.ai_evidence_owner ON dbo.ai_memory_evidence AFTER INSERT, UPDATE AS BEGIN SET NOCOUNT ON; IF EXISTS(SELECT 1 FROM inserted i WHERE
(i.ref_kind=N''message'' AND NOT EXISTS(SELECT 1 FROM dbo.messages m JOIN dbo.conversations c ON c.id=m.conversation_id WHERE m.id=i.ref_id AND c.user_id=i.user_id)) OR
(i.ref_kind=N''item'' AND NOT EXISTS(SELECT 1 FROM dbo.ai_memory_items m WHERE m.id=i.ref_id AND m.user_id=i.user_id)) OR
(i.ref_kind=N''event'' AND NOT EXISTS(SELECT 1 FROM dbo.ai_turn_events e WHERE e.id=i.ref_id AND e.user_id=i.user_id))) THROW 51001, N''AI reference owner mismatch'', 1; END');
EXEC(N'CREATE TRIGGER dbo.ai_embedding_owner ON dbo.ai_embeddings AFTER INSERT, UPDATE AS BEGIN SET NOCOUNT ON; IF EXISTS(SELECT 1 FROM inserted i WHERE
(i.ref_kind=N''message'' AND NOT EXISTS(SELECT 1 FROM dbo.messages m JOIN dbo.conversations c ON c.id=m.conversation_id WHERE m.id=i.ref_id AND c.user_id=i.user_id)) OR
(i.ref_kind=N''memory'' AND NOT EXISTS(SELECT 1 FROM dbo.ai_memory_items m WHERE m.id=i.ref_id AND m.user_id=i.user_id))) THROW 51001, N''AI reference owner mismatch'', 1; END');
EXEC(N'CREATE TRIGGER dbo.ai_protocol_consent_owner ON dbo.ai_protocol_instances AFTER INSERT, UPDATE AS BEGIN SET NOCOUNT ON; IF EXISTS(SELECT 1 FROM inserted i WHERE i.consent_message_id IS NOT NULL AND NOT EXISTS(SELECT 1 FROM dbo.messages m JOIN dbo.conversations c ON c.id=m.conversation_id WHERE m.id=i.consent_message_id AND c.user_id=i.user_id AND m.role=N''user'')) THROW 51001, N''AI reference owner mismatch'', 1; END');

-- Privacy invalidation is conservative until full provenance is implemented.
EXEC(N'CREATE PROCEDURE dbo.ai_purge_user_derivatives @user_id uniqueidentifier AS BEGIN SET NOCOUNT ON;
DELETE dbo.ai_user_turn_leases WHERE user_id=@user_id;
DELETE dbo.ai_embeddings WHERE user_id=@user_id;
DELETE dbo.ai_memory_evidence WHERE user_id=@user_id;
DELETE dbo.ai_hypothesis_cooldowns WHERE user_id=@user_id;
DELETE dbo.ai_memory_items WHERE user_id=@user_id;
DELETE dbo.ai_formulations WHERE user_id=@user_id;
DELETE dbo.ai_protocol_instances WHERE user_id=@user_id;
DELETE dbo.ai_turn_context WHERE user_id=@user_id;
DELETE dbo.ai_turn_events WHERE user_id=@user_id;
DELETE dbo.ai_jobs WHERE user_id=@user_id;
DELETE dbo.ai_outbox WHERE user_id=@user_id;
DELETE dbo.ai_turn_traces WHERE user_id=@user_id;
DELETE dbo.ai_sessions WHERE user_id=@user_id;
DELETE dbo.ai_threads WHERE user_id=@user_id;
DELETE dbo.ai_open_loops WHERE user_id=@user_id;
DELETE dbo.ai_follow_ups WHERE user_id=@user_id;
DELETE dbo.ai_memory_tombstones WHERE user_id=@user_id;
DELETE dbo.ai_user_state WHERE user_id=@user_id;
END');
ALTER TABLE dbo.ai_memory_items ADD CONSTRAINT ai_memory_sensitive_ciphertext CHECK (
 sensitivity<>N'highly_sensitive' OR (content IS NULL AND content_enc IS NOT NULL)),
 CONSTRAINT ai_memory_counts CHECK(support_n>=0 AND contradict_n>=0 AND sessions_n>=0);
CREATE UNIQUE INDEX ai_open_loop_reference ON dbo.ai_open_loops(user_id,kind,ref_id) WHERE status=N'open' AND ref_id IS NOT NULL;
EXEC(N'CREATE TRIGGER dbo.ai_context_conversation ON dbo.ai_turn_context AFTER INSERT, UPDATE AS BEGIN SET NOCOUNT ON;
 IF EXISTS(SELECT 1 FROM inserted i JOIN dbo.turns t ON t.id=i.turn_id JOIN dbo.ai_sessions s ON s.id=i.session_id WHERE t.conversation_id<>s.conversation_id)
 THROW 51002,N''AI session conversation mismatch'',1; END');
-- Block unsupported partial raw deletion while it would orphan derived data.
-- Conversation.Delete explicitly invalidates the graph before deleting messages.
EXEC(N'CREATE TRIGGER dbo.ai_raw_delete_guard ON dbo.messages AFTER DELETE AS BEGIN SET NOCOUNT ON;
 IF EXISTS(SELECT 1 FROM deleted d JOIN dbo.conversations c ON c.id=d.conversation_id
 JOIN dbo.ai_user_state s ON s.user_id=c.user_id)
 THROW 51003,N''Use the owner-scoped deletion unit of work'',1; END');
