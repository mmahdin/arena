#!/usr/bin/env bash
# Theia local development launcher (Linux / Ubuntu).
# No migrations, no source edits, no Docker image rebuilds.
set -Eeuo pipefail
umask 077

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"

usage() {
  cat <<'HELP'
Usage:
  ./theia-dev.sh
  ./theia-dev.sh /absolute/path/to/ai-therapist-backend /absolute/path/to/ai-therapist-frontend

Place this script alongside the two repository directories, or inside either
repository, to enable automatic directory discovery. For a different layout,
pass both paths explicitly (or set THEIA_BACKEND_DIR and THEIA_FRONTEND_DIR).

Starts: MSSQL + Keycloak/Postgres (Docker), Go API (:18080), Vite (:5173),
and same-origin Auth Edge (:8080).
Open: http://app.localhost:8080  (NOT http://127.0.0.1:5173)
Does not run migrations. Ctrl+C stops Go/Vite/Auth Edge; Docker stays up.
Logs: ${THEIA_LOG_DIR:-$HOME/.local/state/theia-dev}
HELP
}

if [[ "${1:-}" == '-h' || "${1:-}" == '--help' ]]; then
  usage
  exit 0
fi
if (( $# > 2 )); then
  usage >&2
  exit 2
fi

find_repo() {
  local marker="$1" sibling="$2" candidate
  for candidate in "$SCRIPT_DIR" "$SCRIPT_DIR/$sibling" "$SCRIPT_DIR/../$sibling" \
                   "$PWD" "$PWD/$sibling" "$PWD/../$sibling"; do
    if [[ -f "$candidate/$marker" ]]; then
      (cd -- "$candidate" && pwd -P)
      return 0
    fi
  done
  return 1
}

BACKEND_DIR="${1:-${THEIA_BACKEND_DIR:-}}"
FRONTEND_DIR="${2:-${THEIA_FRONTEND_DIR:-}}"
if [[ -z "$BACKEND_DIR" ]]; then
  BACKEND_DIR="$(find_repo 'compose.auth-sms.yaml' 'ai-therapist-backend')" || true
fi
if [[ -z "$FRONTEND_DIR" ]]; then
  FRONTEND_DIR="$(find_repo 'scripts/auth-dev-edge.mjs' 'ai-therapist-frontend')" || true
fi
if [[ ! -f "$BACKEND_DIR/Makefile" || ! -f "$BACKEND_DIR/compose.auth-sms.yaml" ]]; then
  printf 'Cannot find the backend repository. Pass its directory as argument 1.\n' >&2
  usage >&2
  exit 2
fi
if [[ ! -f "$FRONTEND_DIR/package.json" || ! -f "$FRONTEND_DIR/scripts/auth-dev-edge.mjs" ]]; then
  printf 'Cannot find the frontend repository. Pass its directory as argument 2.\n' >&2
  usage >&2
  exit 2
fi
BACKEND_DIR="$(cd -- "$BACKEND_DIR" && pwd -P)"
FRONTEND_DIR="$(cd -- "$FRONTEND_DIR" && pwd -P)"

for cmd in docker make go python3 node npm curl setsid; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    printf 'Missing command: %s\n' "$cmd" >&2
    exit 1
  fi
done
if [[ ! -f "$BACKEND_DIR/.env" || ! -f "$BACKEND_DIR/.env.auth" ]]; then
  printf 'Missing backend .env or .env.auth. Complete local Auth setup first.\n' >&2
  exit 1
fi

# Both ports must remain free: Go on 18080; Edge owns 8080, Vite owns 5173.
for port in 18080 5173 8080; do
  if (exec 3<>"/dev/tcp/127.0.0.1/$port") 2>/dev/null; then
    printf 'Port %s is already in use. Stop the previous Go/Vite/Edge process and retry.\n' "$port" >&2
    exit 1
  fi
done

LOG_DIR="${THEIA_LOG_DIR:-${XDG_STATE_HOME:-$HOME/.local/state}/theia-dev}"
mkdir -p -- "$LOG_DIR"
for service in api vite edge; do : > "$LOG_DIR/$service.log"; done
api_pid='' vite_pid='' edge_pid='' tail_pid=''

stop_group() {
  local pid="$1" pgid
  [[ -n "$pid" ]] || return 0
  if kill -0 "$pid" 2>/dev/null; then
    pgid="$(ps -o pgid= -p "$pid" 2>/dev/null | tr -d ' ')" || true
    if [[ "$pgid" == "$pid" ]]; then
      kill -TERM -- "-$pid" 2>/dev/null || true
    else
      kill -TERM "$pid" 2>/dev/null || true
    fi
  fi
}
cleanup() {
  trap - EXIT INT TERM
  printf '\nStopping Go, Vite and Auth Edge (Docker services remain running)...\n'
  [[ -z "$tail_pid" ]] || kill "$tail_pid" 2>/dev/null || true
  stop_group "$edge_pid"
  stop_group "$vite_pid"
  stop_group "$api_pid"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

start_service() {
  local kind="$1" dir="$2" logfile="$3"
  shift 3
  (
    cd -- "$dir"
    exec setsid "$@"
  ) > "$logfile" 2>&1 &
  started_pid="$!"
  printf 'Started %-8s pid=%s; log=%s\n' "$kind" "$started_pid" "$logfile"
}

wait_for() {
  local label="$1" url="$2" pid="$3" logfile="$4" limit="$5" i
  printf 'Checking %s readiness' "$label"
  for ((i=0; i<limit; i++)); do
    if curl -fsS --max-time 2 -o /dev/null "$url" 2>/dev/null; then
      printf ' OK\n'
      return 0
    fi
    if [[ -n "$pid" ]] && ! kill -0 "$pid" 2>/dev/null; then
      printf ' FAILED (process exited)\n' >&2
      tail -n 35 -- "$logfile" >&2 || true
      return 1
    fi
    printf '.'
    sleep 1
  done
  printf ' TIMEOUT\n' >&2
  [[ ! -f "$logfile" ]] || tail -n 35 -- "$logfile" >&2
  return 1
}

printf 'Backend:  %s\nFrontend: %s\n' "$BACKEND_DIR" "$FRONTEND_DIR"
printf '\n[1/4] Starting MSSQL, PostgreSQL and SMS-enabled Keycloak...\n'
(
  cd -- "$BACKEND_DIR"
  docker compose --env-file .env --env-file .env.auth \
    -f compose.yaml -f compose.auth.yaml -f compose.auth-sms.yaml \
    --profile auth up -d --no-build mssql keycloak
)
wait_for 'Keycloak' \
  'http://auth.localhost:8180/realms/theia-users/.well-known/openid-configuration' \
  '' '' 120

if [[ ! -d "$FRONTEND_DIR/node_modules" ]]; then
  printf '\nFrontend dependencies missing; running npm ci once...\n'
  (cd -- "$FRONTEND_DIR" && npm ci)
fi

printf '\n[2/4] Starting Go API on 127.0.0.1:18080...\n'
start_service api "$BACKEND_DIR" "$LOG_DIR/api.log" \
  env HTTP_ADDR=127.0.0.1:18080 make api
api_pid="$started_pid"
wait_for 'Go API' 'http://127.0.0.1:18080/health/ready' "$api_pid" "$LOG_DIR/api.log" 180

printf '\n[3/4] Starting Vite on 127.0.0.1:5173...\n'
start_service vite "$FRONTEND_DIR" "$LOG_DIR/vite.log" \
  env THEIA_DEV_API_TARGET=http://127.0.0.1:18080 npm run dev
vite_pid="$started_pid"
wait_for 'Vite' 'http://127.0.0.1:5173/' "$vite_pid" "$LOG_DIR/vite.log" 60

printf '\n[4/4] Starting same-origin Auth Edge on app.localhost:8080...\n'
start_service edge "$FRONTEND_DIR" "$LOG_DIR/edge.log" \
  env THEIA_DEV_API_TARGET=http://127.0.0.1:18080 npm run dev:auth-edge
edge_pid="$started_pid"
wait_for 'Auth Edge' 'http://app.localhost:8080/' "$edge_pid" "$LOG_DIR/edge.log" 30

cat <<'DONE'

Theia is ready!
  App:      http://app.localhost:8080
  Keycloak: http://auth.localhost:8180/admin/
  Note:     Use the app.localhost URL, NOT the direct Vite :5173 address.
  Migrate:  Run 'make migrate' yourself when needed (inside backend).

Live service logs follow. Press Ctrl+C to stop Go/Vite/Auth Edge.
Docker MSSQL/Keycloak remain running for faster next startup.
DONE

tail -n 8 -F -- "$LOG_DIR/api.log" "$LOG_DIR/vite.log" "$LOG_DIR/edge.log" &
tail_pid="$!"
wait -n "$api_pid" "$vite_pid" "$edge_pid" || true
printf '\nOne of the local services exited; stopping the other local services.\n' >&2
exit 1
