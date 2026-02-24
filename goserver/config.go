package main

import (
	"encoding/json"
	"os"
)

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
