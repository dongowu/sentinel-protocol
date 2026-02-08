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

// Enhanced Config with activity monitoring settings
type EnhancedConfig struct {
	VaultID           string        `json:"vault_id"`
	OwnerAddress      string        `json:"owner_address"`
	HeartbeatInterval time.Duration `json:"heartbeat_interval"`
	SuiRPCURL         string        `json:"sui_rpc_url"`
	PackageID         string        `json:"package_id"`
	PrivateKey        string        `json:"private_key,omitempty"` // Optional, for SDK signing

	// Activity monitoring settings
	ActivityCheckInterval time.Duration `json:"activity_check_interval"` // How often to check activity (default: 1 minute)
	InactivityThreshold   time.Duration `json:"inactivity_threshold"`    // When to stop heartbeats (default: 24 hours)
	EmergencyThreshold    time.Duration `json:"emergency_threshold"`     // When to trigger emergency (default: 72 hours)

	// Heartbeat strategy
	SmartHeartbeat bool `json:"smart_heartbeat"` // Only send heartbeat when active (default: true)

	// OpenClaw integration
	OpenClaw *OpenClawConfig `json:"openclaw,omitempty"`
	Sentinel *SentinelConfig `json:"sentinel,omitempty"`
}

// DaemonState tracks the daemon's operational state
type DaemonState struct {
	activityMonitor *ActivityMonitor
	alertSystem     *AlertSystem
	openClawClient  *OpenClawClient
	sentinelGuard   *SentinelGuard
	suiClient       interface{} // Placeholder, using CLI mode
	config          *EnhancedConfig
	lastHeartbeat   time.Time
	emergencyMode   bool
	alertTriggered  bool
}

func mainEnhanced() {
	// Command-line flags
	configPath := flag.String("config", "config.json", "Path to configuration file")
	createVault := flag.Bool("create", false, "Create a new vault")
	filePath := flag.String("file", "", "File to encrypt (required for --create)")
	beneficiary := flag.String("beneficiary", "", "Beneficiary address (required for --create)")
	walrusURL := flag.String("walrus", "https://publisher.walrus-testnet.walrus.space", "Walrus publisher URL")
	epochs := flag.Int("epochs", 5, "Number of epochs to store on Walrus")
	useCLI := flag.Bool("use-cli", false, "Use Sui CLI instead of SDK (fallback mode)")
	flag.Parse()

	if *createVault {
		handleCreateVault(*filePath, *beneficiary, *walrusURL, *epochs)
		return
	}

	// Load configuration
	config, err := loadEnhancedConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Validate configuration
	if err := validateConfig(config); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Start the enhanced daemon
	if *useCLI {
		startCLIDaemon(config)
	} else {
		startEnhancedDaemon(config)
	}
}

// startEnhancedDaemon runs the daemon with activity monitoring and smart heartbeats
func startEnhancedDaemon(config *EnhancedConfig) {
	log.Println("=== Sentinel Protocol Enhanced Daemon ===")
	log.Printf("Vault ID: %s", config.VaultID)
	log.Printf("Owner: %s", config.OwnerAddress)
	log.Printf("Smart Heartbeat: %v", config.SmartHeartbeat)
	log.Printf("Activity Check: %v", config.ActivityCheckInterval)
	log.Printf("Inactivity Threshold: %v", config.InactivityThreshold)
	log.Printf("Emergency Threshold: %v", config.EmergencyThreshold)
	log.Println()

	// Initialize activity monitor
	activityMonitor := NewActivityMonitor(config.ActivityCheckInterval)
	go activityMonitor.Start()

	// Initialize alert system
	alertSystem := NewAlertSystem(config)

	// Initialize Sentinel guard (OpenClaw x Sui policy gate)
	var sentinelGuard *SentinelGuard
	if config.Sentinel != nil && config.Sentinel.Enabled {
		sentinelGuard = NewSentinelGuard(config.Sentinel)
		log.Printf("✓ Sentinel enabled (threshold=%d, audit=%s)", config.Sentinel.RiskThreshold, config.Sentinel.AuditLogPath)
	}

	// Initialize OpenClaw client
	var openClawClient *OpenClawClient
	if config.OpenClaw != nil && config.OpenClaw.Enabled {
		openClawClient = NewOpenClawClient(config.OpenClaw, sentinelGuard)
		if err := openClawClient.TestConnection(); err != nil {
			log.Printf("⚠️  OpenClaw connection test failed: %v", err)
			log.Println("   Continuing without OpenClaw integration")
		} else {
			log.Println("✓ OpenClaw connected successfully")
		}
	}

	// Create daemon state (using CLI mode for simplicity)
	state := &DaemonState{
		activityMonitor: activityMonitor,
		alertSystem:     alertSystem,
		openClawClient:  openClawClient,
		sentinelGuard:   sentinelGuard,
		suiClient:       nil, // Use CLI mode
		config:          config,
		lastHeartbeat:   time.Now(),
		emergencyMode:   false,
		alertTriggered:  false,
	}

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Main monitoring loop
	ticker := time.NewTicker(config.ActivityCheckInterval)
	defer ticker.Stop()

	log.Println("✓ Daemon started successfully")
	log.Println("  Press Ctrl+C to stop")
	log.Println()

	for {
		select {
		case <-ticker.C:
			handleMonitoringCycle(state)

		case <-sigChan:
			log.Println("\n🛑 Shutting down daemon...")
			activityMonitor.Stop()
			return
		}
	}
}

// handleMonitoringCycle processes one monitoring cycle
func handleMonitoringCycle(state *DaemonState) {
	inactiveDuration := state.activityMonitor.GetInactiveDuration()

	// Log current status
	log.Printf("[%s] Status Check:", time.Now().Format("2006-01-02 15:04:05"))
	log.Printf("  Inactive for: %v", inactiveDuration.Round(time.Second))
	log.Printf("  Last heartbeat: %v ago", time.Since(state.lastHeartbeat).Round(time.Second))

	// Check for emergency condition (72+ hours inactive)
	if inactiveDuration > state.config.EmergencyThreshold {
		if !state.emergencyMode {
			log.Println("\n⚠️  EMERGENCY THRESHOLD EXCEEDED!")
			log.Printf("  System inactive for %v (threshold: %v)",
				inactiveDuration.Round(time.Hour),
				state.config.EmergencyThreshold)
			log.Println("  Will execution can now be triggered by anyone")

			// Trigger OpenClaw last words action
			if state.openClawClient != nil {
				log.Println("\n🤖 Triggering OpenClaw: Last Words...")
				if err := state.openClawClient.TriggerLastWords(state.config.VaultID, "BENEFICIARY_ADDRESS"); err != nil {
					log.Printf("  ⚠️  OpenClaw last words failed: %v", err)
				}
			}

			state.emergencyMode = true
		}
		return
	}

	// Check if we should trigger an alert (24-72 hours inactive)
	if state.alertSystem.ShouldTriggerAlert(inactiveDuration) && !state.alertTriggered {
		log.Println("\n🚨 TRIGGERING USER ALERT!")
		state.alertTriggered = true

		// Trigger OpenClaw wake-up action first (most dramatic!)
		if state.openClawClient != nil {
			go func() {
				if err := state.openClawClient.TriggerWakeUp(); err != nil {
					log.Printf("  ⚠️  OpenClaw wake-up failed: %v", err)
				}
			}()
		}

		// Show GUI alert
		go func() {
			if err := state.alertSystem.TriggerAlert(inactiveDuration); err != nil {
				log.Printf("  ⚠️  Alert failed: %v", err)
			}
		}()

		// Also open browser alert for dramatic effect
		go func() {
			if err := state.alertSystem.OpenBrowserAlert(inactiveDuration); err != nil {
				log.Printf("  ⚠️  Browser alert failed: %v", err)
			}
		}()

		// Wait for user response
		select {
		case <-state.alertSystem.userResponded:
			log.Println("\n✓ User responded to alert!")
			log.Println("  Sending immediate heartbeat...")

			// Send immediate heartbeat
			if err := sendHeartbeatCLI(state.config); err != nil {
				log.Printf("  ❌ Emergency heartbeat failed: %v", err)
			} else {
				state.lastHeartbeat = time.Now()
				state.alertTriggered = false
				state.emergencyMode = false
				log.Println("  ✓ Emergency heartbeat sent successfully!")
			}
		case <-time.After(1 * time.Minute):
			// Continue monitoring
		}
	}

	// Reset alert flag if user becomes active again
	if inactiveDuration < state.config.InactivityThreshold {
		state.alertTriggered = false
	}

	// Check if we should send a heartbeat
	shouldSendHeartbeat := false

	if state.config.SmartHeartbeat {
		// Smart mode: Only send heartbeat if user is active
		if inactiveDuration < state.config.InactivityThreshold {
			// User is active, check if it's time for a heartbeat
			timeSinceLastHeartbeat := time.Since(state.lastHeartbeat)
			if timeSinceLastHeartbeat >= state.config.HeartbeatInterval {
				shouldSendHeartbeat = true
				log.Printf("  ✓ User active, sending scheduled heartbeat")
			}
		} else {
			log.Printf("  ⏸  User inactive (>%v), skipping heartbeat", state.config.InactivityThreshold)
		}
	} else {
		// Traditional mode: Always send heartbeat on schedule
		timeSinceLastHeartbeat := time.Since(state.lastHeartbeat)
		if timeSinceLastHeartbeat >= state.config.HeartbeatInterval {
			shouldSendHeartbeat = true
			log.Printf("  ⏰ Scheduled heartbeat time")
		}
	}

	// Send heartbeat if needed
	if shouldSendHeartbeat {
		if err := sendHeartbeatCLI(state.config); err != nil {
			log.Printf("  ❌ Heartbeat failed: %v", err)
			log.Println("  Will retry on next cycle")
		} else {
			state.lastHeartbeat = time.Now()
			state.emergencyMode = false  // Reset emergency mode on successful heartbeat
			state.alertTriggered = false // Reset alert flag
		}
	}

	log.Println()
}

// sendHeartbeatCLI sends a heartbeat using Sui CLI
func sendHeartbeatCLI(config *EnhancedConfig) error {
	log.Printf("[%s] 💓 Sending heartbeat...", time.Now().Format("2006-01-02 15:04:05"))

	cmd := exec.Command(
		"sui", "client", "call",
		"--package", config.PackageID,
		"--module", "lazarus_protocol",
		"--function", "keep_alive",
		"--args", config.VaultID, "0x6", // 0x6 is the Clock object
		"--gas-budget", "10000000",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("transaction failed: %v\nOutput: %s", err, string(output))
	}

	log.Printf("✓ Heartbeat sent successfully")
	return nil
}

// startCLIDaemon runs the daemon using Sui CLI (fallback mode)
func startCLIDaemon(config *EnhancedConfig) {
	log.Println("=== Sentinel Protocol Daemon (CLI Mode) ===")
	log.Println("⚠️  Running in CLI fallback mode")
	log.Println()

	// Convert to old config format
	oldConfig := &Config{
		VaultID:           config.VaultID,
		OwnerAddress:      config.OwnerAddress,
		HeartbeatInterval: config.HeartbeatInterval,
		SuiRPCURL:         config.SuiRPCURL,
		PackageID:         config.PackageID,
	}

	startHeartbeatDaemon(oldConfig)
}

// loadEnhancedConfig loads the enhanced configuration
func loadEnhancedConfig(path string) (*EnhancedConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config EnhancedConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// Set defaults
	if config.ActivityCheckInterval == 0 {
		config.ActivityCheckInterval = 1 * time.Minute
	}
	if config.InactivityThreshold == 0 {
		config.InactivityThreshold = 24 * time.Hour
	}
	if config.EmergencyThreshold == 0 {
		config.EmergencyThreshold = 72 * time.Hour
	}
	if config.HeartbeatInterval == 0 {
		config.HeartbeatInterval = 7 * 24 * time.Hour
	}
	// Smart heartbeat enabled by default
	config.SmartHeartbeat = true

	return &config, nil
}

// validateConfig validates the configuration
func validateConfig(config *EnhancedConfig) error {
	if config.VaultID == "" {
		return fmt.Errorf("vault_id is required")
	}
	if config.OwnerAddress == "" {
		return fmt.Errorf("owner_address is required")
	}
	if config.PackageID == "" {
		return fmt.Errorf("package_id is required")
	}
	if config.EmergencyThreshold <= config.InactivityThreshold {
		return fmt.Errorf("emergency_threshold must be greater than inactivity_threshold")
	}
	return nil
}

// TriggerEmergency manually triggers emergency mode (for testing)
func TriggerEmergency(config *EnhancedConfig) error {
	log.Println("🚨 TRIGGERING EMERGENCY MODE")
	log.Println("This will execute the will on the blockchain")
	log.Println()

	// Call the Rust CLI to re-encrypt and store emergency data
	log.Println("[1/2] Preparing emergency data...")

	// In a real scenario, you might want to encrypt additional emergency instructions
	// For now, we just log the action

	log.Println("[2/2] Executing will on blockchain...")

	// Call execute_will on the smart contract
	cmd := exec.Command(
		"sui", "client", "call",
		"--package", config.PackageID,
		"--module", "lazarus_protocol",
		"--function", "execute_will",
		"--args", config.VaultID, "0x6", // 0x6 is the Clock object
		"--gas-budget", "10000000",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("emergency execution failed: %v\nOutput: %s", err, string(output))
	}

	log.Println("✓ Emergency executed successfully")
	log.Println("  Beneficiary can now access the encrypted data")

	return nil
}
