#!/usr/bin/env bash
# E2E: multipackage-dev forst dev reload keeps the parent-owned Node host pid alive.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO="${REPO:-$(cd "$SCRIPT_DIR/.." && git rev-parse --show-toplevel 2>/dev/null || echo "$SCRIPT_DIR/..")}"
REPO="$(cd "$REPO" && pwd)"

FORST_BINARY="${FORST_BINARY:-$REPO/bin/forst}"
FT_ROOT="$REPO/examples/in/rfc/node-interop/multi-package-dev"
HOST_READY="$FT_ROOT/.forst/node.sock.ready"
RELOAD_MARKER="$FT_ROOT/.forst/reloading"

FORST_PID=""
CLEANED_UP=false

free_port() {
  bash "$SCRIPT_DIR/kill-forst-tcp-listeners.sh" 6321
}

cleanup() {
  local code="${1:-$?}"
  if [[ "$CLEANED_UP" == true ]]; then
    exit "$code"
  fi
  CLEANED_UP=true
  trap - EXIT INT TERM
  if [[ -n "$FORST_PID" ]]; then
    kill -INT "$FORST_PID" 2>/dev/null || true
    sleep 0.5
    kill -KILL "$FORST_PID" 2>/dev/null || true
    wait "$FORST_PID" 2>/dev/null || true
  fi
  free_port
  exit "$code"
}
trap 'cleanup $?' EXIT

if [[ ! -x "$FORST_BINARY" ]]; then
  echo "forst binary missing: $FORST_BINARY" >&2
  exit 1
fi
if [[ ! -f "$REPO/packages/node-runtime/dist/host.js" ]]; then
  echo "node-runtime not built: $REPO/packages/node-runtime/dist/host.js" >&2
  exit 1
fi

free_port
rm -rf "$FT_ROOT/.forst"

export FORST_REPO_ROOT="$REPO"
export FORST_BOUNDARY_ROOT="$FT_ROOT"
"$FORST_BINARY" dev \
  -export-struct-fields \
  -root "$FT_ROOT" \
  -entry main.ft \
  -log-level error &
FORST_PID=$!

bash "$SCRIPT_DIR/wait-for-url.sh" http://127.0.0.1:6321/health invoke 60

read_host_pid() {
  python3 - <<'PY' "$HOST_READY"
import json, sys
with open(sys.argv[1]) as f:
    print(json.load(f)["pid"])
PY
}

pid0="$(read_host_pid)"
if [[ -z "$pid0" || "$pid0" -le 0 ]]; then
  echo "node host pid missing from $HOST_READY" >&2
  exit 1
fi
if ! kill -0 "$pid0" 2>/dev/null; then
  echo "node host pid=$pid0 not alive" >&2
  exit 1
fi

touch "$FT_ROOT/bcrypt.ft"

gen=0
for _ in $(seq 1 60); do
  if [[ -f "$RELOAD_MARKER" ]]; then
    gen="$(python3 - <<'PY' "$RELOAD_MARKER"
import json, sys
with open(sys.argv[1]) as f:
    print(json.load(f).get("generation", 0))
PY
)"
  fi
  if curl -sf http://127.0.0.1:6321/health >/dev/null 2>&1; then
    health_gen="$(curl -sf http://127.0.0.1:6321/health | python3 -c 'import json,sys; d=json.load(sys.stdin); print(d.get("result",{}).get("generation", d.get("generation", 0)))' 2>/dev/null || echo 0)"
    if [[ "${health_gen:-0}" -ge 2 ]]; then
      gen="$health_gen"
      break
    fi
  fi
  sleep 1
done

if [[ "${gen:-0}" -lt 2 ]]; then
  echo "reload generation did not reach 2 (got ${gen:-0})" >&2
  exit 1
fi

pid1="$(read_host_pid)"
if [[ "$pid1" != "$pid0" ]]; then
  echo "node host pid changed after reload: $pid0 -> $pid1" >&2
  exit 1
fi
if ! kill -0 "$pid1" 2>/dev/null; then
  echo "node host pid=$pid1 not alive after reload" >&2
  exit 1
fi

echo "multipackage-dev reload e2e ok: host pid=$pid1 generation=$gen"
