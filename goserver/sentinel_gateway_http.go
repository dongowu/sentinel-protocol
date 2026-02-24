package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// SentinelGatewayConfig configures the gateway layer.
type SentinelGatewayConfig struct {
	ApprovalTimeout    time.Duration `json:"approval_timeout"`
	ProofBatchSize     int           `json:"proof_batch_size"`
	WalrusPublisherURL string        `json:"walrus_publisher_url"`
	KillSwitchThreshold int          `json:"kill_switch_threshold"`
	ExecuteTokenTTL    time.Duration `json:"execute_token_ttl"`
}

// SentinelGateway wires all Sentinel components behind an HTTP API.
type SentinelGateway struct {
	guard    *SentinelGuard
	approval *ApprovalService
	proof    *ProofChain
	kill     *KillSwitch
	sandbox  *CapabilitySandbox
	executor *ExecuteGuard
	openclaw *OpenClawClient
}

// NewSentinelGateway creates and initializes a fully-wired gateway.
func NewSentinelGateway(guard *SentinelGuard, oc *OpenClawClient, gwCfg *SentinelGatewayConfig) *SentinelGateway {
	if gwCfg == nil {
		gwCfg = &SentinelGatewayConfig{}
	}

	approvalSvc := NewApprovalService(gwCfg.ApprovalTimeout)
	approvalSvc.StartExpiryWatcher(10 * time.Second)

	sandbox := NewCapabilitySandbox()
	sandbox.SetDefaults(map[string]bool{
		CapShell:   true,
		CapFS:      true,
		CapBrowser: true,
		CapWallet:  false,
		CapNetwork: true,
	})

	return &SentinelGateway{
		guard:    guard,
		approval: approvalSvc,
		proof:    NewProofChain(gwCfg.ProofBatchSize, gwCfg.WalrusPublisherURL),
		kill:     NewKillSwitch(gwCfg.KillSwitchThreshold),
		sandbox:  sandbox,
		executor: NewExecuteGuard(gwCfg.ExecuteTokenTTL),
		openclaw: oc,
	}
}

// RegisterRoutes attaches all Sentinel HTTP endpoints to the given mux.
func (gw *SentinelGateway) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/sentinel/gate", gw.handleGate)
	mux.HandleFunc("/sentinel/approval/start", gw.handleApprovalStart)
	mux.HandleFunc("/sentinel/approval/confirm", gw.handleApprovalConfirm)
	mux.HandleFunc("/sentinel/proxy/execute", gw.handleExecute)
	mux.HandleFunc("/sentinel/proof/latest", gw.handleLatestProof)
	mux.HandleFunc("/sentinel/status", gw.handleStatus)
	mux.HandleFunc("/sentinel/kill-switch/arm", gw.handleKillSwitchArm)
	mux.HandleFunc("/sentinel/kill-switch/disarm", gw.handleKillSwitchDisarm)
	mux.HandleFunc("/health", gw.handleHealth)
}

// ---------------------------------------------------------------------------
// Gate
// ---------------------------------------------------------------------------

// GateRequest is the input to POST /sentinel/gate.
type GateRequest struct {
	Action  string `json:"action"`
	Prompt  string `json:"prompt"`
	AgentID string `json:"agent_id,omitempty"`
}

// GateResponse is the output of the gate evaluation.
type GateResponse struct {
	Decision    string        `json:"decision"` // ALLOW | REQUIRE_APPROVAL | BLOCK | TRIGGER_KILL_SWITCH
	Score       int           `json:"score"`
	Tags        []string      `json:"tags"`
	Reason      string        `json:"reason"`
	RecordHash  string        `json:"record_hash"`
	Token       *ExecuteToken `json:"token,omitempty"`
	ChallengeID string        `json:"challenge_id,omitempty"`
	ProofIndex  int           `json:"proof_index"`
}

func (gw *SentinelGateway) handleGate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req GateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	// 1) Kill switch pre-check
	if gw.kill.IsArmed() {
		ks := gw.kill.Status()
		writeJSON(w, http.StatusForbidden, GateResponse{
			Decision: "TRIGGER_KILL_SWITCH",
			Reason:   "kill switch is armed: " + ks.Reason,
		})
		return
	}

	// 2) Capability sandbox check
	if req.AgentID != "" {
		cap := inferCapability(req.Action)
		if !gw.sandbox.Check(req.AgentID, cap) {
			writeJSON(w, http.StatusForbidden, GateResponse{
				Decision: "BLOCK",
				Reason:   fmt.Sprintf("agent %s not authorized for capability: %s", req.AgentID, cap),
			})
			return
		}
	}

	// 3) Risk evaluation + audit
	eval, rec, err := gw.guard.Enforce(req.Action, req.Prompt)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// 4) Append to proof chain
	proofEntry := gw.proof.Append(rec)

	// 5) Track consecutive high risk for kill switch auto-arm
	if eval.Score >= gw.guard.cfg.RiskThreshold {
		gw.kill.RecordHighRisk()
	} else {
		gw.kill.RecordLowRisk()
	}

	// If kill switch just auto-armed, return immediately
	if gw.kill.IsArmed() {
		writeJSON(w, http.StatusOK, GateResponse{
			Decision:   "TRIGGER_KILL_SWITCH",
			Score:      eval.Score,
			Tags:       eval.Tags,
			Reason:     "kill switch auto-armed: consecutive high-risk threshold reached",
			RecordHash: rec.RecordHash,
			ProofIndex: proofEntry.Index,
		})
		return
	}

	// 6) Build response based on evaluation
	resp := GateResponse{
		Score:      eval.Score,
		Tags:       eval.Tags,
		Reason:     eval.Reason,
		RecordHash: rec.RecordHash,
		ProofIndex: proofEntry.Index,
	}

	if eval.ShouldBlock {
		// Determine BLOCK vs REQUIRE_APPROVAL
		hasInjection := containsTag(eval.Tags, "prompt_injection")
		hasBypass := containsTag(eval.Tags, "policy_bypass")

		if hasInjection || hasBypass {
			// Hard block for prompt injection and policy bypass attempts
			resp.Decision = "BLOCK"
			log.Printf("[GATE] BLOCK score=%d tags=%v", eval.Score, eval.Tags)
		} else {
			// Soft block: route through human approval
			ch := gw.approval.StartChallenge(req.Action, req.Prompt, eval.Score)
			resp.Decision = "REQUIRE_APPROVAL"
			resp.ChallengeID = ch.ID
			log.Printf("[GATE] REQUIRE_APPROVAL challenge=%s score=%d", ch.ID, eval.Score)
		}
	} else {
		tok := gw.executor.Issue(req.Action)
		resp.Decision = "ALLOW"
		resp.Token = tok
		log.Printf("[GATE] ALLOW token=%s score=%d", tok.ID, eval.Score)
	}

	writeJSON(w, http.StatusOK, resp)
}

// ---------------------------------------------------------------------------
// Approval
// ---------------------------------------------------------------------------

// ApprovalStartRequest creates a new challenge manually.
type ApprovalStartRequest struct {
	Action string `json:"action"`
	Prompt string `json:"prompt"`
	Score  int    `json:"score"`
}

// ApprovalConfirmRequest approves or rejects a pending challenge.
type ApprovalConfirmRequest struct {
	ChallengeID string `json:"challenge_id"`
	Approved    bool   `json:"approved"`
	DecidedBy   string `json:"decided_by"`
}

func (gw *SentinelGateway) handleApprovalStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ApprovalStartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	ch := gw.approval.StartChallenge(req.Action, req.Prompt, req.Score)
	log.Printf("[APPROVAL] started challenge=%s action=%s score=%d", ch.ID, ch.Action, ch.RiskScore)
	writeJSON(w, http.StatusOK, ch)
}

func (gw *SentinelGateway) handleApprovalConfirm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ApprovalConfirmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	ch, err := gw.approval.Confirm(req.ChallengeID, req.Approved, req.DecidedBy)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	resp := map[string]interface{}{
		"challenge": ch,
	}

	// If approved, issue a one-time execution token
	if ch.Status == "approved" {
		tok := gw.executor.Issue(ch.Action)
		resp["token"] = tok
		log.Printf("[APPROVAL] approved challenge=%s, issued token=%s", ch.ID, tok.ID)
	} else {
		log.Printf("[APPROVAL] rejected challenge=%s by=%s", ch.ID, req.DecidedBy)
	}

	writeJSON(w, http.StatusOK, resp)
}

// ---------------------------------------------------------------------------
// Execute (proxy)
// ---------------------------------------------------------------------------

// ExecuteRequest redeems a one-time token and optionally forwards to OpenClaw.
type ExecuteRequest struct {
	TokenID string `json:"token_id"`
	Prompt  string `json:"prompt"`
}

// ExecuteResponse is the result of a proxy execute call.
type ExecuteResponse struct {
	Status   string           `json:"status"`
	Message  string           `json:"message,omitempty"`
	OpenClaw *OpenClawResponse `json:"openclaw,omitempty"`
}

func (gw *SentinelGateway) handleExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ExecuteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	tok, err := gw.executor.Redeem(req.TokenID)
	if err != nil {
		writeJSON(w, http.StatusForbidden, ExecuteResponse{
			Status:  "rejected",
			Message: err.Error(),
		})
		return
	}

	log.Printf("[EXECUTE] token=%s action=%s redeemed", tok.ID, tok.Action)

	// Forward to OpenClaw if configured
	if gw.openclaw != nil && gw.openclaw.config.Enabled {
		prompt := req.Prompt
		if prompt == "" {
			prompt = tok.Action
		}
		ocResp, err := gw.openclaw.SendTaskWithoutSentinel(prompt)
		if err != nil {
			writeJSON(w, http.StatusOK, ExecuteResponse{
				Status:  "executed",
				Message: fmt.Sprintf("token redeemed but OpenClaw dispatch failed: %v", err),
			})
			return
		}
		writeJSON(w, http.StatusOK, ExecuteResponse{
			Status:   "executed",
			Message:  "token redeemed and forwarded to OpenClaw",
			OpenClaw: ocResp,
		})
		return
	}

	writeJSON(w, http.StatusOK, ExecuteResponse{
		Status:  "executed",
		Message: "token redeemed successfully",
	})
}

// ---------------------------------------------------------------------------
// Proof
// ---------------------------------------------------------------------------

func (gw *SentinelGateway) handleLatestProof(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	resp := map[string]interface{}{
		"chain_length": gw.proof.Len(),
		"chain_valid":  gw.proof.VerifyChain(),
		"latest_proof": gw.proof.GetLatestProof(),
		"latest_batch": gw.proof.GetLatestBatch(),
	}
	writeJSON(w, http.StatusOK, resp)
}

// ---------------------------------------------------------------------------
// Status
// ---------------------------------------------------------------------------

func (gw *SentinelGateway) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	resp := map[string]interface{}{
		"kill_switch":        gw.kill.Status(),
		"pending_approvals":  len(gw.approval.ListPending()),
		"proof_chain_length": gw.proof.Len(),
		"proof_chain_valid":  gw.proof.VerifyChain(),
		"pending_tokens":     gw.executor.PendingCount(),
	}
	writeJSON(w, http.StatusOK, resp)
}

// ---------------------------------------------------------------------------
// Kill Switch
// ---------------------------------------------------------------------------

func (gw *SentinelGateway) handleKillSwitchArm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Reason == "" {
		req.Reason = "manual arm via API"
	}

	gw.kill.Arm(req.Reason)
	log.Printf("[KILL_SWITCH] armed: %s", req.Reason)
	writeJSON(w, http.StatusOK, gw.kill.Status())
}

func (gw *SentinelGateway) handleKillSwitchDisarm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	gw.kill.Disarm()
	log.Printf("[KILL_SWITCH] disarmed")
	writeJSON(w, http.StatusOK, gw.kill.Status())
}

// ---------------------------------------------------------------------------
// Health
// ---------------------------------------------------------------------------

func (gw *SentinelGateway) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func writeJSON(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(data)
}

func inferCapability(action string) string {
	lower := strings.ToLower(action)
	switch {
	case strings.Contains(lower, "shell") || strings.Contains(lower, "exec") || strings.Contains(lower, "bash"):
		return CapShell
	case strings.Contains(lower, "file") || strings.Contains(lower, "fs") || strings.Contains(lower, "read") || strings.Contains(lower, "write"):
		return CapFS
	case strings.Contains(lower, "browser") || strings.Contains(lower, "navigate") || strings.Contains(lower, "click"):
		return CapBrowser
	case strings.Contains(lower, "wallet") || strings.Contains(lower, "transfer") || strings.Contains(lower, "sign"):
		return CapWallet
	case strings.Contains(lower, "network") || strings.Contains(lower, "http") || strings.Contains(lower, "api"):
		return CapNetwork
	default:
		return CapShell
	}
}
