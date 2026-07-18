#!/usr/bin/env bash
# Unit tests for kill-forst-tcp-listeners.sh (cmdline matching and --force cleanup).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

is_forst_listener_cmd() {
  local cmd="$1"
  [[ -z "$cmd" ]] && return 1
  echo "$cmd" | grep -Eq \
    '(/forst([[:space:]]|$)|(^|[[:space:]])forst[[:space:]]+(dev|run|lsp|test|build|generate))|\.forst/run/|forst_invoke_server\.gen\.go|(^|[[:space:]/])forst\.run([[:space:]]|$)'
}

assert_matches() {
  local cmd="$1"
  if ! is_forst_listener_cmd "$cmd"; then
    echo "expected match: $cmd" >&2
    exit 1
  fi
}

assert_no_match() {
  local cmd="$1"
  if is_forst_listener_cmd "$cmd"; then
    echo "expected no match: $cmd" >&2
    exit 1
  fi
}

assert_matches "/usr/local/bin/forst run -root /tmp/app -- main.ft"
assert_matches "/tmp/app/.forst/run/forst-4025813837/forst.run"
assert_matches "forst.run"
assert_matches "./forst.run"
assert_matches "/path/forst dev -root /tmp/app -entry main.ft"
assert_matches "forst_invoke_server.gen.go"

assert_no_match "python3 -m http.server 6321"
assert_no_match "node server.js"
assert_no_match "curl http://127.0.0.1:6321/health"

TEST_PORT="$(python3 - <<'PY'
import socket
s = socket.socket()
s.bind(("127.0.0.1", 0))
print(s.getsockname()[1])
s.close()
PY
)"

python3 -m http.server "$TEST_PORT" --bind 127.0.0.1 >/dev/null 2>&1 &
LISTENER_PID=$!
cleanup_listener() {
  kill -KILL "$LISTENER_PID" 2>/dev/null || true
  wait "$LISTENER_PID" 2>/dev/null || true
}
trap cleanup_listener EXIT

sleep 0.2
if ! kill -0 "$LISTENER_PID" 2>/dev/null; then
  echo "failed to start test listener on port $TEST_PORT" >&2
  exit 1
fi

bash "$SCRIPT_DIR/kill-forst-tcp-listeners.sh" "$TEST_PORT" --force
sleep 0.2

if kill -0 "$LISTENER_PID" 2>/dev/null; then
  echo "--force did not kill remaining listener pid=$LISTENER_PID on port $TEST_PORT" >&2
  exit 1
fi

trap - EXIT
echo "kill-forst-tcp-listeners tests ok"
