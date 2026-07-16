#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/env.sh"

docker rm -f "${DB_CONTAINER}" >/dev/null 2>&1 || true
echo "Stopped local Postgres container ${DB_CONTAINER}."
