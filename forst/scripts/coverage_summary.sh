#!/usr/bin/env bash
# Aggregate statement coverage for the same packages as CI (Taskfile: ci:test).
# Usage (from repo root): bash forst/scripts/coverage_summary.sh
# Optional: MIN_COVERAGE=80 exits 1 if total is below the threshold.
set -euo pipefail
cd "$(dirname "$0")/.."
go test -coverprofile=profile.cov ./cmd/forst/... ./internal/... -count=1 >/dev/null
total_line=$(go tool cover -func=profile.cov | tail -1)
echo "$total_line"
pct=$(echo "$total_line" | grep -oE '[0-9]+\.[0-9]+%' | head -1 | tr -d '%')
min="${MIN_COVERAGE:-}"
if [[ -n "$min" ]]; then
	echo "$pct $min" | awk '{ exit !($1 >= $2) }'
fi
