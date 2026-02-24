# Roadmap (Win Hackathon + Production Path)

## 现在（可展示）

- In-path gate + 执行层双保险
- 审批挑战 + 一次性 token
- 能力沙箱 + Kill Switch
- Proof Chain + latest proof 查询
- Sui audit anchor（record_audit）

## 48 小时内建议

1. 接入真实 LLM provider 到语义风险引擎
2. 增加攻击样本集并固化 benchmark
3. 增加 demo 一键脚本（启动 + 场景回放 + 证据导出）

## 1-2 周建议

1. Move 侧补全 Kill Switch 状态模块
2. Walrus 存储从 stub 升级为真实持久化路径
3. 增加多 agent 隔离策略与配额策略

## 当前风险

- 语义风险引擎仍以扩展点为主，生产可用性取决于模型接入质量
- 链上熔断状态与控制面联动尚未完全闭环
- 演示脚本自动化程度还可以再提升
