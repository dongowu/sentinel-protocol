# Sentinel Protocol - Sui Move Package

Sui Move package for Sentinel Protocol (OpenClaw x Sui), focused on **on-chain security audit anchoring** for autonomous-agent runtime decisions.

## Hackathon Focus (Track 1: Safety & Security)

This package provides the on-chain components used by Sentinel to make security decisions auditable:

- `sentinel_audit.move`: canonical audit registry + operator policy + `record_audit`
- `sentinel_audit_integration.move`: enhanced record model for rule/anomaly dimensions
- `community_rules.move`: governance-style community rule submission/voting

A legacy module `lazarus_protocol.move` is still included for backward compatibility but is **not** the primary hackathon path.

## Modules

### 1) `sentinel_audit` (primary)

**Purpose:** Anchor runtime policy decisions on-chain.

**Core objects/events:**
- `Registry` (shared): admin, operator, policy version/hash
- `PolicyUpdatedEvent`
- `OperatorUpdatedEvent`
- `AuditAnchoredEvent`

**Core functions:**
- `update_policy(registry, version, hash, ctx)`
- `set_operator(registry, operator, ctx)`
- `record_audit(registry, record_hash, action_tag, risk_score, blocked, clock, ctx)`

### 2) `sentinel_audit_integration`

**Purpose:** Keep richer dimensions for analytics/testing (rule id, anomaly metadata, block rate).

### 3) `community_rules`

**Purpose:** Community-managed rule lifecycle and voting logic for future policy governance.

### 4) `lazarus_protocol` (legacy)

Legacy dead-man-switch module retained for compatibility with earlier project stage.

## Build & Test

```bash
# from contract/
sui move build
sui move test
```

## Deployment

### Testnet

```bash
sui client publish --gas-budget 100000000
```

Save the package ID and registry object ID from publish output.

## Setup Flow for Sentinel Audit

1. Publish package (creates shared `Registry` in `init`)
2. Grant service wallet as operator

```bash
sui client call \
  --package $PACKAGE_ID \
  --module sentinel_audit \
  --function set_operator \
  --args $REGISTRY_ID $SERVICE_ADDRESS true \
  --gas-budget 10000000
```

3. (Optional) update policy version/hash

```bash
sui client call \
  --package $PACKAGE_ID \
  --module sentinel_audit \
  --function update_policy \
  --args $REGISTRY_ID 2 $POLICY_HASH \
  --gas-budget 10000000
```

4. Anchor one audit decision

```bash
sui client call \
  --package $PACKAGE_ID \
  --module sentinel_audit \
  --function record_audit \
  --args $REGISTRY_ID $RECORD_HASH 1 92 true 0x6 \
  --gas-budget 10000000
```

## Verification Checklist

- `sui move test` passes
- `record_audit` tx succeeds on testnet
- Anchored event can be queried from tx digest
- Off-chain `record_hash` equals local recomputed hash from audit log

## Security Notes

- Only admin can change policy metadata and operator permissions
- `record_audit` is restricted to admin/operator
- Chain events provide immutable audit trail for incident review

## Related Docs

- `../README.md` - full Sentinel architecture and runtime controls
- `../docs/DEMO_RUNBOOK.md` - end-to-end demo script
- `../docs/SECURITY_WORKFLOWS.md` - air-gap + evidence verification scripts
