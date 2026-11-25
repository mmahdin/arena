#!/usr/bin/env python3
"""OBS-02: conservative local Phoenix auth, OTLP ingestion and persistence smoke.

Posts a single synthetic metadata-only OTLP/protobuf span. This test is not a
substitute for manually inspecting a real Go provider attempt with OBS-01B.
Only ever contacts numeric IPv4 loopback; no proxies or redirects are used.
No passwords, API keys, payloads or sensitive server responses are printed.
"""
from __future__ import annotations

import argparse
import json
import os
from pathlib import Path
import struct
import time
import urllib.error
import urllib.request
import uuid

from init_local_phoenix_env import read_env

PROJECT = "theia-obs02-smoke"


def _varint(value: int) -> bytes:
    if value < 0:
        raise ValueError("Negative protobuf varint")
    out = bytearray()
    while value >= 128:
        out.append((value & 0x7F) | 0x80)
        value >>= 7
    out.append(value)
    return bytes(out)


def _field(num: int, data: bytes) -> bytes:
    return _varint((num << 3) | 2) + _varint(len(data)) + data


def _str(num: int, value: str) -> bytes:
    return _field(num, value.encode("utf-8"))


def _fixed64(num: int, value: int) -> bytes:
    return _varint((num << 3) | 1) + struct.pack("<Q", value)


def _attr(key: str, val: str) -> bytes:
    # OTLP Common KeyValue { key = 1, value = 2 } / AnyValue.string_value=1
    return _field(1, _str(1, key) + _field(2, _str(1, val)))


def make_otlp_span(trace_id: str, span_id: str, *, now_ns: int | None = None) -> bytes:
    if (len(trace_id) != 32 or len(span_id) != 16 or
            not all(c in "0123456789abcdef" for c in trace_id + span_id)):
        raise ValueError("Invalid synthetic OTLP trace/span identifier")
    now_ns = now_ns if now_ns is not None else time.time_ns()
    span = (
        _field(1, bytes.fromhex(trace_id)) + _field(2, bytes.fromhex(span_id))
        + _str(5, "theia.obs02.smoke") + _varint((6 << 3) | 0) + _varint(1)
        + _fixed64(7, now_ns) + _fixed64(8, now_ns + 10_000_000)
    )
    scope_spans = _field(1, _str(1, "theia.obs02.smoke")) + _field(2, span)
    resource = _attr("service.name", "theia-obs02-smoke") + _attr("openinference.project.name", PROJECT)
    resource_spans = _field(1, resource) + _field(2, scope_spans)
    return _field(1, resource_spans)  # ExportTraceServiceRequest.resource_spans


class _NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


_OPENER = urllib.request.build_opener(urllib.request.ProxyHandler({}), _NoRedirect())


def _call(base: str, path: str, *, method: str = "GET", data: bytes | None = None,
          key: str | None = None, project: str | None = None) -> tuple[int, bytes]:
    if not (base.startswith("http://127.0.0.1:") and base.count(":") == 2 and
            base.rsplit(":", 1)[1].isdigit() and 1 <= int(base.rsplit(":", 1)[1]) <= 65535):
        raise ValueError("Only local 127.0.0.1 HTTP Phoenix endpoints are supported")
    headers = {"Accept": "application/json"}
    if key:
        headers["Authorization"] = "Bearer " + key
    if data is not None:
        headers["Content-Type"] = "application/x-protobuf"
    if project:
        headers["x-project-name"] = project
    request = urllib.request.Request(base + path, method=method, data=data, headers=headers)
    try:
        with _OPENER.open(request, timeout=5) as response:
            return response.status, response.read(1_000_000)
    except urllib.error.HTTPError as error:
        # Read no error body: it may include reflected request data or secrets.
        error.close()
        return error.code, b""
    except (urllib.error.URLError, OSError, TimeoutError) as error:
        raise RuntimeError("Phoenix localhost is unavailable; check container health") from None


def preflight(base: str = "http://127.0.0.1:6006") -> None:
    health, _ = _call(base, "/healthz")
    if health != 200:
        raise RuntimeError("Phoenix health endpoint did not return 200")
    projects, _ = _call(base, "/v1/projects")
    if projects not in (401, 403):
        raise RuntimeError("SECURITY FAILURE: unauthenticated Phoenix API read was not denied")
    empty_otlp = make_otlp_span("11" * 16, "22" * 8)
    write, _ = _call(base, "/v1/traces", method="POST", data=empty_otlp)
    if write not in (401, 403):
        raise RuntimeError("SECURITY FAILURE: unauthenticated OTLP ingest was not denied")


def has_trace(base: str, key: str, trace_id: str) -> bool:
    status, raw = _call(base, f"/v1/projects/{PROJECT}/traces?include_spans=true&limit=100", key=key)
    if status == 404:
        return False
    if status != 200:
        raise RuntimeError("Authorized Phoenix trace lookup failed")
    try:
        payload = json.loads(raw)
    except (json.JSONDecodeError, UnicodeError):
        raise RuntimeError("Phoenix returned an invalid trace-list response") from None
    return trace_id in json.dumps(payload).lower()


def ingest(base: str, key: str) -> str:
    preflight(base)
    status, _ = _call(base, "/v1/projects", key=key)
    if status != 200:
        raise RuntimeError("Ingestion key cannot access authenticated Phoenix API")
    trace_id = uuid.uuid4().hex
    span_id = os.urandom(8).hex()
    status, _ = _call(base, "/v1/traces", method="POST", data=make_otlp_span(trace_id, span_id), key=key, project=PROJECT)
    if status not in (200, 202):
        raise RuntimeError("Authenticated Phoenix OTLP ingestion failed")
    # Trace materialization can lag successful ingestion.
    for _ in range(20):
        if has_trace(base, key, trace_id):
            return trace_id
        time.sleep(1)
    raise RuntimeError("Phoenix accepted OTLP but the trace was not queryable; check project and retention")


def verify_after_restart(base: str, key: str, trace_id: str) -> None:
    preflight(base)
    if not has_trace(base, key, trace_id):
        raise RuntimeError("Trace not found after Phoenix restart; verify volume persistence")


def main() -> None:
    parser = argparse.ArgumentParser(description="Check authenticated local Phoenix ingestion")
    mode = parser.add_mutually_exclusive_group(required=True)
    mode.add_argument("--preflight", action="store_true", help="Probe unauthenticated read/ingestion denial")
    mode.add_argument("--ingest", action="store_true", help="Ingest synthetic span and verify readback")
    mode.add_argument("--after-restart", metavar="TRACE_ID", help="Verify synthetic trace persisted after restart")
    args = parser.parse_args()
    base = "http://127.0.0.1:6006"
    if args.preflight:
        preflight(base)
        print("Phoenix health + negative authentication checks passed.")
        return
    key = read_env().get("THEIA_OTEL_CONTENT_API_KEY", "")
    if not key:
        raise RuntimeError("Phoenix system API key not configured; use --set-ingest-key first")
    if args.ingest:
        trace_id = ingest(base, key)
        print("Phoenix authenticated OTLP ingestion and readback passed.")
        print("Synthetic trace ID (keep for restart test): " + trace_id)
    else:
        trace_id = args.after_restart
        if len(trace_id) != 32 or not all(c in "0123456789abcdef" for c in trace_id):
            raise ValueError("--after-restart requires a lowercase 32-character W3C TraceID")
        verify_after_restart(base, key, trace_id)
        print("Phoenix trace is still queryable after restart.")


if __name__ == "__main__":
    try:
        main()
    except (RuntimeError, ValueError, OSError) as error:
        raise SystemExit(f"OBS-02 smoke failed: {error}") from None
