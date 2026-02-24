package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// Config holds the daemon configuration
type Config struct {
	VaultID           string        `json:"vault_id"`
	OwnerAddress      string        `json:"owner_address"`
	HeartbeatInterval time.Duration `json:"heartbeat_interval"`
	SuiRPCURL         string        `json:"sui_rpc_url"`
	PackageID         string        `json:"package_id"`
}

// EncryptionResult represents the output from the Rust CLI tool
type EncryptionResult struct {
	BlobID        string `json:"blob_id"`
	DecryptionKey string `json:"decryption_key"`
	Checksum      string `json:"checksum"`
	OriginalSize  int    `json:"original_size"`
	EncryptedSize int    `json:"encrypted_size"`
}

type SentinelEvalOutput struct {
	Action       string   `json:"action"`
	Prompt       string   `json:"prompt"`
	Score        int      `json:"score"`
	Tags         []string `json:"tags"`
	Decision     string   `json:"decision"`
	Reason       string   `json:"reason"`
	RecordHash   string   `json:"record_hash"`
	AuditLogPath string   `json:"audit_log_path"`
}

type SentinelOneClickOutput struct {
	Action              string   `json:"action"`
	Prompt              string   `json:"prompt"`
	Score               int      `json:"score"`
	Tags                []string `json:"tags"`
	Decision            string   `json:"decision"`
	Reason              string   `json:"reason"`
	RecordHash          string   `json:"record_hash"`
	AuditLogPath        string   `json:"audit_log_path"`
	RustCLIHashVerified bool     `json:"rustcli_hash_verified"`
	TxDigest            string   `json:"tx_digest,omitempty"`
	AnchorError         string   `json:"anchor_error,omitempty"`
	OnchainQueryCmd     string   `json:"onchain_query_cmd,omitempty"`
	OnchainExplorerURL  string   `json:"onchain_explorer_url,omitempty"`
	OpenClawSubmitted   bool     `json:"openclaw_submitted"`
	OpenClawStatus      string   `json:"openclaw_status,omitempty"`
	OpenClawMessage     string   `json:"openclaw_message,omitempty"`
	OpenClawTaskID      string   `json:"openclaw_task_id,omitempty"`
}

type SentinelOneClickConfig struct {
	SuiRPCURL string          `json:"sui_rpc_url"`
	OpenClaw  *OpenClawConfig `json:"openclaw,omitempty"`
	Sentinel  *SentinelConfig `json:"sentinel,omitempty"`
}

func main() {
	// Command-line flags
	configPath := flag.String("config", "config.json", "Path to configuration file")
	createVault := flag.Bool("create", false, "Create a new vault")
	filePath := flag.String("file", "", "File to encrypt (required for --create)")
	beneficiary := flag.String("beneficiary", "", "Beneficiary address (required for --create)")
	walrusURL := flag.String("walrus", "https://publisher.walrus-testnet.walrus.space", "Walrus publisher URL")
	epochs := flag.Int("epochs", 5, "Number of epochs to store on Walrus")
	enhanced := flag.Bool("enhanced", false, "Use enhanced mode with activity monitoring")
	useCLI := flag.Bool("use-cli", true, "Use Sui CLI instead of SDK (default: true)")
	sentinelBenchmark := flag.String("sentinel-benchmark", "", "Path to Sentinel benchmark JSON file")
	sentinelEvalAction := flag.String("sentinel-eval-action", "", "Action to evaluate with Sentinel (requires --sentinel-eval-prompt)")
	sentinelEvalPrompt := flag.String("sentinel-eval-prompt", "", "Prompt to evaluate with Sentinel (requires --sentinel-eval-action)")
	sentinelOneClickAction := flag.String("sentinel-oneclick-action", "", "One-click action sent to OpenClaw with Sentinel audit/enforcement")
	sentinelOneClickPrompt := flag.String("sentinel-oneclick-prompt", "", "One-click prompt sent to OpenClaw (requires --sentinel-oneclick-action)")
	sentinelProxy := flag.Bool("sentinel-proxy", false, "Start Sentinel in-path proxy HTTP server")
	sentinelProxyAddr := flag.String("sentinel-proxy-addr", "127.0.0.1:18080", "Listen address for the Sentinel proxy server")
	flag.Parse()

	if *createVault {
		handleCreateVault(*filePath, *beneficiary, *walrusURL, *epochs)
		return
	}

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

		if err := RunSentinelBenchmark(*sentinelBenchmark, guard); err != nil {
			log.Fatalf("Sentinel benchmark failed: %v", err)
		}
		return
	}

	// Check if enhanced mode is requested
	if *enhanced {
		// Load enhanced configuration
		enhancedConfig, err := loadEnhancedConfig(*configPath)
		if err != nil {
			log.Fatalf("Failed to load enhanced config: %v", err)
		}

		if *useCLI {
			startCLIDaemon(enhancedConfig)
		} else {
			startEnhancedDaemon(enhancedConfig)
		}
		return
	}

	// Standard mode (backward compatible)
	config, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	startHeartbeatDaemon(config)
}

// handleCreateVault creates a new vault by encrypting a file and deploying to Sui
func handleCreateVault(filePath, beneficiary, walrusURL string, epochs int) {
	if filePath == "" || beneficiary == "" {
		log.Fatal("Both --file and --beneficiary are required when using --create")
	}

	log.Println("=== Creating New Lazarus Vault ===")

	// Step 1: Encrypt and upload to Walrus
	log.Println("[1/3] Encrypting file and uploading to Walrus...")
	encResult, err := encryptAndStore(filePath, walrusURL, epochs)
	if err != nil {
		log.Fatalf("Encryption failed: %v", err)
	}

	log.Printf("✓ File encrypted successfully")
	log.Printf("  Blob ID: %s", encResult.BlobID)
	log.Printf("  Decryption Key: %s", encResult.DecryptionKey)
	log.Printf("  Checksum: %s", encResult.Checksum)

	// Step 2: Create vault on Sui
	log.Println("\n[2/3] Creating vault on Sui blockchain...")
	vaultID, err := createSuiVault(beneficiary, encResult.BlobID)
	if err != nil {
		log.Fatalf("Failed to create vault: %v", err)
	}

	log.Printf("✓ Vault created successfully")
	log.Printf("  Vault ID: %s", vaultID)

	// Step 3: Save configuration
	log.Println("\n[3/3] Saving configuration...")
	config := Config{
		VaultID:           vaultID,
		OwnerAddress:      "YOUR_ADDRESS_HERE", // User should update this
		HeartbeatInterval: 7 * 24 * time.Hour,  // 7 days
		SuiRPCURL:         "https://fullnode.testnet.sui.io:443",
		PackageID:         "YOUR_PACKAGE_ID_HERE", // User should update this
	}

	if err := saveConfig("config.json", config); err != nil {
		log.Fatalf("Failed to save config: %v", err)
	}

	log.Println("\n✓ Vault creation complete!")
	log.Println("\n⚠️  CRITICAL: Save the decryption key securely!")
	log.Printf("   Decryption Key: %s\n", encResult.DecryptionKey)
	log.Println("\n📝 Next steps:")
	log.Println("   1. Update config.json with your owner address and package ID")
	log.Println("   2. Run the daemon: ./goserver --config config.json")
}

// encryptAndStore calls the Rust CLI tool to encrypt and upload a file
func encryptAndStore(filePath, walrusURL string, epochs int) (*EncryptionResult, error) {
	cmd := exec.Command(
		"../rustcli/target/release/lazarus-vault",
		"encrypt-and-store",
		"--file", filePath,
		"--publisher", walrusURL,
		"--epochs", fmt.Sprintf("%d", epochs),
	)

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("rust tool failed: %s", string(exitErr.Stderr))
		}
		return nil, err
	}

	var result EncryptionResult
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse output: %v", err)
	}

	return &result, nil
}

// createSuiVault creates a vault on the Sui blockchain
func createSuiVault(beneficiary, blobID string) (string, error) {
	// Note: This is a placeholder. In production, you would:
	// 1. Use the Sui Go SDK to interact with the blockchain
	// 2. Call the create_vault function on the smart contract
	// 3. Parse the transaction result to get the vault ID

	cmd := exec.Command(
		"sui", "client", "call",
		"--package", "YOUR_PACKAGE_ID",
		"--module", "lazarus_protocol",
		"--function", "create_vault",
		"--args", beneficiary, blobID,
		"--gas-budget", "10000000",
		"--json",
	)

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("sui call failed: %v", err)
	}

	// Parse the transaction result to extract vault ID
	// This is simplified - in production you'd parse the JSON properly
	log.Printf("Transaction output: %s", string(output))

	return "VAULT_ID_FROM_TRANSACTION", nil
}

// startHeartbeatDaemon runs the heartbeat loop
func startHeartbeatDaemon(config *Config) {
	log.Println("=== Sentinel Protocol Heartbeat Daemon ===")
	log.Printf("Vault ID: %s", config.VaultID)
	log.Printf("Owner: %s", config.OwnerAddress)
	log.Printf("Heartbeat Interval: %v", config.HeartbeatInterval)
	log.Println("\nDaemon started. Press Ctrl+C to stop.")

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Create ticker for heartbeat
	ticker := time.NewTicker(config.HeartbeatInterval)
	defer ticker.Stop()

	// Send initial heartbeat
	if err := sendHeartbeat(config); err != nil {
		log.Printf("⚠️  Initial heartbeat failed: %v", err)
	}

	// Main loop
	for {
		select {
		case <-ticker.C:
			if err := sendHeartbeat(config); err != nil {
				log.Printf("⚠️  Heartbeat failed: %v", err)
			}
		case <-sigChan:
			log.Println("\n🛑 Shutting down daemon...")
			return
		}
	}
}

// sendHeartbeat sends a heartbeat transaction to the Sui blockchain
func sendHeartbeat(config *Config) error {
	log.Printf("[%s] Sending heartbeat...", time.Now().Format("2006-01-02 15:04:05"))

	// Note: This is a placeholder. In production, you would:
	// 1. Use the Sui Go SDK to interact with the blockchain
	// 2. Call the keep_alive function on the smart contract
	// 3. Handle transaction errors and retries

	cmd := exec.Command(
		"sui", "client", "call",
		"--package", config.PackageID,
		"--module", "lazarus_protocol",
		"--function", "keep_alive",
		"--args", config.VaultID,
		"--gas-budget", "10000000",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("transaction failed: %v\nOutput: %s", err, string(output))
	}

	log.Printf("✓ Heartbeat sent successfully")
	return nil
}

// loadConfig loads configuration from a JSON file
func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// saveConfig saves configuration to a JSON file
func saveConfig(path string, config Config) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func loadSentinelConfigOnly(path string) (*SentinelConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var raw struct {
		Sentinel *SentinelConfig `json:"sentinel"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	return raw.Sentinel, nil
}

func loadSentinelOneClickConfig(path string) (*SentinelOneClickConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg SentinelOneClickConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func defaultSentinelConfig() *SentinelConfig {
	return &SentinelConfig{
		Enabled:       true,
		RiskThreshold: 70,
		AuditLogPath:  "./audit/sentinel-audit.jsonl",
	}
}

func resolveSentinelConfig(cfg *SentinelConfig) *SentinelConfig {
	if cfg == nil {
		return defaultSentinelConfig()
	}
	return cfg
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

// runSentinelProxyMode starts the Sentinel in-path proxy HTTP server.
func runSentinelProxyMode(configPath, listenAddr, walrusURL string) {
	log.Println("=== Sentinel In-Path Proxy ===")

	// Load config
	cfg, err := loadSentinelOneClickConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	guard := NewSentinelGuard(resolveSentinelConfig(cfg.Sentinel))
	if guard == nil {
		log.Fatalf("Sentinel guard is not configured")
	}

	// Optional OpenClaw client
	var oc *OpenClawClient
	if cfg.OpenClaw != nil && cfg.OpenClaw.Enabled {
		oc = NewOpenClawClient(cfg.OpenClaw, guard)
		log.Printf("  OpenClaw: enabled (%s)", cfg.OpenClaw.ServerURL)
	}

	// Build gateway config
	gwCfg := &SentinelGatewayConfig{
		ApprovalTimeout:    5 * time.Minute,
		ProofBatchSize:     10,
		WalrusPublisherURL: walrusURL,
		KillSwitchThreshold: 3,
		ExecuteTokenTTL:    30 * time.Second,
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

	// Graceful shutdown
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
