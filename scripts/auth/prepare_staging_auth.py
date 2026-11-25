#!/usr/bin/env python3
"""Prepare private Theia staging Auth secrets; optionally apply prerequisites.

Never prints credentials or sends SMS. Existing generated credentials are reused.
Only the dedicated Theia PostgreSQL resources and named Theia config/secrets
are managed; the shared Ramona identity service is not changed.
"""
import argparse
import base64
import ipaddress
import json
import os
from pathlib import Path
import re
import secrets
import subprocess
import sys

import yaml

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from run_local import env_entries

ROOT = Path(__file__).resolve().parents[2]
AUTH_ORIGIN = "https://ai-therapist-auth.alibaba.stg.agidp.ir"
USER_ORIGIN = "https://ai-therapist-frontend.alibaba.stg.agidp.ir"
STAFF_ORIGIN = "https://theia-backoffice.alibaba.stg.agidp.ir"
INTERNAL_ORIGIN = "http://ai-therapist-backend-service.ramona.svc.cluster.local:80"
SECRET_KEYS = (
    "KEYCLOAK_DB_PASSWORD", "KEYCLOAK_BOOTSTRAP_ADMIN_PASSWORD",
    "THEIA_USERS_BFF_CLIENT_SECRET", "THEIA_STAFF_BFF_CLIENT_SECRET",
    "AUTH_BFF_KEY_BASE64", "THEIA_SMS_HMAC_KEY_BASE64",
)


def read_env(path):
    return {key: value for key, value, _ in env_entries(path.read_text())}


def private_write(path, text):
    if path.is_symlink() or (path.exists() and not path.is_file()):
        raise ValueError("Private output must be a regular file")
    fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_TRUNC, 0o600)
    os.fchmod(fd, 0o600)
    with os.fdopen(fd, "w") as stream:
        stream.write(text)


def proxy_networks(value):
    result = []
    for item in value.split(","):
        network = ipaddress.ip_network(item.strip(), strict=False)
        if network.prefixlen == 0 or not network.is_private:
            raise ValueError("Only explicit private proxy networks are allowed")
        if str(network) not in result:
            result.append(str(network))
    if not result:
        raise ValueError("Private staging proxy networks are required")
    return ",".join(result)


def prepare(output, provider_env, template, proxies):
    output.mkdir(parents=True, exist_ok=True, mode=0o700)
    if output.is_symlink() or output.stat().st_mode & 0o077:
        raise ValueError("Private output directory must have mode 0700")
    provider = read_env(provider_env)
    api_key = provider.get("THEIA_SMS_KAVENEGAR_API_KEY", "")
    if not re.fullmatch(r"[A-Za-z0-9]{20,200}", api_key):
        raise ValueError("A valid Kavenegar key is required in the private provider env")
    if not re.fullmatch(r"[A-Za-z0-9_-]{1,64}", template):
        raise ValueError("Invalid Kavenegar template name")
    credentials = output / "credentials.env"
    values = read_env(credentials) if credentials.exists() else {}
    if values and any(not values.get(key) for key in SECRET_KEYS):
        raise ValueError("Existing staging credentials are incomplete; refusing rotation")
    if not values:
        values = {key: base64.b64encode(secrets.token_bytes(32)).decode()
                  if key.endswith("BASE64") else secrets.token_urlsafe(36)
                  for key in SECRET_KEYS}
        private_write(credentials, "".join(f"{key}={value}\n" for key, value in values.items()))
    for key in ("AUTH_BFF_KEY_BASE64", "THEIA_SMS_HMAC_KEY_BASE64"):
        if len(base64.b64decode(values[key], validate=True)) != 32:
            raise ValueError("Invalid stored staging encryption key")

    def resource(kind, name, data, secret=False):
        return {"apiVersion": "v1", "kind": kind,
                "metadata": {"name": name, "namespace": "ramona"},
                **({"type": "Opaque", "stringData": data} if secret else {"data": data})}

    auth_config = {
        "KC_DB": "postgres", "KC_DB_URL": "jdbc:postgresql://ai-therapist-auth-postgres:5432/theia_keycloak",
        "KC_DB_USERNAME": "theia_keycloak", "KC_HTTP_ENABLED": "true", "KC_HTTP_PORT": "8080",
        # A full canonical HTTPS hostname pins Keycloak's public scheme/host.
        # Do not enable forwarded-header parsing or require Gateway CIDRs.
        "KC_HOSTNAME": AUTH_ORIGIN, "KC_HOSTNAME_STRICT": "true", "KC_CACHE": "ispn",
        "KC_HEALTH_ENABLED": "true", "KC_METRICS_ENABLED": "true",
        "KC_BOOTSTRAP_ADMIN_USERNAME": "theia-stage-bootstrap",
        "THEIA_SMS_ENV": "stage", "THEIA_SMS_TRANSPORT": "kavenegar",
        "THEIA_SMS_KAVENEGAR_TEMPLATE": template, "AUTH_USER_ORIGIN": USER_ORIGIN,
        "AUTH_STAFF_ORIGIN": STAFF_ORIGIN, "THEIA_BACKEND_INTERNAL_ORIGIN": INTERNAL_ORIGIN,
    }
    auth_secret = {
        "KC_DB_PASSWORD": values["KEYCLOAK_DB_PASSWORD"],
        "KC_BOOTSTRAP_ADMIN_PASSWORD": values["KEYCLOAK_BOOTSTRAP_ADMIN_PASSWORD"],
        "THEIA_USERS_BFF_CLIENT_SECRET": values["THEIA_USERS_BFF_CLIENT_SECRET"],
        "THEIA_STAFF_BFF_CLIENT_SECRET": values["THEIA_STAFF_BFF_CLIENT_SECRET"],
        "THEIA_SMS_HMAC_KEY_BASE64": values["THEIA_SMS_HMAC_KEY_BASE64"],
        "THEIA_SMS_KAVENEGAR_API_KEY": api_key,
    }
    backend_config = {
        "ALLOW_DEV_LOGIN": "false", "AUTH_BFF_ENABLED": "true",
        "AUTH_USER_LOGIN_ENABLED": "true", "AUTH_USER_SMS_FLOW_VERIFIED": "true",
        "AUTH_STAFF_LOGIN_ENABLED": "false", "AUTH_BROWSER_SMOKE_ENABLED": "false",
        "AUTH_KEYCLOAK_BASE_URL": AUTH_ORIGIN, "AUTH_USER_ORIGIN": USER_ORIGIN,
        "AUTH_STAFF_ORIGIN": STAFF_ORIGIN, "AUTH_TRUSTED_PROXY_CIDRS": proxies,
        "APP_ALLOWED_ORIGIN": USER_ORIGIN,
    }
    # Merge-patch files deliberately omit all unrelated AI/database settings.
    backend_secret = {key: base64.b64encode(values[key].encode()).decode()
                      for key in ("AUTH_BFF_KEY_BASE64", "THEIA_USERS_BFF_CLIENT_SECRET", "THEIA_STAFF_BFF_CLIENT_SECRET")}
    manifests = {
        "auth-configmap.yaml": resource("ConfigMap", "ai-therapist-auth-configmap", auth_config),
        "auth-secret.yaml": resource("Secret", "ai-therapist-auth-secret", auth_secret, True),
        "postgres-secret.yaml": resource("Secret", "ai-therapist-auth-postgres-secret", {
            "POSTGRES_DB": "theia_keycloak", "POSTGRES_USER": "theia_keycloak",
            "POSTGRES_PASSWORD": values["KEYCLOAK_DB_PASSWORD"]}, True),
        "frontend-configmap.yaml": resource("ConfigMap", "ai-therapist-frontend-configmap", {
            "API_UPSTREAM": INTERNAL_ORIGIN, "PUBLIC_SCHEME": "https"}),
        "backend-configmap.patch.yaml": {"data": backend_config},
        "backend-secret.patch.yaml": {"data": backend_secret},
    }
    for name, document in manifests.items():
        private_write(output / name, yaml.safe_dump(document, sort_keys=False))


def kubectl(context, *args, capture=False):
    result = subprocess.run(["kubectl", "--context", context, "--namespace", "ramona", "--request-timeout=30s", *args],
                            capture_output=True, text=True)
    if result.returncode:
        # Server errors may echo Secret bodies. Never forward arbitrary stderr.
        raise RuntimeError("kubectl operation failed; inspect the cluster privately")
    return result.stdout if capture else None


def discovered_proxies(context):
    # Namespace-scoped operators need no Node/CNI/Gateway read permission.
    # Trust only currently ready frontend peers; refresh after their replacement.
    pods = json.loads(kubectl(context, "get", "pods", "-l", "app=ai-therapist-frontend", "-o", "json", capture=True))
    addresses = []
    for pod in pods.get("items", []):
        if pod.get("metadata", {}).get("deletionTimestamp") or not any(
            c.get("type") == "Ready" and c.get("status") == "True"
            for c in pod.get("status", {}).get("conditions", [])
        ):
            continue
        for address in pod.get("status", {}).get("podIPs", []):
            ip = ipaddress.ip_address(address["ip"])
            addresses.append(f"{ip}/{ip.max_prefixlen}")
    return proxy_networks(",".join(addresses))


def verify_existing_credentials(context, output):
    # A lost local credential file must never rotate an existing PostgreSQL
    # password or make a persisted realm's confidential clients unusable.
    for filename, name in (("postgres-secret.yaml", "ai-therapist-auth-postgres-secret"),
                           ("auth-secret.yaml", "ai-therapist-auth-secret")):
        raw = kubectl(context, "get", "secret", name, "--ignore-not-found", "-o", "json", capture=True)
        if not raw.strip():
            continue
        existing = json.loads(raw).get("data", {})
        desired = yaml.safe_load((output / filename).read_text())["stringData"]
        for key, value in desired.items():
            if key == "THEIA_SMS_KAVENEGAR_API_KEY":
                continue  # An explicitly configured provider-key update is allowed.
            if base64.b64decode(existing.get(key, ""), validate=True).decode() != value:
                raise ValueError("Existing staging credentials differ; restore the original private credential file")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", type=Path, default=ROOT / ".local/staging-auth")
    parser.add_argument("--provider-env", type=Path, default=ROOT / ".env.auth")
    parser.add_argument("--sms-template", default="loginotp")
    parser.add_argument("--trusted-proxy-cidrs", default="")
    parser.add_argument("--apply", action="store_true")
    parser.add_argument("--config-only", action="store_true", help="Apply only ConfigMaps/Secrets; workload creation requires separate deployment permission")
    parser.add_argument("--context", help="Explicit staging kubectl context; never changes the default context")
    parser.add_argument("--launchpad-root", type=Path)
    args = parser.parse_args()
    if args.apply and (not args.context or (not args.config_only and not args.launchpad_root)):
        parser.error("--apply requires --context and either --launchpad-root or --config-only")
    if args.config_only and not args.apply:
        parser.error("--config-only requires --apply")
    if args.apply:
        # Both resources must already exist: never replace a backend installation.
        kubectl(args.context, "get", "configmap", "ai-therapist-backend-configmap")
        kubectl(args.context, "get", "secret", "ai-therapist-backend-secret")
        if not args.config_only:
            manifest = args.launchpad_root / "manifests/ai-therapist-auth-postgres.yaml"
            if not manifest.is_file():
                parser.error("Dedicated Theia PostgreSQL manifest is missing")
    proxies = (proxy_networks(args.trusted_proxy_cidrs) if args.trusted_proxy_cidrs else
               discovered_proxies(args.context) if args.apply else "127.0.0.1/32")
    prepare(args.output, args.provider_env, args.sms_template, proxies)
    print("Private staging credentials and Auth/BFF manifests prepared; existing credentials retained.")
    if not args.apply:
        print("Not applied. Run --apply with the staging context to discover the ready frontend peer addresses.")
        return
    verify_existing_credentials(args.context, args.output)
    for name in ("postgres-secret.yaml", "auth-configmap.yaml", "auth-secret.yaml", "frontend-configmap.yaml"):
        kubectl(args.context, "apply", "-f", str(args.output / name))
    if not args.config_only:
        kubectl(args.context, "apply", "-f", str(manifest))
        kubectl(args.context, "rollout", "status", "statefulset/ai-therapist-auth-postgres", "--timeout=180s")
    for kind, name, file in (("configmap", "ai-therapist-backend-configmap", "backend-configmap.patch.yaml"),
                             ("secret", "ai-therapist-backend-secret", "backend-secret.patch.yaml")):
        kubectl(args.context, "patch", kind, name, "--type=merge", "--patch-file", str(args.output / file))
    print("Auth ConfigMaps/Secrets applied; unrelated backend settings preserved.")
    if args.config_only:
        print("Workloads were not created or restarted. PostgreSQL/Auth deployment requires the deployment operator or CI.")
    else:
        print("Staging database ready. Deploy the dedicated Auth image, then the backend and frontend.")


if __name__ == "__main__":
    try:
        main()
    except (OSError, ValueError, RuntimeError, yaml.YAMLError) as error:
        # Do not reveal raw exceptions: filesystem, provider and cluster data may be private.
        raise SystemExit(f"Staging Auth setup failed ({type(error).__name__}); inspect configuration privately.")
