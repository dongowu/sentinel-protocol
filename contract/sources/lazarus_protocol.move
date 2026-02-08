/// LazarusProtocol: A Dead Man's Switch on Sui
///
/// This module implements a dead man's switch mechanism where a vault owner
/// must periodically send heartbeat signals. If the heartbeat threshold is exceeded,
/// the vault can be executed and transferred to a designated beneficiary.
module lazarus_protocol::lazarus_protocol {
    use sui::clock::Clock;
    use std::string::String;

    /// Threshold in milliseconds (e.g., 30 days = 30 * 24 * 60 * 60 * 1000)
    const HEARTBEAT_THRESHOLD_MS: u64 = 2592000000; // 30 days

    /// Error: Caller is not the owner
    const ENotOwner: u64 = 1;

    /// Error: Heartbeat threshold not exceeded yet
    const EThresholdNotExceeded: u64 = 2;

    /// Error: Vault already executed
    const EVaultAlreadyExecuted: u64 = 3;

    /// The main Vault object that holds the dead man's switch state
    public struct Vault has key, store {
        id: sui::object::UID,
        owner: address,
        beneficiary: address,
        encrypted_blob_id: String,
        last_heartbeat_ms: u64,
        is_executed: bool,
    }

    /// Event emitted when a heartbeat is received
    public struct HeartbeatEvent has copy, drop {
        vault_id: address,
        owner: address,
        timestamp_ms: u64,
    }

    /// Event emitted when the will is executed
    public struct WillExecutedEvent has copy, drop {
        vault_id: address,
        owner: address,
        beneficiary: address,
        timestamp_ms: u64,
        encrypted_blob_id: String,
    }

    /// Event emitted when a vault is created
    public struct VaultCreatedEvent has copy, drop {
        vault_id: address,
        owner: address,
        beneficiary: address,
        timestamp_ms: u64,
    }

    /// Creates a new Vault with the specified beneficiary and encrypted blob ID
    public fun create_vault(
        beneficiary: address,
        encrypted_blob_id: String,
        clock: &Clock,
        ctx: &mut sui::tx_context::TxContext
    ) {
        let sender = sui::tx_context::sender(ctx);
        let current_time = sui::clock::timestamp_ms(clock);

        let vault = Vault {
            id: sui::object::new(ctx),
            owner: sender,
            beneficiary,
            encrypted_blob_id,
            last_heartbeat_ms: current_time,
            is_executed: false,
        };

        let vault_id = sui::object::uid_to_address(&vault.id);

        sui::event::emit(VaultCreatedEvent {
            vault_id,
            owner: sender,
            beneficiary,
            timestamp_ms: current_time,
        });

        sui::transfer::share_object(vault);
    }

    /// Updates the heartbeat timestamp to keep the vault alive
    public fun keep_alive(
        vault: &mut Vault,
        clock: &Clock,
        ctx: &mut sui::tx_context::TxContext
    ) {
        let sender = sui::tx_context::sender(ctx);

        assert!(sender == vault.owner, ENotOwner);
        assert!(!vault.is_executed, EVaultAlreadyExecuted);

        let current_time = sui::clock::timestamp_ms(clock);
        vault.last_heartbeat_ms = current_time;

        sui::event::emit(HeartbeatEvent {
            vault_id: sui::object::uid_to_address(&vault.id),
            owner: vault.owner,
            timestamp_ms: current_time,
        });
    }

    /// Executes the will if the heartbeat threshold has been exceeded
    public fun execute_will(
        vault: &mut Vault,
        clock: &Clock,
        _ctx: &mut sui::tx_context::TxContext
    ) {
        let current_time = sui::clock::timestamp_ms(clock);

        assert!(!vault.is_executed, EVaultAlreadyExecuted);

        let time_since_heartbeat = current_time - vault.last_heartbeat_ms;
        assert!(time_since_heartbeat > HEARTBEAT_THRESHOLD_MS, EThresholdNotExceeded);

        vault.is_executed = true;

        sui::event::emit(WillExecutedEvent {
            vault_id: sui::object::uid_to_address(&vault.id),
            owner: vault.owner,
            beneficiary: vault.beneficiary,
            timestamp_ms: current_time,
            encrypted_blob_id: vault.encrypted_blob_id,
        });
    }

    public fun get_owner(vault: &Vault): address { vault.owner }
    public fun get_beneficiary(vault: &Vault): address { vault.beneficiary }
    public fun get_encrypted_blob_id(vault: &Vault): String { vault.encrypted_blob_id }
    public fun get_last_heartbeat_ms(vault: &Vault): u64 { vault.last_heartbeat_ms }
    public fun is_executed(vault: &Vault): bool { vault.is_executed }
    public fun get_threshold_ms(): u64 { HEARTBEAT_THRESHOLD_MS }

    public fun can_execute(vault: &Vault, clock: &Clock): bool {
        if (vault.is_executed) {
            return false
        };

        let current_time = sui::clock::timestamp_ms(clock);
        let time_since_heartbeat = current_time - vault.last_heartbeat_ms;
        time_since_heartbeat > HEARTBEAT_THRESHOLD_MS
    }

    #[test_only]
    public fun test_init(_ctx: &mut sui::tx_context::TxContext) {
        // Reserved for future unit tests.
    }
}
