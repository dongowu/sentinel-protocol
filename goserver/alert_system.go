package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// AlertSystem handles user alerts when inactivity is detected
type AlertSystem struct {
	config          *EnhancedConfig
	lastAlertTime   time.Time
	alertCooldown   time.Duration
	userResponded   chan bool
}

// NewAlertSystem creates a new alert system
func NewAlertSystem(config *EnhancedConfig) *AlertSystem {
	return &AlertSystem{
		config:        config,
		alertCooldown: 6 * time.Hour, // Don't spam alerts more than every 6 hours
		userResponded: make(chan bool, 1),
	}
}

// ShouldTriggerAlert checks if we should show an alert
func (as *AlertSystem) ShouldTriggerAlert(inactiveDuration time.Duration) bool {
	// Trigger alert if:
	// 1. User has been inactive for 24+ hours
	// 2. But not yet reached emergency threshold (72 hours)
	// 3. Cooldown period has passed since last alert
	if inactiveDuration < as.config.InactivityThreshold {
		return false
	}

	if inactiveDuration >= as.config.EmergencyThreshold {
		return false // Too late, already in emergency mode
	}

	// Check cooldown
	if time.Since(as.lastAlertTime) < as.alertCooldown {
		return false
	}

	return true
}

// TriggerAlert shows a GUI alert to the user
func (as *AlertSystem) TriggerAlert(inactiveDuration time.Duration) error {
	as.lastAlertTime = time.Now()

	log.Println("\n⚠️  LAZARUS PROTOCOL WARNING ⚠️")
	log.Printf("   System inactive for: %v", inactiveDuration.Round(time.Hour))
	log.Printf("   Emergency threshold: %v", as.config.EmergencyThreshold)
	log.Println("   Triggering user alert...")

	// Show platform-specific GUI alert
	go as.showGUIAlert(inactiveDuration)

	// Also play alert sound
	go as.playAlertSound()

	// Wait for user response (with timeout)
	select {
	case <-as.userResponded:
		log.Println("✓ User confirmed they are alive!")
		return nil
	case <-time.After(30 * time.Minute):
		log.Println("⏱  User did not respond to alert")
		return fmt.Errorf("user did not respond to alert")
	}
}

// showGUIAlert displays a platform-specific GUI alert
func (as *AlertSystem) showGUIAlert(inactiveDuration time.Duration) {
	message := fmt.Sprintf(
		"⚠️ LAZARUS PROTOCOL WARNING ⚠️\n\n"+
			"System inactive for: %v\n"+
			"Emergency threshold: %v\n\n"+
			"Click 'I'm Alive' to confirm you are still active.\n"+
			"Otherwise, your will may be executed in %v.",
		inactiveDuration.Round(time.Hour),
		as.config.EmergencyThreshold,
		as.config.EmergencyThreshold-inactiveDuration,
	)

	switch runtime.GOOS {
	case "windows":
		as.showWindowsAlert(message)
	case "darwin":
		as.showMacOSAlert(message)
	case "linux":
		as.showLinuxAlert(message)
	default:
		log.Printf("⚠️  GUI alerts not supported on %s", runtime.GOOS)
		as.showTerminalAlert(message)
	}
}

// showWindowsAlert shows a Windows message box
func (as *AlertSystem) showWindowsAlert(message string) {
	// Use PowerShell to show a GUI dialog
	script := fmt.Sprintf(`
Add-Type -AssemblyName System.Windows.Forms
$result = [System.Windows.Forms.MessageBox]::Show(
	"%s",
	"Lazarus Protocol Alert",
	[System.Windows.Forms.MessageBoxButtons]::YesNo,
	[System.Windows.Forms.MessageBoxIcon]::Warning
)
if ($result -eq [System.Windows.Forms.DialogResult]::Yes) {
	exit 0
} else {
	exit 1
}
`, message)

	cmd := exec.Command("powershell", "-Command", script)
	err := cmd.Run()
	if err == nil {
		// User clicked "Yes" (I'm Alive)
		as.userResponded <- true
	}
}

// showMacOSAlert shows a macOS notification
func (as *AlertSystem) showMacOSAlert(message string) {
	// Use osascript to show a dialog
	script := fmt.Sprintf(`
display dialog "%s" buttons {"Cancel", "I'm Alive"} default button "I'm Alive" with icon caution with title "Lazarus Protocol Alert"
`, message)

	cmd := exec.Command("osascript", "-e", script)
	err := cmd.Run()
	if err == nil {
		// User clicked "I'm Alive"
		as.userResponded <- true
	}
}

// showLinuxAlert shows a Linux notification
func (as *AlertSystem) showLinuxAlert(message string) {
	// Try zenity first (most common)
	cmd := exec.Command("zenity", "--question",
		"--title=Lazarus Protocol Alert",
		"--text="+message,
		"--ok-label=I'm Alive",
		"--cancel-label=Cancel",
		"--width=400")

	err := cmd.Run()
	if err == nil {
		as.userResponded <- true
		return
	}

	// Fallback to kdialog (KDE)
	cmd = exec.Command("kdialog", "--yesno", message,
		"--title", "Lazarus Protocol Alert")
	err = cmd.Run()
	if err == nil {
		as.userResponded <- true
		return
	}

	// Fallback to terminal alert
	as.showTerminalAlert(message)
}

// showTerminalAlert shows a terminal-based alert
func (as *AlertSystem) showTerminalAlert(message string) {
	separator := strings.Repeat("=", 60)
	log.Println("\n" + separator)
	log.Println(message)
	log.Println(separator)
	log.Println("\nType 'alive' and press Enter to confirm you are alive:")

	// This would need user input handling in the main loop
	// For now, just log the message
}

// playAlertSound plays an alert sound
func (as *AlertSystem) playAlertSound() {
	switch runtime.GOOS {
	case "windows":
		// Use PowerShell to play system sound
		exec.Command("powershell", "-Command",
			"[console]::beep(800,500); [console]::beep(800,500); [console]::beep(800,500)").Run()

	case "darwin":
		// Use afplay to play system sound
		exec.Command("afplay", "/System/Library/Sounds/Sosumi.aiff").Run()
		time.Sleep(500 * time.Millisecond)
		exec.Command("afplay", "/System/Library/Sounds/Sosumi.aiff").Run()
		time.Sleep(500 * time.Millisecond)
		exec.Command("afplay", "/System/Library/Sounds/Sosumi.aiff").Run()

	case "linux":
		// Use paplay (PulseAudio) or aplay (ALSA)
		exec.Command("paplay", "/usr/share/sounds/freedesktop/stereo/alarm-clock-elapsed.oga").Run()
	}
}

// OpenBrowserAlert opens a browser with an alert page
func (as *AlertSystem) OpenBrowserAlert(inactiveDuration time.Duration) error {
	// Create a simple HTML alert page
	htmlContent := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
	<title>Lazarus Protocol Alert</title>
	<style>
		body {
			font-family: Arial, sans-serif;
			background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
			display: flex;
			justify-content: center;
			align-items: center;
			height: 100vh;
			margin: 0;
		}
		.alert-box {
			background: white;
			padding: 40px;
			border-radius: 20px;
			box-shadow: 0 20px 60px rgba(0,0,0,0.3);
			text-align: center;
			max-width: 500px;
		}
		h1 {
			color: #e74c3c;
			font-size: 32px;
			margin-bottom: 20px;
		}
		.warning-icon {
			font-size: 80px;
			margin-bottom: 20px;
		}
		.info {
			font-size: 18px;
			color: #333;
			margin: 20px 0;
			line-height: 1.6;
		}
		.countdown {
			font-size: 48px;
			color: #e74c3c;
			font-weight: bold;
			margin: 30px 0;
		}
		button {
			background: #27ae60;
			color: white;
			border: none;
			padding: 20px 60px;
			font-size: 24px;
			border-radius: 50px;
			cursor: pointer;
			transition: all 0.3s;
			box-shadow: 0 10px 30px rgba(39, 174, 96, 0.3);
		}
		button:hover {
			background: #229954;
			transform: translateY(-2px);
			box-shadow: 0 15px 40px rgba(39, 174, 96, 0.4);
		}
		.details {
			margin-top: 30px;
			padding: 20px;
			background: #f8f9fa;
			border-radius: 10px;
			font-size: 14px;
			color: #666;
		}
	</style>
</head>
<body>
	<div class="alert-box">
		<div class="warning-icon">⚠️</div>
		<h1>LAZARUS PROTOCOL WARNING</h1>
		<div class="info">
			Your system has been inactive for <strong>%v</strong>.<br>
			If you do not respond, your digital will may be executed.
		</div>
		<div class="countdown" id="countdown">%v</div>
		<button onclick="confirmAlive()">I'M ALIVE! 💚</button>
		<div class="details">
			<strong>What happens if I don't respond?</strong><br>
			After %v of total inactivity, your vault will be unlocked<br>
			and your beneficiary will be able to access your encrypted data.
		</div>
	</div>
	<script>
		function confirmAlive() {
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
			document.getElementById('countdown').textContent =
				hours + 'h ' + minutes + 'm remaining';
		}, 1000);

		// Auto-play alert sound
		const audio = new Audio('data:audio/wav;base64,UklGRnoGAABXQVZFZm10IBAAAAABAAEAQB8AAEAfAAABAAgAZGF0YQoGAACBhYqFbF1fdJivrJBhNjVgodDbq2EcBj+a2/LDciUFLIHO8tiJNwgZaLvt559NEAxQp+PwtmMcBjiR1/LMeSwFJHfH8N2QQAoUXrTp66hVFApGn+DyvmwhBSuBzvLZiTYIGGS57OihUBELTKXh8bllHAU2jdXvzn0pBSh+zPDajzsKElyx6OyrWBUIQ5zd8sFuJAUuhM/z24k2CBhku+zooVARC0yl4fG5ZRwFNo3V7859KQUofsz');
		audio.play().catch(() => {});
	</script>
</body>
</html>
`,
		inactiveDuration.Round(time.Hour),
		as.config.EmergencyThreshold-inactiveDuration,
		as.config.EmergencyThreshold,
		int((as.config.EmergencyThreshold - inactiveDuration).Seconds()),
	)

	// Save HTML to temp file
	tmpFile := "/tmp/lazarus_alert.html"
	if runtime.GOOS == "windows" {
		tmpFile = "C:\\Windows\\Temp\\lazarus_alert.html"
	}

	if err := os.WriteFile(tmpFile, []byte(htmlContent), 0644); err != nil {
		return fmt.Errorf("failed to create alert page: %w", err)
	}

	// Open in default browser
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", tmpFile)
	case "darwin":
		cmd = exec.Command("open", tmpFile)
	case "linux":
		cmd = exec.Command("xdg-open", tmpFile)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to open browser: %w", err)
	}

	log.Println("✓ Alert page opened in browser")
	return nil
}

// HandleUserResponse processes user confirmation
func (as *AlertSystem) HandleUserResponse() {
	as.userResponded <- true
}
