package config

import (
	"os"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// RuntimeConfig 运行时可修改配置
type RuntimeConfig struct {
	Agent              AgentConfig              `json:"agent" yaml:"agent"`
	Tools              ToolsConfig              `json:"tools" yaml:"tools"`
	ContextCompression ContextCompressionConfig `json:"context_compression" yaml:"context_compression"`
	Plugin             PluginConfig             `json:"plugin" yaml:"plugin"`
	SummaryLLM         LLMConfig                `json:"summary_llm" yaml:"summary_llm"`
	COS                COSConfig                `json:"cos" yaml:"cos"`
	APMPlus            APMPlusConfig            `json:"apmplus" yaml:"apmplus"`
}

// RuntimeConfigResponse 前端配置响应（敏感字段脱敏）
type RuntimeConfigResponse struct {
	Agent              AgentConfig              `json:"agent"`
	Tools              ToolsConfigResponse      `json:"tools"`
	ContextCompression ContextCompressionConfig `json:"context_compression"`
	Plugin             PluginConfig             `json:"plugin"`
	SummaryLLM         LLMConfigResponse        `json:"summary_llm"`
	COS                COSConfigResponse        `json:"cos"`
	APMPlus            APMPlusConfigResponse    `json:"apmplus"`
}

// LLMConfigResponse LLM配置响应（敏感字段脱敏）
type LLMConfigResponse struct {
	Model    string `json:"model"`
	APIKey   string `json:"api_key,omitempty"`
	BaseURL  string `json:"base_url"`
	Provider string `json:"provider"`
}

// COSConfigResponse COS配置响应（敏感字段脱敏）
type COSConfigResponse struct {
	SecretID  string `json:"secret_id,omitempty"`
	SecretKey string `json:"secret_key,omitempty"`
	Bucket    string `json:"bucket"`
	Region    string `json:"region"`
}

// APMPlusConfigResponse APMPlus配置响应（敏感字段脱敏）
type APMPlusConfigResponse struct {
	Host        string `json:"host"`
	AppKey      string `json:"app_key,omitempty"`
	ServiceName string `json:"service_name"`
	Release     string `json:"release"`
}

// ToolsConfigResponse 工具配置响应（敏感字段脱敏）
type ToolsConfigResponse struct {
	Sandbox SandboxConfig        `json:"sandbox"`
	Memory  MemoryProviderConfig `json:"memory"`
}

// 全局配置管理
var (
	globalConfig   *Config
	globalRuntime  *RuntimeConfig
	configMu       sync.RWMutex
	configFilePath string
)

// InitRuntime 初始化运行时配置
func InitRuntime(cfg *Config, cfgPath string) {
	configMu.Lock()
	defer configMu.Unlock()

	globalConfig = cfg
	configFilePath = cfgPath

	// 初始化运行时配置
	globalRuntime = &RuntimeConfig{
		Agent:              cfg.Agent,
		Tools:              cfg.Tools,
		ContextCompression: cfg.ContextCompression,
		Plugin:             cfg.Plugin,
		SummaryLLM:         cfg.SummaryLLM,
		COS:                cfg.COS,
		APMPlus:            cfg.APMPlus,
	}
}

// GetRuntimeConfig 获取运行时配置
func GetRuntimeConfig() *RuntimeConfig {
	configMu.RLock()
	defer configMu.RUnlock()
	if globalRuntime == nil {
		return &RuntimeConfig{}
	}
	return globalRuntime
}

// GetConfig 获取完整配置
func GetConfig() *Config {
	configMu.RLock()
	defer configMu.RUnlock()
	if globalConfig == nil {
		return &Config{}
	}
	return globalConfig
}

// UpdateRuntimeConfig 更新运行时配置
func UpdateRuntimeConfig(updates map[string]interface{}) error {
	configMu.Lock()
	defer configMu.Unlock()

	// 更新运行时配置
	if agentUpdates, ok := updates["agent"].(map[string]interface{}); ok {
		updateAgentConfig(&globalRuntime.Agent, agentUpdates)
		globalConfig.Agent = globalRuntime.Agent
	}
	if toolsUpdates, ok := updates["tools"].(map[string]interface{}); ok {
		updateToolsConfig(&globalRuntime.Tools, toolsUpdates)
		globalConfig.Tools = globalRuntime.Tools
	}
	if ccUpdates, ok := updates["context_compression"].(map[string]interface{}); ok {
		updateContextCompressionConfig(&globalRuntime.ContextCompression, ccUpdates)
		globalConfig.ContextCompression = globalRuntime.ContextCompression
	}
	if pluginUpdates, ok := updates["plugin"].(map[string]interface{}); ok {
		updatePluginConfig(&globalRuntime.Plugin, pluginUpdates)
		globalConfig.Plugin = globalRuntime.Plugin
	}
	if summaryLLMUpdates, ok := updates["summary_llm"].(map[string]interface{}); ok {
		updateLLMConfig(&globalRuntime.SummaryLLM, summaryLLMUpdates)
		globalConfig.SummaryLLM = globalRuntime.SummaryLLM
	}
	if cosUpdates, ok := updates["cos"].(map[string]interface{}); ok {
		updateCOSConfig(&globalRuntime.COS, cosUpdates)
		globalConfig.COS = globalRuntime.COS
	}
	if apmPlusUpdates, ok := updates["apmplus"].(map[string]interface{}); ok {
		updateAPMPlusConfig(&globalRuntime.APMPlus, apmPlusUpdates)
		globalConfig.APMPlus = globalRuntime.APMPlus
	}

	// 写回配置文件
	return saveConfigToFile()
}

// saveConfigToFile 保存配置到文件
func saveConfigToFile() error {
	if configFilePath == "" {
		return nil
	}

	data, err := yaml.Marshal(globalConfig)
	if err != nil {
		return err
	}

	return os.WriteFile(configFilePath, data, 0644)
}

// ReloadConfig 重新加载配置
func ReloadConfig() error {
	configMu.Lock()
	defer configMu.Unlock()

	// 重新加载配置文件
	cfg := LoadConfig(configFilePath)
	globalConfig = cfg

	globalRuntime = &RuntimeConfig{
		Agent:              cfg.Agent,
		Tools:              cfg.Tools,
		ContextCompression: cfg.ContextCompression,
		Plugin:             cfg.Plugin,
		SummaryLLM:         cfg.SummaryLLM,
		COS:                cfg.COS,
		APMPlus:            cfg.APMPlus,
	}

	return nil
}

// ToResponse 转换为前端响应格式（脱敏敏感字段）
func (r *RuntimeConfig) ToResponse() RuntimeConfigResponse {
	return RuntimeConfigResponse{
		Agent: r.Agent,
		Tools: ToolsConfigResponse{
			Sandbox: r.Tools.Sandbox,
			Memory:  r.Tools.Memory,
		},
		ContextCompression: r.ContextCompression,
		Plugin:             r.Plugin,
		SummaryLLM: LLMConfigResponse{
			Model:    r.SummaryLLM.Model,
			APIKey:   maskAPIKey(r.SummaryLLM.APIKey),
			BaseURL:  r.SummaryLLM.BaseURL,
			Provider: r.SummaryLLM.Provider,
		},
		COS: COSConfigResponse{
			SecretID:  maskAPIKey(r.COS.SecretID),
			SecretKey: maskAPIKey(r.COS.SecretKey),
			Bucket:    r.COS.Bucket,
			Region:    r.COS.Region,
		},
		APMPlus: APMPlusConfigResponse{
			Host:        r.APMPlus.Host,
			AppKey:      maskAPIKey(r.APMPlus.AppKey),
			ServiceName: r.APMPlus.ServiceName,
			Release:     r.APMPlus.Release,
		},
	}
}

// maskAPIKey 脱敏 API Key
func maskAPIKey(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	if len(key) <= 8 {
		return "***"
	}
	// 显示前3个和后3个字符
	return key[:3] + "***" + key[len(key)-3:]
}

// 辅助函数：更新配置
func updateAgentConfig(cfg *AgentConfig, updates map[string]interface{}) {
	if v, ok := updates["max_iterations"].(float64); ok {
		cfg.MaxIterations = int(v)
	}
}

func updateToolsConfig(cfg *ToolsConfig, updates map[string]interface{}) {
	if sandboxUpdates, ok := updates["sandbox"].(map[string]interface{}); ok {
		if enabled, ok := sandboxUpdates["enabled"].(bool); ok {
			cfg.Sandbox.Enabled = &enabled
		}
		if provider, ok := sandboxUpdates["provider"].(string); ok {
			cfg.Sandbox.Provider = provider
		}
		if idleTTL, ok := sandboxUpdates["idle_ttl_seconds"].(float64); ok {
			cfg.Sandbox.IdleTTLSeconds = int(idleTTL)
		}
		if createTimeout, ok := sandboxUpdates["create_timeout_seconds"].(float64); ok {
			cfg.Sandbox.CreateTimeoutSeconds = int(createTimeout)
		}
		if recycleInterval, ok := sandboxUpdates["recycle_interval_seconds"].(float64); ok {
			cfg.Sandbox.RecycleIntervalSeconds = int(recycleInterval)
		}
		if healthcheckTimeout, ok := sandboxUpdates["healthcheck_timeout_seconds"].(float64); ok {
			cfg.Sandbox.HealthcheckTimeoutSeconds = int(healthcheckTimeout)
		}
		if dockerUpdates, ok := sandboxUpdates["docker"].(map[string]interface{}); ok {
			if image, ok := dockerUpdates["image"].(string); ok {
				cfg.Sandbox.Docker.Image = image
			}
			if host, ok := dockerUpdates["host"].(string); ok {
				cfg.Sandbox.Docker.Host = host
			}
			if network, ok := dockerUpdates["network"].(string); ok {
				cfg.Sandbox.Docker.Network = network
			}
		}
	}
	if memoryUpdates, ok := updates["memory"].(map[string]interface{}); ok {
		if provider, ok := memoryUpdates["provider"].(string); ok {
			cfg.Memory.Provider = provider
		}
	}
}

func updateContextCompressionConfig(cfg *ContextCompressionConfig, updates map[string]interface{}) {
	if v, ok := updates["enabled"].(bool); ok {
		cfg.Enabled = &v
	}
	if v, ok := updates["soft_limit_tokens"].(float64); ok {
		cfg.SoftLimitTokens = int(v)
	}
	if v, ok := updates["hard_limit_tokens"].(float64); ok {
		cfg.HardLimitTokens = int(v)
	}
	if v, ok := updates["output_reserve_tokens"].(float64); ok {
		cfg.OutputReserveTokens = int(v)
	}
	if v, ok := updates["max_recent_messages"].(float64); ok {
		cfg.MaxRecentMessages = int(v)
	}
	if v, ok := updates["estimated_chars_per_token"].(float64); ok {
		cfg.EstimatedCharsPerToken = int(v)
	}
}

func updatePluginConfig(cfg *PluginConfig, updates map[string]interface{}) {
	if v, ok := updates["builtin_manifest_path"].(string); ok {
		cfg.BuiltinManifestPath = v
	}
	if v, ok := updates["default_icon_url"].(string); ok {
		cfg.DefaultIconURL = v
	}
}

func updateLLMConfig(cfg *LLMConfig, updates map[string]interface{}) {
	if v, ok := updates["model"].(string); ok {
		cfg.Model = v
	}
	if v, ok := updates["api_key"].(string); ok && v != "" {
		cfg.APIKey = v
	}
	if v, ok := updates["base_url"].(string); ok {
		cfg.BaseURL = v
	}
	if v, ok := updates["provider"].(string); ok {
		cfg.Provider = v
	}
}

func updateCOSConfig(cfg *COSConfig, updates map[string]interface{}) {
	if v, ok := updates["secret_id"].(string); ok && v != "" {
		cfg.SecretID = v
	}
	if v, ok := updates["secret_key"].(string); ok && v != "" {
		cfg.SecretKey = v
	}
	if v, ok := updates["bucket"].(string); ok {
		cfg.Bucket = v
	}
	if v, ok := updates["region"].(string); ok {
		cfg.Region = v
	}
}

func updateAPMPlusConfig(cfg *APMPlusConfig, updates map[string]interface{}) {
	if v, ok := updates["host"].(string); ok {
		cfg.Host = v
	}
	if v, ok := updates["app_key"].(string); ok && v != "" {
		cfg.AppKey = v
	}
	if v, ok := updates["service_name"].(string); ok {
		cfg.ServiceName = v
	}
	if v, ok := updates["release"].(string); ok {
		cfg.Release = v
	}
}

// GetConfigFilePath 获取配置文件路径
func GetConfigFilePath() string {
	return configFilePath
}
