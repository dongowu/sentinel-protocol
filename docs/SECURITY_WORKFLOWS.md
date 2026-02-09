# Security Workflows

## 1) Wallet Air-Gap (proposal only)

Goal: OpenClaw/Sentinel can propose a transaction, but cannot directly move funds.

```bash
./scripts/propose_wallet_tx.sh \
  <package> <module> <function> <gas_budget> <arg1> [arg2 ...]
```

This writes a proposal JSON containing `unsigned_tx_bytes` into `./proposals/`.
A human/hardware wallet must sign and execute externally:

```bash
sui client execute-signed-tx --tx-bytes <TX_BYTES> --signatures <SIGS>
```

## 2) One-click audit evidence verification

Goal: verify an anchored decision is both on-chain successful and locally consistent.

```bash
./scripts/verify_audit_evidence.sh <tx_digest> <audit_log_path>
```

Checks:
- on-chain tx status is `Success`
- local audit entry exists by `tx_digest`
- local record hash recomputes deterministically from fields

If all pass, output is `PASS`.
