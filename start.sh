#!/usr/bin/env bash
# Local convenience launcher: API on :8080 + optional notes for the web console.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"

if ! command -v go >/dev/null 2>&1; then
  echo "Go is required. Install Go 1.26+ and retry." >&2
  exit 1
fi

if [[ ! -f "${STRAND_CONFIG:-$HOME/.strand/config.toml}" ]] && [[ ! -f "$HOME/.strand/config.toml" ]]; then
  echo "No Strand config found — initializing ~/.strand …"
  go run ./cmd/strand config init
fi

echo "Starting Strand API (Ctrl+C to stop)."
echo "  API:  http://127.0.0.1:8080"
echo "  Web:  cd web && pnpm install && pnpm run dev   → http://127.0.0.1:3000"
echo

exec go run ./cmd/strand start
