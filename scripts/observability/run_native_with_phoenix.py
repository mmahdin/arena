#!/usr/bin/env python3
"""Launch the native Go API with authenticated tracing to local Phoenix.

No shell interpolation, no secrets in command arguments, and no modifications
of the ordinary .env file. Requires independently running authenticated Phoenix.
"""
from __future__ import annotations
import os
from pathlib import Path
import sys
from init_local_phoenix_env import INGEST_PATTERN, read_env
from smoke_phoenix import _call, preflight

ROOT = Path(__file__).resolve().parents[2]


def local_go_environment(inherited: dict[str, str], secrets: dict[str, str], *, content: bool = False) -> dict[str, str]:
    key = secrets.get("THEIA_OTEL_CONTENT_API_KEY", "")
    if not INGEST_PATTERN.fullmatch(key):
        raise ValueError("Create a Phoenix system API key and run --set-ingest-key before exporting traces")
    env = dict(inherited)
    for name in ("PHOENIX_SECRET", "PHOENIX_DEFAULT_ADMIN_INITIAL_PASSWORD"):
        env.pop(name, None)
    env.update({
        "APP_ENV": "local",
        "HTTP_ADDR": "127.0.0.1:8080",
        "ALLOW_DEV_LOGIN": "false",
        "AI_LOCAL_INSPECTOR": "false",
        "THEIA_OTEL_ENABLED": "true",
        "THEIA_OTEL_CAPTURE_CONTENT": "true" if content else "false",
        "OTEL_SERVICE_NAME": "theia-api",
        "OTEL_SERVICE_VERSION": "dev",
        "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT": "http://127.0.0.1:6006/v1/traces",
        "THEIA_OTEL_CONTENT_API_KEY": key,
    })
    return env


def main() -> None:
    read = read_env()
    if len(sys.argv) != 1 and sys.argv[1:] != ["--content"]:
        raise ValueError("usage: run_native_with_phoenix.py [--content]")
    content = sys.argv[1:] == ["--content"]
    if content:
        print("OBS-04A: exporting full provider payloads. Use synthetic test accounts and fictional content ONLY.", file=sys.stderr)
    env = local_go_environment(os.environ, read, content=content)
    # Opt-in *dev tracer* refuses a dead or unauthenticated collector instead
    # of quietly telling an engineer that missing spans mean no AI activity.
    base = "http://127.0.0.1:6006"
    preflight(base)
    status, _ = _call(base, "/v1/projects", key=read["THEIA_OTEL_CONTENT_API_KEY"])
    if status != 200:
        raise ValueError("The provided Phoenix ingestion key is invalid or unauthorized")
    # The local command loader picks up DB/model configuration from .env, but
    # cannot overwrite the explicit metadata-only tracing settings above.
    command = [sys.executable, str(ROOT / "scripts/run_local.py"), "go", "run", "./cmd/api"]
    os.execvpe(command[0], command, env)


if __name__ == "__main__":
    try:
        main()
    except (ValueError, OSError, RuntimeError) as error:
        raise SystemExit(f"Native Phoenix tracer launch failed: {type(error).__name__}: {error}") from None
