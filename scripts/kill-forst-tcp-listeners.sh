#!/usr/bin/env bash
# kill-forst-tcp-listeners.sh <port> [--force] [signal]
# Kill processes listening on TCP <port> that look like Forst (CLI, dev/run, or generated invoke).
# With --force (or FORST_KILL_ALL_PORT_LISTENERS=1), SIGKILL any remaining listener on the port.
set -euo pipefail

PORT=${1:?usage: kill-forst-tcp-listeners.sh PORT [--force] [SIGNAL]}
shift

FORCE=false
SIGNAL=9
while [[ $# -gt 0 ]]; do
  case "$1" in
    --force) FORCE=true; shift ;;
    *)
      SIGNAL=$1
      shift
      ;;
  esac
done
if [[ "${FORST_KILL_ALL_PORT_LISTENERS:-}" == "1" ]]; then
  FORCE=true
fi

is_forst_listener_cmd() {
  local cmd="$1"
  [[ -z "$cmd" ]] && return 1
  echo "$cmd" | grep -Eq \
    '(/forst([[:space:]]|$)|(^|[[:space:]])forst[[:space:]]+(dev|run|lsp|test|build|generate))|\.forst/run/|forst_invoke_server\.gen\.go|(^|[[:space:]/])forst\.run([[:space:]]|$)'
}

listener_pids() {
  lsof -ti "tcp:${PORT}" 2>/dev/null || true
}

kill_remaining_listeners() {
  local pid
  for pid in $(listener_pids); do
    kill "-${SIGNAL}" "$pid" 2>/dev/null || true
  done
}

for pid in $(listener_pids); do
  cmd=$(ps -p "$pid" -o args= 2>/dev/null || ps -p "$pid" -o command= 2>/dev/null || true)
  if is_forst_listener_cmd "$cmd"; then
    kill "-${SIGNAL}" "$pid" 2>/dev/null || true
  fi
done

if [[ "$FORCE" == true ]]; then
  kill_remaining_listeners
fi
