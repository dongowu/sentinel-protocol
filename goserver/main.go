package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

func main() {
	// Command-line flags
	configPath := flag.String("config", "configs/config.json", "Path to configuration file")
	walrusURL := flag.String("walrus", "https://publisher.walrus-testnet.walrus.space", "Walrus publisher URL")
	sentinelBenchmark := flag.String("sentinel-benchmark", "", "Path to Sentinel benchmark JSON file")
	sentinelBenchmarkOut := flag.String("sentinel-benchmark-out", "", "Optional path to write Sentinel benchmark report JSON")
	sentinelEvalAction := flag.String("sentinel-eval-action", "", "Action to evaluate with Sentinel (requires --sentinel-eval-prompt)")
	sentinelEvalPrompt := flag.String("sentinel-eval-prompt", "", "Prompt to evaluate with Sentinel (requires --sentinel-eval-action)")
	sentinelOneClickAction := flag.String("sentinel-oneclick-action", "", "One-click action sent to OpenClaw with Sentinel audit/enforcement")
	sentinelOneClickPrompt := flag.String("sentinel-oneclick-prompt", "", "One-click prompt sent to OpenClaw (requires --sentinel-oneclick-action)")
	sentinelProxy := flag.Bool("sentinel-proxy", false, "Start Sentinel in-path proxy HTTP server")
	sentinelProxyAddr := flag.String("sentinel-proxy-addr", "127.0.0.1:18080", "Listen address for the Sentinel proxy server")
	flag.Parse()

	if *sentinelEvalAction != "" || *sentinelEvalPrompt != "" {
		if err := runSentinelEvalMode(*configPath, *sentinelEvalAction, *sentinelEvalPrompt, os.Stdout); err != nil {
			log.Fatalf("Sentinel eval failed: %v", err)
		}
		return
	}

	if *sentinelOneClickAction != "" || *sentinelOneClickPrompt != "" {
		if err := runSentinelOneClickMode(*configPath, *sentinelOneClickAction, *sentinelOneClickPrompt, os.Stdout); err != nil {
			log.Fatalf("Sentinel one-click failed: %v", err)
		}
		return
	}

	if *sentinelProxy {
		runSentinelProxyMode(*configPath, *sentinelProxyAddr, *walrusURL)
		return
	}

	if *sentinelBenchmark != "" {
		sentinelCfg, err := loadSentinelConfigOnly(*configPath)
		if err != nil {
			log.Fatalf("Failed to load sentinel config for benchmark: %v", err)
		}

		guard := NewSentinelGuard(resolveSentinelConfig(sentinelCfg))

		report, err := RunSentinelBenchmarkWithReport(*sentinelBenchmark, guard)
		if err != nil {
			log.Fatalf("Sentinel benchmark failed: %v", err)
		}
		b, _ := json.MarshalIndent(report, "", "  ")
		fmt.Println("\nSentinel benchmark report:")
		fmt.Println(string(b))

		if strings.TrimSpace(*sentinelBenchmarkOut) != "" {
			if err := writeBenchmarkReport(*sentinelBenchmarkOut, report); err != nil {
				log.Fatalf("Failed to write benchmark report: %v", err)
			}
			log.Printf("Benchmark report written to %s", *sentinelBenchmarkOut)
		}
		return
	}

	// No mode selected — print usage.
	flag.Usage()
}

func runSentinelEvalMode(configPath, action, prompt string, out io.Writer) error {
	action = strings.TrimSpace(action)
	prompt = strings.TrimSpace(prompt)

	if action == "" {
		return fmt.Errorf("--sentinel-eval-action is required in sentinel eval mode")
	}
	if prompt == "" {
		return fmt.Errorf("--sentinel-eval-prompt is required in sentinel eval mode")
	}

	sentinelCfg, err := loadSentinelConfigOnly(configPath)
	if err != nil {
		return fmt.Errorf("failed to load sentinel config: %w", err)
	}

	guard := NewSentinelGuard(resolveSentinelConfig(sentinelCfg))
	if guard == nil {
		return fmt.Errorf("sentinel guard is not configured")
	}

	eval, rec, err := guard.Enforce(action, prompt)
	if err != nil {
		return fmt.Errorf("sentinel enforce failed: %w", err)
	}

	result := SentinelEvalOutput{
		Action:       action,
		Prompt:       prompt,
		Score:        eval.Score,
		Tags:         eval.Tags,
		Decision:     rec.Decision,
		Reason:       eval.Reason,
		RecordHash:   rec.RecordHash,
		AuditLogPath: guard.cfg.AuditLogPath,
	}

	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

func runSentinelOneClickMode(configPath, action, prompt string, out io.Writer) error {
	return runSentinelOneClickModeWithSender(configPath, action, prompt, out, defaultOpenClawTaskSender)
}

type openClawTaskSender func(cfg *OpenClawConfig, prompt string) (*OpenClawResponse, error)

func defaultOpenClawTaskSender(cfg *OpenClawConfig, prompt string) (*OpenClawResponse, error) {
	client := NewOpenClawClient(cfg, nil)
	return client.SendTaskWithoutSentinel(prompt)
}

func runSentinelOneClickModeWithSender(configPath, action, prompt string, out io.Writer, sender openClawTaskSender) error {
	action = strings.TrimSpace(action)
	prompt = strings.TrimSpace(prompt)

	if action == "" {
		return fmt.Errorf("--sentinel-oneclick-action is required in one-click mode")
	}
	if prompt == "" {
		return fmt.Errorf("--sentinel-oneclick-prompt is required in one-click mode")
	}

	cfg, err := loadSentinelOneClickConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load one-click config: %w", err)
	}
	if cfg.OpenClaw == nil || !cfg.OpenClaw.Enabled {
		return fmt.Errorf("openclaw.enabled=true is required in config for one-click mode")
	}
	if sender == nil {
		return fmt.Errorf("openclaw sender is not configured")
	}

	guard := NewSentinelGuard(resolveSentinelConfig(cfg.Sentinel))
	if guard == nil {
		return fmt.Errorf("sentinel guard is not configured")
	}

	eval, rec, err := guard.Enforce(action, prompt)
	if err != nil {
		return fmt.Errorf("sentinel enforce failed: %w", err)
	}

	if err := verifyRustCLIHash(guard, rec); err != nil {
		return err
	}

	result := SentinelOneClickOutput{
		Action:              action,
		Prompt:              prompt,
		Score:               eval.Score,
		Tags:                eval.Tags,
		Decision:            rec.Decision,
		Reason:              eval.Reason,
		RecordHash:          rec.RecordHash,
		AuditLogPath:        guard.cfg.AuditLogPath,
		RustCLIHashVerified: true,
		TxDigest:            rec.TxDigest,
		AnchorError:         rec.AnchorError,
	}

	if rec.TxDigest != "" {
		result.OnchainQueryCmd = fmt.Sprintf("sui client tx-block %s --json", rec.TxDigest)
		result.OnchainExplorerURL = buildSuiExplorerURL(cfg.SuiRPCURL, rec.TxDigest)
	}

	if eval.ShouldBlock {
		result.OpenClawSubmitted = false
		result.OpenClawStatus = "blocked_by_policy"
		return encodeSentinelOutput(out, result)
	}

	resp, err := sender(cfg.OpenClaw, prompt)
	if err != nil {
		return fmt.Errorf("openclaw dispatch failed: %w", err)
	}

	result.OpenClawSubmitted = true
	result.OpenClawStatus = resp.Status
	result.OpenClawMessage = resp.Message
	result.OpenClawTaskID = resp.TaskID
	return encodeSentinelOutput(out, result)
}

func verifyRustCLIHash(guard *SentinelGuard, rec *AuditRecord) error {
	if guard == nil || rec == nil {
		return fmt.Errorf("sentinel audit record is not available")
	}

	hashOut, err := guard.hashViaRust(rec)
	if err != nil {
		return fmt.Errorf("rustcli hash verification failed: %w (build rustcli first: cargo build --release in rustcli/)", err)
	}
	if hashOut.RecordHash != rec.RecordHash {
		return fmt.Errorf("rustcli hash mismatch: expected %s, got %s", rec.RecordHash, hashOut.RecordHash)
	}
	return nil
}

func buildSuiExplorerURL(rpcURL, txDigest string) string {
	if txDigest == "" {
		return ""
	}

	lower := strings.ToLower(rpcURL)
	network := "testnet"
	switch {
	case strings.Contains(lower, "mainnet"):
		network = "mainnet"
	case strings.Contains(lower, "devnet"):
		network = "devnet"
	case strings.Contains(lower, "localhost"), strings.Contains(lower, "127.0.0.1"):
		return ""
	}

	return fmt.Sprintf("https://suiexplorer.com/txblock/%s?network=%s", txDigest, network)
}

func encodeSentinelOutput(out io.Writer, data interface{}) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

func writeBenchmarkReport(path string, report *BenchmarkReport) error {
	if report == nil {
		return fmt.Errorf("benchmark report is nil")
	}
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// runSentinelProxyMode starts the Sentinel in-path proxy HTTP server.
func runSentinelProxyMode(configPath, listenAddr, walrusURL string) {
	log.Println("=== Sentinel In-Path Proxy ===")

	cfg, err := loadSentinelOneClickConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	guard := NewSentinelGuard(resolveSentinelConfig(cfg.Sentinel))
	if guard == nil {
		log.Fatalf("Sentinel guard is not configured")
	}
	log.Printf("  Anchor: enabled=%v package=%s registry=%s", guard.cfg.AnchorEnabled, guard.cfg.AnchorPackage, guard.cfg.AnchorRegistry)

	var oc *OpenClawClient
	if cfg.OpenClaw != nil && cfg.OpenClaw.Enabled {
		oc = NewOpenClawClient(cfg.OpenClaw, guard)
		log.Printf("  OpenClaw: enabled (%s)", cfg.OpenClaw.ServerURL)
	}

	gwCfg := &SentinelGatewayConfig{
		ApprovalTimeout:     5 * time.Minute,
		ProofBatchSize:      10,
		WalrusPublisherURL:  walrusURL,
		KillSwitchThreshold: 3,
		ExecuteTokenTTL:     30 * time.Second,
	}

	gateway := NewSentinelGateway(guard, oc, gwCfg)

	mux := http.NewServeMux()
	gateway.RegisterRoutes(mux)

	log.Printf("  Listen: %s", listenAddr)
	log.Println("  Endpoints:")
	log.Println("    POST /sentinel/gate           - Evaluate agent action")
	log.Println("    POST /sentinel/approval/start  - Start approval challenge")
	log.Println("    POST /sentinel/approval/confirm - Approve/reject challenge")
	log.Println("    POST /sentinel/proxy/execute    - Execute with one-time token")
	log.Println("    GET  /sentinel/proof/latest     - Latest proof chain entry")
	log.Println("    GET  /sentinel/status           - System status")
	log.Println("    POST /sentinel/kill-switch/arm  - Arm kill switch")
	log.Println("    POST /sentinel/kill-switch/disarm - Disarm kill switch")
	log.Println("    GET  /health                    - Health check")
	log.Println()

	srv := &http.Server{
		Addr:    listenAddr,
		Handler: mux,
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("\nShutting down Sentinel proxy...")
		srv.Close()
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Proxy server failed: %v", err)
	}
}
