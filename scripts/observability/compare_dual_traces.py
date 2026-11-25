#!/usr/bin/env python3
"""OBS-05B: compare one synthetic W3C trace ingested by Phoenix and Langfuse.

Only synthetic-data workspaces are supported. Raw traces are inspected in memory;
reports contain IDs, counts and equality flags, never prompts or retrieved text.
No model or RAG execution is performed by this tool.
"""
from __future__ import annotations

import argparse
from collections import Counter
from datetime import datetime, timedelta, timezone
import json
import os
from pathlib import Path
import re
import time
import urllib.parse
import urllib.error
import urllib.request
import uuid

from init_dual_otlp_env import read_env as read_dual
from init_local_phoenix_env import read_env as read_phoenix_env
from smoke_dual_otlp import COLLECTOR, PHOENIX, OPENER, request
from smoke_phoenix import _attr, _field, _fixed64, _str, _varint

SCHEMA = "theia.obs05b.parity.v1"
# Must match OBS-05A's Phoenix exporter x-project-name header: that header
# pins all dual-export spans to the same existing private project.
FIXTURE_PROJECT = "theia-obs05a-dual"
TRACE_RE = re.compile(r"[0-9a-f]{32}\Z")
SPAN_RE = re.compile(r"[0-9a-f]{16}\Z")
PROJECT_RE = re.compile(r"[A-Za-z][A-Za-z0-9_-]{0,79}\Z")
MAX_PAGES = 16
PAGE_SIZE = 100
MAX_REPORT_SPANS = 800
MAX_RESPONSE_BYTES = 4_000_000
EXPECTED_KINDS = frozenset({"AGENT", "RETRIEVER", "RERANKER", "LLM", "GUARDRAIL", "CHAIN"})


class CompareError(Exception):
    """A safe, non-content diagnostic."""


def _json_response(body: bytes, service: str) -> dict:
    if len(body) > MAX_RESPONSE_BYTES:
        raise CompareError(f"{service}: response too large")
    try:
        value = json.loads(body)
        if not isinstance(value, dict):
            raise ValueError("Unexpected JSON shape")
        return value
    except (ValueError, UnicodeError):
        raise CompareError(f"{service}: invalid JSON response") from None


def _list_page(document: dict, service: str) -> tuple[list, str | None]:
    data = document.get("data")
    if not isinstance(data, list):
        raise CompareError(f"{service}: expected a list in data")
    meta = document.get("meta") or {}
    if not isinstance(meta, dict):
        raise CompareError(f"{service}: unexpected pagination metadata")
    cursor = meta.get("cursor") or document.get("next_cursor") or document.get("nextCursor")
    if cursor is not None and (not isinstance(cursor, str) or len(cursor) > 4096):
        raise CompareError(f"{service}: invalid cursor")
    return data, cursor


def _request_json(url: str, authorization: str, service: str) -> dict:
    # The OBS-05A opener explicitly disallows HTTP redirects and system proxy
    # settings. Body limits apply BEFORE parsing; never print server responses.
    req = urllib.request.Request(url, headers={"Authorization": authorization, "Accept": "application/json"})
    try:
        with OPENER.open(req, timeout=12) as response:
            if response.status != 200:
                raise CompareError(f"{service}: authenticated read failed (HTTP {response.status})")
            body = response.read(MAX_RESPONSE_BYTES + 1)
    except urllib.error.HTTPError as exc:
        status = exc.code
        exc.close()
        if service == "Phoenix" and status == 404:
            # The isolated fixture project may not exist before first ingestion.
            return {"data": [], "next_cursor": None}
        raise CompareError(f"{service}: authenticated read failed (HTTP {status})") from None
    except (urllib.error.URLError, OSError, TimeoutError):
        raise CompareError(f"{service}: request unavailable") from None
    return _json_response(body, service)


def read_phoenix(trace_id: str, project: str, key: str) -> list[dict]:
    """Fetch exact-ID spans with cursor paging; never search arbitrary trace content."""
    if not TRACE_RE.fullmatch(trace_id) or not PROJECT_RE.fullmatch(project):
        raise ValueError("Invalid trace/project selector")
    rows: list[dict] = []
    cursor = None
    seen = set()
    for _ in range(MAX_PAGES):
        query = {"trace_id": trace_id, "limit": PAGE_SIZE}
        if cursor:
            query["cursor"] = cursor
        path = f"/v1/projects/{urllib.parse.quote(project, safe='')}/spans?" + urllib.parse.urlencode(query)
        doc = _request_json(PHOENIX + path, "Bearer " + key, "Phoenix")
        data, next_cursor = _list_page(doc, "Phoenix")
        rows.extend(data)
        if len(rows) > MAX_REPORT_SPANS:
            raise CompareError("Phoenix: span count exceeds safe comparison limit")
        if not next_cursor:
            return rows
        if next_cursor in seen:
            raise CompareError("Phoenix: repeated pagination cursor")
        seen.add(next_cursor)
        cursor = next_cursor
    raise CompareError("Phoenix: pagination exhausted before completion")


def read_langfuse(trace_id: str, credentials: dict[str, str], start: datetime, end: datetime,
                  *, read_origin: str | None = None) -> list[dict]:
    if not TRACE_RE.fullmatch(trace_id):
        raise ValueError("Invalid trace selector")
    base = credentials["THEIA_LANGFUSE_OTLP_ENDPOINT"].removesuffix("/api/public/otel")
    if read_origin is not None:
        if base != "http://langfuse-web:3000":
            raise CompareError("Read origin override is permitted only for local Docker Langfuse")
        # The exporter may talk to Docker's langfuse-web name, while this script
        # runs on the host. Only send Basic auth to an EXPLICIT loopback port.
        parsed = urllib.parse.urlsplit(read_origin)
        if (parsed.scheme != "http" or parsed.hostname != "127.0.0.1" or not parsed.port or
            parsed.netloc != f"127.0.0.1:{parsed.port}" or parsed.path not in ("", "/") or
            parsed.query or parsed.fragment):
            raise CompareError("Langfuse read override must be explicit 127.0.0.1 HTTP loopback")
        base = f"http://127.0.0.1:{parsed.port}"
    elif base == "http://langfuse-web:3000":
        raise CompareError("Docker Langfuse requires --langfuse-read-origin pointing to a private loopback port")
    rows: list[dict] = []
    cursor = None
    seen = set()
    for _ in range(MAX_PAGES):
        query = {
            "traceId": trace_id, "fromStartTime": start.isoformat(timespec="seconds"),
            "toStartTime": end.isoformat(timespec="seconds"),
            "fields": "core,basic,io,metadata,model,usage", "limit": PAGE_SIZE,
        }
        if cursor:
            query["cursor"] = cursor
        doc = _request_json(base + "/api/public/v2/observations?" + urllib.parse.urlencode(query),
                            "Basic " + credentials["THEIA_LANGFUSE_AUTH_BASE64"], "Langfuse")
        data, next_cursor = _list_page(doc, "Langfuse")
        rows.extend(data)
        if len(rows) > MAX_REPORT_SPANS:
            raise CompareError("Langfuse: observation count exceeds safe comparison limit")
        if not next_cursor:
            return rows
        if next_cursor in seen:
            raise CompareError("Langfuse: repeated pagination cursor")
        seen.add(next_cursor)
        cursor = next_cursor
    raise CompareError("Langfuse: pagination exhausted before completion")


def _clean_id(raw: object, size: int) -> str | None:
    # OTel hex IDs in both services. Reject non-OTel identifiers, never guess.
    if isinstance(raw, str) and len(raw) == size and all(c in "0123456789abcdefABCDEF" for c in raw):
        return raw.lower()
    return None


def _flatten(value: object) -> object:
    if isinstance(value, dict):
        return {str(k): v for k, v in value.items()}
    return {}


def normalize_phoenix(rows: list, trace_id: str) -> dict[str, dict]:
    result = {}
    for row in rows:
        if not isinstance(row, dict):
            raise CompareError("Phoenix: invalid span row")
        ctx = _flatten(row.get("context"))
        row_trace = _clean_id(ctx.get("trace_id") or row.get("trace_id") or row.get("traceId"), 32)
        if row_trace != trace_id:
            raise CompareError("Phoenix: unrelated or malformed trace row")
        span_id = _clean_id(ctx.get("span_id") or row.get("span_id") or row.get("spanId"), 16)
        if not span_id or span_id in result:
            raise CompareError("Phoenix: invalid or repeated span identifier")
        attrs = _flatten(row.get("attributes"))
        kind = row.get("span_kind") or attrs.get("openinference.span.kind")
        result[span_id] = {
            "name": row.get("name") if isinstance(row.get("name"), str) else "",
            "parent": _clean_id(row.get("parent_id") or row.get("parentSpanId"), 16),
            "kind": kind if isinstance(kind, str) else None,
            "input": attrs.get("input.value"), "output": attrs.get("output.value"),
            "tokens_in": _token(attrs.get("llm.token_count.prompt", attrs.get("gen_ai.usage.input_tokens"))),
            "tokens_out": _token(attrs.get("llm.token_count.completion", attrs.get("gen_ai.usage.output_tokens"))),
            "rag_document": any(key.startswith("retrieval.documents.") for key in attrs),
            "rag_fields": {key: _canonical(value) for key, value in attrs.items()
                           if key.startswith("retrieval.documents.")},
            "rag_events": sum(1 for event in (row.get("events") or [])
                              if isinstance(event, dict) and isinstance(event.get("name"), str)
                              and event["name"].startswith("theia.rag.")),
        }
    return result


def _token(value: object) -> int | None:
    if isinstance(value, bool):
        return None
    if isinstance(value, int) and value >= 0:
        return value
    if isinstance(value, str) and value.isascii() and value.isdigit():
        return int(value)
    return None


def normalize_langfuse(rows: list, trace_id: str) -> dict[str, dict]:
    result = {}
    for row in rows:
        if not isinstance(row, dict):
            raise CompareError("Langfuse: invalid observation row")
        if _clean_id(row.get("traceId"), 32) != trace_id:
            raise CompareError("Langfuse: unrelated or malformed trace row")
        span_id = _clean_id(row.get("id"), 16)
        if not span_id or span_id in result:
            raise CompareError("Langfuse: unexpected or repeated observation ID")
        metadata = _flatten(row.get("metadata"))
        # Langfuse v4 places unmapped OTLP span attributes under attributes.*.
        rag_fields = {}
        for key, value in metadata.items():
            attribute = key.removeprefix("attributes.")
            if attribute.startswith("retrieval.documents."):
                rag_fields[attribute] = _canonical(value)
        usage = _flatten(row.get("usageDetails"))
        kind = row.get("type")
        result[span_id] = {
            "name": row.get("name") if isinstance(row.get("name"), str) else "",
            "parent": _clean_id(row.get("parentObservationId"), 16),
            "kind": kind if isinstance(kind, str) else None,
            "input": row.get("input"), "output": row.get("output"),
            "tokens_in": _token(row.get("inputUsage", usage.get("input"))),
            "tokens_out": _token(row.get("outputUsage", usage.get("output"))),
            "rag_document": bool(rag_fields),
            "rag_fields": rag_fields,
            "rag_events": None,  # v2 rows do not expose raw OTLP events.
        }
    return result


def _canonical(value: object) -> str | None:
    if value is None:
        return None
    if isinstance(value, str):
        try:
            parsed = json.loads(value)
        except (ValueError, TypeError):
            return value
        return json.dumps(parsed, ensure_ascii=False, sort_keys=True, separators=(",", ":"))
    return json.dumps(value, ensure_ascii=False, sort_keys=True, separators=(",", ":"))


def compare(trace_id: str, phoenix_rows: list, langfuse_rows: list,
            *, fixture: bool = False) -> dict:
    """Return a content-free report. Never echo raw attributes or output strings."""
    p = normalize_phoenix(phoenix_rows, trace_id)
    l = normalize_langfuse(langfuse_rows, trace_id)
    shared = p.keys() & l.keys()
    flags = {"names": True, "parents": True, "rag_document_presence": True,
             "rag_document_fields_equal": True, "input_presence": True,
             "output_presence": True, "equal_inputs": True, "equal_outputs": True,
             "known_token_usage": True}
    iop = {"phoenix_inputs": 0, "langfuse_inputs": 0, "phoenix_outputs": 0, "langfuse_outputs": 0}
    kind_mismatch = Counter()
    known_kinds = EXPECTED_KINDS | frozenset({"PROMPT", "EMBEDDING", "TOOL", "EVALUATOR", "UNKNOWN", "UNCLASSIFIED"})
    compatible = {"LLM": {"GENERATION", "LLM"}, "AGENT": {"AGENT"},
                  "RETRIEVER": {"RETRIEVER"}, "GUARDRAIL": {"GUARDRAIL"},
                  "CHAIN": {"CHAIN", "SPAN"}, "RERANKER": {"SPAN", "CHAIN", "RERANKER"},
                  "PROMPT": {"SPAN", "CHAIN", "PROMPT"},
                  "UNKNOWN": {"SPAN", "UNKNOWN"},
                  "UNCLASSIFIED": {"SPAN", "UNCLASSIFIED"}}
    for obj, prefix in ((p, "phoenix"), (l, "langfuse")):
        iop[prefix + "_inputs"] = sum(x["input"] is not None for x in obj.values())
        iop[prefix + "_outputs"] = sum(x["output"] is not None for x in obj.values())
    for key in shared:
        left, right = p[key], l[key]
        flags["names"] &= left["name"] == right["name"]
        flags["rag_document_presence"] &= left["rag_document"] == right["rag_document"]
        flags["rag_document_fields_equal"] &= left["rag_fields"] == right["rag_fields"]
        flags["parents"] &= left["parent"] == right["parent"]
        flags["input_presence"] &= (left["input"] is None) == (right["input"] is None)
        flags["output_presence"] &= (left["output"] is None) == (right["output"] is None)
        if left["input"] is not None and right["input"] is not None:
            flags["equal_inputs"] &= _canonical(left["input"]) == _canonical(right["input"])
        if left["output"] is not None and right["output"] is not None:
            flags["equal_outputs"] &= _canonical(left["output"]) == _canonical(right["output"])
        for suffix in ("tokens_in", "tokens_out"):
            if left[suffix] is not None:
                flags["known_token_usage"] &= left[suffix] == right[suffix]
        if left["kind"] and right["kind"] and right["kind"] not in compatible.get(left["kind"], {left["kind"]}):
            # Never reflect an arbitrary remote label in report content.
            kind_mismatch[left["kind"] if left["kind"] in known_kinds else "UNCLASSIFIED"] += 1

    only_p = sorted(p.keys() - l.keys())
    only_l = sorted(l.keys() - p.keys())
    known_langfuse = known_kinds | frozenset({"SPAN", "GENERATION", "EVENT"})
    p_kinds = Counter(x["kind"] if x["kind"] in known_kinds else "UNCLASSIFIED" for x in p.values())
    l_kinds = Counter(x["kind"] if x["kind"] in known_langfuse else "UNCLASSIFIED" for x in l.values())
    issues = []
    if not p or not l: issues.append("MISSING_TRACE_OR_SPANS")
    if only_p or only_l: issues.append("SPAN_ID_SET_MISMATCH")
    for key, ok in flags.items():
        if not ok: issues.append("MISMATCH_" + key.upper())
    if kind_mismatch: issues.append("SEMANTIC_TYPE_MISMATCH")
    if not fixture and (not iop["phoenix_inputs"] or not iop["phoenix_outputs"] or
                        not iop["langfuse_inputs"] or not iop["langfuse_outputs"]):
        issues.append("INCOMPLETE_FULL_CONTENT_CAPTURE")
    if fixture:
        for role in sorted(EXPECTED_KINDS - p_kinds.keys()):
            issues.append("PHOENIX_MISSING_" + role)
        # There are two physical LLM attempts, each with a transport child.
        if p_kinds.get("LLM", 0) != 2: issues.append("PHOENIX_PHYSICAL_LLM_ATTEMPTS")
        if len(p) != 10: issues.append("PHOENIX_FIXTURE_SPAN_COUNT")
        if sum(x["rag_events"] for x in p.values()) < 3:
            issues.append("PHOENIX_MISSING_RAG_EVENTS")
        if not iop["phoenix_inputs"] or not iop["phoenix_outputs"]:
            issues.append("PHOENIX_MISSING_IO")
        if not iop["langfuse_inputs"] or not iop["langfuse_outputs"]:
            issues.append("LANGFUSE_MISSING_IO")
    return {
        "schema": SCHEMA, "traceId": trace_id, "fixture": fixture,
        "generatedAt": datetime.now(timezone.utc).isoformat(timespec="seconds"),
        "phoenix": {"spans": len(p), "kinds": dict(sorted(p_kinds.items())),
                    "inputs": iop["phoenix_inputs"], "outputs": iop["phoenix_outputs"],
                    "rag_document_spans": sum(x["rag_document"] for x in p.values()),
                    "rag_events": sum(x["rag_events"] for x in p.values())},
        "langfuse": {"observations": len(l), "types": dict(sorted(l_kinds.items())),
                     "inputs": iop["langfuse_inputs"], "outputs": iop["langfuse_outputs"],
                     "rag_document_spans": sum(x["rag_document"] for x in l.values()),
                     "rag_events": "NOT_AVAILABLE_THROUGH_V2_API"},
        "parity": {"shared_span_ids": len(shared), "only_phoenix_span_ids": only_p,
                   "only_langfuse_span_ids": only_l, "checks": flags,
                   "semantic_mismatches_by_phoenix_role": dict(sorted(kind_mismatch.items()))},
        "result": "PASS" if not issues else "FAIL",
        "issues": issues,
        "platform_choice": "NOT_DETERMINED",  # Needs manual UX/quality comparison.
    }


def _int_attr(key: str, value: int) -> bytes:
    # Raw OTLP KeyValue bytes (the enclosing Span.attributes is field 9).
    return _str(1, key) + _field(2, _varint(3 << 3) + _varint(value))


def _string_attr(key: str, value: str) -> bytes:
    return _str(1, key) + _field(2, _str(1, value))


def _span(trace_id: str, span_id: str, parent_id: str | None, name: str, kind: str | None,
          attributes: dict[str, str | int], start_ns: int, end_ns: int,
          *, error: bool = False, events: tuple[str, ...] = ()) -> bytes:
    content = _field(1, bytes.fromhex(trace_id)) + _field(2, bytes.fromhex(span_id))
    if parent_id: content += _field(4, bytes.fromhex(parent_id))
    content += _str(5, name) + _varint(6 << 3) + _varint(3 if name == "gen_ai.client" else 1)
    content += _fixed64(7, start_ns) + _fixed64(8, end_ns)
    if kind: attributes = {**attributes, "openinference.span.kind": kind}
    for key, value in attributes.items():
        content += _field(9, _int_attr(key, value) if isinstance(value, int) else _string_attr(key, value))
    for event in events:
        content += _field(11, _fixed64(1, start_ns + 1000) + _str(2, event))
    content += _field(15, _varint(3 << 3) + _varint(2 if error else 1))
    return content


def synthetic_fixture(trace_id: str, *, now_ns: int | None = None) -> tuple[bytes, set[str]]:
    """One 10-span tree: retriever, RRF, brain, two attempts, HTTP, guardrail."""
    if not TRACE_RE.fullmatch(trace_id): raise ValueError("Invalid trace ID")
    now_ns = now_ns if now_ns is not None else time.time_ns()
    ids = [os.urandom(8).hex() for _ in range(10)]
    # Avoid theoretical collisions, so the test can assert the exact span-id set.
    while len(set(ids)) < len(ids): ids = [os.urandom(8).hex() for _ in range(10)]
    data = [
        (0, None, "theia.ai.turn", "AGENT", {"input.value": "SYNTHETIC_QUESTION", "output.value": "SYNTHETIC_ANSWER"}, ()),
        (1, 0, "theia.ai.application.retrieval.service.Retrieve", "RETRIEVER", {
            "input.value": "SYNTHETIC_QUERY", "output.value": '{"selected":["SYNTHETIC_DOC_A"]}',
            "retrieval.documents.0.document.id": "SYNTHETIC_DOC_A",
            "retrieval.documents.0.document.content": "SYNTHETIC_CONTEXT_A"}, ("theia.rag.vector_search", "theia.rag.bm25_search")),
        (2, 1, "theia.ai.retrieval.rank", "RERANKER", {
            "input.value": "SYNTHETIC_CANDIDATES", "output.value": "SYNTHETIC_RRF_SELECTED"}, ("theia.rag.rrf",)),
        (3, 0, "theia.ai.application.brain.Decide", "AGENT", {
            "input.value": "SYNTHETIC_BRAIN_INPUT", "output.value": "SYNTHETIC_BRAIN_DECISION"}, ()),
        (4, 0, "gen_ai.client", "LLM", {"gen_ai.operation.name": "chat", "gen_ai.provider.name": "synthetic",
            "gen_ai.request.model": "synthetic-model", "llm.model_name": "synthetic-model",
            "llm.token_count.prompt": 12}, ("theia.retry.attempt_1",)),
        (5, 4, "theia.ai.http.client", None, {"http.request.method": "POST", "http.response.status_code": 429}, ()),
        (6, 0, "gen_ai.client", "LLM", {"gen_ai.operation.name": "chat", "gen_ai.provider.name": "synthetic",
            "gen_ai.request.model": "synthetic-model", "llm.model_name": "synthetic-model",
            "input.value": '{"prompt":"SYNTHETIC_PROMPT"}', "output.value": "SYNTHETIC_MODEL_OUTPUT",
            "llm.token_count.prompt": 12, "llm.token_count.completion": 7}, ("theia.retry.attempt_2",)),
        (7, 6, "theia.ai.http.client", None, {"http.request.method": "POST", "http.response.status_code": 200}, ()),
        (8, 0, "theia.ai.application.validator.Validate", "GUARDRAIL", {
            "input.value": "SYNTHETIC_DRAFT", "output.value": "SYNTHETIC_ACCEPTED"}, ()),
        (9, 0, "theia.ai.finalizer.Finalize", "CHAIN", {"output.value": "SYNTHETIC_ANSWER"}, ()),
    ]
    records = []
    for number, parent, name, kind, attrs, events in data:
        begin = now_ns + number * 3_000_000
        records.append(_span(trace_id, ids[number], ids[parent] if parent is not None else None,
                             name, kind, attrs, begin, begin + 2_000_000, error=number == 4,
                             events=events))
    scope = _field(1, _str(1, "theia.obs05b.comparison")) + b"".join(_field(2, record) for record in records)
    resource = _attr("service.name", "theia-obs05b-fixture") + _attr("openinference.project.name", FIXTURE_PROJECT)
    return _field(1, _field(1, resource) + _field(2, scope)), set(ids)


def collect(trace_id: str, project: str, credentials: dict, phoenix_key: str,
            *, start: datetime, end: datetime, attempts: int, delay: float,
            read_origin: str | None = None, min_spans: int = 1) -> tuple[list, list]:
    phoenix: list = []
    langfuse: list = []
    previous: tuple[frozenset, frozenset] | None = None
    for i in range(attempts):
        # Refresh BOTH readbacks on each poll. A nonempty first page is not
        # proof that queued exporters have delivered the entire trace.
        phoenix = read_phoenix(trace_id, project, phoenix_key)
        langfuse = read_langfuse(trace_id, credentials, start, end, read_origin=read_origin)
        if len(phoenix) >= min_spans and len(langfuse) >= min_spans:
            if min_spans > 1:
                return phoenix, langfuse
            signature = (
                frozenset(row.get("context", {}).get("span_id", "") for row in phoenix),
                frozenset(row.get("id", "") for row in langfuse),
            )
            if signature == previous:
                return phoenix, langfuse  # Two consecutive stable snapshots.
            previous = signature
        else:
            previous = None
        if i + 1 < attempts:
            time.sleep(delay)
    return phoenix, langfuse


def _safe_report_path(raw: str) -> Path:
    target = Path(raw).expanduser()
    if target.is_symlink() or (target.exists() and not target.is_file()):
        raise CompareError("Report path must be a regular file, not a symlink")
    return target


def save_report(report: dict, destination: str | None) -> None:
    # Only counters, approved enum labels, IDs and booleans; no source content.
    serialized = json.dumps(report, ensure_ascii=True, indent=2, sort_keys=True) + "\n"
    if destination is None:
        print(serialized, end="")
        return
    target = _safe_report_path(destination)
    if target.exists():
        raise CompareError("Refusing to overwrite an existing comparison report")
    flags = os.O_WRONLY | os.O_CREAT | os.O_EXCL
    fd = os.open(target, flags, 0o600)
    with os.fdopen(fd, "w", encoding="ascii") as f:
        f.write(serialized)
    print("Saved content-free comparison report (0600).")


def main() -> None:
    parser = argparse.ArgumentParser(description="OBS-05B: same synthetic trace comparison")
    modes = parser.add_mutually_exclusive_group(required=True)
    modes.add_argument("--fixture", action="store_true", help="Ingest one synthetic 10-span comparison fixture")
    modes.add_argument("--inspect", metavar="TRACE_ID", help="Inspect one already-captured synthetic Turn")
    parser.add_argument("--project", help="Phoenix project (required for --inspect)")
    parser.add_argument("--lookback-hours", type=int, default=24, help="Langfuse query lookback, 1..168")
    parser.add_argument("--output", help="Create a new private JSON report; never write raw payloads")
    parser.add_argument("--langfuse-read-origin", help="For Docker Langfuse: host-local 127.0.0.1 HTTP read origin")
    parser.add_argument("--synthetic-only", action="store_true", help="Confirm synthetic source and private access")
    args = parser.parse_args()
    if not args.synthetic_only:
        parser.error("--synthetic-only is required; never compare real user content with this tool")
    if not 1 <= args.lookback_hours <= 168:
        parser.error("--lookback-hours must be between 1 and 168")
    trace_id = uuid.uuid4().hex if args.fixture else args.inspect
    if not TRACE_RE.fullmatch(trace_id):
        parser.error("--inspect requires a lowercase 32-character W3C TraceID")
    project = args.project or (FIXTURE_PROJECT if args.fixture else None)
    if project is None or not PROJECT_RE.fullmatch(project):
        parser.error("--project must be an explicit simple project name for --inspect")
    if args.fixture and project != FIXTURE_PROJECT:
        parser.error("--fixture must use the existing OBS-05A dual-export Phoenix project")
    credentials = read_dual()
    phoenix_key = read_phoenix_env().get("THEIA_OTEL_CONTENT_API_KEY", "")
    if len(phoenix_key) < 32:
        raise CompareError("Phoenix system API key missing")
    expected: set[str] | None = None
    if args.fixture:
        payload, expected = synthetic_fixture(trace_id)
        status, _ = request(COLLECTOR + "/v1/traces", method="POST", data=payload,
                            authorization="Bearer " + credentials["THEIA_COLLECTOR_INGEST_KEY"])
        if status not in (200, 202, 204):
            raise CompareError("Collector did not accept the synthetic multi-span fixture")
    end = datetime.now(timezone.utc) + timedelta(minutes=5)
    start = end - timedelta(hours=args.lookback_hours)
    phoenix_rows, langfuse_rows = collect(trace_id, project, credentials, phoenix_key,
                                          start=start, end=end, attempts=12, delay=3,
                                          read_origin=args.langfuse_read_origin,
                                          min_spans=10 if args.fixture else 1)
    report = compare(trace_id, phoenix_rows, langfuse_rows, fixture=args.fixture)
    report["project"] = project
    if expected is not None:
        p_ids = set(normalize_phoenix(phoenix_rows, trace_id))
        l_ids = set(normalize_langfuse(langfuse_rows, trace_id))
        if expected != p_ids or expected != l_ids:
            report["issues"].append("SYNTHETIC_SOURCE_ID_SET_MISMATCH")
            report["result"] = "FAIL"
    save_report(report, args.output)
    if args.output:
        print(f"TraceID: {trace_id}; parity={report['result']}; "
              f"Phoenix={report['phoenix']['spans']} spans; "
              f"Langfuse={report['langfuse']['observations']} observations")
    if report["result"] != "PASS":
        raise SystemExit("OBS-05B comparison failed; inspect issue codes in the sanitized report")


if __name__ == "__main__":
    try:
        main()
    except (CompareError, ValueError, OSError) as error:
        # All error text is locally authored. Do not echo URLs, body, tokens, or server errors.
        raise SystemExit("OBS-05B unavailable: " + str(error)) from None
