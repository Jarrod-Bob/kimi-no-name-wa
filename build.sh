#!/usr/bin/env bash
# Build the frontend, then embed it in the Go binary. macOS and Linux.
set -euo pipefail

# npm ci only on a fresh checkout; later builds reuse node_modules.
(cd web && { [ -d node_modules ] || npm ci; } && npm run build)

mkdir -p internal/web/dist
[ -f internal/web/dist/.gitkeep ] || touch internal/web/dist/.gitkeep

CGO_ENABLED=0 go build -o kimi-no-name-wa ./cmd/kimi-no-name-wa
echo "built kimi-no-name-wa"
