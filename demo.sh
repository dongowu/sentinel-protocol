#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"

echo "== Sentinel Protocol Demo =="
echo "[1/4] Go tests"
( cd "$ROOT_DIR/goserver" && go test ./... -v )

echo "[2/4] Move build"
( cd "$ROOT_DIR/contract" && sui move build >/dev/null )

echo "[3/4] Rust tests"
( cd "$ROOT_DIR/rustcli" && cargo test --all >/dev/null )

echo "[4/4] Sentinel benchmark (sample)"
if [ -f "$ROOT_DIR/goserver/benchmark_cases.example.json" ]; then
  ( cd "$ROOT_DIR/goserver" && go run . --config config.openclaw.example.json --sentinel-benchmark benchmark_cases.example.json )
else
  echo "benchmark_cases.example.json not found, skip"
fi

echo "Demo complete."
