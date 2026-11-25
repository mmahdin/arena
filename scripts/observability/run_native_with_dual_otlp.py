#!/usr/bin/env python3
"""Run the native Go API with private authenticated dual Phoenix/Langfuse OTLP.

Does not change the original run_native_with_phoenix.py workflow or execute
external shell commands. The Langfuse project secrets stay in the Collector.
"""
from __future__ import annotations

import os
from pathlib import Path
import sys

from init_dual_otlp_env import read_env
from smoke_dual_otlp import preflight

ROOT = Path(__file__).resolve().parents[2]


def go_environment(inherited: dict[str, str], credentials: dict[str, str], *, content: bool = False) -> dict[str, str]:
    token = credentials["THEIA_COLLECTOR_INGEST_KEY"]
    env = dict(inherited)
    # No Phoenix, Langfuse, or Collector server secrets are inherited by Go;
    # its one receiver token authorizes only loopback ingest to the Collector.
    for name in ("PHOENIX_SECRET", "PHOENIX_DEFAULT_ADMIN_INITIAL_PASSWORD", "THEIA_LANGFUSE_AUTH_BASE64",
                 "THEIA_LANGFUSE_OTLP_ENDPOINT", "THEIA_PHOENIX_API_KEY", "THEIA_COLLECTOR_INGEST_KEY"):
        env.pop(name, None)
    env.update({
        "APP_ENV": "local", "HTTP_ADDR": "127.0.0.1:8080", "ALLOW_DEV_LOGIN": "false",
        "AI_LOCAL_INSPECTOR": "false", "THEIA_OTEL_ENABLED": "true",
        "THEIA_OTEL_CAPTURE_CONTENT": "true" if content else "false",
        "THEIA_OTEL_CONTENT_API_KEY": token,
        "OTEL_SERVICE_NAME": "theia-api", "OTEL_SERVICE_VERSION": "dev",
        "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT": "http://127.0.0.1:4318/v1/traces",
    })
    return env


def main() -> None:
    if len(sys.argv) != 1 and sys.argv[1:] != ["--content"]:
        raise ValueError("Usage: run_native_with_dual_otlp.py [--content]")
    credentials = read_env()
    preflight(credentials["THEIA_COLLECTOR_INGEST_KEY"])
    content = sys.argv[1:] == ["--content"]
    if content:
        print("OBS-05A: FULL synthetic-only capture to BOTH Phoenix and Langfuse. No real user content.", file=sys.stderr)
    env = go_environment(os.environ, credentials, content=content)
    command = [sys.executable, str(ROOT / "scripts/run_local.py"), "go", "run", "./cmd/api"]
    os.execvpe(command[0], command, env)


if __name__ == "__main__":
    try:
        main()
    except (ValueError, RuntimeError, OSError) as exc:
        raise SystemExit(f"Native OBS-05A dual exporter failed: {type(exc).__name__}. Verify Collector, keys and local configuration.") from None
