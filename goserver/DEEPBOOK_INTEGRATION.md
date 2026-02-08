# Sentinel Protocol - DeepBook "Panic Sell" Integration

## 概述

**"紧急变现"功能**：在遗嘱执行时，自动将波动性代币（如 meme coin）通过 DeepBook 市价卖成 USDC，然后转给受益人，防止资产因币价暴跌而缩水。

## 痛点

传统的遗嘱执行只是简单地转移资产所有权，但如果持有的是高波动性代币：
- 📉 在受益人获得资产前，币价可能已经暴跌
- 💸 Meme coin 可能在几天内归零
- ⏰ 受益人可能不懂加密货币，无法及时变现

## 解决方案

使用 **Programmable Transaction Block (PTB)** 在单个交易中完成：

```
1. 执行遗嘱 (execute_will)
   ↓
2. 在 DeepBook 上市价卖出代币
   ↓
3. 将 USDC 转给受益人
```

## 技术架构

### PTB 流程

```
┌─────────────────────────────────────────────────────────┐
│           Programmable Transaction Block                 │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  Command 0: MoveCall                                    │
│  ├─ Package: lazarus_protocol                           │
│  ├─ Function: execute_will                              │
│  └─ Args: [vault, clock]                                │
│                                                          │
│  Command 1: MoveCall                                    │
│  ├─ Package: 0xdee9 (DeepBook)                         │
│  ├─ Module: clob_v2                                     │
│  ├─ Function: place_market_order                        │
│  ├─ Type Args: [MEME, USDC]                            │
│  └─ Args: [pool, base_coin, min_quote, clock]          │
│                                                          │
│  Command 2: TransferObjects                             │
│  ├─ Objects: [Result(1)]  ← USDC from trade           │
│  └─ Recipient: beneficiary                              │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

### DeepBook 集成

DeepBook 是 Sui 的原生 DEX，提供：
- **中央限价订单簿 (CLOB)**：高效的价格发现
- **市价单**：立即执行，无需等待
- **低滑点**：深度流动性池
- **原子性**：PTB 保证要么全部成功，要么全部失败

## 配置

### config.deepbook.json

```json
{
  "deepbook": {
    "enabled": true,
    "pool_id": "0xDEEPBOOK_POOL_ID",
    "base_asset": "0xMEME_COIN_TYPE",
    "quote_asset": "0xUSDC_TYPE",
    "slippage_bps": 100,
    "min_output_amount": "1000000",
    "asset_balances": {
      "0xMEME_COIN_TYPE": "0xYOUR_MEME_COIN_OBJECT_ID"
    }
  }
}
```

### 参数说明

| 参数 | 类型 | 说明 |
|------|------|------|
| `enabled` | bool | 是否启用紧急变现 |
| `pool_id` | string | DeepBook 交易池 ID |
| `base_asset` | string | 要卖出的代币类型（如 MEME） |
| `quote_asset` | string | 要买入的代币类型（如 USDC） |
| `slippage_bps` | int | 滑点容忍度（100 = 1%） |
| `min_output_amount` | string | 最小接收 USDC 数量 |
| `asset_balances` | map | 资产对象 ID 映射 |

## 使用方法

### 1. 模拟执行（测试）

```bash
cd goserver
go run . --simulate-panic-sell \
  --vault 0xVAULT_ID \
  --beneficiary 0xBENEFICIARY \
  --config config.deepbook.json
```

输出：
```
🧪 SIMULATING PANIC SELL 🧪
============================================================

📊 Simulation Results:
   Input: 1000000 MEME
   Market Price: 0.05 USDC
   Slippage: 100 bps (1.00%)
   Min Output: 1000000 USDC

   Transaction Flow:
   1. Execute will on vault
   2. Place market sell order on DeepBook
   3. Receive USDC
   4. Transfer USDC to beneficiary

============================================================
✓ Simulation complete (no actual transaction)
============================================================
```

### 2. 实际执行

```bash
go run . --execute-panic-sell \
  --vault 0xVAULT_ID \
  --beneficiary 0xBENEFICIARY \
  --package 0xPACKAGE_ID \
  --config config.deepbook.json
```

输出：
```
🚨 EXECUTING PANIC SELL 🚨
============================================================

[1/4] Checking asset balance...
   Balance: 1000000 MEME

[2/4] Fetching market price from DeepBook...
   Current price: 0.05 USDC per token

[3/4] Building Programmable Transaction Block...
   ✓ PTB executed successfully

[4/4] Verifying transaction...
   Transaction digest: 0xABC123...

============================================================
✓ PANIC SELL COMPLETED SUCCESSFULLY!
  Assets liquidated and USDC transferred to beneficiary
============================================================
```

## 代码实现

### Go 实现 (PTB 构建)

文件：`goserver/deepbook_integration.go`

关键函数：
- `BuildPanicSellPTB()`: 构建 PTB JSON
- `ExecutePanicSell()`: 执行紧急变现
- `SimulatePanicSell()`: 模拟执行（测试用）

### Move 合约（可选增强）

如果想在合约层面支持，可以修改 `execute_will`：

```move
public entry fun execute_will_with_liquidation<BaseAsset, QuoteAsset>(
    vault: &mut Vault,
    pool: &mut Pool<BaseAsset, QuoteAsset>,
    base_coin: Coin<BaseAsset>,
    min_quote: u64,
    clock: &Clock,
    ctx: &mut TxContext
) {
    // 1. Check threshold
    let current_time = clock::timestamp_ms(clock);
    let time_since_heartbeat = current_time - vault.last_heartbeat_ms;
    assert!(time_since_heartbeat > HEARTBEAT_THRESHOLD_MS, EThresholdNotExceeded);

    // 2. Mark as executed
    vault.is_executed = true;

    // 3. Liquidate on DeepBook
    let quote_coin = clob_v2::place_market_order(
        pool,
        base_coin,
        min_quote,
        clock,
        ctx
    );

    // 4. Transfer USDC to beneficiary
    transfer::public_transfer(quote_coin, vault.beneficiary);

    // 5. Emit event
    event::emit(WillExecutedWithLiquidation {
        vault_id: object::uid_to_address(&vault.id),
        beneficiary: vault.beneficiary,
        base_amount: coin::value(&base_coin),
        quote_amount: coin::value(&quote_coin),
    });
}
```

## 演示场景

### 场景 1：Meme Coin 暴跌

```
时间线：
T0:  用户持有 1,000,000 DOGE (价值 $50,000)
T+30d: 用户 30 天无活动，触发警报
T+72h: 用户仍无响应，进入紧急模式
T+73h: 受益人触发 execute_will
      ├─ DOGE 价格已跌至 $0.01 (价值 $10,000)
      ├─ 🚨 触发 Panic Sell
      ├─ 在 DeepBook 上市价卖出 DOGE
      ├─ 获得 9,900 USDC (扣除 1% 滑点)
      └─ 转给受益人

结果：受益人获得 $9,900 USDC，而不是价值 $10,000 的 DOGE
```

### 场景 2：正常执行（无 Panic Sell）

```
如果 deepbook.enabled = false：
T+73h: 受益人触发 execute_will
      ├─ 仅执行遗嘱
      ├─ Vault 标记为已执行
      └─ 受益人可以手动处理资产
```

## 技术亮点

### 1. PTB 的优势

- **原子性**：要么全部成功，要么全部失败
- **Gas 效率**：单个交易完成多个操作
- **灵活性**：可以动态组合任意操作
- **安全性**：无需预先授权，临时组合

### 2. DeepBook 集成

- **原生 DEX**：Sui 官方支持
- **深度流动性**：主流交易对流动性充足
- **低滑点**：CLOB 机制保证价格稳定
- **即时执行**：市价单立即成交

### 3. 风险控制

- **最小输出量**：`min_output_amount` 防止过度滑点
- **滑点保护**：`slippage_bps` 限制最大滑点
- **模拟执行**：测试前先模拟
- **事件日志**：完整的链上审计追踪

## 与其他 DeFi 协议对比

| 协议 | 优势 | 劣势 |
|------|------|------|
| **DeepBook** | 原生支持、低滑点、CLOB | 需要流动性 |
| Cetus | AMM 简单 | 滑点较大 |
| Turbos | 集中流动性 | 复杂度高 |
| Aftermath | 稳定币优化 | 仅限稳定币对 |

## Hackathon 加分项

这个功能展示了：

1. **DeFi 集成能力**：深度理解 DeepBook
2. **PTB 应用**：Sui 核心特性的实际应用
3. **实用性**：解决真实痛点
4. **创新性**：将遗嘱执行与 DeFi 结合
5. **完整性**：从问题到解决方案的闭环

## 未来增强

- [ ] 支持多种交易对
- [ ] 智能路由（自动选择最优 DEX）
- [ ] 限价单支持（设置最低价格）
- [ ] 分批卖出（减少市场冲击）
- [ ] 价格预言机集成（防止价格操纵）
- [ ] 多资产组合清算

## 参考资料

- [DeepBook 文档](https://docs.sui.io/standards/deepbook)
- [PTB 指南](https://docs.sui.io/concepts/transactions/prog-txn-blocks)
- [Sui Move 示例](https://github.com/MystenLabs/sui/tree/main/examples)

---

**这个功能是 Track 2 (DeFi) 的大杀器！** 🚀
