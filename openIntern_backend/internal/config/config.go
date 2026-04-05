package config

import (
	"log"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Port               string                   `yaml:"port"`
	MySQL              MySQLConfig              `yaml:"mysql"`
	Redis              RedisConfig              `yaml:"redis"`
	JWT                JWTConfig                `yaml:"jwt"`
	COS                COSConfig                `yaml:"cos"`
	Plugin             PluginConfig             `yaml:"plugin"`
	SummaryLLM         LLMConfig                `yaml:"summary_llm"`
	Tools              ToolsConfig              `yaml:"tools"`
	Agent              AgentConfig              `yaml:"agent"`
	ContextCompression ContextCompressionConfig `yaml:"context_compression"`
	APMPlus            APMPlusConfig            `yaml:"apmplus"`
}

type MySQLConfig struct {
	DSN string
}

type JWTConfig struct {
	Secret        string `yaml:"secret"`
	ExpireMinutes int    `yaml:"expire_minutes"`
}

type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type COSConfig struct {
	SecretID  string `yaml:"secret_id" json:"secret_id,omitempty"`
	SecretKey string `yaml:"secret_key" json:"secret_key,omitempty"`
	Bucket    string `yaml:"bucket" json:"bucket"`
	Region    string `yaml:"region" json:"region"`
}

type PluginConfig struct {
	BuiltinManifestPath    string `yaml:"builtin_manifest_path" json:"builtin_manifest_path"`
	DefaultIconURL         string `yaml:"default_icon_url" json:"default_icon_url"`
	MCPSyncDelaySeconds    int    `yaml:"mcp_sync_delay_seconds" json:"mcp_sync_delay_seconds"`
	MCPSyncPollSeconds     int    `yaml:"mcp_sync_poll_seconds" json:"mcp_sync_poll_seconds"`
	MCPSyncIntervalSeconds int    `yaml:"mcp_sync_interval_seconds" json:"mcp_sync_interval_seconds"`
	MCPSyncTimeoutSeconds  int    `yaml:"mcp_sync_timeout_seconds" json:"mcp_sync_timeout_seconds"`
	MCPSyncRetrySeconds    int    `yaml:"mcp_sync_retry_seconds" json:"mcp_sync_retry_seconds"`
}

type LLMConfig struct {
	Model    string `yaml:"model" json:"model"`
	APIKey   string `yaml:"api_key" json:"api_key,omitempty"`
	BaseURL  string `yaml:"base_url" json:"base_url"`
	Provider string `yaml:"provider" json:"provider"`
}

type ToolsConfig struct {
	Sandbox    SandboxConfig        `yaml:"sandbox" json:"sandbox"`
	Memory     MemoryProviderConfig `yaml:"memory" json:"memory"`
	OpenViking OpenVikingConfig     `yaml:"openviking" json:"openviking"`
}

type SandboxDockerConfig struct {
	Image   string `yaml:"image" json:"image"`
	Host    string `yaml:"host" json:"host"`
	Network string `yaml:"network" json:"network"`
}

type SandboxConfig struct {
	Enabled                   *bool               `yaml:"enabled" json:"enabled"`
	Provider                  string              `yaml:"provider" json:"provider"`
	IdleTTLSeconds            int                 `yaml:"idle_ttl_seconds" json:"idle_ttl_seconds"`
	CreateTimeoutSeconds      int                 `yaml:"create_timeout_seconds" json:"create_timeout_seconds"`
	RecycleIntervalSeconds    int                 `yaml:"recycle_interval_seconds" json:"recycle_interval_seconds"`
	HealthcheckTimeoutSeconds int                 `yaml:"healthcheck_timeout_seconds" json:"healthcheck_timeout_seconds"`
	Docker                    SandboxDockerConfig `yaml:"docker" json:"docker"`
}

// MemoryProviderConfig controls which long-term memory backend is active.
type MemoryProviderConfig struct {
	Provider string `yaml:"provider" json:"provider"`
}

type OpenVikingConfig struct {
	BaseURL                    string `yaml:"base_url" json:"base_url"`
	APIKey                     string `yaml:"api_key" json:"api_key,omitempty"`
	SkillsRoot                 string `yaml:"skills_root" json:"skills_root"`
	TimeoutSeconds             int    `yaml:"timeout_seconds" json:"timeout_seconds"`
	MemorySearchTimeoutSeconds int    `yaml:"memory_search_timeout_seconds" json:"memory_search_timeout_seconds"`
	MemorySyncDelaySeconds     int    `yaml:"memory_sync_delay_seconds" json:"memory_sync_delay_seconds"`
	MemorySyncPollSeconds      int    `yaml:"memory_sync_poll_seconds" json:"memory_sync_poll_seconds"`
	MemorySyncTimeoutSeconds   int    `yaml:"memory_sync_timeout_seconds" json:"memory_sync_timeout_seconds"`
	MemorySyncRetrySeconds     int    `yaml:"memory_sync_retry_seconds" json:"memory_sync_retry_seconds"`
}

type APMPlusConfig struct {
	Host        string `yaml:"host" json:"host"`
	AppKey      string `yaml:"app_key" json:"app_key,omitempty"`
	ServiceName string `yaml:"service_name" json:"service_name"`
	Release     string `yaml:"release" json:"release"`
}

type AgentConfig struct {
	MaxIterations int `yaml:"max_iterations" json:"max_iterations"`
}

type ContextCompressionConfig struct {
	Enabled                *bool `yaml:"enabled" json:"enabled"`
	SoftLimitTokens        int   `yaml:"soft_limit_tokens" json:"soft_limit_tokens"`
	HardLimitTokens        int   `yaml:"hard_limit_tokens" json:"hard_limit_tokens"`
	OutputReserveTokens    int   `yaml:"output_reserve_tokens" json:"output_reserve_tokens"`
	MaxRecentMessages      int   `yaml:"max_recent_messages" json:"max_recent_messages"`
	EstimatedCharsPerToken int   `yaml:"estimated_chars_per_token" json:"estimated_chars_per_token"`
}

func LoadConfig(configFile string) *Config {
	cfg := &Config{}
	data, err := os.ReadFile(configFile)
	if err != nil {
		return cfg
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		log.Fatalf("failed to parse config.yaml: %v", err)
	}
	// 配置文件里的相对路径统一按 config.yaml 所在目录解析，避免不同启动目录下表现不一致。
	if cfg.Plugin.BuiltinManifestPath != "" && !filepath.IsAbs(cfg.Plugin.BuiltinManifestPath) {
		cfg.Plugin.BuiltinManifestPath = filepath.Join(filepath.Dir(configFile), cfg.Plugin.BuiltinManifestPath)
	}
	return cfg
}
