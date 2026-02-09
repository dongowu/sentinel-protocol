package main

import "time"

// PolicyResult is the final policy decision for a command.
type PolicyResult struct {
	Action        string // ALLOW | REQUIRE_APPROVAL | BLOCK
	Reason        string
	RiskScore     float32
	AnomalyType   string
	NeedsApproval bool
}

// PolicyAuditEntry is a normalized audit event from PolicyGate.
type PolicyAuditEntry struct {
	Timestamp   time.Time `json:"timestamp"`
	AgentID     string    `json:"agent_id"`
	Command     string    `json:"command"`
	Action      string    `json:"action"`
	RiskScore   float32   `json:"risk_score"`
	AnomalyType string    `json:"anomaly_type"`
	Reason      string    `json:"reason"`
}

// PolicyGate wraps behavior detection into user-friendly decisions.
type PolicyGate struct {
	agentID string
	profile *AgentProfile
}

func NewPolicyGate(agentID string) *PolicyGate {
	return &PolicyGate{
		agentID: agentID,
		profile: NewAgentProfile(agentID),
	}
}

func (pg *PolicyGate) CheckCommand(command string) PolicyResult {
	anomaly := pg.profile.DetectAnomaly(command)

	// Hard blocks: extreme anomaly or known high-risk classes.
	if anomaly.Score >= 0.90 || anomaly.OpType == "PRIVILEGE_ESCALATION" || anomaly.OpType == "SYSTEM_MODIFICATION" {
		return PolicyResult{
			Action:        "BLOCK",
			Reason:        anomaly.Reason,
			RiskScore:     anomaly.Score,
			AnomalyType:   anomaly.OpType,
			NeedsApproval: false,
		}
	}

	if anomaly.Score >= 0.50 {
		return PolicyResult{
			Action:        "REQUIRE_APPROVAL",
			Reason:        anomaly.Reason,
			RiskScore:     anomaly.Score,
			AnomalyType:   anomaly.OpType,
			NeedsApproval: true,
		}
	}

	return PolicyResult{
		Action:        "ALLOW",
		Reason:        "within learned behavior profile",
		RiskScore:     anomaly.Score,
		AnomalyType:   anomaly.OpType,
		NeedsApproval: false,
	}
}

func (pg *PolicyGate) RecordSuccessfulOperation(command string) {
	pg.profile.RecordOperation(command)
}

func (pg *PolicyGate) GetAgentProfile() *AgentProfile {
	return pg.profile
}

func (pg *PolicyGate) LogToAudit(result PolicyResult, command string) PolicyAuditEntry {
	return PolicyAuditEntry{
		Timestamp:   time.Now().UTC(),
		AgentID:     pg.agentID,
		Command:     command,
		Action:      result.Action,
		RiskScore:   result.RiskScore,
		AnomalyType: result.AnomalyType,
		Reason:      result.Reason,
	}
}
