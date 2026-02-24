package main

import (
	"sync"
	"time"
)

// KillSwitchStatus is the public snapshot returned by the Status() method
// and serialized in API responses.
type KillSwitchStatus struct {
	Armed               bool      `json:"armed"`
	Reason              string    `json:"reason,omitempty"`
	ArmedAt             time.Time `json:"armed_at,omitempty"`
	ConsecutiveHighRisk int       `json:"consecutive_high_risk"`
	Threshold           int       `json:"threshold"`
}

// KillSwitch is a global safety latch for the Sentinel system.
// When armed, all agent actions must be rejected until a human operator
// explicitly disarms the switch.
//
// The switch can be armed manually (Arm) or automatically when
// consecutiveHighRisk evaluations reach the configured threshold.
type KillSwitch struct {
	mu                  sync.RWMutex
	armed               bool
	consecutiveHighRisk int
	threshold           int
	armedAt             time.Time
	reason              string
}

// NewKillSwitch creates a KillSwitch that auto-arms after `threshold`
// consecutive high-risk evaluations. A threshold <= 0 is clamped to 3.
func NewKillSwitch(threshold int) *KillSwitch {
	if threshold <= 0 {
		threshold = 3
	}
	return &KillSwitch{
		threshold: threshold,
	}
}

// Arm activates the kill switch with a human-readable reason.
func (ks *KillSwitch) Arm(reason string) {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	ks.armed = true
	ks.armedAt = time.Now().UTC()
	ks.reason = reason
}

// Disarm deactivates the kill switch and resets the consecutive counter.
func (ks *KillSwitch) Disarm() {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	ks.armed = false
	ks.reason = ""
	ks.armedAt = time.Time{}
	ks.consecutiveHighRisk = 0
}

// IsArmed reports whether the kill switch is currently active.
func (ks *KillSwitch) IsArmed() bool {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	return ks.armed
}

// RecordHighRisk increments the consecutive high-risk counter. If the
// counter reaches the threshold the switch arms itself automatically.
func (ks *KillSwitch) RecordHighRisk() {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	ks.consecutiveHighRisk++
	if ks.consecutiveHighRisk >= ks.threshold && !ks.armed {
		ks.armed = true
		ks.armedAt = time.Now().UTC()
		ks.reason = "auto-armed: consecutive high-risk threshold reached"
	}
}

// RecordLowRisk resets the consecutive high-risk counter because the
// latest evaluation was not high-risk.
func (ks *KillSwitch) RecordLowRisk() {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	ks.consecutiveHighRisk = 0
}

// Status returns a point-in-time snapshot of the kill switch state.
func (ks *KillSwitch) Status() KillSwitchStatus {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	return KillSwitchStatus{
		Armed:               ks.armed,
		Reason:              ks.reason,
		ArmedAt:             ks.armedAt,
		ConsecutiveHighRisk: ks.consecutiveHighRisk,
		Threshold:           ks.threshold,
	}
}

// ---------------------------------------------------------------------------
// CapabilitySandbox
// ---------------------------------------------------------------------------

// Well-known capability names. Callers should use these constants rather than
// raw strings so that typos are caught at compile time.
const (
	CapShell   = "shell"
	CapFS      = "fs"
	CapBrowser = "browser"
	CapWallet  = "wallet"
	CapNetwork = "network"
)

// CapabilitySandbox enforces a per-agent allowlist of capabilities.
// Each agent can be granted or revoked individual capabilities; anything
// not explicitly granted falls back to the default policy.
type CapabilitySandbox struct {
	mu          sync.RWMutex
	allowedCaps map[string]map[string]bool // agentID -> capability -> allowed
	defaultCaps map[string]bool            // capability -> allowed
}

// NewCapabilitySandbox creates a sandbox with all default capabilities denied.
func NewCapabilitySandbox() *CapabilitySandbox {
	return &CapabilitySandbox{
		allowedCaps: make(map[string]map[string]bool),
		defaultCaps: make(map[string]bool),
	}
}

// SetDefaults configures the default policy applied when an agent has no
// explicit override for a capability.
func (cs *CapabilitySandbox) SetDefaults(caps map[string]bool) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cs.defaultCaps = make(map[string]bool, len(caps))
	for k, v := range caps {
		cs.defaultCaps[k] = v
	}
}

// Grant allows the specified capability for an agent.
func (cs *CapabilitySandbox) Grant(agentID, capability string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if cs.allowedCaps[agentID] == nil {
		cs.allowedCaps[agentID] = make(map[string]bool)
	}
	cs.allowedCaps[agentID][capability] = true
}

// Revoke explicitly denies the specified capability for an agent.
func (cs *CapabilitySandbox) Revoke(agentID, capability string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if cs.allowedCaps[agentID] == nil {
		cs.allowedCaps[agentID] = make(map[string]bool)
	}
	cs.allowedCaps[agentID][capability] = false
}

// Check returns true if the agent is allowed the given capability.
// Per-agent overrides take precedence over the default policy.
func (cs *CapabilitySandbox) Check(agentID, capability string) bool {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	if agentCaps, ok := cs.allowedCaps[agentID]; ok {
		if allowed, exists := agentCaps[capability]; exists {
			return allowed
		}
	}
	return cs.defaultCaps[capability]
}
