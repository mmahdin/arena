-- BO-08: immutable sanitized WP-15 synthetic evaluation reports.
-- This is NOT a transcript store, test executor, release approval or a clinical
-- review ledger. Report source identity is UNATTESTED in this first phase.
CREATE TABLE dbo.admin_eval_runs (
    id uniqueidentifier NOT NULL CONSTRAINT admin_eval_runs_pk PRIMARY KEY,
    report_sha256 char(64) COLLATE Latin1_General_100_BIN2 NOT NULL,
    imported_at datetime2(3) NOT NULL CONSTRAINT admin_eval_runs_imported_at_default DEFAULT SYSUTCDATETIME(),
    imported_by_staff_id uniqueidentifier NOT NULL,
    baseline_run_id uniqueidentifier NULL,
    method varchar(32) NOT NULL,
    evidence_type varchar(32) NOT NULL,
    status varchar(16) NOT NULL,
    scenario_count int NOT NULL,
    failed_scenarios int NOT NULL,
    blockers int NOT NULL,
    regression_count int NOT NULL,
    totals_json nvarchar(1024) NOT NULL,
    coverage_json nvarchar(1024) NOT NULL,
    report_json nvarchar(max) NOT NULL,
    baseline_regressions_json nvarchar(max) NOT NULL,
    artifact_hash_verified bit NOT NULL,
    source_attested bit NOT NULL CONSTRAINT admin_eval_runs_source_default DEFAULT 0,
    clinically_approved bit NOT NULL CONSTRAINT admin_eval_runs_clinical_default DEFAULT 0,
    CONSTRAINT admin_eval_runs_staff_fk FOREIGN KEY (imported_by_staff_id) REFERENCES dbo.staff_accounts(id),
    CONSTRAINT admin_eval_runs_baseline_fk FOREIGN KEY (baseline_run_id) REFERENCES dbo.admin_eval_runs(id),
    CONSTRAINT admin_eval_runs_method_check CHECK (method IN ('synthetic_recorded','synthetic_trace_replay','external_executor')),
    CONSTRAINT admin_eval_runs_evidence_check CHECK (evidence_type IN ('recorded_fixture','partial_trace_replay','external_executor_claimed')),
    CONSTRAINT admin_eval_runs_status_check CHECK (status IN ('passed','failed','blocked')),
    CONSTRAINT admin_eval_runs_counts_check CHECK (scenario_count BETWEEN 1 AND 512 AND failed_scenarios BETWEEN 0 AND scenario_count AND blockers>=0 AND regression_count>=0),
    CONSTRAINT admin_eval_runs_no_attestation_check CHECK (artifact_hash_verified=1 AND source_attested=0 AND clinically_approved=0),
    CONSTRAINT admin_eval_runs_json_check CHECK (ISJSON(totals_json)=1 AND ISJSON(coverage_json)=1 AND ISJSON(report_json)=1 AND ISJSON(baseline_regressions_json)=1)
);
CREATE UNIQUE INDEX admin_eval_runs_artifact_baseline_unique ON dbo.admin_eval_runs(report_sha256,baseline_run_id);
CREATE INDEX admin_eval_runs_date_idx ON dbo.admin_eval_runs(imported_at DESC,id DESC) INCLUDE (status,method,scenario_count,failed_scenarios,blockers,regression_count);
-- Tamper resistance for the application account; privileged DBAs can bypass
-- this trigger. There is no claim of cryptographic immutability.
EXEC(N'CREATE TRIGGER dbo.admin_eval_runs_block_mutation
ON dbo.admin_eval_runs AFTER UPDATE, DELETE
AS BEGIN
    SET NOCOUNT ON;
    THROW 51030, ''admin eval runs are append only'', 1;
END');
