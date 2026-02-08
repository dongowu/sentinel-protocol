# Sentinel Protocol (OpenClaw x Sui)

Verifiable security layer for local autonomous agents.

This repository started as Lazarus Protocol and now includes Sentinel, a policy gate + audit anchoring workflow for OpenClaw and Sui.

## What This Project Does

Sentinel adds a security control loop to agent execution:

1. Detect risky prompts/actions (prompt injection, dangerous exec, wallet risk)
2. Block or allow based on policy
3. Record structured audit logs locally
4. Anchor audit evidence on Sui for public verification

Core loop: **Detect -> Block -> Record -> Verify**

## Architecture

- `goserver/`: policy gate, benchmark runner, OpenClaw integration, Sui anchoring calls
- `contract/`: Move modules (`lazarus_protocol.move`, `sentinel_audit.move`)
- `rustcli/`: deterministic audit hashing/signing + legacy encryption/Walrus tooling

## Quick Start

### 1) Build tools

```bash
cd rustcli
cargo build --release
cd ../goserver
go build ./...
```

### 2) Build and publish Move package

```bash
cd ../contract
sui move build
sui client publish --gas-budget 100000000
```

Save these from publish output:

- `PackageID`
- `RegistryID` (object type `sentinel_audit::Registry`)

### 3) Configure Sentinel

Edit `goserver/config.openclaw.json`:

- `sentinel.anchor_enabled = true`
- `sentinel.anchor_package = <PackageID>`
- `sentinel.anchor_registry = <RegistryID>`

Then set operator:

```bash
sui client call \
  --package <PackageID> \
  --module sentinel_audit \
  --function set_operator \
  --args <RegistryID> <YOUR_ADDRESS> \
  --gas-budget 10000000
```

### 4) Run benchmark

```bash
cd ../goserver
go run . --config config.openclaw.json --sentinel-benchmark benchmark_cases.example.json
```

## Verification Evidence (Hackathon)

### A) Benchmark Result

Current sample set result:

- `total=4`
- `correct=4`
- `accuracy=1.0`
- `true_positive=2`
- `false_positive=0`
- `true_negative=2`
- `false_negative=0`

> Add screenshot here:

```markdown
![Benchmark result](docs/images/benchmark-result.png)
```

### B) On-chain Proof

Reference values used in demo:

- `PackageID`: `0x350839c3f676f8f0f1c8e2a97401e9cfe63425599daa31d2f060e9461d309c2b`
- `RegistryID`: `0x8a8aebd897753bfa384ff3341bec31d4051f95e0e870bdf2c06a4196fa08df60`
- `record_audit tx digest`: `GXc1bZNEsd6rNR8wjum1MpwACX3VQMXVeNTvPKTckVvV`

Query example:

```bash
sui client tx-block GXc1bZNEsd6rNR8wjum1MpwACX3VQMXVeNTvPKTckVvV --json
```

Expected evidence: `sentinel_audit::AuditAnchoredEvent` with `action_tag`, `risk_score`, `blocked`.

> Add screenshots here:

```markdown
![Tx digest proof](docs/images/tx-digest-proof.png)
![AuditAnchoredEvent proof](docs/images/audit-event-proof.png)
```

## Demo Video

Recommended structure (90-140s):

1. Threat input
2. Before (no gate) vs After (Sentinel)
3. Benchmark metrics
4. On-chain event proof
5. Final takeaway

Suggested video file:

- `sentinel-demo-single-paused-novoice-*.mp4` (raw)
- edited version with overlays in CapCut

## Security Notes

- Do not use exposed seed phrases/keys from chats in real wallets.
- Keep `sign_private_key` empty unless you explicitly need signing.
- This is a hackathon prototype; review before production use.

## License

MIT
