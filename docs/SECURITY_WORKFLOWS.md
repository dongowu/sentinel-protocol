# Security Workflows

## 1) Wallet Air-Gap（仅提案，不代签）

Sentinel 只生成提案，不直接代替用户签名转账。

```bash
./scripts/propose_wallet_tx.sh <package> <module> <function> <gas_budget> <arg1> [arg2 ...]
```

输出：`./proposals/proposal_*.json`，包含 `unsigned_tx_bytes`。

## 2) 一键审计证据验证（链上 + 本地）

```bash
./scripts/verify_audit_evidence.sh <tx_digest> <audit_log_path>
```

验证内容：

1. 链上交易成功
2. 本地审计记录存在
3. 本地重算哈希与记录值一致

全部通过返回 `PASS`。

## 3) 审批与熔断策略（演示重点）

- 高风险动作进入 `REQUIRE_APPROVAL`
- 人工拒绝后动作阻断
- 连续高风险可触发 `TRIGGER_KILL_SWITCH`
