#!/usr/bin/env bash
# E2E or dev: remix-serve from a temp dir with local @forst/* file: deps (no registry Forst downloads).
#
# Usage:
#   scripts/remix-serve-standalone-e2e.sh           # run checks, then exit (cleans up)
#   scripts/remix-serve-standalone-e2e.sh --dev       # keep servers until Ctrl+C, then clean up
#   REMIX_SERVE_STANDALONE_DEV=1 ...                  # same as --dev
set -euo pipefail
set +m

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO="${REPO:-$(cd "$SCRIPT_DIR/.." && git rev-parse --show-toplevel 2>/dev/null || echo "$SCRIPT_DIR/..")}"
REPO="$(cd "$REPO" && pwd)"

FORST_BINARY="${FORST_BINARY:-$REPO/bin/forst}"
FORST_GOMOD_ROOT="${FORST_GOMOD_ROOT:-$REPO/forst}"
EXAMPLE_SRC="$REPO/examples/in/rfc/node-interop/remix-serve"
REGISTER_MJS="$REPO/packages/node-runtime/dist/host/register.mjs"
LOG_FILE="${LOG_FILE:-$REPO/.cursor-remix-serve-standalone-e2e.log}"

DEV_MODE=false
for arg in "$@"; do
  case "$arg" in
    --dev) DEV_MODE=true ;;
    -h|--help)
      echo "usage: $0 [--dev]"
      echo "  (default) build temp project, smoke-test :8081/:3000, clean up"
      echo "  --dev     same setup, print URLs, block until Ctrl+C, then clean up"
      exit 0
      ;;
  esac
done
if [[ "${REMIX_SERVE_STANDALONE_DEV:-}" == "1" ]]; then
  DEV_MODE=true
fi

TMP=""
FORST_PID=""
CLEANED_UP=false

free_ports() {
  lsof -ti tcp:8081 2>/dev/null | xargs kill -9 2>/dev/null || true
  lsof -ti tcp:3000 2>/dev/null | xargs kill -9 2>/dev/null || true
}

kill_forst_tree() {
  if [[ -z "$FORST_PID" ]]; then
    return 0
  fi
  kill -TERM "$FORST_PID" 2>/dev/null || true
  sleep 0.2
  kill -KILL "$FORST_PID" 2>/dev/null || true
  wait "$FORST_PID" 2>/dev/null || true
  FORST_PID=""
}

cleanup() {
  local code="${1:-$?}"
  if [[ "$CLEANED_UP" == true ]]; then
    exit "$code"
  fi
  CLEANED_UP=true
  trap - EXIT INT TERM

  echo "=== cleanup ==="
  kill_forst_tree
  if [[ -n "$TMP" ]]; then
    pkill -f "forst run.*${TMP}/main.ft" 2>/dev/null || true
    pkill -f "remix-serve ${TMP}/build/server/index.js" 2>/dev/null || true
  fi
  free_ports
  if [[ -n "$TMP" && -d "$TMP" ]]; then
    rm -rf "$TMP"
    echo "removed temp project: $TMP"
  fi
  echo "=== cleanup done ==="
  exit "$code"
}
trap 'cleanup $?' EXIT
trap 'cleanup 130' INT
trap 'cleanup 143' TERM

if [[ ! -x "$FORST_BINARY" ]]; then
  echo "missing compiler binary: $FORST_BINARY (run: task build)" >&2
  exit 1
fi
if [[ ! -f "$REGISTER_MJS" ]]; then
  echo "missing node-runtime host register: $REGISTER_MJS (run: task build:node-runtime)" >&2
  exit 1
fi

echo "=== build monorepo artifacts ==="
(cd "$REPO" && task build:node-runtime build:client build:sidecar build)

TMP="$(mktemp -d)"

echo "=== copy example to $TMP ==="
rsync -a \
  --exclude node_modules \
  --exclude .forst \
  --exclude build \
  --exclude client \
  --exclude generated \
  --exclude 'app/lib/forst.invoke.ts' \
  "$EXAMPLE_SRC/" "$TMP/"

echo "=== patch @forst/* deps ==="
node "$SCRIPT_DIR/patch-remix-serve-standalone-deps.mjs" "$TMP" "$REPO"

echo "=== bun install in temp ==="
(cd "$TMP" && bun install 2>&1 | tee -a "$LOG_FILE")

echo "=== assert local @forst packages ==="
node "$SCRIPT_DIR/assert-local-forst-packages.mjs" "$TMP" "$REPO"

echo "=== forst generate ==="
FORST_BOUNDARY_ROOT="$TMP" "$FORST_BINARY" generate "$TMP"

echo "=== remix build ==="
(cd "$TMP" && bun run build)

echo "=== free ports ==="
free_ports
sleep 0.3

echo "=== forst run ==="
export FORST_BOUNDARY_ROOT="$TMP"
export FORST_GOMOD_ROOT

if [[ "$DEV_MODE" == true ]]; then
  (
    cd "$TMP"
    exec "$FORST_BINARY" run -export-struct-fields -root "$TMP" -- "$TMP/main.ft"
  ) &
  FORST_PID=$!
else
  (
    cd "$TMP"
    "$FORST_BINARY" run -export-struct-fields -root "$TMP" -- "$TMP/main.ft"
  ) >"$LOG_FILE" 2>&1 &
  FORST_PID=$!
fi

bash "$SCRIPT_DIR/wait-for-url.sh" http://127.0.0.1:8081/health "invoke :8081" 120
bash "$SCRIPT_DIR/wait-for-url.sh" http://127.0.0.1:3000/ "remix :3000" 120

if [[ "$DEV_MODE" == true ]]; then
  echo ""
  echo "=== standalone remix-serve dev ==="
  echo "temp project: $TMP"
  echo "Remix:        http://127.0.0.1:3000/"
  echo "Invoke:       http://127.0.0.1:8081/health"
  echo "log:         stdout/stderr (this terminal)"
  echo "Press Ctrl+C to stop — temp dir and listeners will be cleaned up."
  echo ""
  wait "$FORST_PID" || true
  cleanup 0
fi

for line in sync:2 sync:3 async:ok gen:1 events:2; do
  grep -q "$line" "$LOG_FILE" || {
    echo "missing stdout line: $line" >&2
    tail -50 "$LOG_FILE" >&2 || true
    exit 1
  }
done

curl -sf http://127.0.0.1:8081/health | grep -q 'healthy'
curl -sf http://127.0.0.1:3000/ | grep -q 'Todos'

echo "=== standalone remix-serve e2e OK ==="
# EXIT trap runs cleanup before the shell terminates.
