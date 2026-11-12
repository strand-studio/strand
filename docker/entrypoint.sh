#!/bin/sh
set -eu

DATA_DIR="${STRAND_DATA_DIR:-/data}"
CONFIG="${STRAND_CONFIG:-$DATA_DIR/config.toml}"
ENV_FILE="$DATA_DIR/.env"

mkdir -p "$DATA_DIR"

if [ ! -f "$CONFIG" ]; then
  echo "[strand] no config at $CONFIG — running config init…"
  strand config init --dir "$DATA_DIR" || true
fi

# Prefer explicit env; otherwise load secrets next to config if present.
if [ -f "$ENV_FILE" ]; then
  # shellcheck disable=SC1090
  set -a
  . "$ENV_FILE"
  set +a
fi

export STRAND_CONFIG="${STRAND_CONFIG:-$CONFIG}"

exec strand "$@"
