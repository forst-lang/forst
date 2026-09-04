#!/usr/bin/env bash
# Publish cwd package to npm via OIDC trusted publishing.
# Skips when the exact name@version is already on the registry (safe re-runs).
# Extra args are forwarded to `npm publish` (e.g. --verbose).
set -euo pipefail

NAME="$(node -p "require('./package.json').name")"
VER="$(node -p "require('./package.json').version")"

if npm view "${NAME}@${VER}" version >/dev/null 2>&1; then
  echo "${NAME}@${VER} already on npm; skipping publish."
  exit 0
fi

echo "Publishing ${NAME}@${VER} (OIDC / trusted publisher)…"
# Empty NODE_AUTH_TOKEN so setup-node's token auth does not shadow OIDC.
NODE_AUTH_TOKEN="" npm publish --access public --workspaces=false --provenance "$@"
