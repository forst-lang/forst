#!/usr/bin/env bash
# kill-forst-tcp-listeners.sh <port> [signal]
# Kill only processes listening on TCP <port> that look like Forst (CLI, dev/run, or generated invoke).
set -euo pipefail

PORT=${1:?usage: kill-forst-tcp-listeners.sh PORT [SIGNAL]}
SIGNAL=${2:-9}

is_forst_listener_cmd() {
  local cmd="$1"
  [[ -z "$cmd" ]] && return 1
  echo "$cmd" | grep -Eq \
    '(/forst([[:space:]]|$)|(^|[[:space:]])forst[[:space:]]+(dev|run|lsp|test|build|generate))|\.forst/run/|forst_invoke_server\.gen\.go)'
}

listener_pids() {
  lsof -ti "tcp:${PORT}" 2>/dev/null || true
}

for pid in $(listener_pids); do
  cmd=$(ps -p "$pid" -o args= 2>/dev/null || ps -p "$pid" -o command= 2>/dev/null || true)
  if is_forst_listener_cmd "$cmd"; then
    kill "-${SIGNAL}" "$pid" 2>/dev/null || true
  fi
done
