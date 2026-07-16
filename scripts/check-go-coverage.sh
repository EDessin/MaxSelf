#!/usr/bin/env bash
set -euo pipefail

threshold="${1:-90}"
profile="${2:-coverage.out}"

go test ./internal/... -coverprofile="$profile"

coverage="$(
  go tool cover -func="$profile" |
    awk '/^total:/ { gsub("%", "", $3); print $3 }'
)"

go tool cover -func="$profile"

awk -v coverage="$coverage" -v threshold="$threshold" 'BEGIN {
  if (coverage + 0 < threshold + 0) {
    printf "backend coverage %.1f%% is below %.1f%%\n", coverage, threshold
    exit 1
  }
  printf "backend coverage %.1f%% meets %.1f%%\n", coverage, threshold
}'
