/// Sentinel Audit: policy-aware, on-chain audit anchoring for OpenClaw actions.
///
/// The off-chain service computes deterministic record hashes for decisions and
/// stores detailed payloads locally/Walrus. This module stores the governance
/// state and emits verifiable anchor events.
module lazarus_protocol::sentinel_audit {
    use sui::clock::{Self, Clock};
    use sui::event;
    use sui::object::{Self, UID};
    use sui::transfer;
    use sui::tx_context::{Self, TxContext};

    /// Error: caller is not admin.
    const E_NOT_ADMIN: u64 = 1;

    /// Error: caller is not an approved operator.
    const E_NOT_OPERATOR: u64 = 2;

    /// Error: invalid policy version.
    const E_INVALID_POLICY_VERSION: u64 = 3;

    /// Shared registry that defines who can anchor records and policy metadata.
    public struct Registry has key {
        id: UID,
        admin: address,
        policy_version: u64,
        policy_hash: address,
    }

    /// Event emitted whenever policy metadata changes.
    public struct PolicyUpdatedEvent has copy, drop {
        operator: address,
        policy_version: u64,
        policy_hash: address,
    }

    /// Event emitted when operator permissions change.
    public struct OperatorUpdatedEvent has copy, drop {
        operator: address,
        target: address,
        enabled: bool,
    }

    /// Event emitted for each anchored decision.
    public struct AuditAnchoredEvent has copy, drop {
        operator: address,
        record_hash: address,
        action_tag: u8,
        risk_score: u8,
        blocked: bool,
        policy_version: u64,
        timestamp_ms: u64,
    }

    /// Dynamic field key for operator allowlist.
    public struct OperatorKey has copy, drop, store {
        value: address,
    }

    /// Package init: creates shared registry with initial policy metadata.
    fun init(ctx: &mut TxContext) {
        let sender = tx_context::sender(ctx);
        let registry = Registry {
            id: object::new(ctx),
            admin: sender,
            policy_version: 1,
            policy_hash: @0x0,
        };
        transfer::share_object(registry);
    }

    /// Admin updates policy version/hash used by off-chain Sentinel logic.
    public entry fun update_policy(
        registry: &mut Registry,
        policy_version: u64,
        policy_hash: address,
        ctx: &mut TxContext,
    ) {
        assert!(tx_context::sender(ctx) == registry.admin, E_NOT_ADMIN);
        assert!(policy_version > 0, E_INVALID_POLICY_VERSION);

        registry.policy_version = policy_version;
        registry.policy_hash = policy_hash;

        event::emit(PolicyUpdatedEvent {
            operator: tx_context::sender(ctx),
            policy_version,
            policy_hash,
        });
    }

    /// Admin grants/revokes an address that can anchor audit records.
    public entry fun set_operator(
        registry: &mut Registry,
        operator: address,
        enabled: bool,
        ctx: &mut TxContext,
    ) {
        assert!(tx_context::sender(ctx) == registry.admin, E_NOT_ADMIN);

        let key = OperatorKey { value: operator };
        if (enabled) {
            if (!sui::dynamic_field::exists_<OperatorKey>(&registry.id, key)) {
                sui::dynamic_field::add(&mut registry.id, key, true);
            };
        } else {
            if (sui::dynamic_field::exists_<OperatorKey>(&registry.id, key)) {
                let _ = sui::dynamic_field::remove<OperatorKey, bool>(&mut registry.id, key);
            };
        };

        event::emit(OperatorUpdatedEvent {
            operator: tx_context::sender(ctx),
            target: operator,
            enabled,
        });
    }

    /// Anchor one Sentinel decision hash under current policy context.
    ///
    /// - `record_hash`: deterministic hash of the off-chain decision payload.
    /// - `action_tag`: app-defined action enum (e.g. 1=exec,2=browser,3=message).
    /// - `risk_score`: normalized 0-100 risk score.
    /// - `blocked`: true if request was blocked.
    public entry fun record_audit(
        registry: &Registry,
        record_hash: address,
        action_tag: u8,
        risk_score: u8,
        blocked: bool,
        clock: &Clock,
        ctx: &mut TxContext,
    ) {
        let sender = tx_context::sender(ctx);
        assert!(is_operator(registry, sender), E_NOT_OPERATOR);

        event::emit(AuditAnchoredEvent {
            operator: sender,
            record_hash,
            action_tag,
            risk_score,
            blocked,
            policy_version: registry.policy_version,
            timestamp_ms: clock::timestamp_ms(clock),
        });
    }

    /// View helpers
    public fun policy_version(registry: &Registry): u64 {
        registry.policy_version
    }

    public fun policy_hash(registry: &Registry): address {
        registry.policy_hash
    }

    public fun admin(registry: &Registry): address {
        registry.admin
    }

    public fun is_operator(registry: &Registry, operator: address): bool {
        if (operator == registry.admin) {
            return true
        };
        sui::dynamic_field::exists_<OperatorKey>(&registry.id, OperatorKey { value: operator })
    }
}
