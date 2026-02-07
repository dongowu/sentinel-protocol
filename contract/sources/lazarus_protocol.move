/// LazarusProtocol: A Dead Man's Switch on Sui
///
/// This module implements a dead man's switch mechanism where a vault owner
/// must periodically send heartbeat signals. If the heartbeat threshold is exceeded,
/// the vault can be executed and transferred to a designated beneficiary.
module lazarus_protocol::lazarus_protocol {
    use sui::object::{Self, UID};
    use sui::tx_context::{Self, TxContext};
    use sui::transfer;
    use sui::clock::{Self, Clock};
    use sui::event;
    use std::string::String;

    // ======== Constants ========

    /// Threshold in milliseconds (e.g., 30 days = 30 * 24 * 60 * 60 * 1000)
    const HEARTBEAT_THRESHOLD_MS: u64 = 2592000000; // 30 days

    // ======== Errors ========

    /// Error: Caller is not the owner
    const ENotOwner: u64 = 1;

    /// Error: Heartbeat threshold not exceeded yet
    const EThresholdNotExceeded: u64 = 2;

    /// Error: Vault already executed
    const EVaultAlreadyExecuted: u64 = 3;

    // ======== Structs ========

    /// The main Vault object that holds the dead man's switch state
    public struct Vault has key, store {
        id: UID,
        owner: address,
        beneficiary: address,
        encrypted_blob_id: String,
        last_heartbeat_ms: u64,
        is_executed: bool,
    }

    // ======== Events ========

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

    // ======== Entry Functions ========

    /// Creates a new Vault with the specified beneficiary and encrypted blob ID
    ///
    /// # Arguments
    /// * `beneficiary` - The address that will receive the vault when executed
    /// * `encrypted_blob_id` - The ID of the encrypted data blob (from Rust tool)
    /// * `clock` - The shared Clock object for timestamp
    /// * `ctx` - Transaction context
    public entry fun create_vault(
        beneficiary: address,
        encrypted_blob_id: String,
        clock: &Clock,
        ctx: &mut TxContext
    ) {
        let sender = tx_context::sender(ctx);
        let current_time = clock::timestamp_ms(clock);

        let vault = Vault {
            id: object::new(ctx),
            owner: sender,
            beneficiary,
            encrypted_blob_id,
            last_heartbeat_ms: current_time,
            is_executed: false,
        };

        let vault_id = object::uid_to_address(&vault.id);

        // Emit vault creation event
        event::emit(VaultCreatedEvent {
            vault_id,
            owner: sender,
            beneficiary,
            timestamp_ms: current_time,
        });

        // Share the vault object so anyone can read it and execute the will
        transfer::share_object(vault);
    }

    /// Updates the heartbeat timestamp to keep the vault alive
    /// Can only be called by the vault owner (typically via Go Daemon)
    ///
    /// # Arguments
    /// * `vault` - Mutable reference to the Vault object
    /// * `clock` - The shared Clock object for timestamp
    /// * `ctx` - Transaction context
    public entry fun keep_alive(
        vault: &mut Vault,
        clock: &Clock,
        ctx: &mut TxContext
    ) {
        let sender = tx_context::sender(ctx);

        // Assert that the caller is the owner
        assert!(sender == vault.owner, ENotOwner);

        // Assert that the vault hasn't been executed
        assert!(!vault.is_executed, EVaultAlreadyExecuted);

        let current_time = clock::timestamp_ms(clock);
        vault.last_heartbeat_ms = current_time;

        // Emit heartbeat event for frontend indexing
        event::emit(HeartbeatEvent {
            vault_id: object::uid_to_address(&vault.id),
            owner: vault.owner,
            timestamp_ms: current_time,
        });
    }

    /// Executes the will if the heartbeat threshold has been exceeded
    /// Can be called by anyone once the threshold is met
    ///
    /// # Arguments
    /// * `vault` - Mutable reference to the Vault object
    /// * `clock` - The shared Clock object for timestamp
    /// * `ctx` - Transaction context
    public entry fun execute_will(
        vault: &mut Vault,
        clock: &Clock,
        ctx: &mut TxContext
    ) {
        let current_time = clock::timestamp_ms(clock);

        // Assert that the vault hasn't been executed yet
        assert!(!vault.is_executed, EVaultAlreadyExecuted);

        // Check if the threshold has been exceeded
        let time_since_heartbeat = current_time - vault.last_heartbeat_ms;
        assert!(time_since_heartbeat > HEARTBEAT_THRESHOLD_MS, EThresholdNotExceeded);

        // Mark vault as executed
        vault.is_executed = true;

        // Emit will executed event with encrypted blob ID for beneficiary
        event::emit(WillExecutedEvent {
            vault_id: object::uid_to_address(&vault.id),
            owner: vault.owner,
            beneficiary: vault.beneficiary,
            timestamp_ms: current_time,
            encrypted_blob_id: vault.encrypted_blob_id,
        });

        // Note: The vault remains a shared object but is marked as executed.
        // The beneficiary can now access the encrypted_blob_id from the event
        // or by reading the vault object directly.
        // If you want to transfer ownership, you would need to make it a non-shared
        // object initially and use transfer::transfer() here instead.
    }

    // ======== View Functions ========

    /// Returns the vault owner address
    public fun get_owner(vault: &Vault): address {
        vault.owner
    }

    /// Returns the beneficiary address
    public fun get_beneficiary(vault: &Vault): address {
        vault.beneficiary
    }

    /// Returns the encrypted blob ID
    public fun get_encrypted_blob_id(vault: &Vault): String {
        vault.encrypted_blob_id
    }

    /// Returns the last heartbeat timestamp
    public fun get_last_heartbeat_ms(vault: &Vault): u64 {
        vault.last_heartbeat_ms
    }

    /// Returns whether the vault has been executed
    public fun is_executed(vault: &Vault): bool {
        vault.is_executed
    }

    /// Returns the heartbeat threshold constant
    public fun get_threshold_ms(): u64 {
        HEARTBEAT_THRESHOLD_MS
    }

    /// Checks if the vault can be executed based on current time
    public fun can_execute(vault: &Vault, clock: &Clock): bool {
        if (vault.is_executed) {
            return false
        };

        let current_time = clock::timestamp_ms(clock);
        let time_since_heartbeat = current_time - vault.last_heartbeat_ms;
        time_since_heartbeat > HEARTBEAT_THRESHOLD_MS
    }

    // ======== Test Functions ========

    #[test_only]
    public fun test_init(ctx: &mut TxContext) {
        // Test initialization function
    }
}
