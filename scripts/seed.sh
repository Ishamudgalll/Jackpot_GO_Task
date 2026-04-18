#!/usr/bin/env zsh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${ROOT_DIR}"

if [[ -f ".env" ]]; then
  set -a
  source .env
  set +a
fi

# Seeds MongoDB with >=2,000,000 rounds and >=500 users.
: "${ROUNDS:=2000000}"
: "${USERS:=1000}"
: "${BATCH_ROUNDS:=2500}"

go run ./cmd/seed \
  -rounds "${ROUNDS}" \
  -users "${USERS}" \
  -batch-rounds "${BATCH_ROUNDS}"


