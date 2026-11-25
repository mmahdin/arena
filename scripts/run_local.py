#!/usr/bin/env python3
"""Run local Go commands with .env values, never evaluating shell code.

Before launching, copy missing .env.example keys into .env without replacing
existing values. Exported environment variables take precedence over .env.
When Auth BFF is enabled, fill missing BFF credentials from .env.auth.
"""
import fcntl
import os
from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[1]
AUTH_CREDENTIAL_KEYS = {
    "AUTH_BFF_KEY_BASE64",
    "THEIA_USERS_BFF_CLIENT_SECRET",
    "THEIA_STAFF_BFF_CLIENT_SECRET",
}


def env_entries(content: str):
    """Yield (key, literal value, original line); never expand shell syntax."""
    for raw in content.splitlines():
        line = raw.strip()
        if not line or line.startswith("#"):
            continue
        if line.startswith("export "):
            line = line[7:].strip()
        if "=" not in line:
            continue
        key, value = line.split("=", 1)
        key, value = key.strip(), value.strip()
        if not key.isidentifier() or not key.isascii():
            continue
        if value.startswith(('"', "'")) and len(value) >= 2 and value.endswith(value[0]):
            value = value[1:-1]
        else:
            value = value.split(" #", 1)[0].rstrip()
        yield key, value, raw


def sync_env(root: Path) -> list[str]:
    """Append new defaults, including empty values; serialize concurrent starts."""
    example = root / ".env.example"
    if not example.is_file():
        return []
    template = example.read_bytes()
    entries = list(env_entries(template.decode("utf-8-sig")))
    # Append in place so existing bytes, permissions, secrets and comments stay
    # intact. Lock the same file for simultaneous make api / make migrate runs.
    fd = os.open(root / ".env", os.O_RDWR | os.O_CREAT, 0o600)
    with os.fdopen(fd, "r+b") as dotenv:
        fcntl.flock(dotenv, fcntl.LOCK_EX)
        original = dotenv.read()
        present = {key for key, _, _ in env_entries(original.decode("utf-8-sig"))}
        missing = []
        lines = []
        for key, _, raw in entries:
            if key not in present:
                missing.append(key)
                lines.append(raw)
                present.add(key)
        if missing:
            if not original:
                addition = template
            else:
                newline = b"\r\n" if b"\r\n" in original else b"\n"
                separator = b"" if original.endswith(b"\n") else newline
                addition = (separator + newline
                            + b"# Added from .env.example (missing keys only)." + newline
                            + newline.join(line.encode("utf-8") for line in lines) + newline)
            dotenv.write(addition)
            dotenv.flush()
            os.fsync(dotenv.fileno())
    return missing


def local_env(root: Path, inherited: dict[str, str]) -> dict[str, str]:
    env = inherited.copy()
    dotenv = root / ".env"
    if dotenv.is_file():
        for key, value, _ in env_entries(dotenv.read_text(encoding="utf-8-sig")):
            env.setdefault(key, value)
    auth_dotenv = root / ".env.auth"
    if env.get("AUTH_BFF_ENABLED") == "true" and auth_dotenv.is_file():
        for key, value, _ in env_entries(auth_dotenv.read_text(encoding="utf-8-sig")):
            if key in AUTH_CREDENTIAL_KEYS:
                env.setdefault(key, value)
    return env


def main(args: list[str]) -> None:
    if not args or (args[0].startswith("--") and args != ["--sync-env"]):
        sys.exit("Usage: python3 scripts/run_local.py [--sync-env | go run ./cmd/api]")
    added = sync_env(ROOT)
    if added:
        print("Updated .env with missing keys: " + ", ".join(added), file=sys.stderr)
    if args == ["--sync-env"]:
        return
    env = local_env(ROOT, os.environ)
    os.chdir(ROOT / "apps/api")
    os.execvpe(args[0], args, env)


if __name__ == "__main__":
    try:
        main(sys.argv[1:])
    except (OSError, UnicodeError) as error:
        sys.exit(f"Local setup failed ({type(error).__name__}); check .env, .env.example, .env.auth and command permissions.")
