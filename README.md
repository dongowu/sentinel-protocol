# Lazarus Protocol

A decentralized "Dead Man's Switch" system built on Sui blockchain for the Sui Hackathon.

## 🎯 Overview

Lazarus Protocol enables secure digital inheritance through a dead man's switch mechanism. Users encrypt sensitive files locally, store them on Walrus Protocol (Sui's decentralized storage), and set up automated heartbeat monitoring. If heartbeats stop for 30 days, beneficiaries can access the encrypted data.

## 🏗️ Project Structure

```
lazarus-protocol/
├── contract/              # Sui Move Smart Contract
│   ├── sources/
│   │   ├── lazarus_protocol.move
│   │   └── sentinel_audit.move
│   ├── Move.toml
│   └── README.md
│
├── rustcli/              # Rust CLI Tool (Encryption & Storage)
│   ├── src/
│   │   └── main.rs
│   ├── Cargo.toml
│   └── README.md
│
├── goserver/             # Go Heartbeat Daemon
│   ├── main.go
│   ├── go.mod
│   ├── config.example.json
│   └── README.md
│
└── README.md            # This file
```

## 🚀 Quick Start

### 1. Build the Rust CLI Tool

```bash
cd rustcli
cargo build --release
cd ..
```

### 2. Deploy the Smart Contract

```bash
cd contract
sui move build
sui client publish --gas-budget 100000000
# Save the package ID from output
cd ..
```

### 3. Create a Vault

```bash
cd goserver
go build -o lazarus-daemon

./lazarus-daemon --create \
  --file /path/to/your/will.pdf \
  --beneficiary 0xBENEFICIARY_ADDRESS \
  --walrus https://publisher.walrus-testnet.walrus.space \
  --epochs 5
```

**CRITICAL**: Save the `decryption_key` printed to console!

### 4. Configure and Run Daemon

Edit `config.json`:
- Update `owner_address` with your Sui address
- Update `package_id` with deployed contract package ID

Run the daemon:
```bash
./lazarus-daemon --config config.json
```

## 📦 Components

### Smart Contract (`contract/`)

**Technology**: Sui Move
**Purpose**: On-chain dead man's switch logic

- Vault creation with beneficiary designation
- Heartbeat tracking (30-day threshold)
- Will execution after threshold exceeded
- Event emission for frontend indexing

[View Contract Documentation](contract/README.md)

### Rust CLI Tool (`rustcli/`)

**Technology**: Rust
**Purpose**: Zero-knowledge encryption and storage

- AES-256-GCM encryption with random keys
- Walrus Protocol integration
- SHA-256 checksums for integrity
- JSON output for easy integration

[View CLI Documentation](rustcli/README.md)

### Go Daemon (`goserver/`)

**Technology**: Go
**Purpose**: Automated heartbeat service

- Periodic heartbeat transactions (every 7 days)
- Vault creation workflow
- Configuration management
- Graceful shutdown handling

[View Daemon Documentation](goserver/README.md)

## 🔄 System Workflow

```
┌─────────────────────────────────────────────────────────────┐
│                     1. VAULT CREATION                        │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
                    ┌──────────────────┐
                    │   User's File    │
                    │   (will.pdf)     │
                    └────────┬─────────┘
                             │
                             ▼
                    ┌──────────────────┐
                    │  Rust CLI Tool   │
                    │  - Encrypt       │
                    │  - Generate Key  │
                    └────────┬─────────┘
                             │
                             ▼
                    ┌──────────────────┐
                    │ Walrus Protocol  │
                    │ (Encrypted Blob) │
                    └────────┬─────────┘
                             │
                             ▼
                    ┌──────────────────┐
                    │  Sui Blockchain  │
                    │  (Vault Object)  │
                    └────────┬─────────┘
                             │
┌─────────────────────────────────────────────────────────────┐
│                   2. HEARTBEAT MONITORING                    │
└─────────────────────────────────────────────────────────────┘
                             │
                             ▼
                    ┌──────────────────┐
                    │   Go Daemon      │
                    │  Every 7 days    │
                    └────────┬─────────┘
                             │
                             ▼
                    ┌──────────────────┐
                    │  keep_alive()    │
                    │  Transaction     │
                    └────────┬─────────┘
                             │
┌─────────────────────────────────────────────────────────────┐
│              3. WILL EXECUTION (After 30 days)              │
└─────────────────────────────────────────────────────────────┘
                             │
                             ▼
                    ┌──────────────────┐
                    │  execute_will()  │
                    │  (Anyone)        │
                    └────────┬─────────┘
                             │
                             ▼
                    ┌──────────────────┐
                    │  Event Emitted   │
                    │  (blob_id)       │
                    └────────┬─────────┘
                             │
                             ▼
                    ┌──────────────────┐
                    │  Beneficiary     │
                    │  - Get blob_id   │
                    │  - Download blob │
                    │  - Decrypt       │
                    └──────────────────┘
```

## 🔒 Security Features

1. **Zero-Knowledge Encryption**: Decryption keys never leave local machine
2. **Decentralized Storage**: No single point of failure
3. **On-Chain Logic**: Transparent and auditable
4. **Access Control**: Owner-only heartbeat enforcement
5. **Event Logging**: All actions emit events for auditing
6. **Sentinel Audit Anchoring**: OpenClaw security decisions can be hashed and anchored on Sui

## 🛠️ Technology Stack

| Component | Technology | Purpose |
|-----------|-----------|---------|
| Smart Contract | Sui Move | On-chain logic & state |
| Encryption | Rust (AES-256-GCM) | Zero-knowledge encryption |
| Storage | Walrus Protocol | Decentralized blob storage |
| Daemon | Go | Automated heartbeats |
| Blockchain | Sui | Transaction execution |

## 📋 Requirements

- **Rust**: 1.70+ (for CLI tool)
- **Go**: 1.21+ (for daemon)
- **Sui CLI**: Latest version (for contract deployment)
- **Walrus Access**: Testnet publisher URL

## 🧪 Testing

### Test Rust CLI
```bash
cd rustcli
cargo test
```

### Test Smart Contract
```bash
cd contract
sui move test
```

### Test Go Daemon
```bash
cd goserver
go test ./...
```

## 📖 Documentation

- [Smart Contract Documentation](contract/README.md)
- [Rust CLI Documentation](rustcli/README.md)
- [Go Daemon Documentation](goserver/README.md)
- [System Overview](SYSTEM_OVERVIEW.md)

## 🎓 Use Cases

1. **Digital Inheritance**: Pass cryptocurrency wallets to heirs
2. **Emergency Access**: Backup recovery for critical systems
3. **Business Continuity**: Ensure access to critical credentials
4. **Personal Archives**: Secure time-capsule functionality

## 🚧 Future Enhancements

- [ ] Frontend web application
- [ ] Mobile app for heartbeat management
- [ ] Multi-signature support
- [ ] Configurable threshold periods
- [ ] Email/SMS notifications
- [ ] Multiple beneficiaries
- [ ] Partial data release (tiered access)

## 🤝 Contributing

This project was built for the Sui Hackathon. Contributions welcome!

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Submit a pull request

## 📄 License

MIT License - See LICENSE file for details

## 🏆 Hackathon Submission

**Built for**: Sui Hackathon 2026
**Category**: DeFi / Infrastructure
**Team**: Lazarus Protocol

### Innovation Highlights

- First dead man's switch implementation on Sui
- Zero-knowledge encryption with decentralized storage
- Seamless integration between Move, Rust, and Go
- Production-ready code with comprehensive testing

## 📞 Support

For issues and questions:
- Open an issue on GitHub
- Check component-specific README files
- Review the [System Overview](SYSTEM_OVERVIEW.md)

---

**⚠️ IMPORTANT**: This is a hackathon project. Audit thoroughly before production use.
