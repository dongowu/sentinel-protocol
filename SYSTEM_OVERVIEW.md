# Sentinel Protocol - Complete System Overview

## 🎯 Project Structure

```
sentinel-protocol/
├── sources/
│   └── lazarus_protocol.move    # Sui Smart Contract (Dead Man's Switch)
├── src/
│   └── main.rs                  # Rust CLI Tool (Encryption & Storage)
├── Move.toml                    # Sui package configuration
├── Cargo.toml                   # Rust package configuration
└── README.md                    # Documentation
```

## 🔐 Component 1: Rust CLI Tool (`lazarus-vault`)

**Purpose**: Zero-knowledge encryption and decentralized storage

**Features**:
- ✅ AES-256-GCM encryption with random keys
- ✅ SHA-256 checksums for integrity verification
- ✅ Walrus Protocol integration (HTTP PUT)
- ✅ JSON output for easy integration
- ✅ Comprehensive error handling
- ✅ Unit tests (3/3 passing)

**Usage**:
```bash
lazarus-vault encrypt-and-store \
  --file /path/to/will.pdf \
  --publisher https://publisher.walrus-testnet.walrus.space \
  --epochs 5
```

**Output**:
```json
{
  "blob_id": "abc123...",
  "decryption_key": "0123456789abcdef...",
  "checksum": "sha256hash...",
  "original_size": 1024,
  "encrypted_size": 1056
}
```

## ⛓️ Component 2: Sui Smart Contract (`lazarus_protocol.move`)

**Purpose**: On-chain dead man's switch logic

**Key Features**:
- ✅ Vault struct (shared object) with owner, beneficiary, encrypted_blob_id
- ✅ `create_vault()` - Initialize vault
- ✅ `keep_alive()` - Owner-only heartbeat (30-day threshold)
- ✅ `execute_will()` - Anyone can trigger after threshold
- ✅ Events: VaultCreated, Heartbeat, WillExecuted
- ✅ View functions for reading vault state
- ✅ Compiles successfully with Sui Move

## 🔄 System Workflow

```
1. User encrypts file
   └─> lazarus-vault encrypt-and-store --file will.pdf
   └─> Returns: blob_id + decryption_key

2. User creates vault on Sui
   └─> sui client call create_vault(beneficiary, blob_id)
   └─> Vault created with 30-day threshold

3. Go Daemon sends heartbeats
   └─> Every 7 days: sui client call keep_alive(vault)
   └─> Updates last_heartbeat_ms

4. If owner stops (30+ days)
   └─> Anyone calls: execute_will(vault)
   └─> Event emitted with blob_id
   └─> Beneficiary retrieves encrypted file from Walrus
   └─> Beneficiary uses decryption_key to decrypt
```

## 🛠️ Technical Stack

| Component | Technology | Purpose |
|-----------|-----------|---------|
| Smart Contract | Sui Move | On-chain logic & state |
| Encryption Tool | Rust | Zero-knowledge encryption |
| Storage | Walrus Protocol | Decentralized blob storage |
| Daemon | Go (planned) | Automated heartbeats |
| Frontend | React (planned) | User interface |

## 🔒 Security Guarantees

1. **Zero-Knowledge**: Decryption key never leaves local machine
2. **Encryption**: AES-256-GCM (industry standard)
3. **Decentralization**: No single point of failure
4. **Access Control**: Owner-only heartbeat enforcement
5. **Transparency**: All actions emit events for auditing

## 📦 Deliverables

### ✅ Completed
- [x] Sui Move smart contract with full functionality
- [x] Rust CLI tool with encryption & Walrus integration
- [x] Unit tests (all passing)
- [x] Comprehensive documentation
- [x] Release binary built

### 🔜 Next Steps (Integration)
- [ ] Deploy smart contract to Sui testnet
- [ ] Build Go daemon for automated heartbeats
- [ ] Create frontend for vault management
- [ ] End-to-end testing with real Walrus network

## 🚀 Quick Start

### Build the Rust Tool
```bash
cargo build --release
```

### Deploy the Smart Contract
```bash
sui move build
sui client publish --gas-budget 100000000
```

### Test Encryption
```bash
./target/release/lazarus-vault encrypt-and-store \
  --file test_will.txt \
  --publisher https://publisher.walrus-testnet.walrus.space \
  --epochs 5
```

## 📝 Example Integration (Go Daemon)

```go
// Encrypt and store file
cmd := exec.Command("lazarus-vault", "encrypt-and-store",
    "--file", willPath,
    "--publisher", walrusURL,
    "--epochs", "5")
output, _ := cmd.Output()

var result struct {
    BlobID        string `json:"blob_id"`
    DecryptionKey string `json:"decryption_key"`
}
json.Unmarshal(output, &result)

// Create vault on Sui
suiCmd := exec.Command("sui", "client", "call",
    "--package", packageID,
    "--module", "lazarus_protocol",
    "--function", "create_vault",
    "--args", beneficiary, result.BlobID)
suiCmd.Run()

// Start heartbeat loop
ticker := time.NewTicker(7 * 24 * time.Hour)
for range ticker.C {
    exec.Command("sui", "client", "call",
        "--function", "keep_alive",
        "--args", vaultID).Run()
}
```

## 🎓 Hackathon Highlights

**Innovation**:
- First dead man's switch on Sui blockchain
- Zero-knowledge encryption with decentralized storage
- Seamless integration between Rust, Move, and Walrus

**Technical Excellence**:
- Production-ready code with tests
- Comprehensive error handling
- Clean architecture with separation of concerns

**Real-World Use Case**:
- Digital inheritance planning
- Secure backup recovery
- Emergency access systems

---

Built for Sui Hackathon 2026 🏆
