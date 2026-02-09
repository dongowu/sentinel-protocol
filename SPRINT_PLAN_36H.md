# Sprint Plan (36h)

## Goal

Deliver a demo-ready and test-backed Sentinel Protocol stack with reproducible commands.

## Workstream A - Runtime Gate (Go)

- Behavioral detection quality tests
- Policy gate threshold calibration
- Sentinel enforcement integration checks

## Workstream B - On-chain Rules/Audit (Move)

- Rule lifecycle tests (submit/vote/against/active)
- Audit stats correctness tests
- Lint cleanup where safe

## Workstream C - Cryptography/Storage (Rust)

- Encrypt/decrypt CLI integration test
- Better error messages for Walrus endpoint compatibility
- Optional retries for transient network errors

## Definition of Done

- `go test ./... -v` passes
- `cargo test --all` passes
- `sui move build` passes
- `./demo.sh` runs end-to-end without manual edits
