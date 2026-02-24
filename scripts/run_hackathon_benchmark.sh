#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
GO_DIR="$ROOT_DIR/goserver"
CASES_PATH="${1:-$GO_DIR/testdata/benchmark_cases.hackathon.json}"
OUT_DIR="${2:-$ROOT_DIR/docs/evidence}"

if [[ "$CASES_PATH" != /* ]]; then
  CASES_PATH="$ROOT_DIR/$CASES_PATH"
fi

if [ ! -f "$CASES_PATH" ]; then
  echo "benchmark cases not found: $CASES_PATH" >&2
  exit 1
fi

mkdir -p "$OUT_DIR"
TS="$(date +%Y%m%d-%H%M%S)"
REPORT_JSON="$OUT_DIR/sentinel-benchmark-$TS.json"
REPORT_LOG="$OUT_DIR/sentinel-benchmark-$TS.log"

echo "Running Sentinel benchmark..."
echo "  cases:  $CASES_PATH"
echo "  report: $REPORT_JSON"
echo "  log:    $REPORT_LOG"

(
  cd "$GO_DIR"
  env -u GOROOT GOCACHE=/tmp/go-build-cache-lazarus \
    go run . --config configs/config.openclaw.json \
      --sentinel-benchmark "$CASES_PATH" \
      --sentinel-benchmark-out "$REPORT_JSON"
) | tee "$REPORT_LOG"

echo "Done. Attach these files in submission evidence:"
echo "  - $REPORT_JSON"
echo "  - $REPORT_LOG"
