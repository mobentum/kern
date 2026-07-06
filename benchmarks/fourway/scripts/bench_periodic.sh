#!/usr/bin/env bash
set -euo pipefail

# Usage:
#   ./scripts/bench_periodic.sh [interval_seconds] [out_dir] [max_runs]
# Examples:
#   ./scripts/bench_periodic.sh
#   ./scripts/bench_periodic.sh 300 ./results
#   ./scripts/bench_periodic.sh 120 ./results 10

INTERVAL_SECONDS="${1:-300}"
OUT_DIR="${2:-./results}"
MAX_RUNS="${3:-0}" # 0 means run forever

mkdir -p "$OUT_DIR"

run=0
while true; do
  run=$((run + 1))
  timestamp="$(date +%Y%m%d-%H%M%S)"
  out_file="$OUT_DIR/bench-${timestamp}.txt"

  echo "[$(date '+%Y-%m-%d %H:%M:%S')] Run #${run} -> ${out_file}"
  go test -run '^$' -bench '^BenchmarkFramework(QueryAccess|Plaintext|PlaintextMiddleware|DecodeJSON|PathParams)$' -benchmem | tee "$out_file"

  if [[ "$MAX_RUNS" -gt 0 && "$run" -ge "$MAX_RUNS" ]]; then
    echo "Completed ${run} periodic benchmark runs."
    exit 0
  fi

  echo "Sleeping ${INTERVAL_SECONDS}s before next run..."
  sleep "$INTERVAL_SECONDS"
done
