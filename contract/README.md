# Lazarus Protocol - Sui Smart Contract

Dead man's switch implementation on Sui blockchain using Move language.

## Overview

This smart contract implements a vault system where:
- Owner must send periodic heartbeats (every 30 days)
- If heartbeat threshold is exceeded, anyone can execute the will
- Beneficiary receives access to encrypted data stored on Walrus

## Contract Structure

### Vault Object

```move
public struct Vault has key, store {
    id: UID,
    owner: address,
    beneficiary: address,
    encrypted_blob_id: String,
    last_heartbeat_ms: u64,
    is_executed: bool,
}
```

### Entry Functions

#### `create_vault`
Creates a new vault with beneficiary and encrypted blob ID.

```move
public entry fun create_vault(
    beneficiary: address,
    encrypted_blob_id: String,
    clock: &Clock,
    ctx: &mut TxContext
)
```

#### `keep_alive`
Updates heartbeat timestamp. Owner-only function.

```move
public entry fun keep_alive(
    vault: &mut Vault,
    clock: &Clock,
    ctx: &mut TxContext
)
```

#### `execute_will`
Executes the will after 30-day threshold. Callable by anyone.

```move
public entry fun execute_will(
    vault: &mut Vault,
    clock: &Clock,
    ctx: &mut TxContext
)
```

### Events

- `VaultCreatedEvent`: Emitted when vault is created
- `HeartbeatEvent`: Emitted on each heartbeat
- `WillExecutedEvent`: Emitted when will is executed

## Building

```bash
sui move build
```

## Testing

```bash
sui move test
```

## Deployment

### Testnet

```bash
sui client publish --gas-budget 100000000
```

Save the package ID from the output for use in the Go daemon.

### Mainnet

```bash
sui client publish --gas-budget 100000000 --network mainnet
```

## Usage Examples

### Create a Vault

```bash
sui client call \
  --package $PACKAGE_ID \
  --module lazarus_protocol \
  --function create_vault \
  --args $BENEFICIARY_ADDRESS $BLOB_ID 0x6 \
  --gas-budget 10000000
```

Note: `0x6` is the shared Clock object on Sui.

### Send Heartbeat

```bash
sui client call \
  --package $PACKAGE_ID \
  --module lazarus_protocol \
  --function keep_alive \
  --args $VAULT_ID 0x6 \
  --gas-budget 10000000
```

### Execute Will

```bash
sui client call \
  --package $PACKAGE_ID \
  --module lazarus_protocol \
  --function execute_will \
  --args $VAULT_ID 0x6 \
  --gas-budget 10000000
```

## View Functions

Query vault state:

```bash
sui client object $VAULT_ID
```

Or use the view functions in your application:
- `get_owner(vault: &Vault): address`
- `get_beneficiary(vault: &Vault): address`
- `get_encrypted_blob_id(vault: &Vault): String`
- `get_last_heartbeat_ms(vault: &Vault): u64`
- `is_executed(vault: &Vault): bool`
- `can_execute(vault: &Vault, clock: &Clock): bool`

## Security Features

1. **Access Control**: Only owner can send heartbeats
2. **Threshold Enforcement**: 30-day minimum before execution
3. **Execution Lock**: Vault can only be executed once
4. **Event Logging**: All actions emit events for auditing

## Constants

- `HEARTBEAT_THRESHOLD_MS`: 2592000000 (30 days in milliseconds)

To modify the threshold, edit the constant in `sources/lazarus_protocol.move` and redeploy.

## Integration

### With Go Daemon

The Go daemon automatically calls `keep_alive` every 7 days.

### With Frontend

Use Sui TypeScript SDK to:
1. Query vault state
2. Display countdown to execution
3. Allow beneficiary to execute will
4. Retrieve encrypted blob ID from events

## Upgrading

To upgrade the contract:

```bash
sui client upgrade --gas-budget 100000000
```

Note: Ensure upgrade policies are set correctly to allow upgrades.

## License

MIT License - Built for Sui Hackathon
