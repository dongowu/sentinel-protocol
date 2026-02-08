package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
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
	flag.Parse()

	if *createVault {
		handleCreateVault(*filePath, *beneficiary, *walrusURL, *epochs)
		return
	}

	if *sentinelBenchmark != "" {
		enhancedConfig, err := loadEnhancedConfig(*configPath)
		if err != nil {
			log.Fatalf("Failed to load enhanced config for benchmark: %v", err)
		}

		guard := NewSentinelGuard(enhancedConfig.Sentinel)
		if guard == nil {
			guard = NewSentinelGuard(&SentinelConfig{Enabled: true, RiskThreshold: 70, AuditLogPath: "./audit/sentinel-audit.jsonl"})
		}

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
	log.Println("=== Lazarus Protocol Heartbeat Daemon ===")
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
