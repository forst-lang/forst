#!/usr/bin/env bash
# wait-for-file.sh <path> [label] [timeout_seconds]
# Exits 0 when path exists, 1 after timeout.
set -euo pipefail

PATH_TO_WAIT=${1:?usage: wait-for-file.sh PATH [label] [timeout_seconds]}
LABEL=${2:-$PATH_TO_WAIT}
TIMEOUT=${3:-120}

for i in $(seq 1 "$TIMEOUT"); do
  if [[ -f "$PATH_TO_WAIT" ]]; then
    echo "ready: $LABEL (${i}s)"
    exit 0
  fi
  sleep 1
done

echo "timeout: $LABEL not ready after ${TIMEOUT}s" >&2
exit 1
