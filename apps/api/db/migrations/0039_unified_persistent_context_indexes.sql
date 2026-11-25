-- SQL Server 2022 additive indexes; the migration runner owns the transaction.
-- Additive indexes only. Original records, rowversions and evidence are retained.
CREATE INDEX IX_ai_memory_baseline ON dbo.ai_memory_items(user_id,status,epistemic,pinned DESC,importance DESC,updated_at DESC)
 INCLUDE(kind,sensitivity,valid_from,valid_to,superseded_by_id) WHERE deleted_at IS NULL;
CREATE INDEX IX_ai_memory_evidence_owner_item ON dbo.ai_memory_evidence(user_id,item_id,evidence_at DESC) INCLUDE(ref_kind,ref_id,role);
CREATE INDEX IX_ai_formulations_owner_latest ON dbo.ai_formulations(user_id,thread_id,version DESC) WHERE deleted_at IS NULL;

-- Prevent historical retries from reading state produced by a later turn.
CREATE INDEX IX_ai_turns_owner_completed_order ON dbo.turns(user_id,admission_order) WHERE status=N'completed';
