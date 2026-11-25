-- OBS-03: OTel root span correlation only. No user identifiers, raw AI text,
-- Inspector files, endpoint URLs, payloads, credentials or provider errors.
-- Forward-only SQL Server migration; migration runner owns the transaction.
CREATE TABLE dbo.ai_trace_refs (
    id uniqueidentifier NOT NULL CONSTRAINT ai_trace_refs_pk PRIMARY KEY,
    turn_id uniqueidentifier NOT NULL,
    kind varchar(16) NOT NULL,
    job_id uniqueidentifier NULL,
    job_attempt int NULL,
    trace_id varchar(32) COLLATE Latin1_General_100_BIN2 NOT NULL,
    root_span_id varchar(16) COLLATE Latin1_General_100_BIN2 NOT NULL,
    recorded_at datetime2(3) NOT NULL CONSTRAINT ai_trace_refs_recorded_default DEFAULT SYSUTCDATETIME(),
    CONSTRAINT ai_trace_refs_turn_fk FOREIGN KEY (turn_id) REFERENCES dbo.turns(id),
    CONSTRAINT ai_trace_refs_job_fk FOREIGN KEY (job_id) REFERENCES dbo.ai_jobs(id),
    CONSTRAINT ai_trace_refs_kind_check CHECK (kind IN ('turn','memory','reflection')),
    CONSTRAINT ai_trace_refs_job_shape_check CHECK (
        (kind='turn' AND job_id IS NULL AND job_attempt IS NULL)
        OR (kind IN ('memory','reflection') AND job_id IS NOT NULL AND job_attempt IS NOT NULL AND job_attempt BETWEEN 1 AND 1000000)),
    CONSTRAINT ai_trace_refs_trace_id_check CHECK (
        LEN(trace_id)=32 AND trace_id <> REPLICATE('0',32)
        AND trace_id NOT LIKE '%[^0-9a-f]%' COLLATE Latin1_General_100_BIN2),
    CONSTRAINT ai_trace_refs_root_span_check CHECK (
        LEN(root_span_id)=16 AND root_span_id <> REPLICATE('0',16)
        AND root_span_id NOT LIKE '%[^0-9a-f]%' COLLATE Latin1_General_100_BIN2),
    CONSTRAINT ai_trace_refs_unique_root UNIQUE (trace_id,root_span_id)
);
CREATE INDEX ai_trace_refs_turn_page_idx
    ON dbo.ai_trace_refs(turn_id,recorded_at DESC,id DESC)
    INCLUDE (kind,job_id,job_attempt,trace_id,root_span_id);
