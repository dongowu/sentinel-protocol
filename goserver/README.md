# Sentinel Protocol - Go Heartbeat Daemon

Automated heartbeat service that keeps your Lazarus vault alive by periodically calling the `keep_alive` function on the Sui blockchain.

## Features

- **Automated Heartbeats**: Sends periodic heartbeat transactions to Sui
- **Vault Creation**: Integrates with Rust CLI to create new vaults
- **Graceful Shutdown**: Handles SIGINT/SIGTERM signals properly
- **Configuration Management**: JSON-based configuration
- **Error Handling**: Robust error handling with logging

## Installation

```bash
cd goserver
go build -o lazarus-daemon
```

## Usage

### Create a New Vault

```bash
./lazarus-daemon --create \
  --file /path/to/will.pdf \
  --beneficiary 0x1234567890abcdef1234567890abcdef12345678 \
  --walrus https://publisher.walrus-testnet.walrus.space \
  --epochs 5
```

This will:
1. Encrypt your file using the Rust CLI tool
2. Upload encrypted data to Walrus Protocol
3. Create a vault on Sui blockchain
4. Generate a `config.json` file

**CRITICAL**: Save the decryption key that is printed to the console!

### Run the Heartbeat Daemon

After creating a vault and updating `config.json`:

```bash
./lazarus-daemon --config config.json
```

The daemon will:
- Send an initial heartbeat immediately
- Send heartbeats every 7 days (configurable)
- Run until you stop it with Ctrl+C

## Configuration

Edit `config.json` after vault creation:

```json
{
  "vault_id": "0xabc123...",
  "owner_address": "0xYOUR_ADDRESS",
  "heartbeat_interval": "168h",
  "sui_rpc_url": "https://fullnode.testnet.sui.io:443",
  "package_id": "0xYOUR_PACKAGE_ID"
}
```

**Required Updates**:
- `owner_address`: Your Sui wallet address
- `package_id`: The deployed smart contract package ID

## Architecture

```
┌─────────────┐
│   Go Daemon │
└──────┬──────┘
       │
       ├─> Calls Rust CLI (vault creation)
       │   └─> Encrypts file
       │   └─> Uploads to Walrus
       │
       └─> Calls Sui CLI (heartbeats)
           └─> Invokes keep_alive()
```

## Production Deployment

### Systemd Service (Linux)

Create `/etc/systemd/system/lazarus-daemon.service`:

```ini
[Unit]
Description=Sentinel Protocol Heartbeat Daemon
After=network.target

[Service]
Type=simple
User=youruser
WorkingDirectory=/path/to/goserver
ExecStart=/path/to/goserver/lazarus-daemon --config /path/to/config.json
Restart=always
RestartSec=10

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
WORKDIR /app
COPY . .
RUN go build -o lazarus-daemon

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/lazarus-daemon .
COPY config.json .
CMD ["./lazarus-daemon", "--config", "config.json"]
```

Build and run:
```bash
docker build -t lazarus-daemon .
docker run -d --name lazarus --restart unless-stopped lazarus-daemon
```

## Development

### Run Tests

```bash
go test ./...
```

### Build for Multiple Platforms

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o lazarus-daemon-linux

# macOS
GOOS=darwin GOARCH=amd64 go build -o lazarus-daemon-macos

# Windows
GOOS=windows GOARCH=amd64 go build -o lazarus-daemon.exe
```

## Monitoring

View daemon logs:
```bash
# Systemd
sudo journalctl -u lazarus-daemon -f

# Docker
docker logs -f lazarus
```

## Security Considerations

1. **Private Keys**: Never commit `config.json` with real addresses
2. **File Permissions**: Ensure config file is readable only by daemon user
3. **Network Security**: Use HTTPS for all API calls
4. **Backup**: Keep backup of config.json in secure location

## Troubleshooting

### Heartbeat Fails

Check:
- Sui CLI is installed and configured
- Wallet has sufficient SUI for gas fees
- Package ID and Vault ID are correct
- Network connectivity to Sui RPC

### Vault Creation Fails

Check:
- Rust CLI tool is built (`cd ../rustcli && cargo build --release`)
- Walrus Publisher URL is accessible
- File path is correct and readable

## Future Enhancements

- [ ] Use Sui Go SDK instead of CLI
- [ ] Add metrics and monitoring (Prometheus)
- [ ] Implement retry logic with exponential backoff
- [ ] Add health check endpoint
- [ ] Support multiple vaults per daemon
- [ ] Email/SMS notifications on heartbeat failure

## License

MIT License - Built for Sui Hackathon
