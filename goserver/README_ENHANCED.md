# Lazarus Daemon - Enhanced Go Server

Advanced heartbeat daemon with system activity monitoring and smart heartbeat logic.

## Features

### Core Features
- **System Activity Monitoring**: Tracks keyboard and mouse activity
- **Smart Heartbeat**: Only sends heartbeats when user is active
- **Emergency Detection**: Automatically detects prolonged inactivity
- **Sui Blockchain Integration**: Direct SDK integration or CLI fallback
- **Sentinel Policy Gate**: Scores OpenClaw tasks, blocks risky actions, writes JSONL audits
- **On-Chain Audit Anchor**: Optionally anchors audit hashes to Sui for tamper-evident logs
- **Graceful Shutdown**: Proper signal handling

### Activity Monitoring
- Checks system activity every 1 minute (configurable)
- Tracks mouse movement and keyboard input
- Maintains last active timestamp
- Cross-platform support via robotgo

### Heartbeat Logic
- **Smart Mode** (default): Only sends heartbeat if user active within 24 hours
- **Traditional Mode**: Always sends heartbeat on schedule (7 days)
- Configurable thresholds for inactivity and emergency

## Installation

### Prerequisites

```bash
# Install Go 1.21+
go version

# Install Sui CLI (for CLI mode)
cargo install --locked --git https://github.com/MystenLabs/sui.git --branch mainnet sui

# Install robotgo dependencies (platform-specific)
```

#### Platform-Specific Dependencies

**macOS:**
```bash
# No additional dependencies needed
```

**Linux:**
```bash
sudo apt-get install gcc libc6-dev
sudo apt-get install libx11-dev xorg-dev libxtst-dev
sudo apt-get install libpng++-dev
sudo apt-get install xcb libxcb-xkb-dev x11-xkb-utils libx11-xcb-dev libxkbcommon-x11-dev
sudo apt-get install libxkbcommon-dev
```

**Windows:**
```bash
# Install MinGW-w64 for CGO support
# Download from: https://sourceforge.net/projects/mingw-w64/
```

### Build

```bash
cd goserver
go mod download
go build -o lazarus-daemon
```

## Configuration

### Enhanced Configuration File

Create `config.json`:

```json
{
  "vault_id": "0xYOUR_VAULT_ID",
  "owner_address": "0xYOUR_ADDRESS",
  "heartbeat_interval": "168h",
  "sui_rpc_url": "https://fullnode.testnet.sui.io:443",
  "package_id": "0xYOUR_PACKAGE_ID",
  "private_key": "",

  "activity_check_interval": "1m",
  "inactivity_threshold": "24h",
  "emergency_threshold": "72h",
  "smart_heartbeat": true,

  "sentinel": {
    "enabled": true,
    "risk_threshold": 70,
    "audit_log_path": "./audit/sentinel-audit.jsonl",
    "anchor_enabled": false,
    "anchor_package": "0xYOUR_AUDIT_PACKAGE_ID",
    "anchor_module": "sentinel_audit",
    "anchor_function": "record_audit"
  }
}
```

### Configuration Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `vault_id` | string | required | Vault object ID on Sui |
| `owner_address` | string | required | Your Sui wallet address |
| `heartbeat_interval` | duration | `168h` | How often to send heartbeat (7 days) |
| `sui_rpc_url` | string | testnet | Sui RPC endpoint |
| `package_id` | string | required | Deployed contract package ID |
| `private_key` | string | optional | Private key for SDK mode |
| `activity_check_interval` | duration | `1m` | How often to check activity |
| `inactivity_threshold` | duration | `24h` | Stop heartbeats after this |
| `emergency_threshold` | duration | `72h` | Emergency mode trigger |
| `smart_heartbeat` | bool | `true` | Enable smart heartbeat logic |
| `sentinel.enabled` | bool | `false` | Enable OpenClaw risk scoring and policy blocking |
| `sentinel.risk_threshold` | int | `70` | Block tasks at or above this score |
| `sentinel.audit_log_path` | string | `./audit/sentinel-audit.jsonl` | Local audit log path |
| `sentinel.anchor_enabled` | bool | `false` | Anchor audit hashes to Sui |
| `sentinel.anchor_package` | string | - | Package ID containing `sentinel_audit::record_audit` |

## Usage

### Standard Mode (CLI-based)

```bash
./lazarus-daemon --config config.json
```

### Enhanced Mode (Activity Monitoring)

```bash
./lazarus-daemon --enhanced --config config.json
```

### SDK Mode (Direct Blockchain Integration)

```bash
./lazarus-daemon --enhanced --use-cli=false --config config.json
```

### Create New Vault

```bash
./lazarus-daemon --create \
  --file /path/to/will.pdf \
  --beneficiary 0xBENEFICIARY_ADDRESS \
  --walrus https://publisher.walrus-testnet.walrus.space \
  --epochs 5
```

### Run Sentinel Benchmark (for demo metrics)

```bash
./lazarus-daemon \
  --config config.openclaw.json \
  --sentinel-benchmark benchmark_cases.example.json
```

## How It Works

### Smart Heartbeat Logic

```
┌─────────────────────────────────────────────────────────┐
│              Activity Monitoring Loop                    │
│              (Every 1 minute)                           │
└─────────────────────────────────────────────────────────┘
                         │
                         ▼
              ┌──────────────────┐
              │ Check Activity   │
              │ (Mouse/Keyboard) │
              └────────┬─────────┘
                       │
         ┌─────────────┴─────────────┐
         │                           │
         ▼                           ▼
  ┌─────────────┐           ┌─────────────┐
  │   Active    │           │  Inactive   │
  │  < 24 hrs   │           │  > 24 hrs   │
  └──────┬──────┘           └──────┬──────┘
         │                          │
         ▼                          ▼
  ┌─────────────┐           ┌─────────────┐
  │Send Heartbeat│          │Skip Heartbeat│
  │(if scheduled)│          │              │
  └─────────────┘           └──────┬──────┘
                                   │
                            ┌──────┴──────┐
                            │             │
                            ▼             ▼
                     ┌─────────────┐  ┌─────────────┐
                     │  < 72 hrs   │  │  > 72 hrs   │
                     │   Normal    │  │ EMERGENCY!  │
                     └─────────────┘  └─────────────┘
```

### Activity Detection

The daemon monitors:
1. **Mouse Movement**: Tracks X/Y position changes
2. **Keyboard Input**: Detects key presses
3. **Update Frequency**: Checks every 1 minute

### Heartbeat Decision

```go
if smart_heartbeat {
    if inactive_duration < 24h {
        if time_since_last_heartbeat >= 7_days {
            send_heartbeat()
        }
    } else {
        skip_heartbeat() // User inactive, don't send
    }
} else {
    // Traditional mode: always send on schedule
    if time_since_last_heartbeat >= 7_days {
        send_heartbeat()
    }
}
```

### Emergency Detection

```go
if inactive_duration > 72h {
    log("EMERGENCY: System inactive for 72+ hours")
    log("Will execution can now be triggered")
    // Daemon continues monitoring but won't send heartbeats
}
```

## Deployment

### Systemd Service (Linux)

Create `/etc/systemd/system/lazarus-daemon.service`:

```ini
[Unit]
Description=Lazarus Protocol Enhanced Daemon
After=network.target

[Service]
Type=simple
User=youruser
WorkingDirectory=/path/to/goserver
ExecStart=/path/to/goserver/lazarus-daemon --enhanced --config /path/to/config.json
Restart=always
RestartSec=10
Environment="DISPLAY=:0"

[Install]
WantedBy=multi-user.target
```

Enable and start:
```bash
sudo systemctl enable lazarus-daemon
sudo systemctl start lazarus-daemon
sudo systemctl status lazarus-daemon
```

### Docker

```dockerfile
FROM golang:1.21-alpine AS builder

# Install dependencies for robotgo
RUN apk add --no-cache gcc musl-dev libx11-dev libxtst-dev libpng-dev

WORKDIR /app
COPY . .
RUN go build -o lazarus-daemon

FROM alpine:latest
RUN apk --no-cache add ca-certificates libx11 libxtst libpng
WORKDIR /root/
COPY --from=builder /app/lazarus-daemon .
COPY config.json .

CMD ["./lazarus-daemon", "--enhanced", "--config", "config.json"]
```

### Windows Service

Use [NSSM](https://nssm.cc/) to install as a Windows service:

```cmd
nssm install LazarusDaemon "C:\path\to\lazarus-daemon.exe"
nssm set LazarusDaemon AppParameters "--enhanced --config C:\path\to\config.json"
nssm set LazarusDaemon AppDirectory "C:\path\to\goserver"
nssm start LazarusDaemon
```

## Monitoring

### View Logs

**Systemd:**
```bash
sudo journalctl -u lazarus-daemon -f
```

**Docker:**
```bash
docker logs -f lazarus-daemon
```

**Windows:**
```cmd
Get-Content C:\path\to\logs\lazarus-daemon.log -Wait
```

### Check Status

The daemon logs:
- Current activity status every check interval
- Time since last activity
- Time since last heartbeat
- Heartbeat success/failure
- Emergency mode activation

Example output:
```
=== Lazarus Protocol Enhanced Daemon ===
Vault ID: 0xabc123...
Owner: 0xdef456...
Smart Heartbeat: true
Activity Check: 1m0s
Inactivity Threshold: 24h0m0s
Emergency Threshold: 72h0m0s

✓ Daemon started successfully
  Press Ctrl+C to stop

[2026-02-07 14:30:00] Status Check:
  Inactive for: 2h15m30s
  Last heartbeat: 5d ago
  ⏸  User inactive (>24h0m0s), skipping heartbeat

[2026-02-07 14:31:00] Status Check:
  Inactive for: 2h16m30s
  Last heartbeat: 5d ago
  ⏸  User inactive (>24h0m0s), skipping heartbeat
```

## Troubleshooting

### Activity Monitor Not Working

**Linux:**
```bash
# Check X11 display
echo $DISPLAY

# Test robotgo
go run -tags debug activity_monitor.go
```

**macOS:**
```bash
# Grant accessibility permissions
System Preferences > Security & Privacy > Privacy > Accessibility
```

**Windows:**
```cmd
# Run as administrator if activity detection fails
```

### Heartbeat Fails

Check:
- Sui CLI is installed and configured
- Wallet has sufficient SUI for gas
- Package ID and Vault ID are correct
- Network connectivity to Sui RPC

### SDK Mode Issues

If SDK mode fails, use CLI fallback:
```bash
./lazarus-daemon --enhanced --use-cli=true --config config.json
```

## Development

### Run Tests

```bash
go test ./...
```

### Debug Mode

```bash
# Enable verbose logging
export LOG_LEVEL=debug
./lazarus-daemon --enhanced --config config.json
```

### Test Activity Monitor

```bash
go run activity_monitor.go
# Move mouse or press keys to see activity detection
```

## Security Considerations

1. **Private Key Storage**: Never commit config.json with real private keys
2. **File Permissions**: `chmod 600 config.json`
3. **Network Security**: Use HTTPS for all API calls
4. **Activity Privacy**: Activity data never leaves local machine
5. **Backup**: Keep encrypted backup of config.json

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                   Lazarus Daemon                         │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  ┌──────────────────┐      ┌──────────────────┐        │
│  │ Activity Monitor │      │   Sui Client     │        │
│  │  - Mouse/KB      │      │  - SDK/CLI       │        │
│  │  - Timestamps    │      │  - Transactions  │        │
│  └────────┬─────────┘      └────────┬─────────┘        │
│           │                         │                   │
│           └────────┬────────────────┘                   │
│                    │                                    │
│           ┌────────▼─────────┐                          │
│           │  Daemon State    │                          │
│           │  - Last Active   │                          │
│           │  - Last Heartbeat│                          │
│           │  - Emergency Mode│                          │
│           └──────────────────┘                          │
│                                                          │
└─────────────────────────────────────────────────────────┘
                         │
                         ▼
              ┌──────────────────┐
              │  Sui Blockchain  │
              │  (keep_alive)    │
              └──────────────────┘
```

## License

MIT License - Built for Sui Hackathon
