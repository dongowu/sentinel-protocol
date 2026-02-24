# Sentinel Benchmark Guide

This benchmark demonstrates Sentinel detection quality on a curated red-team and benign mix for hackathon judging.

## Cases

- File: `goserver/testdata/benchmark_cases.hackathon.json`
- Total cases: 18
- Mix: prompt injection, wallet abuse, privilege/system attacks, exfiltration, benign developer tasks

## Run (One Command)

```bash
./scripts/run_hackathon_benchmark.sh
```

Optional custom inputs:

```bash
./scripts/run_hackathon_benchmark.sh \
  goserver/testdata/benchmark_cases.hackathon.json \
  docs/evidence
```

To generate benchmark artifacts together with all test logs:

```bash
./scripts/build_judge_evidence.sh
```

## Output Artifacts

The script emits two files:

- `sentinel-benchmark-<timestamp>.json` (machine-readable metrics)
- `sentinel-benchmark-<timestamp>.log` (per-case prediction trace)

## Metrics Reported

- Accuracy
- Precision
- Recall
- F1
- Confusion matrix (TP/FP/TN/FN)
- Block rate

## Submission Usage

Attach the JSON and log files to DeepSurge evidence. These files support technical-merit claims with reproducible scoring output.
