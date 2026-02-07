package main

import (
	"log"
	"sync"
	"time"
)

// ActivityMonitor tracks system activity without external dependencies
// This is a simplified version that works on all platforms
type ActivityMonitor struct {
	lastActiveTime time.Time
	mu             sync.RWMutex
	stopChan       chan struct{}
	checkInterval  time.Duration
	manualUpdate   chan bool
}

// NewActivityMonitor creates a new activity monitor
func NewActivityMonitor(checkInterval time.Duration) *ActivityMonitor {
	return &ActivityMonitor{
		lastActiveTime: time.Now(),
		stopChan:       make(chan struct{}),
		checkInterval:  checkInterval,
		manualUpdate:   make(chan bool, 10),
	}
}

// Start begins monitoring system activity
// Note: This simplified version requires manual updates via MarkActive()
// For production, integrate with OS-specific APIs or use a lightweight library
func (am *ActivityMonitor) Start() {
	log.Println("🔍 Starting activity monitor (manual mode)...")
	log.Printf("   Check interval: %v", am.checkInterval)
	log.Println("   Note: Activity detection requires manual confirmation")
	log.Println("   Type 'alive' in terminal or respond to GUI alerts")

	ticker := time.NewTicker(am.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// In manual mode, we don't auto-detect activity
			// Activity must be marked via MarkActive() or alert responses

		case <-am.manualUpdate:
			am.updateActivity()
			log.Println("✓ Activity manually confirmed")

		case <-am.stopChan:
			log.Println("🛑 Activity monitor stopped")
			return
		}
	}
}

// MarkActive manually marks the system as active
// This is called when user responds to alerts or types 'alive'
func (am *ActivityMonitor) MarkActive() {
	am.manualUpdate <- true
}

// updateActivity updates the last active timestamp
func (am *ActivityMonitor) updateActivity() {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.lastActiveTime = time.Now()
}

// GetLastActiveTime returns the last activity timestamp
func (am *ActivityMonitor) GetLastActiveTime() time.Time {
	am.mu.RLock()
	defer am.mu.RUnlock()
	return am.lastActiveTime
}

// GetInactiveDuration returns how long the system has been inactive
func (am *ActivityMonitor) GetInactiveDuration() time.Duration {
	return time.Since(am.GetLastActiveTime())
}

// IsActive checks if the system was active within the given duration
func (am *ActivityMonitor) IsActive(within time.Duration) bool {
	return am.GetInactiveDuration() < within
}

// Stop stops the activity monitor
func (am *ActivityMonitor) Stop() {
	close(am.stopChan)
}
