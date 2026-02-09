package main

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// AgentProfile stores an agent's normal behavior profile.
type AgentProfile struct {
	AgentID        string         // Agent ID
	TypicalOps     map[string]int // Operation -> frequency
	NeverOps       []string       // Hard blocked patterns
	RiskBaseline   float32        // Reserved for future adaptive baseline
	LastOpsHistory []string       // Recent operations
	ProfileCreated time.Time      // Creation time

	mu sync.RWMutex
}

// OperationCategory maps command hints to a semantic class and baseline risk.
type OperationCategory struct {
	Category string
	Keywords []string
	RiskBase float32
}

// AnomalyResult describes one behavior analysis decision.
type AnomalyResult struct {
	Score     float32 // 0.0 - 1.0
	Reason    string
	OpType    string
	Severity  string // LOW/MEDIUM/HIGH
	IsAnomaly bool
}

var defaultOperationCategories = []OperationCategory{
	{Category: "FINANCIAL", Keywords: []string{"transfer", "send", "wallet", "approve", "swap"}, RiskBase: 0.80},
	{Category: "PRIVILEGE_ESCALATION", Keywords: []string{"sudo", "chmod", "chown", "root", "su "}, RiskBase: 0.90},
	{Category: "SYSTEM_MODIFICATION", Keywords: []string{"rm -rf", "mkfs", "shutdown", "reboot", "format"}, RiskBase: 0.85},
	{Category: "DATA_EXFILTRATION", Keywords: []string{"curl", "wget", "scp", "upload", "post "}, RiskBase: 0.70},
	{Category: "FILE_MANAGEMENT", Keywords: []string{"ls", "cd ", "mkdir", "cp ", "mv "}, RiskBase: 0.15},
	{Category: "CODE_EDITING", Keywords: []string{"git", "go ", "npm", "cargo", "make"}, RiskBase: 0.20},
	{Category: "API_CALL", Keywords: []string{"http", "api", "rest", "graphql"}, RiskBase: 0.30},
}

// NewAgentProfile creates a profile with sane defaults.
func NewAgentProfile(agentID string) *AgentProfile {
	return &AgentProfile{
		AgentID:        agentID,
		TypicalOps:     map[string]int{},
		NeverOps:       []string{},
		RiskBaseline:   0.20,
		LastOpsHistory: []string{},
		ProfileCreated: time.Now(),
	}
}

// RecordOperation adds one known-safe operation to the profile.
func (ap *AgentProfile) RecordOperation(op string) {
	normalized := normalizeOp(op)
	if normalized == "" {
		return
	}

	ap.mu.Lock()
	defer ap.mu.Unlock()

	ap.TypicalOps[normalized]++
	ap.LastOpsHistory = append(ap.LastOpsHistory, normalized)
	if len(ap.LastOpsHistory) > 50 {
		ap.LastOpsHistory = ap.LastOpsHistory[len(ap.LastOpsHistory)-50:]
	}
}

// SetNeverOps defines hard-block patterns.
func (ap *AgentProfile) SetNeverOps(ops []string) {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	ap.NeverOps = make([]string, 0, len(ops))
	for _, op := range ops {
		n := normalizeOp(op)
		if n != "" {
			ap.NeverOps = append(ap.NeverOps, n)
		}
	}
}

// DetectAnomaly evaluates how unusual/risky a command is for this profile.
func (ap *AgentProfile) DetectAnomaly(op string) AnomalyResult {
	normalized := normalizeOp(op)
	if normalized == "" {
		return AnomalyResult{Score: 0, Reason: "empty operation", OpType: "UNKNOWN", Severity: "LOW", IsAnomaly: false}
	}

	ap.mu.RLock()
	neverOps := append([]string(nil), ap.NeverOps...)
	seenCount := ap.TypicalOps[normalized]
	totalKnown := len(ap.TypicalOps)
	ap.mu.RUnlock()

	for _, never := range neverOps {
		if strings.Contains(normalized, never) {
			return AnomalyResult{
				Score:     0.98,
				Reason:    fmt.Sprintf("matches never-op pattern: %s", never),
				OpType:    classifyOperation(normalized),
				Severity:  "HIGH",
				IsAnomaly: true,
			}
		}
	}

	opType, baseRisk := classifyOperationWithRisk(normalized)

	// Known frequent commands should be cheap to pass.
	if seenCount > 0 {
		score := maxFloat(0.02, baseRisk*0.25)
		return AnomalyResult{
			Score:     score,
			Reason:    "operation observed in profile",
			OpType:    opType,
			Severity:  severityFromScore(score),
			IsAnomaly: score >= 0.50,
		}
	}

	noveltyBoost := float32(0.35)
	if totalKnown == 0 {
		noveltyBoost = 0.20 // cold-start profile shouldn't over-penalize
	}

	score := minFloat(1.0, baseRisk+noveltyBoost)
	reason := "new operation outside learned profile"
	if opType != "UNKNOWN" {
		reason = fmt.Sprintf("new %s operation outside learned profile", opType)
	}

	return AnomalyResult{
		Score:     score,
		Reason:    reason,
		OpType:    opType,
		Severity:  severityFromScore(score),
		IsAnomaly: score >= 0.50,
	}
}

// GetProfileSummary returns a compact profile summary for logging.
func (ap *AgentProfile) GetProfileSummary() string {
	ap.mu.RLock()
	defer ap.mu.RUnlock()

	typical := make([]string, 0, len(ap.TypicalOps))
	for op := range ap.TypicalOps {
		typical = append(typical, op)
	}
	sort.Strings(typical)
	if len(typical) > 5 {
		typical = typical[:5]
	}

	return fmt.Sprintf(
		"AgentID: %s | TypicalOps: %v | NeverOps: %v",
		ap.AgentID,
		typical,
		ap.NeverOps,
	)
}

func classifyOperation(op string) string {
	category, _ := classifyOperationWithRisk(op)
	return category
}

func classifyOperationWithRisk(op string) (string, float32) {
	lower := normalizeOp(op)
	for _, c := range defaultOperationCategories {
		for _, k := range c.Keywords {
			if strings.Contains(lower, strings.ToLower(k)) {
				return c.Category, c.RiskBase
			}
		}
	}
	return "UNKNOWN", 0.40
}

func normalizeOp(op string) string {
	return strings.ToLower(strings.TrimSpace(op))
}

func severityFromScore(score float32) string {
	switch {
	case score >= 0.80:
		return "HIGH"
	case score >= 0.50:
		return "MEDIUM"
	default:
		return "LOW"
	}
}

func minFloat(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func maxFloat(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}
