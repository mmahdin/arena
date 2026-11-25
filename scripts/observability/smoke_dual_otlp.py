#!/usr/bin/env python3
"""OBS-05A: one synthetic OTLP TraceID, readback from Phoenix and Langfuse v4.

Never runs a model or submits any user content. No token/secret in URLs, stdout,
error output or service logs. Requires configured Langfuse and local Phoenix.
"""
from __future__ import annotations

import argparse
from datetime import datetime, timedelta, timezone
import json
import os
import time
import urllib.error
import urllib.parse
import urllib.request
import uuid

from init_dual_otlp_env import read_env as read_dual
from init_local_phoenix_env import read_env as read_phoenix
from smoke_phoenix import _call as phoenix_call, _attr, _field, _fixed64, _str, _varint

COLLECTOR = "http://127.0.0.1:4318"
PHOENIX = "http://127.0.0.1:6006"
PROJECT = "theia-obs05a-dual"


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, *args):
        return None


OPENER = urllib.request.build_opener(urllib.request.ProxyHandler({}), NoRedirect())


def make_synthetic_span(trace_id: str, span_id: str) -> bytes:
    """Syntactically valid OTLP/protobuf span with ONLY synthetic metadata."""
    if len(trace_id) != 32 or len(span_id) != 16 or any(c not in "0123456789abcdef" for c in trace_id + span_id):
        raise ValueError("Invalid W3C trace/span ID")
    now = time.time_ns()
    span = (_field(1, bytes.fromhex(trace_id)) + _field(2, bytes.fromhex(span_id))
            + _str(5, "theia.obs05a.dual-smoke") + _varint(6 << 3) + _varint(1)
            + _fixed64(7, now) + _fixed64(8, now + 10_000_000))
    scope = _field(1, _str(1, "theia.obs05a.smoke")) + _field(2, span)
    resource = _attr("service.name", "theia-obs05a-smoke") + _attr("openinference.project.name", PROJECT)
    return _field(1, _field(1, resource) + _field(2, scope))


def request(url: str, *, method: str = "GET", data: bytes | None = None,
            authorization: str | None = None, timeout: float = 5.0) -> tuple[int, bytes]:
    headers = {"Accept": "application/json"}
    if data is not None:
        headers["Content-Type"] = "application/x-protobuf"
    if authorization is not None:
        headers["Authorization"] = authorization
    req = urllib.request.Request(url, method=method, data=data, headers=headers)
    try:
        with OPENER.open(req, timeout=timeout) as response:
            return response.status, response.read(262_144)
    except urllib.error.HTTPError as err:
        err.close()  # Never reflect response content; it might contain secrets.
        return err.code, b""
    except (urllib.error.URLError, OSError, TimeoutError):
        raise RuntimeError("OTLP endpoint unavailable or TLS/connectivity failed") from None


def ingest_collector(trace_id: str, key: str | None) -> int:
    auth = "Bearer " + key if key is not None else None
    status, _ = request(COLLECTOR + "/v1/traces", method="POST",
                        data=make_synthetic_span(trace_id, os.urandom(8).hex()), authorization=auth)
    return status


def preflight(key: str) -> None:
    status = ingest_collector(uuid.uuid4().hex, None)
    if status not in (401, 403):
        raise RuntimeError("SECURITY FAILURE: unauthenticated Collector accepted trace ingestion")
    status = ingest_collector(uuid.uuid4().hex, "not-the-private-token")
    if status not in (401, 403):
        raise RuntimeError("SECURITY FAILURE: invalid bearer token accepted by Collector")
    status = ingest_collector(uuid.uuid4().hex, key)
    if status not in (200, 202, 204):
        raise RuntimeError("Collector did not accept authenticated synthetic OTLP")


def phoenix_has_trace(trace_id: str, key: str) -> bool:
    status, body = phoenix_call(PHOENIX, f"/v1/projects/{PROJECT}/traces?include_spans=true&limit=100", key=key)
    if status == 404:
        return False
    if status != 200:
        raise RuntimeError("Authorized Phoenix trace readback failed")
    try:
        doc = json.loads(body)
    except (json.JSONDecodeError, UnicodeError):
        raise RuntimeError("Invalid Phoenix trace list") from None
    return trace_id in json.dumps(doc).lower()


def langfuse_has_trace(trace_id: str, credentials: dict[str, str]) -> bool:
    # Langfuse v4: legacy GET /traces/{id} has been removed. Use Observations API v2.
    base = credentials["THEIA_LANGFUSE_OTLP_ENDPOINT"].removesuffix("/api/public/otel")
    now = datetime.now(timezone.utc)
    query = urllib.parse.urlencode({
        "traceId": trace_id,
        "fromStartTime": (now - timedelta(days=1)).isoformat(timespec="seconds"),
        "toStartTime": (now + timedelta(minutes=5)).isoformat(timespec="seconds"),
        "fields": "core,basic", "limit": "100",
    })
    status, body = request(base + "/api/public/v2/observations?" + query,
                           authorization="Basic " + credentials["THEIA_LANGFUSE_AUTH_BASE64"])
    if status != 200:
        raise RuntimeError("Langfuse v4 Observations API readback failed")
    try:
        doc = json.loads(body)
        rows = doc.get("data", [])
        if not isinstance(rows, list):
            raise ValueError("Invalid response")
        return any(isinstance(row, dict) and row.get("traceId", "").lower() == trace_id for row in rows)
    except (ValueError, AttributeError, TypeError):
        raise RuntimeError("Unexpected Langfuse v4 Observations API response") from None


def inspect(trace_id: str, credentials: dict[str, str], phoenix_key: str,
            *, attempts: int = 1, delay: float = 1) -> tuple[bool, bool]:
    phoenix = langfuse = False
    for i in range(attempts):
        phoenix = phoenix or phoenix_has_trace(trace_id, phoenix_key)
        langfuse = langfuse or langfuse_has_trace(trace_id, credentials)
        if phoenix and langfuse:
            break
        if i + 1 < attempts:
            time.sleep(delay)
    return phoenix, langfuse


def main() -> None:
    parser = argparse.ArgumentParser(description="OBS-05A dual authenticated synthetic tracing smoke")
    group = parser.add_mutually_exclusive_group(required=True)
    group.add_argument("--preflight", action="store_true", help="Verify private Collector authentication")
    group.add_argument("--ingest", action="store_true", help="Write one synthetic trace, verify both destinations")
    group.add_argument("--inspect", metavar="TRACE_ID", help="Read back the same real/synthetic Turn TraceID in both destinations")
    args = parser.parse_args()
    creds = read_dual()
    if args.preflight:
        preflight(creds["THEIA_COLLECTOR_INGEST_KEY"])
        print("Collector authenticated OTLP acceptance and negative authentication checks passed.")
        return
    phoenix_key = read_phoenix().get("THEIA_OTEL_CONTENT_API_KEY", "")
    if len(phoenix_key) < 32:
        raise ValueError("Missing Phoenix system ingestion API key")
    if args.ingest:
        preflight(creds["THEIA_COLLECTOR_INGEST_KEY"])
        trace_id = uuid.uuid4().hex
        status = ingest_collector(trace_id, creds["THEIA_COLLECTOR_INGEST_KEY"])
        if status not in (200, 202, 204):
            raise RuntimeError("Collector did not accept the synthetic trace")
    else:
        trace_id = args.inspect
        if len(trace_id) != 32 or any(ch not in "0123456789abcdef" for ch in trace_id):
            raise ValueError("Use a lowercase 32-character W3C TraceID")
    phoenix, langfuse = inspect(trace_id, creds, phoenix_key, attempts=12, delay=3)
    print("TraceID: " + trace_id)
    print("Phoenix readback: " + ("found" if phoenix else "MISSING"))
    print("Langfuse v4 readback: " + ("found" if langfuse else "MISSING"))
    if not phoenix or not langfuse:
        raise RuntimeError("Dual export not proven; check Collector exporter queues, destination access and retention")
    print("OBS-05A PASS: identical TraceID queryable in both destinations.")


if __name__ == "__main__":
    try:
        main()
    except (ValueError, RuntimeError, OSError) as exc:
        # Trusted, constant text only: never print server bodies, keys or trace content.
        raise SystemExit(f"OBS-05A smoke failed: {type(exc).__name__}: {exc}") from None
