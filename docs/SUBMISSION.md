# Sui x OpenClaw Agent Hackathon Submission

## 0. Submission Metadata (Copy to DeepSurge)

- **Project Name:** Sentinel Protocol
- **Track:** Track 1 — Safety & Security
- **Repo:** `https://github.com/dongowu/lazarus-protocol`
- **Demo Environment:** macOS + OpenClaw + local Sentinel proxy + Sui testnet
- **Wallet Address (Sui):** `0x79ee84d793ed41f9868a63c7d0f2e62b2752ea0078944db44940b751d27a05a1`
- **Submission Date (PST):** `2026-02-24` _(draft date; update on final submit day before 2026-03-03 11:00 PM PST)_

## 1. One-Liner

Sentinel is a **pre-execution security gate** for autonomous agents that blocks risky actions before execution, enforces human approval for wallet operations, and anchors tamper-evident audit proofs to Sui.

## 1.1 AI-Agent Build Statement (Required by Eligibility)

This project was developed **by and with AI agents during the hackathon window**, with human supervision for scope decisions, environment setup, and final submission packaging.

## 2. Problem & Why Now

Autonomous local agents can access terminal, browser, and wallets with high privilege. Existing approaches mostly detect risk **after** execution. Sentinel shifts security left by enforcing policy **before** execution and emitting verifiable evidence (proof chain + on-chain anchor) for post-incident accountability.

## 3. What We Built

### 3.1 Core Security Pipeline

1. Agent requests action authorization at `/sentinel/gate`
2. Risk engine scores prompt + behavior
3. Policy engine returns one of:
   - `ALLOW` (one-time token issued)
   - `REQUIRE_APPROVAL` (human challenge required)
   - `BLOCK` (hard stop)
   - `TRIGGER_KILL_SWITCH` (emergency halt)
4. Every decision is hashed into local proof chain and can be anchored on Sui

### 3.2 Security Controls

- **One-time execution token** with TTL and replay prevention
- **Human approval workflow** for high-risk operations
- **Capability sandbox** per agent class (shell/fs/browser/wallet/network)
- **Manual + automatic kill switch** for containment
- **Proof chain** (hash chain + Merkle metadata + optional Walrus CID + Sui anchor)

## 4. Sui Stack Integration (Eligibility Mapping)

- Move module: `contract/sources/sentinel_audit.move`
- On-chain anchor function: `sentinel_audit::record_audit`
- Testnet package info is documented in `README.md`
- Audit event model supports querying anchored policy decisions

Known testnet deployment refs in repo:

- Package ID: `0x9ab7b272a0e6c959835ff29e3fdf050dc4c432f6794b8aa54533fefcad985eca`
- Registry ID: `0xde4a42164d2ea5bfcdecdf8d3bc67b3fd5487dda8c67a26e09227a49d699641d`

## 5. OpenClaw Integration

Sentinel integrates with OpenClaw via plugin-style tools:

- `sentinel_gate`
- `sentinel_status`
- `sentinel_approval`

These tools force risk evaluation before sensitive agent actions.

## 6. Demo Proof (Human-Verifiable)

Follow `docs/DEMO_RUNBOOK.md` to reproduce:

1. Low-risk command -> `ALLOW`
2. Prompt injection -> `BLOCK`
3. Wallet transfer -> `REQUIRE_APPROVAL`
4. Kill switch arm/disarm -> global enforcement
5. Proof/status endpoints -> verifiable chain health

## 7. Innovation Highlights

- Security moves from passive monitoring to **active enforcement**
- Wallet flow supports **air-gap style external signing** proposal model
- Evidence is designed for both incident response and public auditability
- Combines runtime policy, behavioral detection, and chain anchoring in one control plane

## 8. Judge Checklist (Fill Before Final Submit)

- [ ] DeepSurge profile is complete and wallet address is correct
- [ ] Demo video link is public
- [ ] Repo README quick-start works on clean machine
- [ ] `go test` + `cargo test` + `sui move test` outputs attached
- [ ] At least one on-chain anchor tx digest attached
- [ ] Submission text explicitly states "built by/with AI agents during hackathon"

## 9. Evidence to Attach in DeepSurge

- Screenshot/clip: BLOCK decision on injection prompt
- Screenshot/clip: REQUIRE_APPROVAL challenge and approval result
- Screenshot/clip: kill switch triggered state
- Screenshot/clip: `/sentinel/proof/latest`
- Tx digest(s): `8xQ7qp1TydrZrGCM3v7VkgYSKLCGU8Rp5Kkec9E3wxWm` _(verified testnet anchor, add more before final submission)_
- Evidence links bundle: _(add Drive/Notion/GitHub release links to screenshots + short clips)_

Quick way to produce one anchor tx digest for submission evidence:

```bash
cd contract
sui client call \
  --package 0x9ab7b272a0e6c959835ff29e3fdf050dc4c432f6794b8aa54533fefcad985eca \
  --module sentinel_audit \
  --function record_audit \
  --args 0xde4a42164d2ea5bfcdecdf8d3bc67b3fd5487dda8c67a26e09227a49d699641d \
         0x1111111111111111111111111111111111111111111111111111111111111111 \
         1 92 true 0x6 \
  --gas-budget 10000000 --json | jq -r '.digest'
```

## 10. Known Limits & Next Steps

- Semantic risk scoring quality depends on model/provider configuration
- Walrus persistence path can be upgraded from stub to production persistence
- Additional Move tests and policy governance automation are planned

## 11. Quick Verification Commands

```bash
# Go server
cd goserver
env -u GOROOT GOCACHE=/tmp/go-build-cache-lazarus go test -count=1 ./...

# Rust CLI
cd ../rustcli
cargo test -q

# Move package
cd ../contract
sui move test
```
