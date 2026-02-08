# Sentinel Protocol - 警报系统演示指南

## 功能概述

当用户 24 小时无活动时，系统会触发**戏剧性的警报**，要求用户确认存活。这个功能非常适合视频演示！

## 警报触发条件

```
时间线：
├─ 0-24h:  正常活动，定期发送心跳
├─ 24-72h: ⚠️ 警报区间（触发 GUI 弹窗 + 浏览器警报）
└─ 72h+:   🚨 紧急模式（遗嘱可被执行）
```

## 警报系统功能

### 1. GUI 弹窗（跨平台）

**Windows:**
- PowerShell 消息框
- 显示警告图标
- "I'm Alive" / "Cancel" 按钮

**macOS:**
- osascript 对话框
- 系统原生样式
- "I'm Alive" / "Cancel" 按钮

**Linux:**
- zenity 或 kdialog 对话框
- 适配不同桌面环境

### 2. 浏览器警报页面

自动打开浏览器显示：
- 🎨 渐变背景（紫色主题）
- ⚠️ 大号警告图标
- ⏱️ 实时倒计时
- 💚 "I'M ALIVE!" 大按钮
- 📊 详细信息面板

### 3. 声音警报

**Windows:** 系统蜂鸣声（3次）
**macOS:** Sosumi 系统音（3次）
**Linux:** 闹钟音效

## 演示流程

### 场景 1：正常响应

```bash
# 1. 启动守护进程
./lazarus-daemon --enhanced --config config.json

# 2. 等待 24 小时（演示时可修改阈值）
# 系统检测到 24 小时无活动

# 3. 触发警报
[2026-02-07 14:30:00] Status Check:
  Inactive for: 24h0m0s
  Last heartbeat: 7d ago

🚨 TRIGGERING USER ALERT!
  ✓ Alert page opened in browser
  ♪ Playing alert sound...

# 4. 用户点击 "I'm Alive"
✓ User responded to alert!
  Sending immediate heartbeat...
  ✓ Emergency heartbeat sent successfully!
```

### 场景 2：用户未响应

```bash
# 1-3. 同上

# 4. 用户 30 分钟内未响应
⏱  User did not respond to alert

# 5. 继续监控，6 小时后再次警报
[2026-02-07 20:30:00] Status Check:
  Inactive for: 30h0m0s

🚨 TRIGGERING USER ALERT!
  (再次弹窗)

# 6. 如果 72 小时仍无响应
⚠️  EMERGENCY THRESHOLD EXCEEDED!
  System inactive for 72h (threshold: 72h)
  Will execution can now be triggered by anyone
```

## 配置文件

```json
{
  "vault_id": "0xYOUR_VAULT_ID",
  "owner_address": "0xYOUR_ADDRESS",
  "heartbeat_interval": "168h",
  "sui_rpc_url": "https://fullnode.testnet.sui.io:443",
  "package_id": "0xYOUR_PACKAGE_ID",

  "activity_check_interval": "1m",
  "inactivity_threshold": "24h",
  "emergency_threshold": "72h",
  "smart_heartbeat": true
}
```

### 演示用配置（快速测试）

```json
{
  "activity_check_interval": "10s",
  "inactivity_threshold": "2m",
  "emergency_threshold": "5m",
  "smart_heartbeat": true
}
```

这样可以在 2 分钟后触发警报，5 分钟后进入紧急模式。

## 浏览器警报页面预览

```
┌─────────────────────────────────────────┐
│                                         │
│              ⚠️                         │
│                                         │
│    LAZARUS PROTOCOL WARNING             │
│                                         │
│  Your system has been inactive for      │
│  24 hours. If you do not respond,      │
│  your digital will may be executed.     │
│                                         │
│         48h 0m remaining                │
│                                         │
│     ┌─────────────────────┐            │
│     │  I'M ALIVE! 💚      │            │
│     └─────────────────────┘            │
│                                         │
│  ┌───────────────────────────────┐    │
│  │ What happens if I don't       │    │
│  │ respond?                      │    │
│  │                               │    │
│  │ After 72h of total inactivity,│    │
│  │ your vault will be unlocked   │    │
│  │ and your beneficiary will be  │    │
│  │ able to access your encrypted │    │
│  │ data.                         │    │
│  └───────────────────────────────┘    │
│                                         │
└─────────────────────────────────────────┘
```

## 视频演示脚本

### 第一幕：正常运行

```
旁白："Sentinel Protocol 守护进程正在后台运行..."
屏幕：终端显示正常心跳日志
```

### 第二幕：警报触发

```
旁白："24 小时后，系统检测到用户无活动..."
屏幕：
  1. 终端显示 "🚨 TRIGGERING USER ALERT!"
  2. GUI 弹窗出现（带警告图标）
  3. 浏览器自动打开警报页面
  4. 播放警报音效
```

### 第三幕：用户响应

```
旁白："用户点击 'I'm Alive' 按钮..."
屏幕：
  1. 按钮点击动画
  2. 终端显示 "✓ User responded to alert!"
  3. 发送心跳交易到区块链
  4. 显示交易哈希
```

### 第四幕：紧急模式（可选）

```
旁白："如果用户 72 小时未响应..."
屏幕：
  1. 终端显示 "⚠️ EMERGENCY THRESHOLD EXCEEDED!"
  2. 显示遗嘱可被执行的提示
  3. 展示受益人如何触发 execute_will
```

## 手动测试命令

### 测试 GUI 弹窗（Windows）

```powershell
powershell -Command "Add-Type -AssemblyName System.Windows.Forms; [System.Windows.Forms.MessageBox]::Show('Test Alert', 'Sentinel Protocol', [System.Windows.Forms.MessageBoxButtons]::YesNo, [System.Windows.Forms.MessageBoxIcon]::Warning)"
```

### 测试浏览器警报

```bash
# 创建测试 HTML 文件
cat > /tmp/test_alert.html << 'EOF'
<!DOCTYPE html>
<html>
<head><title>Test Alert</title></head>
<body style="background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); display: flex; justify-content: center; align-items: center; height: 100vh; margin: 0;">
  <div style="background: white; padding: 40px; border-radius: 20px; text-align: center;">
    <h1 style="color: #e74c3c;">⚠️ TEST ALERT</h1>
    <button onclick="alert('Confirmed!')" style="background: #27ae60; color: white; border: none; padding: 20px 60px; font-size: 24px; border-radius: 50px; cursor: pointer;">I'M ALIVE!</button>
  </div>
</body>
</html>
EOF

# 打开浏览器
xdg-open /tmp/test_alert.html  # Linux
open /tmp/test_alert.html      # macOS
start /tmp/test_alert.html     # Windows
```

## 代码位置

- **警报系统**: `goserver/alert_system.go`
- **集成逻辑**: `goserver/main_enhanced.go:146-210`
- **配置示例**: `goserver/config.enhanced.json`

## 技术亮点

1. **跨平台支持**: Windows/macOS/Linux 原生 GUI
2. **多重警报**: GUI + 浏览器 + 声音
3. **实时倒计时**: JavaScript 动态更新
4. **优雅降级**: GUI 失败时回退到终端
5. **防骚扰**: 6 小时冷却期，避免频繁弹窗
6. **即时响应**: 用户确认后立即发送链上心跳

## 演示效果

这个功能在视频演示中会非常有戏剧性：

1. 📱 **视觉冲击**: 突然弹出的警告窗口
2. 🎵 **听觉提醒**: 警报音效
3. ⏱️ **紧迫感**: 实时倒计时
4. 🔗 **区块链交互**: 点击按钮 → 链上交易
5. ✅ **即时反馈**: 交易成功确认

完美展示了 "系统报警 → 人工解除 → 链上状态更新" 的完整流程！
