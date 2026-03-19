package config

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Port               string                   `yaml:"port"`
	MySQL              MySQLConfig              `yaml:"mysql"`
	Redis              RedisConfig              `yaml:"redis"`
	JWT                JWTConfig                `yaml:"jwt"`
	COS                COSConfig                `yaml:"cos"`
	Plugin             PluginConfig             `yaml:"plugin"`
	LLM                LLMConfig                `yaml:"llm"`
	SummaryLLM         LLMConfig                `yaml:"summary_llm"`
	EmbeddingLLM       EmbeddingLLMConfig       `yaml:"embedding_llm"`
	Milvus             MilvusConfig             `yaml:"milvus"`
	Tools              ToolsConfig              `yaml:"tools"`
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
	SecretID  string `yaml:"secret_id"`
	SecretKey string `yaml:"secret_key"`
	Bucket    string `yaml:"bucket"`
	Region    string `yaml:"region"`
}

type PluginConfig struct {
	DefaultIconURL               string `yaml:"default_icon_url"`
	MCPSyncDelaySeconds          int    `yaml:"mcp_sync_delay_seconds"`
	MCPSyncPollSeconds           int    `yaml:"mcp_sync_poll_seconds"`
	MCPSyncIntervalSeconds       int    `yaml:"mcp_sync_interval_seconds"`
	MCPSyncTimeoutSeconds        int    `yaml:"mcp_sync_timeout_seconds"`
	MCPSyncRetrySeconds          int    `yaml:"mcp_sync_retry_seconds"`
	OpenVikingSyncDelaySeconds   int    `yaml:"openviking_sync_delay_seconds"`
	OpenVikingSyncPollSeconds    int    `yaml:"openviking_sync_poll_seconds"`
	OpenVikingSyncTimeoutSeconds int    `yaml:"openviking_sync_timeout_seconds"`
	OpenVikingSyncRetrySeconds   int    `yaml:"openviking_sync_retry_seconds"`
}

type LLMConfig struct {
	Model  string `yaml:"model"`
	APIKey string `yaml:"api_key"`
}

type EmbeddingLLMConfig struct {
	Model  string `yaml:"model"`
	APIKey string `yaml:"api_key"`
}

type MilvusConfig struct {
	Address      string `yaml:"address"`
	Username     string `yaml:"username"`
	Password     string `yaml:"password"`
	APIKey       string `yaml:"api_key"`
	Dimension    int64  `yaml:"dimension"`
	MetricType   string `yaml:"metric_type"`
	EnableHybrid bool   `yaml:"enable_hybrid"`
	AnalyzerType string `yaml:"analyzer_type"`
}

type ToolsConfig struct {
	Sandbox    SandboxConfig        `yaml:"sandbox"`
	Memory     MemoryProviderConfig `yaml:"memory"`
	OpenViking OpenVikingConfig     `yaml:"openviking"`
}

type SandboxConfig struct {
	Url string `yaml:"url"`
}

// MemoryProviderConfig controls which long-term memory backend is active.
type MemoryProviderConfig struct {
	Provider string `yaml:"provider"`
}

type OpenVikingConfig struct {
	BaseURL                    string `yaml:"base_url"`
	APIKey                     string `yaml:"api_key"`
	SkillsRoot                 string `yaml:"skills_root"`
	ToolsRoot                  string `yaml:"tools_root"`
	TimeoutSeconds             int    `yaml:"timeout_seconds"`
	MemorySearchTimeoutSeconds int    `yaml:"memory_search_timeout_seconds"`
	MemorySyncDelaySeconds     int    `yaml:"memory_sync_delay_seconds"`
	MemorySyncPollSeconds      int    `yaml:"memory_sync_poll_seconds"`
	MemorySyncTimeoutSeconds   int    `yaml:"memory_sync_timeout_seconds"`
	MemorySyncRetrySeconds     int    `yaml:"memory_sync_retry_seconds"`
}

type APMPlusConfig struct {
	Host        string `yaml:"host"`
	AppKey      string `yaml:"app_key"`
	ServiceName string `yaml:"service_name"`
	Release     string `yaml:"release"`
}

type ContextCompressionConfig struct {
	Enabled                *bool `yaml:"enabled"`
	SoftLimitTokens        int   `yaml:"soft_limit_tokens"`
	HardLimitTokens        int   `yaml:"hard_limit_tokens"`
	OutputReserveTokens    int   `yaml:"output_reserve_tokens"`
	MaxRecentMessages      int   `yaml:"max_recent_messages"`
	EstimatedCharsPerToken int   `yaml:"estimated_chars_per_token"`
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
	return cfg
}
