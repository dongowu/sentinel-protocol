#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
OUT_DIR="${1:-$ROOT_DIR/docs/evidence}"
TS="$(date +%Y%m%d-%H%M%S)"
MOVE_HOME_DIR="${MOVE_HOME:-/tmp/move-home-lazarus}"

GO_LOG="$OUT_DIR/verify-go-$TS.log"
RUST_LOG="$OUT_DIR/verify-rust-$TS.log"
MOVE_LOG="$OUT_DIR/verify-move-$TS.log"
SUMMARY_MD="$OUT_DIR/judge-evidence-$TS.md"

mkdir -p "$OUT_DIR"

echo "Building judge evidence bundle..."
echo "  output directory: $OUT_DIR"

echo
echo "[1/4] Go tests"
(
  cd "$ROOT_DIR/goserver"
  env -u GOROOT GOCACHE=/tmp/go-build-cache-lazarus go test -count=1 ./...
) | tee "$GO_LOG"

echo
echo "[2/4] Rust tests"
(
  cd "$ROOT_DIR/rustcli"
  cargo test -q
) | tee "$RUST_LOG"

echo
echo "[3/4] Move tests"
if [ ! -d "$MOVE_HOME_DIR" ]; then
  mkdir -p "$MOVE_HOME_DIR"
  if [ -d "$HOME/.move" ]; then
    cp -R "$HOME/.move/." "$MOVE_HOME_DIR/" || true
  fi
fi
(
  cd "$ROOT_DIR/contract"
  env MOVE_HOME="$MOVE_HOME_DIR" sui move test
) | tee "$MOVE_LOG"

echo
echo "[4/4] Sentinel benchmark"
"$ROOT_DIR/scripts/run_hackathon_benchmark.sh" "$ROOT_DIR/goserver/testdata/benchmark_cases.hackathon.json" "$OUT_DIR"

LATEST_BENCH_JSON="$(ls -t "$OUT_DIR"/sentinel-benchmark-*.json | head -n 1)"
LATEST_BENCH_LOG="$(ls -t "$OUT_DIR"/sentinel-benchmark-*.log | head -n 1)"

cat >"$SUMMARY_MD" <<EOF
# Judge Evidence Bundle

- Generated at: $(date -u +"%Y-%m-%dT%H:%M:%SZ")
- Go tests log: $(basename "$GO_LOG")
- Rust tests log: $(basename "$RUST_LOG")
- Move tests log: $(basename "$MOVE_LOG")
- Benchmark report JSON: $(basename "$LATEST_BENCH_JSON")
- Benchmark trace log: $(basename "$LATEST_BENCH_LOG")

Use these files as DeepSurge submission evidence.
EOF

echo
echo "Done."
echo "Generated files:"
echo "  - $GO_LOG"
echo "  - $RUST_LOG"
echo "  - $MOVE_LOG"
echo "  - $LATEST_BENCH_JSON"
echo "  - $LATEST_BENCH_LOG"
echo "  - $SUMMARY_MD"
