package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// OpenClawConfig holds OpenClaw integration settings
type OpenClawConfig struct {
	Enabled    bool   `json:"enabled"`
	ServerURL  string `json:"server_url"`
	WakeUpTask string `json:"wake_up_task"`
	LastWords  string `json:"last_words"`
}

// OpenClawRequest represents a request to OpenClaw
type OpenClawRequest struct {
	Task string `json:"task"`
}

// OpenClawResponse represents a response from OpenClaw
type OpenClawResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	TaskID  string `json:"task_id,omitempty"`
}

// OpenClawClient handles OpenClaw integration
type OpenClawClient struct {
	config   *OpenClawConfig
	sentinel *SentinelGuard
}

// NewOpenClawClient creates a new OpenClaw client
func NewOpenClawClient(config *OpenClawConfig, sentinel *SentinelGuard) *OpenClawClient {
	return &OpenClawClient{
		config:   config,
		sentinel: sentinel,
	}
}

// TriggerWakeUp triggers the wake-up action (browser alert + sound)
func (oc *OpenClawClient) TriggerWakeUp() error {
	if !oc.config.Enabled {
		log.Println("⚠️  OpenClaw is disabled, skipping wake-up action")
		return nil
	}

	log.Println("\n🤖 TRIGGERING OPENCLAW: WAKE UP ACTION")
	log.Println(strings.Repeat("=", 60))

	prompt := oc.config.WakeUpTask
	if prompt == "" {
		prompt = `Open the default browser and navigate to a page that plays an alarm sound.
		Display a large warning message: "⚠️ LAZARUS PROTOCOL WARNING ⚠️ - Confirm you are alive!"
		Play an alarm sound repeatedly until the user interacts with the page.`
	}

	return oc.sendTask("WAKE_UP", prompt)
}

// TriggerLastWords triggers the last words action (Twitter post)
func (oc *OpenClawClient) TriggerLastWords(vaultID, beneficiary string) error {
	if !oc.config.Enabled {
		log.Println("⚠️  OpenClaw is disabled, skipping last words action")
		return nil
	}

	log.Println("\n🤖 TRIGGERING OPENCLAW: LAST WORDS")
	log.Println(strings.Repeat("=", 60))

	prompt := oc.config.LastWords
	if prompt == "" {
		prompt = fmt.Sprintf(`Open Twitter (X.com) and draft a tweet (DO NOT POST IT, just draft):

"This is an automated message from Sui-Lazarus Protocol.

My owner has been inactive for 72 hours. The digital legacy protocol has been triggered on Sui Network.

Vault ID: %s
Beneficiary: %s

Goodbye, world. 🕯️

#Sui #LazarusProtocol #DigitalLegacy"

IMPORTANT: Only draft the tweet, do not post it. This is for demonstration purposes only.`, vaultID, beneficiary)
	}

	return oc.sendTask("LAST_WORDS", prompt)
}

// TriggerCustomAction triggers a custom OpenClaw action
func (oc *OpenClawClient) TriggerCustomAction(actionName, prompt string) error {
	if !oc.config.Enabled {
		log.Println("⚠️  OpenClaw is disabled, skipping custom action")
		return nil
	}

	log.Printf("\n🤖 TRIGGERING OPENCLAW: %s", actionName)
	log.Println(strings.Repeat("=", 60))

	return oc.sendTask(actionName, prompt)
}

// sendTask sends a task to OpenClaw
func (oc *OpenClawClient) sendTask(actionType, prompt string) error {
	log.Printf("   Action: %s", actionType)
	log.Printf("   Server: %s", oc.config.ServerURL)
	log.Println()

	if oc.sentinel != nil {
		eval, rec, err := oc.sentinel.Enforce(actionType, prompt)
		if err != nil {
			return fmt.Errorf("sentinel enforcement failed: %w", err)
		}
		log.Printf("   Sentinel score: %d (%v)", eval.Score, eval.Tags)
		if rec.TxDigest != "" {
			log.Printf("   Sui anchor tx: %s", rec.TxDigest)
		}
		if eval.ShouldBlock {
			return fmt.Errorf("blocked by sentinel policy: %s", eval.Reason)
		}
	}

	// Create request payload
	payload := OpenClawRequest{
		Task: prompt,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Send HTTP POST request
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Post(oc.config.ServerURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("OpenClaw connection failed: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var response OpenClawResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("OpenClaw returned error: %s - %s", response.Status, response.Message)
	}

	log.Println("✓ OpenClaw accepted the task")
	if response.TaskID != "" {
		log.Printf("  Task ID: %s", response.TaskID)
	}
	log.Println("  Browser should open shortly...")
	log.Println()

	return nil
}

// TestConnection tests the connection to OpenClaw
func (oc *OpenClawClient) TestConnection() error {
	if !oc.config.Enabled {
		return fmt.Errorf("OpenClaw is disabled")
	}

	log.Println("🔌 Testing OpenClaw connection...")

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(oc.config.ServerURL + "/health")
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	log.Println("✓ OpenClaw connection successful")
	return nil
}

// OpenClawAlertPage creates an HTML alert page for OpenClaw to display
func (oc *OpenClawClient) OpenClawAlertPage(inactiveDuration time.Duration, emergencyThreshold time.Duration) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
	<title>Lazarus Protocol - OpenClaw Alert</title>
	<style>
		body {
			font-family: 'Arial', sans-serif;
			background: linear-gradient(135deg, #ff0000 0%%, #8b0000 100%%);
			display: flex;
			justify-content: center;
			align-items: center;
			height: 100vh;
			margin: 0;
			animation: pulse 2s infinite;
		}
		@keyframes pulse {
			0%%, 100%% { opacity: 1; }
			50%% { opacity: 0.8; }
		}
		.alert-box {
			background: white;
			padding: 60px;
			border-radius: 30px;
			box-shadow: 0 30px 80px rgba(0,0,0,0.5);
			text-align: center;
			max-width: 600px;
			animation: shake 0.5s infinite;
		}
		@keyframes shake {
			0%%, 100%% { transform: translateX(0); }
			25%% { transform: translateX(-10px); }
			75%% { transform: translateX(10px); }
		}
		h1 {
			color: #ff0000;
			font-size: 48px;
			margin-bottom: 30px;
			text-shadow: 2px 2px 4px rgba(0,0,0,0.3);
		}
		.warning-icon {
			font-size: 120px;
			margin-bottom: 30px;
			animation: rotate 2s linear infinite;
		}
		@keyframes rotate {
			from { transform: rotate(0deg); }
			to { transform: rotate(360deg); }
		}
		.info {
			font-size: 24px;
			color: #333;
			margin: 30px 0;
			line-height: 1.8;
		}
		.countdown {
			font-size: 72px;
			color: #ff0000;
			font-weight: bold;
			margin: 40px 0;
			text-shadow: 3px 3px 6px rgba(0,0,0,0.3);
		}
		button {
			background: #00ff00;
			color: #000;
			border: none;
			padding: 30px 80px;
			font-size: 32px;
			font-weight: bold;
			border-radius: 60px;
			cursor: pointer;
			transition: all 0.3s;
			box-shadow: 0 15px 40px rgba(0, 255, 0, 0.4);
			animation: glow 1s infinite;
		}
		@keyframes glow {
			0%%, 100%% { box-shadow: 0 15px 40px rgba(0, 255, 0, 0.4); }
			50%% { box-shadow: 0 15px 60px rgba(0, 255, 0, 0.8); }
		}
		button:hover {
			background: #00cc00;
			transform: scale(1.1);
		}
	</style>
</head>
<body>
	<div class="alert-box">
		<div class="warning-icon">🚨</div>
		<h1>LAZARUS PROTOCOL</h1>
		<h1>CRITICAL WARNING</h1>
		<div class="info">
			System inactive for <strong>%v</strong><br>
			<br>
			<strong>IMMEDIATE ACTION REQUIRED</strong><br>
			Click the button below to confirm you are alive!
		</div>
		<div class="countdown" id="countdown">%v</div>
		<button onclick="confirmAlive()">I'M ALIVE! ✅</button>
	</div>
	<audio id="alarm" loop>
		<source src="data:audio/wav;base64,UklGRnoGAABXQVZFZm10IBAAAAABAAEAQB8AAEAfAAABAAgAZGF0YQoGAACBhYqFbF1fdJivrJBhNjVgodDbq2EcBj+a2/LDciUFLIHO8tiJNwgZaLvt559NEAxQp+PwtmMcBjiR1/LMeSwFJHfH8N2QQAoUXrTp66hVFApGn+DyvmwhBSuBzvLZiTYIGGS57OihUBELTKXh8bllHAU2jdXvzn0pBSh+zPDajzsKElyx6OyrWBUIQ5zd8sFuJAUuhM/z24k2CBhku+zooVARC0yl4fG5ZRwFNo3V7859KQUofsz" type="audio/wav">
	</audio>
	<script>
		// Auto-play alarm
		const audio = document.getElementById('alarm');
		audio.volume = 0.5;
		audio.play().catch(() => {
			// Fallback: use Web Audio API
			const audioContext = new (window.AudioContext || window.webkitAudioContext)();
			const oscillator = audioContext.createOscillator();
			const gainNode = audioContext.createGain();
			oscillator.connect(gainNode);
			gainNode.connect(audioContext.destination);
			oscillator.frequency.value = 800;
			gainNode.gain.value = 0.3;
			oscillator.start();
			setInterval(() => {
				oscillator.frequency.value = oscillator.frequency.value === 800 ? 1000 : 800;
			}, 500);
		});

		function confirmAlive() {
			audio.pause();
			alert('✓ Confirmed! Sending heartbeat to blockchain...');
			// In production, this would call the Go daemon's HTTP endpoint
			window.close();
		}

		// Countdown timer
		let remaining = %d; // seconds until emergency
		setInterval(() => {
			remaining--;
			const hours = Math.floor(remaining / 3600);
			const minutes = Math.floor((remaining %% 3600) / 60);
			const seconds = remaining %% 60;
			document.getElementById('countdown').textContent =
				hours + 'h ' + minutes + 'm ' + seconds + 's';
		}, 1000);

		// Flash title
		setInterval(() => {
			document.title = document.title === 'LAZARUS PROTOCOL - ALERT!' ? '🚨 RESPOND NOW! 🚨' : 'LAZARUS PROTOCOL - ALERT!';
		}, 1000);
	</script>
</body>
</html>`,
		inactiveDuration.Round(time.Hour),
		emergencyThreshold-inactiveDuration,
		int((emergencyThreshold - inactiveDuration).Seconds()),
	)
}
