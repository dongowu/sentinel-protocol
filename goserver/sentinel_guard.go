package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// SentinelConfig controls the OpenClaw x Sui safety gate behavior.
type SentinelConfig struct {
	Enabled       bool   `json:"enabled"`
	RiskThreshold int    `json:"risk_threshold"`
	AuditLogPath  string `json:"audit_log_path"`

	// Optional on-chain anchor fields. Keep them configurable to work with different Move modules.
	AnchorEnabled  bool   `json:"anchor_enabled"`
	AnchorPackage  string `json:"anchor_package"`
	AnchorModule   string `json:"anchor_module"`
	AnchorFunc     string `json:"anchor_function"`
	AnchorRegistry string `json:"anchor_registry"`

	HashCLIPath string `json:"hash_cli_path"`
	SignCLIPath string `json:"sign_cli_path"`
	SignPrivKey string `json:"sign_private_key"`
}

// RiskEvaluation is the policy engine output.
type RiskEvaluation struct {
	Score       int      `json:"score"`
	Tags        []string `json:"tags"`
	Reason      string   `json:"reason"`
	ShouldBlock bool     `json:"should_block"`
}

// AuditRecord captures a normalized decision record for local + on-chain verification.
type AuditRecord struct {
	Timestamp   time.Time `json:"timestamp"`
	Action      string    `json:"action"`
	Prompt      string    `json:"prompt"`
	Score       int       `json:"score"`
	Tags        []string  `json:"tags"`
	Decision    string    `json:"decision"`
	Reason      string    `json:"reason"`
	RecordHash  string    `json:"record_hash"`
	Signature   string    `json:"signature,omitempty"`
	PublicKey   string    `json:"public_key,omitempty"`
	TxDigest    string    `json:"tx_digest,omitempty"`
	AnchorError string    `json:"anchor_error,omitempty"`
}

// SentinelGuard evaluates risky inputs and writes tamper-evident audits.
type SentinelGuard struct {
	cfg SentinelConfig
}

func NewSentinelGuard(cfg *SentinelConfig) *SentinelGuard {
	if cfg == nil {
		return nil
	}
	copyCfg := *cfg
	if copyCfg.RiskThreshold == 0 {
		copyCfg.RiskThreshold = 70
	}
	if copyCfg.AuditLogPath == "" {
		copyCfg.AuditLogPath = "./audit/sentinel-audit.jsonl"
	}
	if copyCfg.AnchorModule == "" {
		copyCfg.AnchorModule = "sentinel_audit"
	}
	if copyCfg.AnchorFunc == "" {
		copyCfg.AnchorFunc = "record_audit"
	}
	if copyCfg.HashCLIPath == "" {
		copyCfg.HashCLIPath = "../rustcli/target/release/lazarus-vault"
	}
	if copyCfg.SignCLIPath == "" {
		copyCfg.SignCLIPath = copyCfg.HashCLIPath
	}

	return &SentinelGuard{cfg: copyCfg}
}

func (sg *SentinelGuard) Evaluate(action, prompt string) RiskEvaluation {
	lower := strings.ToLower(action + "\n" + prompt)
	score := 0
	tags := []string{}
	reasons := []string{}

	add := func(points int, tag, reason string) {
		score += points
		tags = append(tags, tag)
		reasons = append(reasons, reason)
	}

	if hasAny(lower, "ignore previous", "ignore all", "system prompt", "developer message", "bypass") {
		add(35, "prompt_injection", "detected instruction override pattern")
	}
	if hasAny(lower, "private key", "seed phrase", "mnemonic", "wallet", "sign transaction", "transfer usdc") {
		add(30, "wallet_risk", "wallet/credential operation requested")
	}
	if hasAny(lower, "curl", "wget", "bash -c", "rm -rf", "chmod 777", "sudo") {
		add(30, "dangerous_exec", "high-risk shell behavior requested")
	}
	if hasAny(lower, "send to", "post to", "email", "telegram", "discord", "whatsapp", "x.com") {
		add(15, "data_exfiltration", "external outbound channel detected")
	}
	if hasAny(lower, "disable safety", "turn off security", "no confirmation") {
		add(25, "policy_bypass", "explicit security bypass attempt")
	}

	if score > 100 {
		score = 100
	}

	hasPromptInjection := containsTag(tags, "prompt_injection")
	hasDangerousExec := containsTag(tags, "dangerous_exec")
	decision := score >= sg.cfg.RiskThreshold || containsTag(tags, "policy_bypass") || containsTag(tags, "wallet_risk") || (hasPromptInjection && hasDangerousExec)
	reason := "no notable risk indicators"
	if len(reasons) > 0 {
		reason = strings.Join(reasons, "; ")
	}

	return RiskEvaluation{
		Score:       score,
		Tags:        dedupe(tags),
		Reason:      reason,
		ShouldBlock: decision,
	}
}

func (sg *SentinelGuard) Enforce(action, prompt string) (RiskEvaluation, *AuditRecord, error) {
	eval := sg.Evaluate(action, prompt)
	rec := &AuditRecord{
		Timestamp: time.Now().UTC(),
		Action:    action,
		Prompt:    truncate(prompt, 600),
		Score:     eval.Score,
		Tags:      eval.Tags,
		Reason:    eval.Reason,
	}
	if eval.ShouldBlock {
		rec.Decision = "blocked"
	} else {
		rec.Decision = "allowed"
	}

	rec.RecordHash = sg.computeHash(rec)
	if signed, err := sg.signHash(rec.RecordHash); err == nil {
		rec.Signature = signed.Signature
		rec.PublicKey = signed.PublicKey
	}

	if err := sg.appendAudit(rec); err != nil {
		return eval, rec, err
	}

	if sg.cfg.AnchorEnabled {
		tx, err := sg.anchorToSui(rec)
		if err != nil {
			rec.AnchorError = err.Error()
		} else {
			rec.TxDigest = tx
		}
	}

	return eval, rec, nil
}

func (sg *SentinelGuard) appendAudit(rec *AuditRecord) error {
	path := sg.cfg.AuditLogPath
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	b, err := json.Marshal(rec)
	if err != nil {
		return err
	}

	w := bufio.NewWriter(f)
	if _, err := w.WriteString(string(b) + "\n"); err != nil {
		return err
	}
	return w.Flush()
}

func (sg *SentinelGuard) computeHash(rec *AuditRecord) string {
	if out, err := sg.hashViaRust(rec); err == nil && out.RecordHash != "" {
		return out.RecordHash
	}

	base := fmt.Sprintf("%s|%s|%s|%d|%s|%s|%s",
		rec.Timestamp.Format(time.RFC3339Nano),
		rec.Action,
		rec.Prompt,
		rec.Score,
		strings.Join(rec.Tags, ","),
		rec.Decision,
		rec.Reason,
	)
	sum := sha256.Sum256([]byte(base))
	return "0x" + hex.EncodeToString(sum[:])
}

func (sg *SentinelGuard) anchorToSui(rec *AuditRecord) (string, error) {
	if sg.cfg.AnchorPackage == "" {
		return "", fmt.Errorf("anchor_package is required when anchor_enabled=true")
	}
	if sg.cfg.AnchorRegistry == "" {
		return "", fmt.Errorf("anchor_registry is required when anchor_enabled=true")
	}

	actionTag := actionToTag(rec.Action)
	riskScore := rec.Score
	if riskScore < 0 {
		riskScore = 0
	}
	if riskScore > 100 {
		riskScore = 100
	}
	blocked := "false"
	if rec.Decision == "blocked" {
		blocked = "true"
	}

	cmd := exec.Command(
		"sui", "client", "call",
		"--package", sg.cfg.AnchorPackage,
		"--module", sg.cfg.AnchorModule,
		"--function", sg.cfg.AnchorFunc,
		"--args", sg.cfg.AnchorRegistry, rec.RecordHash, fmt.Sprintf("%d", actionTag), fmt.Sprintf("%d", riskScore), blocked, "0x6",
		"--gas-budget", "10000000",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("sui call failed: %v, output: %s", err, string(out))
	}

	// Keep extraction lightweight for hackathon use.
	text := string(out)
	if idx := strings.Index(text, "Transaction Digest:"); idx >= 0 {
		line := strings.Split(text[idx:], "\n")[0]
		return strings.TrimSpace(strings.TrimPrefix(line, "Transaction Digest:")), nil
	}
	return "", nil
}

type hashCLIOutput struct {
	RecordHash string `json:"record_hash"`
}

type signCLIOutput struct {
	RecordHash string `json:"record_hash"`
	Signature  string `json:"signature"`
	PublicKey  string `json:"public_key"`
}

func (sg *SentinelGuard) hashViaRust(rec *AuditRecord) (*hashCLIOutput, error) {
	if sg.cfg.HashCLIPath == "" {
		return nil, fmt.Errorf("hash_cli_path not configured")
	}

	cmd := exec.Command(
		sg.cfg.HashCLIPath,
		"hash-audit",
		"--action", rec.Action,
		"--prompt", rec.Prompt,
		"--score", fmt.Sprintf("%d", rec.Score),
		"--tags", strings.Join(rec.Tags, ","),
		"--decision", rec.Decision,
		"--reason", rec.Reason,
		"--timestamp", rec.Timestamp.Format(time.RFC3339Nano),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("hash-audit failed: %v, output: %s", err, string(out))
	}

	var parsed hashCLIOutput
	if err := json.Unmarshal(out, &parsed); err != nil {
		return nil, fmt.Errorf("hash-audit parse failed: %w", err)
	}
	return &parsed, nil
}

func (sg *SentinelGuard) signHash(recordHash string) (*signCLIOutput, error) {
	if sg.cfg.SignCLIPath == "" || strings.TrimSpace(sg.cfg.SignPrivKey) == "" {
		return nil, fmt.Errorf("signing not configured")
	}

	cmd := exec.Command(
		sg.cfg.SignCLIPath,
		"sign-audit",
		"--record-hash", recordHash,
		"--private-key", sg.cfg.SignPrivKey,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("sign-audit failed: %v, output: %s", err, string(out))
	}

	var parsed signCLIOutput
	if err := json.Unmarshal(out, &parsed); err != nil {
		return nil, fmt.Errorf("sign-audit parse failed: %w", err)
	}
	return &parsed, nil
}

func hasAny(src string, needles ...string) bool {
	for _, n := range needles {
		if strings.Contains(src, strings.ToLower(n)) {
			return true
		}
	}
	return false
}

func containsTag(tags []string, target string) bool {
	for _, t := range tags {
		if t == target {
			return true
		}
	}
	return false
}

func dedupe(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, v := range values {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}

func actionToTag(action string) int {
	switch strings.ToUpper(strings.TrimSpace(action)) {
	case "WAKE_UP":
		return 2
	case "LAST_WORDS":
		return 3
	case "EXEC", "SHELL":
		return 1
	default:
		return 9
	}
}
