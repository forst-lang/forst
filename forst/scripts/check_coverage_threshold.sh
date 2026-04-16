#!/usr/bin/env bash
# Enforce minimum merged statement coverage from an existing cover profile.
# Usage (from forst/): bash scripts/check_coverage_threshold.sh profile.cov 80
set -euo pipefail
PROFILE="${1:?cover profile path}"
MIN="${2:?minimum percent}"
cd "$(dirname "$0")/.."
total_line=$(go tool cover -func="$PROFILE" | tail -1)
echo "$total_line"
pct=$(echo "$total_line" | grep -oE '[0-9]+\.[0-9]+%' | head -1 | tr -d '%')
awk -v p="$pct" -v m="$MIN" 'BEGIN { if (p + 0 < m + 0) { print "coverage " p "% is below minimum " m "%" > "/dev/stderr"; exit 1 }; exit 0 }'
