/// Sentinel Audit: On-chain anchor for OpenClaw security decisions.
///
/// The Go service stores full audit logs locally/Walrus and anchors a
/// deterministic hash on Sui for verifiability.
module lazarus_protocol::sentinel_audit {
    use sui::event;
    use sui::tx_context::{Self, TxContext};

    /// Emitted for each anchored security decision.
    public struct AuditAnchoredEvent has copy, drop {
        operator: address,
        record_hash: address,
    }

    /// Anchor a local audit record hash on-chain.
    ///
    /// `record_hash` should be derived deterministically from the local
    /// decision payload (action, score, tags, decision, reason, timestamp).
    public entry fun record_audit(record_hash: address, ctx: &mut TxContext) {
        event::emit(AuditAnchoredEvent {
            operator: tx_context::sender(ctx),
            record_hash,
        });
    }
}
