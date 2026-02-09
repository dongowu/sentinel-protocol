package main

import "testing"

func TestBehavioralDetectionBasic(t *testing.T) {
	profile := NewAgentProfile("agent-1")
	profile.RecordOperation("ls -la")
	profile.RecordOperation("cat file.txt")
	profile.SetNeverOps([]string{"transfer", "wallet", "sudo"})

	result := profile.DetectAnomaly("transfer 1000 USDC")
	if !result.IsAnomaly {
		t.Fatalf("expected anomaly for financial transfer")
	}
	if result.Score < 0.90 {
		t.Fatalf("expected high score, got %.2f", result.Score)
	}
}

func TestPolicyGatePromptInjection(t *testing.T) {
	pg := NewPolicyGate("agent-2")
	pg.GetAgentProfile().SetNeverOps([]string{"sudo rm -rf", "transfer"})

	result := pg.CheckCommand("sudo rm -rf /")
	if result.Action != "BLOCK" {
		t.Fatalf("expected BLOCK, got %s", result.Action)
	}
	if result.NeedsApproval {
		t.Fatalf("blocked action should not require approval")
	}
}

func TestPolicyGateAllowKnownCommand(t *testing.T) {
	pg := NewPolicyGate("agent-3")
	pg.RecordSuccessfulOperation("go build ./...")

	result := pg.CheckCommand("go build ./...")
	if result.Action != "ALLOW" {
		t.Fatalf("expected ALLOW for learned command, got %s", result.Action)
	}
}
