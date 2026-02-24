package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

// OpenClawConfig holds OpenClaw integration settings.
// Integration uses the OpenClaw CLI (`openclaw agent`) which communicates with
// the Gateway via WebSocket. The sentinel-guard plugin (installed at
// ~/.openclaw/extensions/sentinel-guard/) provides sentinel_gate, sentinel_status,
// and sentinel_approval tools to the agent, routing enforcement through the
// Sentinel HTTP proxy (default http://127.0.0.1:18080).
type OpenClawConfig struct {
	Enabled   bool   `json:"enabled"`
	ServerURL string `json:"server_url"` // Sentinel proxy URL (default http://127.0.0.1:18080)
	AgentID   string `json:"agent_id"`   // OpenClaw agent id (default "main")
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

// SendTaskWithoutSentinel sends a task directly to OpenClaw.
// Use this only when caller already ran policy checks and audit.
func (oc *OpenClawClient) SendTaskWithoutSentinel(prompt string) (*OpenClawResponse, error) {
	if !oc.config.Enabled {
		return nil, fmt.Errorf("OpenClaw is disabled")
	}
	return oc.sendTaskWithoutSentinel(prompt)
}

func (oc *OpenClawClient) sendTaskWithoutSentinel(prompt string) (*OpenClawResponse, error) {
	agentID := oc.config.AgentID
	if agentID == "" {
		agentID = "main"
	}

	// Primary path: use the OpenClaw CLI which talks to the WebSocket gateway.
	cmd := exec.Command("openclaw", "agent", "--agent", agentID, "--local", "--message", prompt)
	out, err := cmd.CombinedOutput()
	if err != nil {
		// Fallback: try a direct HTTP POST to the configured server URL (legacy path).
		return oc.sendTaskHTTP(prompt)
	}

	log.Println("✓ OpenClaw accepted the task via CLI")
	log.Printf("  Agent output: %s", strings.TrimSpace(string(out)))
	return &OpenClawResponse{Status: "ok", Message: strings.TrimSpace(string(out))}, nil
}

// sendTaskHTTP sends a task via the legacy HTTP POST path (fallback).
func (oc *OpenClawClient) sendTaskHTTP(prompt string) (*OpenClawResponse, error) {
	payload := OpenClawRequest{
		Task: prompt,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Send HTTP POST request
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Post(oc.config.ServerURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("OpenClaw connection failed: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var response OpenClawResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenClaw returned error: %s - %s", response.Status, response.Message)
	}

	log.Println("✓ OpenClaw accepted the task")
	if response.TaskID != "" {
		log.Printf("  Task ID: %s", response.TaskID)
	}
	log.Println("  Browser should open shortly...")
	log.Println()

	return &response, nil
}

