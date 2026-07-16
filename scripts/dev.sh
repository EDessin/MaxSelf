#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/env.sh"
bash "${PROJECT_ROOT}/scripts/db-up.sh"

cd "${PROJECT_ROOT}"

pids=()
cleanup() {
  echo
  echo "Stopping local MaxSelf processes..."
  for pid in "${pids[@]}"; do
    kill "${pid}" 2>/dev/null || true
  done
  wait 2>/dev/null || true
}
trap cleanup EXIT INT TERM

echo "Starting MaxSelf Go services locally..."
PORT=8081 go run ./cmd/identity & pids+=("$!")
PORT=8082 go run ./cmd/activity & pids+=("$!")
PORT=8083 go run ./cmd/progress & pids+=("$!")
PORT=8080 go run ./cmd/api & pids+=("$!")

echo "Starting Angular frontend on ${FRONTEND_URL}..."
(cd web && npm start -- --port 4201) & pids+=("$!")

echo
echo "MaxSelf local stack is starting."
echo "Frontend: ${FRONTEND_URL}"
echo "API:      http://localhost:8080"
echo "Press Ctrl+C to stop local processes. Run 'make db-down' to stop Postgres."
echo

while true; do
  sleep 2
  running_jobs="$(jobs -r | wc -l | tr -d ' ')"
  if [[ "${running_jobs}" -lt "${#pids[@]}" ]]; then
    echo "A local process exited. Current jobs:"
    jobs
    exit 1
  fi
done
