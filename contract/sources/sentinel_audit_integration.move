/// Enhanced audit registry with rule/anomaly dimensions.
module lazarus_protocol::sentinel_audit_integration {
    use std::string;
    use std::string::String;
    use std::vector;

    public struct EnhancedAuditRecord has store, copy, drop {
        id: u64,
        timestamp: u64,
        action_tag: String,
        risk_score: u8,
        blocked: bool,
        rule_id: u64,
        anomaly_type: String,
        anomaly_score: u8,
    }

    public struct AuditRegistry has key {
        id: sui::object::UID,
        records: vector<EnhancedAuditRecord>,
        next_record_id: u64,
        total_blocked: u64,
    }

    fun init(ctx: &mut sui::tx_context::TxContext) {
        let registry = AuditRegistry {
            id: sui::object::new(ctx),
            records: vector::empty<EnhancedAuditRecord>(),
            next_record_id: 1,
            total_blocked: 0,
        };
        sui::transfer::share_object(registry);
    }

    public entry fun record_audit_with_rules(
        registry: &mut AuditRegistry,
        action_tag: String,
        risk_score: u8,
        blocked: bool,
        rule_id: u64,
        anomaly_type: String,
        anomaly_score: u8,
        ctx: &mut sui::tx_context::TxContext,
    ) {
        let id = registry.next_record_id;
        registry.next_record_id = id + 1;

        if (blocked) {
            registry.total_blocked = registry.total_blocked + 1;
        };

        vector::push_back(&mut registry.records, EnhancedAuditRecord {
            id,
            timestamp: sui::tx_context::epoch_timestamp_ms(ctx),
            action_tag,
            risk_score,
            blocked,
            rule_id,
            anomaly_type,
            anomaly_score,
        });
    }

    public entry fun record_rule_match(
        registry: &mut AuditRegistry,
        rule_id: u64,
        _command_hash: vector<u8>,
        risk_score: u8,
        action: String,
        ctx: &mut sui::tx_context::TxContext,
    ) {
        record_audit_with_rules(
            registry,
            action,
            risk_score,
            false,
            rule_id,
            string::utf8(b"RULE_MATCH"),
            0,
            ctx,
        );
    }

    public entry fun record_behavioral_anomaly(
        registry: &mut AuditRegistry,
        anomaly_type: String,
        anomaly_score: u8,
        risk_score: u8,
        blocked: bool,
        ctx: &mut sui::tx_context::TxContext,
    ) {
        record_audit_with_rules(
            registry,
            string::utf8(b"BEHAVIORAL_ANOMALY"),
            risk_score,
            blocked,
            0,
            anomaly_type,
            anomaly_score,
            ctx,
        );
    }

    public fun get_audit_stats(registry: &AuditRegistry): (u64, u64, u64) {
        let total = vector::length(&registry.records);
        if (total == 0) {
            return (0, 0, 0)
        };

        let block_rate = (registry.total_blocked * 100) / total;
        (total, registry.total_blocked, block_rate)
    }

    public fun query_records_by_rule(registry: &AuditRegistry, rule_id: u64): (u64, u64) {
        let mut total_matched = 0;
        let mut total_blocked = 0;

        let mut i = 0;
        let len = vector::length(&registry.records);
        while (i < len) {
            let rec = vector::borrow(&registry.records, i);
            if (rec.rule_id == rule_id) {
                total_matched = total_matched + 1;
                if (rec.blocked) {
                    total_blocked = total_blocked + 1;
                };
            };
            i = i + 1;
        };

        (total_matched, total_blocked)
    }

    public fun get_recent_records(registry: &AuditRegistry, count: u64): vector<u64> {
        let mut out = vector::empty<u64>();
        let total = vector::length(&registry.records);
        if (total == 0 || count == 0) {
            return out
        };

        let mut i = 0;
        let limit = if (count > total) { total } else { count };
        while (i < limit) {
            let idx = total - 1 - i;
            let rec = vector::borrow(&registry.records, idx);
            vector::push_back(&mut out, rec.id);
            i = i + 1;
        };
        out
    }
}
