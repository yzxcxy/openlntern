package config

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Port         string             `yaml:"port"`
	MySQL        MySQLConfig        `yaml:"mysql"`
	Redis        RedisConfig        `yaml:"redis"`
	JWT          JWTConfig          `yaml:"jwt"`
	COS          COSConfig          `yaml:"cos"`
	Plugin       PluginConfig       `yaml:"plugin"`
	LLM          LLMConfig          `yaml:"llm"`
	SummaryLLM   LLMConfig          `yaml:"summary_llm"`
	EmbeddingLLM EmbeddingLLMConfig `yaml:"embedding_llm"`
	Milvus       MilvusConfig       `yaml:"milvus"`
	Tools        ToolsConfig        `yaml:"tools"`
	APMPlus      APMPlusConfig      `yaml:"apmplus"`
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
	DefaultIconURL         string `yaml:"default_icon_url"`
	MCPSyncDelaySeconds    int    `yaml:"mcp_sync_delay_seconds"`
	MCPSyncPollSeconds     int    `yaml:"mcp_sync_poll_seconds"`
	MCPSyncScanSeconds     int    `yaml:"mcp_sync_scan_seconds"`
	MCPSyncIntervalSeconds int    `yaml:"mcp_sync_interval_seconds"`
	MCPSyncTimeoutSeconds  int    `yaml:"mcp_sync_timeout_seconds"`
	MCPSyncRetrySeconds    int    `yaml:"mcp_sync_retry_seconds"`
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
	Sandbox    SandboxConfig    `yaml:"sandbox"`
	OpenViking OpenVikingConfig `yaml:"openviking"`
}

type SandboxConfig struct {
	Url string `yaml:"url"`
}

type OpenVikingConfig struct {
	BaseURL        string `yaml:"base_url"`
	APIKey         string `yaml:"api_key"`
	SkillsRoot     string `yaml:"skills_root"`
	TimeoutSeconds int    `yaml:"timeout_seconds"`
}

type APMPlusConfig struct {
	Host        string `yaml:"host"`
	AppKey      string `yaml:"app_key"`
	ServiceName string `yaml:"service_name"`
	Release     string `yaml:"release"`
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
