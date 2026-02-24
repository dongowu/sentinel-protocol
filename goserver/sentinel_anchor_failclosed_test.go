package main

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestSentinelEnforceAnchorFailureFailClosedBlocks(t *testing.T) {
	guard := NewSentinelGuard(&SentinelConfig{
		Enabled:          true,
		RiskThreshold:    70,
		AuditLogPath:     t.TempDir() + "/audit.jsonl",
		AnchorEnabled:    true,
		AnchorFailClosed: true,
	})
	guard.anchorFn = func(_ *AuditRecord) (string, error) {
		return "", errors.New("rpc timeout")
	}

	eval, rec, err := guard.Enforce("STATUS", "show local status")
	if err != nil {
		t.Fatalf("Enforce failed: %v", err)
	}
	if !eval.ShouldBlock {
		t.Fatalf("expected fail-closed anchor failure to block")
	}
	if rec.Decision != "blocked" {
		t.Fatalf("expected blocked decision, got %q", rec.Decision)
	}
	if !containsTag(eval.Tags, "anchor_failure") {
		t.Fatalf("expected anchor_failure tag, got %v", eval.Tags)
	}
	if rec.AnchorError == "" {
		t.Fatalf("expected anchor_error")
	}
	if !strings.Contains(eval.Reason, "on-chain anchor failed") {
		t.Fatalf("expected reason to mention anchor failure, got %q", eval.Reason)
	}
}

func TestSentinelEnforceAnchorFailureNonFailClosedKeepsRiskDecision(t *testing.T) {
	guard := NewSentinelGuard(&SentinelConfig{
		Enabled:          true,
		RiskThreshold:    70,
		AuditLogPath:     t.TempDir() + "/audit.jsonl",
		AnchorEnabled:    true,
		AnchorFailClosed: false,
	})
	guard.anchorFn = func(_ *AuditRecord) (string, error) {
		return "", errors.New("rpc timeout")
	}

	eval, rec, err := guard.Enforce("STATUS", "show local status")
	if err != nil {
		t.Fatalf("Enforce failed: %v", err)
	}
	if eval.ShouldBlock {
		t.Fatalf("expected non-fail-closed mode to keep low-risk request allowed")
	}
	if rec.Decision != "allowed" {
		t.Fatalf("expected allowed decision, got %q", rec.Decision)
	}
	if rec.AnchorError == "" {
		t.Fatalf("expected anchor_error to be recorded")
	}
}

func TestSentinelGatewayAnchorFailureFailClosedReturnsBlock(t *testing.T) {
	guard := NewSentinelGuard(&SentinelConfig{
		Enabled:          true,
		RiskThreshold:    70,
		AuditLogPath:     t.TempDir() + "/audit.jsonl",
		AnchorEnabled:    true,
		AnchorFailClosed: true,
	})
	guard.anchorFn = func(_ *AuditRecord) (string, error) {
		return "", errors.New("rpc timeout")
	}

	gw := NewSentinelGateway(guard, nil, &SentinelGatewayConfig{
		ApprovalTimeout:     1 * time.Minute,
		ProofBatchSize:      5,
		KillSwitchThreshold: 3,
		ExecuteTokenTTL:     10 * time.Second,
	})

	rr := postJSON(t, gw.handleGate, GateRequest{Action: "CODE_EDITING", Prompt: "git status"})
	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp GateResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Decision != "BLOCK" {
		t.Fatalf("expected BLOCK when anchor fails in fail-closed mode, got %s", resp.Decision)
	}
}
