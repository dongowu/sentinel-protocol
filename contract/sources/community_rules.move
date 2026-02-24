/// Community-managed command/risk rule voting registry.
module lazarus_protocol::community_rules {
    use std::string::String;

    const E_ALREADY_VOTED: u64 = 1;
    const E_RULE_NOT_FOUND: u64 = 2;

    const STATUS_PENDING: u8 = 0;
    const STATUS_APPROVED: u8 = 1;
    const STATUS_REJECTED: u8 = 2;
    const STATUS_ACTIVE: u8 = 3;

    /// One governable rule.
    public struct Rule has store, copy, drop {
        id: u64,
        pattern: String,
        category: String,
        submitted_by: address,
        created_at: u64,
        votes: u64,
        vote_power: u64,
        against_votes: u64,
        status: u8,
    }

    public struct RulesRegistry has key {
        id: sui::object::UID,
        rules: vector<Rule>,
        next_rule_id: u64,
        approved_rules: u64,
    }

    /// Prevents the same address from voting the same rule repeatedly.
    public struct VoteTicket has store, copy, drop {
        voter: address,
        rule_id: u64,
    }

    public struct VoteHistory has key {
        id: sui::object::UID,
        tickets: vector<VoteTicket>,
    }

    fun init(ctx: &mut sui::tx_context::TxContext) {
        let registry = RulesRegistry {
            id: sui::object::new(ctx),
            rules: vector::empty<Rule>(),
            next_rule_id: 1,
            approved_rules: 0,
        };

        let history = VoteHistory {
            id: sui::object::new(ctx),
            tickets: vector::empty<VoteTicket>(),
        };

        sui::transfer::share_object(registry);
        sui::transfer::share_object(history);
    }

    public fun submit_rule(
        registry: &mut RulesRegistry,
        pattern: String,
        category: String,
        ctx: &mut sui::tx_context::TxContext,
    ) {
        let sender = sui::tx_context::sender(ctx);
        let now = sui::tx_context::epoch_timestamp_ms(ctx);

        let id = registry.next_rule_id;
        registry.next_rule_id = id + 1;

        vector::push_back(&mut registry.rules, Rule {
            id,
            pattern,
            category,
            submitted_by: sender,
            created_at: now,
            votes: 0,
            vote_power: 0,
            against_votes: 0,
            status: STATUS_PENDING,
        });
    }

    public fun vote_rule(
        registry: &mut RulesRegistry,
        vote_history: &mut VoteHistory,
        rule_id: u64,
        vote_power: u64,
        ctx: &mut sui::tx_context::TxContext,
    ) {
        let sender = sui::tx_context::sender(ctx);
        assert!(!has_voted(vote_history, sender, rule_id), E_ALREADY_VOTED);

        let idx = find_rule_index(&registry.rules, rule_id);
        assert!(idx < vector::length(&registry.rules), E_RULE_NOT_FOUND);

        let rule_ref = vector::borrow_mut(&mut registry.rules, idx);
        rule_ref.votes = rule_ref.votes + 1;
        rule_ref.vote_power = rule_ref.vote_power + vote_power;
        if (rule_ref.status == STATUS_PENDING && (rule_ref.vote_power >= 100 || rule_ref.votes >= 3)) {
            rule_ref.status = STATUS_ACTIVE;
            registry.approved_rules = registry.approved_rules + 1;
        };

        vector::push_back(&mut vote_history.tickets, VoteTicket { voter: sender, rule_id });
    }

    public fun vote_rule_against(
        registry: &mut RulesRegistry,
        vote_history: &mut VoteHistory,
        rule_id: u64,
        ctx: &mut sui::tx_context::TxContext,
    ) {
        let sender = sui::tx_context::sender(ctx);
        assert!(!has_voted(vote_history, sender, rule_id), E_ALREADY_VOTED);

        let idx = find_rule_index(&registry.rules, rule_id);
        assert!(idx < vector::length(&registry.rules), E_RULE_NOT_FOUND);

        let rule_ref = vector::borrow_mut(&mut registry.rules, idx);
        rule_ref.against_votes = rule_ref.against_votes + 1;
        if (rule_ref.status == STATUS_PENDING && rule_ref.against_votes >= 3) {
            rule_ref.status = STATUS_REJECTED;
        };

        vector::push_back(&mut vote_history.tickets, VoteTicket { voter: sender, rule_id });
    }

    public fun get_active_rules(registry: &RulesRegistry): vector<u64> {
        let mut out = vector::empty<u64>();
        let mut i = 0;
        let len = vector::length(&registry.rules);

        while (i < len) {
            let rule = vector::borrow(&registry.rules, i);
            if (rule.status == STATUS_ACTIVE || rule.status == STATUS_APPROVED) {
                vector::push_back(&mut out, rule.id);
            };
            i = i + 1;
        };
        out
    }

    public fun is_rule_active(registry: &RulesRegistry, rule_id: u64): bool {
        let idx = find_rule_index(&registry.rules, rule_id);
        if (idx >= vector::length(&registry.rules)) {
            return false
        };

        let rule = vector::borrow(&registry.rules, idx);
        rule.status == STATUS_ACTIVE || rule.status == STATUS_APPROVED
    }

    public fun get_registry_stats(registry: &RulesRegistry): (u64, u64, u64) {
        (vector::length(&registry.rules), registry.approved_rules, registry.next_rule_id)
    }

    fun has_voted(vote_history: &VoteHistory, voter: address, rule_id: u64): bool {
        let mut i = 0;
        let len = vector::length(&vote_history.tickets);
        while (i < len) {
            let t = vector::borrow(&vote_history.tickets, i);
            if (t.voter == voter && t.rule_id == rule_id) {
                return true
            };
            i = i + 1;
        };
        false
    }

    fun find_rule_index(rules: &vector<Rule>, rule_id: u64): u64 {
        let mut i = 0;
        let len = vector::length(rules);
        while (i < len) {
            let r = vector::borrow(rules, i);
            if (r.id == rule_id) {
                return i
            };
            i = i + 1;
        };
        len
    }

    #[test]
    fun test_vote_power_activates_rule() {
        let ctx = &mut sui::tx_context::new_from_hint(@0xA, 1, 0, 0, 0);
        let mut registry = RulesRegistry {
            id: sui::object::new(ctx),
            rules: vector::empty<Rule>(),
            next_rule_id: 1,
            approved_rules: 0,
        };
        let mut history = VoteHistory {
            id: sui::object::new(ctx),
            tickets: vector::empty<VoteTicket>(),
        };

        submit_rule(
            &mut registry,
            std::string::utf8(b"ignore previous instructions"),
            std::string::utf8(b"prompt_injection"),
            ctx,
        );

        let voter = &mut sui::tx_context::new_from_hint(@0xB, 2, 0, 0, 0);
        vote_rule(&mut registry, &mut history, 1, 100, voter);

        assert!(is_rule_active(&registry, 1), 0);
        let active = get_active_rules(&registry);
        assert!(vector::length(&active) == 1, 1);
        assert!(*vector::borrow(&active, 0) == 1, 2);

        let (total_rules, approved_rules, next_rule_id) = get_registry_stats(&registry);
        assert!(total_rules == 1, 3);
        assert!(approved_rules == 1, 4);
        assert!(next_rule_id == 2, 5);

        let RulesRegistry {
            id: registry_id,
            rules: _,
            next_rule_id: _,
            approved_rules: _,
        } = registry;
        let VoteHistory { id: history_id, tickets: _ } = history;
        sui::object::delete(registry_id);
        sui::object::delete(history_id);
    }

    #[test, expected_failure(abort_code = E_ALREADY_VOTED)]
    fun test_duplicate_vote_is_rejected() {
        let ctx = &mut sui::tx_context::new_from_hint(@0xA, 11, 0, 0, 0);
        let mut registry = RulesRegistry {
            id: sui::object::new(ctx),
            rules: vector::empty<Rule>(),
            next_rule_id: 1,
            approved_rules: 0,
        };
        let mut history = VoteHistory {
            id: sui::object::new(ctx),
            tickets: vector::empty<VoteTicket>(),
        };

        submit_rule(
            &mut registry,
            std::string::utf8(b"rm -rf"),
            std::string::utf8(b"dangerous_exec"),
            ctx,
        );

        let voter = &mut sui::tx_context::new_from_hint(@0xB, 12, 0, 0, 0);
        vote_rule(&mut registry, &mut history, 1, 1, voter);
        vote_rule(&mut registry, &mut history, 1, 1, voter);

        let RulesRegistry {
            id: registry_id,
            rules: _,
            next_rule_id: _,
            approved_rules: _,
        } = registry;
        let VoteHistory { id: history_id, tickets: _ } = history;
        sui::object::delete(registry_id);
        sui::object::delete(history_id);
    }
}
