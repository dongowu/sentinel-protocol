# Final Evidence Package

Generated on: 2026-02-24

This folder is the curated, submission-ready evidence bundle for DeepSurge.

## Included artifacts

- `go-test.log` — Go server test suite pass log
- `rust-test.log` — Rust CLI test suite pass log
- `move-test.log` — Move contract test suite pass log
- `benchmark-report.json` — benchmark metrics (machine-readable)
- `benchmark-trace.log` — per-case benchmark trace
- `openclaw-cli/` — OpenClaw-driven gate/status/proof evidence bundle
- `openclaw-cli/DEEPSURGE_UPLOAD_INDEX.md` — direct upload index for judge review
- `SCREENSHOT_CHECKLIST.md` — required screenshots/clips to capture
- `SHA256SUMS.txt` — file integrity checksums

## Benchmark highlight

- Total: 18
- Correct: 18
- Accuracy: 1.0
- Precision: 1.0
- Recall: 1.0
- F1: 1.0
- Block rate: 0.5555555555555556

## How to use for submission

1. Upload these logs/JSON files as technical evidence.
2. Capture media listed in `SCREENSHOT_CHECKLIST.md` and put them in `media/`.
3. Add on-chain anchor tx digest in the submission form.

## Regenerate OpenClaw evidence

```bash
./scripts/run_openclaw_evidence.sh
```
