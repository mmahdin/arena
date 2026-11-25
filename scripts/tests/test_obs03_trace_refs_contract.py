"""OBS-03 source-level guardrails; Go and disposable SQL tests remain mandatory."""
from pathlib import Path
import re
import unittest
import yaml

ROOT=Path(__file__).resolve().parents[2]

def read(path):
    return (ROOT/path).read_text()

class TraceReferenceContract(unittest.TestCase):
    def test_metadata_only_schema_and_forward_only_migration(self):
        migration=read('apps/api/db/migrations/0034_ai_trace_refs.sql')
        self.assertIn('CREATE TABLE dbo.ai_trace_refs',migration)
        self.assertIn('FOREIGN KEY (turn_id)',migration)
        self.assertIn('FOREIGN KEY (job_id)',migration)
        self.assertIn('CONSTRAINT ai_trace_refs_unique_root',migration)
        self.assertIn('CHECK (kind IN',migration)
        forbidden=re.compile(r'\b(?:payload_json|request_meta_json|result_meta_json|prompt_text|response_text|content_text|user_id|api_key|inspector_path|phoenix_url)\b',re.I)
        sql=re.sub(r'--[^\n]*','',migration)
        self.assertNotRegex(sql,forbidden)
        self.assertNotRegex(sql,re.compile(r'^\s*(?:GO|BEGIN\s+TRAN|COMMIT)\b',re.M|re.I))
    def test_audited_staff_route_and_contract(self):
        spec=yaml.safe_load(read('contracts/openapi/openapi.yaml'))
        api=spec['paths']['/api/v1/admin/turns/{turnId}/trace-references']['get']
        self.assertEqual(api['security'],[{'staffSession':[]}])
        self.assertEqual(set(['200','400','401','404','503'])-set(api['responses']),set())
        schema=spec['components']['schemas']['AdminTraceReference']
        self.assertEqual(set(schema['properties']), {'id','kind','traceId','rootSpanId','jobId','jobAttempt','recordedAt'})
        self.assertIn('adminAPI.Use(bff.RequireStaff())',read('apps/api/internal/app/composition/api.go'))
        self.assertIn('adminAPI.GET("/turns/:turnId/trace-references", adminModule.TraceReferences)',read('apps/api/internal/app/composition/api.go'))
        self.assertIn('ActionTurnTraceRefsRead',read('apps/api/internal/modules/admin/domain/audit.go'))
        app=read('apps/api/internal/modules/admin/application/trace_refs.go')
        self.assertLess(app.index('s.inspectorAudit('),app.index('s.traceRefs.ListTraceReferences('))
    def test_recording_only_after_durable_admission_and_real_claim(self):
        orch=read('apps/api/internal/modules/ai/application/orchestrator/orchestrator.go')
        self.assertGreater(orch.index('trace.RecordAdmittedTurn('),orch.index('if a.Queued || !contract.ValidID(a.LeaseToken)'))
        for kind in ['memory','reflection']:
            worker=read(f'apps/api/internal/modules/ai/application/{kind}/worker.go')
            self.assertGreater(worker.index('trace.RecordClaimedJob('),worker.index('trace.StartJob('))
        runtime=read('apps/api/internal/platform/telemetry/telemetry.go')
        self.assertIn('context.WithValue(ctx, rootSpanKey{}, s.SpanContext())',runtime)
        self.assertNotIn('oteltrace.SpanFromContext(ctx)',runtime[runtime.index('func ActiveRootIDs('):])
    def test_admin_sql_reader_excludes_private_data(self):
        sql=read('apps/api/internal/modules/admin/adapters/mssql/trace_refs.go')
        statements=re.findall(r'`(SELECT\b.*?)`',sql,re.S|re.I)
        self.assertTrue(statements)
        for query in statements:
            self.assertNotRegex(query,re.compile(r'\b(?:user_id|payload_json|prompt|response|message|secret|api_key)\b',re.I))
        self.assertIn('AND lifecycle_version=1',sql)
        self.assertIn('ORDER BY recorded_at DESC,id DESC',sql)
        self.assertIn('f.Limit+1',sql.replace(' ','').replace('\n',''))

if __name__=='__main__':unittest.main()
