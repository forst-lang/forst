#!/usr/bin/env bash
# wait-for-url.sh <url> [label] [timeout_seconds]
# Exits 0 when URL returns HTTP 2xx, 1 after timeout.
set -euo pipefail

URL=$1
LABEL=${2:-$URL}
TIMEOUT=${3:-120}

for i in $(seq 1 "$TIMEOUT"); do
  if curl -sf "$URL" >/dev/null 2>&1; then
    echo "ready: $LABEL (${i}s)"
    exit 0
  fi
  sleep 1
done

echo "timeout: $LABEL not ready after ${TIMEOUT}s" >&2
exit 1
