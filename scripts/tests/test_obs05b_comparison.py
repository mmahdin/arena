"""OBS-05B regression checks: exact trace readback and content-free comparison."""
from __future__ import annotations

from datetime import datetime, timedelta, timezone
import json
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch
from urllib.parse import parse_qs, urlparse

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "scripts" / "observability"))
import compare_dual_traces as compare

TRACE_ID = "ab" * 16
ID_ROOT = "11" * 8
ID_RAG = "22" * 8
ID_LLM = "33" * 8


def phoenix_span(sid, name, kind, parent=None, attrs=None, events=None):
    return {"context": {"trace_id": TRACE_ID, "span_id": sid}, "parent_id": parent,
            "name": name, "span_kind": kind, "attributes": attrs or {}, "events": events or []}


def langfuse_observation(sid, name, kind, parent=None, input=None, output=None, metadata=None,
                         inputUsage=None, outputUsage=None):
    return {"traceId": TRACE_ID, "id": sid, "parentObservationId": parent,
            "name": name, "type": kind, "input": input, "output": output,
            "metadata": metadata or {}, "inputUsage": inputUsage, "outputUsage": outputUsage}


class ComparisonModelTest(unittest.TestCase):
    def sample(self):
        raw = "SYNTHETIC_SECRET_SAMPLE_INPUT_DO_NOT_LOG"
        p = [phoenix_span(ID_ROOT, "theia.ai.turn", "AGENT", attrs={"input.value": raw}),
             phoenix_span(ID_RAG, "theia.ai.application.retrieval.service.Retrieve", "RETRIEVER", parent=ID_ROOT,
                          attrs={"input.value": "SYNTHETIC_QUERY", "output.value": '{"selected": ["doc"]}',
                                 "retrieval.documents.0.document.id": "SYNTHETIC_DOC",
                                 "retrieval.documents.0.document.content": "SYNTHETIC_DOC_BODY"},
                          events=[{"name": "theia.rag.bm25_search"}]),
             phoenix_span(ID_LLM, "gen_ai.client", "LLM", parent=ID_ROOT,
                          attrs={"input.value": "SYNTHETIC_PROMPT", "output.value": "SYNTHETIC_RESULT",
                                 "llm.token_count.prompt": 23, "llm.token_count.completion": 7})]
        l = [langfuse_observation(ID_ROOT, "theia.ai.turn", "AGENT", input=raw),
             langfuse_observation(ID_RAG, "theia.ai.application.retrieval.service.Retrieve", "RETRIEVER",
                                  parent=ID_ROOT, input="SYNTHETIC_QUERY", output='{"selected":["doc"]}',
                                  metadata={"retrieval.documents.0.document.id": "SYNTHETIC_DOC",
                                            "retrieval.documents.0.document.content": "SYNTHETIC_DOC_BODY"}),
             langfuse_observation(ID_LLM, "gen_ai.client", "GENERATION", parent=ID_ROOT,
                                  input="SYNTHETIC_PROMPT", output="SYNTHETIC_RESULT", inputUsage=23, outputUsage=7)]
        return p, l, raw

    def test_parity_does_not_publish_contents_or_choose_product(self):
        p, l, secret = self.sample()
        report = compare.compare(TRACE_ID, p, l)
        self.assertEqual(report["result"], "PASS")
        self.assertEqual(report["platform_choice"], "NOT_DETERMINED")
        self.assertEqual(report["phoenix"]["rag_events"], 1)
        self.assertEqual(report["parity"]["shared_span_ids"], 3)
        serialized = json.dumps(report)
        for forbidden in (secret, "SYNTHETIC_RESULT", "SYNTHETIC_PROMPT", "SYNTHETIC_DOC_BODY"):
            self.assertNotIn(forbidden, serialized)

    def test_lost_or_reparented_spans_and_missing_io_fail(self):
        p, l, _ = self.sample()
        l.pop(1)
        report = compare.compare(TRACE_ID, p, l)
        self.assertEqual(report["result"], "FAIL")
        self.assertIn("SPAN_ID_SET_MISMATCH", report["issues"])
        self.assertEqual(report["parity"]["only_phoenix_span_ids"], [ID_RAG])
        l = self.sample()[1]
        l[2]["parentObservationId"] = ID_RAG
        self.assertIn("MISMATCH_PARENTS", compare.compare(TRACE_ID, p, l)["issues"])
        l[2]["parentObservationId"] = ID_ROOT
        l[2]["output"] = None
        self.assertIn("MISMATCH_OUTPUT_PRESENCE", compare.compare(TRACE_ID, p, l)["issues"])

    def test_token_disagreement_and_retrieval_metadata_mismatch(self):
        p, l, _ = self.sample()
        l[2]["inputUsage"] = 22
        self.assertIn("MISMATCH_KNOWN_TOKEN_USAGE", compare.compare(TRACE_ID, p, l)["issues"])
        l[2]["inputUsage"] = 23
        l[1]["metadata"]["retrieval.documents.0.document.content"] = "TRUNCATED"
        report = compare.compare(TRACE_ID, p, l)
        self.assertIn("MISMATCH_RAG_DOCUMENT_FIELDS_EQUAL", report["issues"])
        self.assertNotIn("SYNTHETIC_DOC_BODY", json.dumps(report))
        l[1]["metadata"] = {}
        self.assertIn("MISMATCH_RAG_DOCUMENT_PRESENCE", compare.compare(TRACE_ID, p, l)["issues"])

    def test_reject_duplicate_and_foreign_span(self):
        p, l, _ = self.sample()
        with self.assertRaises(compare.CompareError):
            compare.compare(TRACE_ID, p + p[:1], l)
        p[1]["context"]["trace_id"] = "cd" * 16
        with self.assertRaises(compare.CompareError):
            compare.compare(TRACE_ID, p, l)

    def test_semantic_type_mismatch_is_reported(self):
        p, l, _ = self.sample()
        l[1]["type"] = "SPAN"
        report = compare.compare(TRACE_ID, p, l)
        self.assertIn("SEMANTIC_TYPE_MISMATCH", report["issues"])
        self.assertEqual(report["parity"]["semantic_mismatches_by_phoenix_role"], {"RETRIEVER": 1})

    def test_langfuse_v4_namespaced_retrieval_attributes(self):
        p, l, _ = self.sample()
        l[1]["metadata"] = {"attributes." + key: value for key, value in l[1]["metadata"].items()}
        report = compare.compare(TRACE_ID, p, l)
        self.assertEqual(report["result"], "PASS")
        self.assertEqual(report["langfuse"]["rag_document_spans"], 1)

    def test_long_content_is_not_in_report_or_logs(self):
        p, l, _ = self.sample()
        very_long = "highly-sensitive-synthetic-" * 500
        p[0]["attributes"]["input.value"] = very_long
        l[0]["input"] = very_long
        report = compare.compare(TRACE_ID, p, l)
        self.assertTrue(report["parity"]["checks"]["equal_inputs"])
        self.assertNotIn("highly-sensitive", json.dumps(report))


class APIReadbackTest(unittest.TestCase):
    def test_phoenix_query_uses_documented_repeated_trace_id_and_cursor(self):
        calls = []
        def mock_request(url, auth, service):
            calls.append((url, auth, service))
            if len(calls) == 1:
                return {"data": [phoenix_span(ID_ROOT, "root", "AGENT")], "next_cursor": "cursor-1"}
            return {"data": [phoenix_span(ID_RAG, "retrieval", "RETRIEVER")], "next_cursor": None}
        with patch.object(compare, "_request_json", side_effect=mock_request):
            rows = compare.read_phoenix(TRACE_ID, "theia-obs05a-dual", "dummy")
        self.assertEqual(len(rows), 2)
        first = parse_qs(urlparse(calls[0][0]).query)
        self.assertEqual(first["trace_id"], [TRACE_ID])
        self.assertNotIn("trace_ids", first)
        self.assertEqual(parse_qs(urlparse(calls[1][0]).query)["cursor"], ["cursor-1"])
        self.assertEqual(calls[0][1], "Bearer dummy")

    def test_langfuse_v4_time_bounds_field_groups_cursor_and_source_auth(self):
        calls = []
        now = datetime.now(timezone.utc)
        def mock_request(url, auth, service):
            calls.append((url, auth, service))
            if len(calls) == 1:
                return {"data": [], "meta": {"cursor": "next-1"}}
            return {"data": [langfuse_observation(ID_ROOT, "root", "SPAN")], "meta": {"cursor": None}}
        credentials = {"THEIA_LANGFUSE_OTLP_ENDPOINT": "https://cloud.langfuse.com/api/public/otel",
                       "THEIA_LANGFUSE_AUTH_BASE64": "EXAMPLEBASE64"}
        with patch.object(compare, "_request_json", side_effect=mock_request):
            rows = compare.read_langfuse(TRACE_ID, credentials, now - timedelta(days=1), now)
        self.assertEqual(len(rows), 1)
        query = parse_qs(urlparse(calls[0][0]).query)
        self.assertEqual(query["traceId"], [TRACE_ID])
        self.assertIn("io", query["fields"][0])
        self.assertIn("metadata", query["fields"][0])
        self.assertIn("usage", query["fields"][0])
        self.assertIn("fromStartTime", query)
        self.assertEqual(parse_qs(urlparse(calls[1][0]).query)["cursor"], ["next-1"])
        self.assertEqual(calls[0][1], "Basic EXAMPLEBASE64")

    def test_read_override_rejects_nonlocal_or_confusing_urls(self):
        now = datetime.now(timezone.utc)
        credentials = {"THEIA_LANGFUSE_OTLP_ENDPOINT": "http://langfuse-web:3000/api/public/otel",
                       "THEIA_LANGFUSE_AUTH_BASE64": "EXAMPLEBASE64"}
        for url in [None, "https://evil.example", "http://evil.example:3000", "http://localhost:3000",
                    "http://127.0.0.1:3000/other", "http://127.0.0.1:3000?key=1",
                    "http://127.0.0.1:3000@evil.example:3000", "http://127.0.0.1:3000#fragment"]:
            with self.subTest(url=url):
                with self.assertRaises(compare.CompareError):
                    compare.read_langfuse(TRACE_ID, credentials, now - timedelta(hours=1), now, read_origin=url)
        with patch.object(compare, "_request_json", return_value={"data": [], "meta": {}}) as mock:
            compare.read_langfuse(TRACE_ID, credentials, now - timedelta(hours=1), now,
                                  read_origin="http://127.0.0.1:3000")
        self.assertTrue(mock.call_args.args[0].startswith("http://127.0.0.1:3000/api/public/v2/observations?"))

    def test_polling_waits_for_complete_fixture_instead_of_first_partial_trace(self):
        phoenix_small = [phoenix_span(ID_ROOT, "root", "AGENT")]
        phoenix_full = [phoenix_span(ID_ROOT, "root", "AGENT"),
                        phoenix_span(ID_RAG, "retrieval", "RETRIEVER", parent=ID_ROOT)]
        langfuse_small = [langfuse_observation(ID_ROOT, "root", "AGENT")]
        langfuse_full = [langfuse_observation(ID_ROOT, "root", "AGENT"),
                         langfuse_observation(ID_RAG, "retrieval", "RETRIEVER", parent=ID_ROOT)]
        now = datetime.now(timezone.utc)
        with patch.object(compare, "read_phoenix", side_effect=[phoenix_small, phoenix_full]) as phoenix, \
             patch.object(compare, "read_langfuse", side_effect=[langfuse_small, langfuse_full]) as langfuse:
            got_p, got_l = compare.collect(TRACE_ID, "theia-obs05a-dual", {}, "key",
                                           start=now - timedelta(hours=1), end=now,
                                           attempts=2, delay=0, min_spans=2)
        self.assertEqual((len(got_p), len(got_l)), (2, 2))
        self.assertEqual(phoenix.call_count, 2)
        self.assertEqual(langfuse.call_count, 2)

    def test_json_limit_and_repeated_pagination_abort_without_reflecting_content(self):
        with self.assertRaises(compare.CompareError):
            compare._json_response(b'"sensitive"', "Phoenix")
        with self.assertRaises(compare.CompareError):
            compare._json_response(b'x' * (compare.MAX_RESPONSE_BYTES + 1), "Phoenix")
        with patch.object(compare, "_request_json", return_value={"data": [], "next_cursor": "loop"}):
            with self.assertRaisesRegex(compare.CompareError, "repeated pagination cursor"):
                compare.read_phoenix(TRACE_ID, "safe", "key")


def wire_fields(data):
    """Small stdlib-only protobuf wire reader for fixture shape regression."""
    index = 0
    result = {}
    def varint():
        nonlocal index
        value = 0
        for shift in range(0, 70, 7):
            if index >= len(data): raise ValueError("Truncated protobuf varint")
            chunk = data[index]
            index += 1
            value |= (chunk & 0x7f) << shift
            if not chunk & 0x80: return value
        raise ValueError("Overlong protobuf varint")
    while index < len(data):
        field = varint()
        key, wire = field >> 3, field & 7
        if wire == 0:
            value = varint()
        elif wire == 1:
            if index + 8 > len(data): raise ValueError("Truncated 64-bit field")
            value = data[index:index + 8]; index += 8
        elif wire == 2:
            length = varint()
            if index + length > len(data): raise ValueError("Truncated length field")
            value = data[index:index + length]; index += length
        elif wire == 5:
            if index + 4 > len(data): raise ValueError("Truncated 32-bit field")
            value = data[index:index + 4]; index += 4
        else:
            raise ValueError("Unsupported wire type")
        result.setdefault(key, []).append(value)
    return result


class FixtureAndReportTest(unittest.TestCase):
    def test_fixture_project_matches_collector_project_header(self):
        collector = (ROOT / "infra/observability/otelcol-dual.yaml").read_text()
        self.assertIn("x-project-name: " + compare.FIXTURE_PROJECT, collector)

    def test_one_fixture_contains_ten_spans_and_no_credentials(self):
        data, ids = compare.synthetic_fixture(TRACE_ID, now_ns=1800000000000000000)
        self.assertEqual(len(ids), 10)
        root = wire_fields(data)
        resource_spans = wire_fields(root[1][0])
        scopes = wire_fields(resource_spans[2][0])
        spans = [wire_fields(blob) for blob in scopes[2]]
        self.assertEqual(len(spans), 10)
        span_ids = {span[2][0].hex() for span in spans}
        self.assertEqual(span_ids, ids)
        self.assertEqual(sum(4 not in span for span in spans), 1)
        self.assertTrue(all(span.get(4, [b''])[0].hex() in ids for span in spans if 4 in span))
        keys = []
        for span in spans:
            for wrapped in span.get(9, []):
                attr = wire_fields(wrapped)
                self.assertIn(1, attr)
                self.assertIn(2, attr)
                keys.append(attr[1][0].decode('utf-8'))
        self.assertIn('llm.token_count.prompt', keys)
        self.assertIn('openinference.span.kind', keys)
        self.assertIn('retrieval.documents.0.document.content', keys)
        self.assertIn(bytes.fromhex(TRACE_ID), data)
        self.assertEqual(data.count(b"gen_ai.client"), 2)
        for expected in [b"theia.rag.bm25_search", b"theia.rag.vector_search", b"theia.rag.rrf",
                         b"openinference.span.kind", b"SYNTHETIC_DOC_A"]:
            self.assertIn(expected, data)
        for sensitive in [b"Authorization", b"sk-lf-", b"Bearer ", b"patient"]:
            self.assertNotIn(sensitive, data)
        with self.assertRaises(ValueError):
            compare.synthetic_fixture("bad")

    def test_report_is_private_and_does_not_overwrite(self):
        p = {"result": "PASS", "traceId": TRACE_ID, "issues": []}
        with tempfile.TemporaryDirectory() as folder:
            path = Path(folder) / "comparison.json"
            compare.save_report(p, str(path))
            self.assertEqual(path.stat().st_mode & 0o777, 0o600)
            self.assertEqual(json.loads(path.read_text()), p)
            with self.assertRaisesRegex(compare.CompareError, "Refusing to overwrite"):
                compare.save_report(p, str(path))
            target = Path(folder) / "alias.json"
            target.symlink_to(path)
            with self.assertRaisesRegex(compare.CompareError, "regular file"):
                compare.save_report(p, str(target))


if __name__ == "__main__":
    unittest.main()
