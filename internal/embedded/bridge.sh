#!/usr/bin/env bash
set -euo pipefail

# Host-service bridge, run on every session start by blvckhole's per-session
# hook. Makes a host service reachable from inside the sandbox under a stable
# hostname/port without touching the app's .env:
#
#   1. a socat tunnel   127.0.0.1:<PORT> -> host.docker.internal:<HOST_PORT>
#   2. name resolution  <NAME> -> 127.0.0.1 (where socat listens)
#
# Raw TCP to the host is only permitted once the host allows localhost:<HOST_PORT>
# (blvckhole does this on provision); this restores the in-sandbox plumbing that a
# restart wipes. Idempotent.
#
# Usage: bridge.sh <name> <port> <host_port> [env_var]

NAME="${1:?bridge name required}"
PORT="${2:?listen port required}"
HOST_PORT="${3:?host port required}"
ENV_VAR="${4:-}"

HOSTS_FILE="${SANDBOX_HOSTS_FILE:-/etc/hosts}"
ENV_FILE="${CLAUDE_ENV_FILE:-/etc/sandbox-persistent.sh}"
LOG_FILE="${BLVCKHOLE_BRIDGE_LOG:-/tmp/blvckhole-bridge.log}"

log() {
    printf '%s %s\n' "$(date -Is 2>/dev/null || true)" "$1" >>"$LOG_FILE" 2>/dev/null || true
}

# The bridge comes up first: it carries the traffic, and a name-resolution
# failure must never stop it. setsid -f detaches it from the sourcing hook.
if pgrep -f "socat.*TCP-LISTEN:${PORT}" >/dev/null 2>&1; then
    log "= bridge already running ${NAME}:${PORT} -> host:${HOST_PORT}"
elif setsid -f socat "TCP-LISTEN:${PORT},bind=127.0.0.1,reuseaddr,fork" \
    "TCP:host.docker.internal:${HOST_PORT}" </dev/null >/dev/null 2>&1; then
    log "+ bridge started ${NAME}:${PORT} -> host:${HOST_PORT}"
else
    log "! bridge failed to start ${NAME}:${PORT} — is socat installed and localhost:${HOST_PORT} allowed?"
    exit 1
fi

# Prefer an /etc/hosts alias; fall back to an env override when the runtime
# mounts /etc/hosts read-only (an alias is then impossible even under sudo).
if grep -qE "^[^#]*[[:space:]]${NAME}([[:space:]]|\$)" "$HOSTS_FILE" 2>/dev/null; then
    log "= ${HOSTS_FILE}: ${NAME} -> 127.0.0.1 (already present)"
elif echo "127.0.0.1  ${NAME}" | sudo tee -a "$HOSTS_FILE" >/dev/null 2>&1; then
    log "+ ${HOSTS_FILE}: ${NAME} -> 127.0.0.1"
elif [ -z "$ENV_VAR" ]; then
    log "! ${HOSTS_FILE} read-only and no env fallback for ${NAME}; connect to 127.0.0.1:${PORT}"
elif grep -q "BLVCKHOLE_BRIDGE_${ENV_VAR}" "$ENV_FILE" 2>/dev/null; then
    log "= ${ENV_VAR} override already in ${ENV_FILE}"
else
    sudo tee -a "$ENV_FILE" >/dev/null <<EOF

if [ -z "\${BLVCKHOLE_BRIDGE_${ENV_VAR}:-}" ]; then
  export ${ENV_VAR}=127.0.0.1
  export BLVCKHOLE_BRIDGE_${ENV_VAR}=1
fi
EOF
    log "+ ${ENV_VAR}=127.0.0.1 via ${ENV_FILE} (${HOSTS_FILE} read-only)"
fi
