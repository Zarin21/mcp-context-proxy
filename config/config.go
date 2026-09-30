package config

import (
	"encoding/json"
	"os"
)

// ServerConfig defines how to connect to an upstream MCP server
type ServerConfig struct {
	Name      string   `json:"name"`
	Transport string   `json:"transport"` // "stdio" or "http"
	Command   string   `json:"command"`   // for stdio: command to run
	Args      []string `json:"args"`      // for stdio: command arguments
	URL       string   `json:"url"`       // for http: server URL
}

// ToolRule defines compression rules for a specific tool
type ToolRule struct {
	Strategies   []string `json:"strategies"`
	MaxTokens    int      `json:"max_tokens"`
	MaxDepth     int      `json:"max_depth"`
	MaxStringLen int      `json:"max_string_len"`
	MaxArrayLen  int      `json:"max_array_len"`
	RemoveNulls  bool     `json:"remove_nulls"`
	RemoveEmpty  bool     `json:"remove_empty"`
}

// ProxyConfig is the top-level configuration
type ProxyConfig struct {
	Listen      string              `json:"listen"`
	Transport   string              `json:"transport"`
	HandleTTL   string              `json:"handle_ttl"`
	DefaultRule ToolRule            `json:"default_rule"`
	ToolRules   map[string]ToolRule `json:"tool_rules"`
	Servers     []ServerConfig      `json:"servers"`
}

// Load reads configuration from a JSON file
func Load(path string) (*ProxyConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := Default()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Default returns a ProxyConfig with sensible defaults
func Default() *ProxyConfig {
	return &ProxyConfig{
		Listen:    ":9800",
		Transport: "stdio",
		HandleTTL: "10m",
		DefaultRule: ToolRule{
			Strategies:   []string{"json_prune", "log_dedup", "truncate"},
			MaxTokens:    4096,
			MaxDepth:     5,
			MaxStringLen: 200,
			MaxArrayLen:  20,
			RemoveNulls:  true,
			RemoveEmpty:  true,
		},
		ToolRules: make(map[string]ToolRule),
		Servers:   []ServerConfig{},
	}
}
