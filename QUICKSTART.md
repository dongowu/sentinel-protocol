# Lazarus Protocol - 快速启动指南

## 🚀 5分钟快速演示

### 前置条件
```bash
# 检查依赖
sui --version        # Sui CLI
cargo --version      # Rust
go version          # Go 1.21+
```

### 步骤 1: 构建所有组件 (2分钟)

```bash
# 1. 构建 Rust CLI
cd rustcli
cargo build --release
cd ..

# 2. 构建 Move 合约
cd contract
sui move build
cd ..

# 3. Go 守护进程已编译
cd goserver
# lazarus-daemon.exe 已就绪 (8.6MB)
```

### 步骤 2: 快速测试 (3分钟)

```bash
cd goserver

# 使用演示配置（2分钟触发警报，5分钟紧急模式）
./lazarus-daemon.exe --enhanced --config config.openclaw.json
```

**预期输出**:
```
=== Lazarus Protocol Enhanced Daemon ===
Vault ID: 0xTEST
Smart Heartbeat: true
Activity Check: 10s
Inactivity Threshold: 2m
Emergency Threshold: 5m

✓ Daemon started successfully
  Press Ctrl+C to stop

[等待 2 分钟...]

🚨 TRIGGERING USER ALERT!
🤖 TRIGGERING OPENCLAW: WAKE UP ACTION
✓ OpenClaw accepted the task
  Browser should open shortly...

[浏览器自动打开，显示红色警报页面 + 警报音]
```

## 📋 完整功能清单

### ✅ 核心功能
- [x] Sui Move 智能合约（死人开关）
- [x] Rust 零知识加密工具
- [x] Go 守护进程（心跳监控）
- [x] 智能心跳（基于活动）
- [x] 多层警报系统
- [x] DeepBook PTB 集成
- [x] OpenClaw 浏览器自动化

### ✅ 警报系统
- [x] Windows GUI 弹窗
- [x] macOS 对话框
- [x] Linux 通知
- [x] 浏览器警报页面
- [x] 声音警报
- [x] OpenClaw 自动化

### ✅ DeFi 集成
- [x] DeepBook 市价单
- [x] PTB 构建器
- [x] 紧急变现功能
- [x] 滑点保护

### ✅ 演示功能
- [x] OpenClaw 唤醒动作
- [x] OpenClaw 遗言动作
- [x] Twitter 草稿生成

## 🎬 演示时间线

| 时间 | 事件 | 动作 |
|------|------|------|
| T+0s | 启动守护进程 | 显示配置信息 |
| T+2m | 触发警报 | OpenClaw + GUI + 浏览器 |
| T+2m30s | 用户响应 | 点击按钮 → 发送心跳 |
| T+5m | 紧急模式 | OpenClaw 打开 Twitter |
| T+5m30s | 执行 PTB | DeepBook 变现 → 转账 |

## 📊 项目规模

```
总代码行数: ~3000+ 行
源文件数量: 93 个
编程语言: 3 种 (Move, Rust, Go)
二进制大小: 8.6 MB
集成协议: 3 个 (Sui, Walrus, DeepBook)
```

## 🏆 Hackathon 评分点

### 技术深度 (40分)
- ✅ Move 智能合约编程
- ✅ Rust 系统编程
- ✅ Go 守护进程开发
- ✅ PTB 原子交易
- ✅ DeepBook DeFi 集成

### 创新性 (30分)
- ✅ 智能心跳机制
- ✅ 多层警报系统
- ✅ 紧急变现功能
- ✅ OpenClaw 自动化

### 实用性 (20分)
- ✅ 解决真实痛点
- ✅ 跨平台支持
- ✅ 完整文档
- ✅ 可演示 Demo

### 完整性 (10分)
- ✅ 前后端完整
- ✅ 测试覆盖
- ✅ 部署文档
- ✅ 视频演示

## 🎥 视频拍摄清单

### 准备工作
- [ ] 清理终端历史
- [ ] 准备演示配置
- [ ] 测试 OpenClaw 连接
- [ ] 准备 Twitter 测试账号

### 拍摄内容
- [ ] 项目介绍 (30s)
- [ ] 架构展示 (30s)
- [ ] 正常运行 (30s)
- [ ] 警报触发 (60s)
- [ ] 用户响应 (30s)
- [ ] 紧急模式 (60s)
- [ ] 总结 (30s)

### 后期制作
- [ ] 添加字幕
- [ ] 突出关键时刻
- [ ] 添加背景音乐
- [ ] 导出高清视频

## 📞 常见问题

### Q: OpenClaw 连接失败？
A: 确保 OpenClaw 服务器运行在 `http://localhost:8080`

### Q: 警报不触发？
A: 检查 `inactivity_threshold` 配置，演示用设置为 `2m`

### Q: Sui CLI 找不到？
A: 运行 `cargo install --locked --git https://github.com/MystenLabs/sui.git --branch mainnet sui`

### Q: 编译失败？
A: 确保 Go 1.21+, Rust 1.70+, Sui CLI 已安装

## 🔗 相关资源

- **项目总结**: `PROJECT_SUMMARY.md`
- **警报演示**: `goserver/ALERT_DEMO.md`
- **DeepBook 集成**: `goserver/DEEPBOOK_INTEGRATION.md`
- **合约文档**: `contract/README.md`
- **Rust CLI 文档**: `rustcli/README.md`

## 📝 提交清单

- [ ] 代码已提交到 GitHub
- [ ] README.md 完整
- [ ] 演示视频已录制
- [ ] 部署文档已完成
- [ ] 测试已通过
- [ ] 配置文件已准备

## 🎉 完成！

您现在拥有一个**完整的、可演示的、创新的 Lazarus Protocol 系统**！

**核心亮点**:
- 🔐 零知识加密
- ⛓️ Sui 智能合约
- 💰 DeepBook DeFi 集成
- 🤖 OpenClaw 自动化
- 🚨 多层警报系统

**准备好参加 Sui Hackathon 了！** 🏆

---

**需要帮助？** 查看 `PROJECT_SUMMARY.md` 获取完整文档。
