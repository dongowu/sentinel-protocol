# Sentinel Protocol (OpenClaw x Sui)

Sentinel secures local autonomous agents with policy-gated execution in OpenClaw and anchors security decisions on Sui, turning safety claims into verifiable on-chain evidence.

## Why This Exists

Autonomous agents now have real local execution power (shell, browser, files). Sentinel adds an immune system before execution:

1. Detect risky intent (prompt injection, wallet exfiltration patterns, dangerous actions)
2. Enforce policy (`ALLOW` / `REQUIRE_APPROVAL` / `BLOCK`)
3. Produce tamper-evident local audit records
4. Anchor critical decisions on Sui for public verification

Core loop: **Detect -> Enforce -> Record -> Anchor -> Verify**

## Current Scope

- **Go runtime security layer** (`goserver/`)
  - Behavioral detection + policy gate
  - Sentinel guard decision engine
  - Local JSONL audit trail
  - Optional on-chain anchoring (`sentinel_audit::record_audit`)
- **Sui Move contracts** (`contract/`)
  - `lazarus_protocol` (dead-man switch primitives)
  - `sentinel_audit` (audit anchoring registry/events)
  - `community_rules` (proposal/voting/governance)
  - `sentinel_audit_integration` (enhanced integration layer)
- **Rust cryptography CLI** (`rustcli/`)
  - deterministic audit hashing/signing
  - encrypt/upload + decrypt flow for Walrus-backed payloads
- **Security workflows** (`scripts/`, `docs/`)
  - Wallet Air-Gap proposal flow (unsigned transaction only)
  - One-click audit evidence verification

## Repository Layout

- `goserver/` - policy gate, behavioral detection, audit logic, daemon, benchmark
- `contract/` - Move sources + published metadata
- `rustcli/` - cryptographic/hash/signing utilities + storage tooling
- `scripts/` - security workflow scripts
- `docs/` - workflow and supporting docs

## Testnet Deployment (Latest)

- **Network:** Sui testnet
- **Package ID:** `0x9ab7b272a0e6c959835ff29e3fdf050dc4c432f6794b8aa54533fefcad985eca`
- **Upgrade Capability:** `0xcc00919bc1bc4b6c3001c2e6f3ee3a19e87b027a64ee834e20960c54ccf97e98`
- **Published tx digest:** `7Lsds61c4pVYWAwtJGuv9NBCVST328ZEymE2m52ZhzqT`
- **Audit Registry:** `0xde4a42164d2ea5bfcdecdf8d3bc67b3fd5487dda8c67a26e09227a49d699641d`

> Full published metadata is tracked in `contract/Published.toml`.

## Quick Start

### 1) Build

```bash
cd rustcli
cargo build --release

cd ../goserver
go build ./...
```

### 2) Configure Go runtime

Use `goserver/config.openclaw.json` or `goserver/config.openclaw.example.json` and ensure:

- `sentinel.enabled = true`
- `sentinel.anchor_enabled = true`
- `sentinel.anchor_package = 0x9ab7...85eca`
- `sentinel.anchor_registry = 0xde4a...9641d`
- `sentinel.hash_cli_path` points to built `lazarus-vault`

### 3) Run benchmark

```bash
cd goserver
go run . --config config.openclaw.json --sentinel-benchmark benchmark_cases.example.json
```

Sample result from current benchmark set:

- `total=4`
- `correct=4`
- `accuracy=1.0`
- `false_positive=0`
- `false_negative=0`

### 4) Run enhanced daemon (demo mode)

```bash
cd goserver
go run . --enhanced --use-cli=false --config config.openclaw.json
```

## Track-1 Security Features (Hackathon)

### A) Wallet Air-Gap (proposal-only execution)

Sentinel can **propose** a transaction but cannot directly execute value-moving actions.

```bash
./scripts/propose_wallet_tx.sh <package> <module> <function> <gas_budget> <arg1> [arg2 ...]
```

Output: `./proposals/proposal_*.json` containing `unsigned_tx_bytes`.
Execution must be completed with an external signer/hardware wallet.

### B) One-click On-chain + Local Evidence Verification

```bash
./scripts/verify_audit_evidence.sh <tx_digest> <audit_log_path>
```

This verifies:

1. the on-chain tx is successful
2. a matching local audit entry exists
3. local deterministic hash recomputes to the recorded `record_hash`

Returns `PASS` only when all checks succeed.

## Example On-chain Proof

A successful `sentinel_audit::record_audit` call during integration testing:

- `tx digest`: `D7NUJANKD7x6xLxa3jTJedDiHJRpvQqWCXNQDArendDt`

Query:

```bash
sui client tx-block D7NUJANKD7x6xLxa3jTJedDiHJRpvQqWCXNQDArendDt --json
```

## Documentation

- `INTEGRATION_GUIDE.md`
- `NEXT_STEPS.md`
- `EXECUTIVE_SUMMARY.md`
- `docs/SECURITY_WORKFLOWS.md`

## Security Notes

- Do not put production private keys in repo config files.
- Keep signer material external where possible (air-gap / hardware wallet flow).
- This is a hackathon prototype and should be hardened before production use.

## License

MIT
