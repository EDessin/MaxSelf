#!/usr/bin/env bash
set -euo pipefail

tmp_dir="$(mktemp -d)"
cleanup() {
  rm -rf "${tmp_dir}"
}
trap cleanup EXIT

env_file="${tmp_dir}/maxself.env"
cat >"${env_file}" <<'ENV'
GOOGLE_CLIENT_ID=test-client-id
GOOGLE_CLIENT_SECRET=test-client-secret
GOOGLE_REDIRECT_URL=http://localhost:8081/custom/google/callback
FRONTEND_URL=http://localhost:4201
ENV

output="$(
  env -i PATH="${PATH}" MAXSELF_ENV_FILE="${env_file}" bash -c '
    source scripts/env.sh
    printf "%s\n%s\n%s\n%s\n" \
      "${GOOGLE_CLIENT_ID}" \
      "${GOOGLE_CLIENT_SECRET}" \
      "${GOOGLE_REDIRECT_URL}" \
      "${DATABASE_URL}"
  '
)"

expected="$(
  printf "%s\n%s\n%s\n%s\n" \
    "test-client-id" \
    "test-client-secret" \
    "http://localhost:8081/custom/google/callback" \
    "postgres://maxself:maxself@localhost:5432/maxself?sslmode=disable"
)"

if [[ "${output}" != "${expected}" ]]; then
  printf "env loader output did not match expected values\n" >&2
  printf "expected:\n%s\n" "${expected}" >&2
  printf "actual:\n%s\n" "${output}" >&2
  exit 1
fi

echo "env loader reads .env values and keeps defaults for missing values"
