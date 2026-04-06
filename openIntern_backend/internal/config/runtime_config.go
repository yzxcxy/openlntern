package config

import (
	"fmt"
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
	MinIO              MinIOConfig              `json:"minio" yaml:"minio"`
	APMPlus            APMPlusConfig            `json:"apmplus" yaml:"apmplus"`
}

// RuntimeConfigResponse 前端配置响应（敏感字段脱敏）
type RuntimeConfigResponse struct {
	Agent              AgentConfig              `json:"agent"`
	Tools              ToolsConfigResponse      `json:"tools"`
	ContextCompression ContextCompressionConfig `json:"context_compression"`
	Plugin             PluginConfig             `json:"plugin"`
	SummaryLLM         LLMConfigResponse        `json:"summary_llm"`
	MinIO              MinIOConfigResponse      `json:"minio"`
	APMPlus            APMPlusConfigResponse    `json:"apmplus"`
}

// LLMConfigResponse LLM配置响应（敏感字段脱敏）
type LLMConfigResponse struct {
	Model    string `json:"model"`
	APIKey   string `json:"api_key,omitempty"`
	BaseURL  string `json:"base_url"`
	Provider string `json:"provider"`
}

// MinIOConfigResponse MinIO配置响应（敏感字段脱敏）
type MinIOConfigResponse struct {
	Endpoint      string `json:"endpoint"`
	AccessKey     string `json:"access_key,omitempty"`
	SecretKey     string `json:"secret_key,omitempty"`
	Bucket        string `json:"bucket"`
	UseSSL        bool   `json:"use_ssl"`
	PublicBaseURL string `json:"public_base_url"`
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
	globalConfig          *Config
	globalRuntime         *RuntimeConfig
	configMu              sync.RWMutex
	configFilePath        string
	minioRuntimeRefresher func(MinIOConfig) error
)

// RegisterMinIORuntimeRefresher 注册 MinIO 运行时刷新函数（由存储层在启动时注入）。
func RegisterMinIORuntimeRefresher(refresher func(MinIOConfig) error) {
	configMu.Lock()
	defer configMu.Unlock()
	minioRuntimeRefresher = refresher
}

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
		MinIO:              cfg.MinIO,
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

// GetRuntimeConfigSnapshot returns a copy of the current runtime config.
func GetRuntimeConfigSnapshot() RuntimeConfig {
	configMu.RLock()
	defer configMu.RUnlock()
	if globalRuntime == nil {
		return RuntimeConfig{}
	}
	return *globalRuntime
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
	if globalConfig == nil || globalRuntime == nil {
		return nil
	}

	// 先在副本上应用更新，确保保存和刷新成功前不污染内存态。
	stagedRuntime := *globalRuntime
	stagedConfig := *globalConfig
	previousMinIO := globalConfig.MinIO
	minioChanged := false

	if agentUpdates, ok := updates["agent"].(map[string]interface{}); ok {
		updateAgentConfig(&stagedRuntime.Agent, agentUpdates)
		stagedConfig.Agent = stagedRuntime.Agent
	}
	if toolsUpdates, ok := updates["tools"].(map[string]interface{}); ok {
		updateToolsConfig(&stagedRuntime.Tools, toolsUpdates)
		stagedConfig.Tools = stagedRuntime.Tools
	}
	if ccUpdates, ok := updates["context_compression"].(map[string]interface{}); ok {
		updateContextCompressionConfig(&stagedRuntime.ContextCompression, ccUpdates)
		stagedConfig.ContextCompression = stagedRuntime.ContextCompression
	}
	if pluginUpdates, ok := updates["plugin"].(map[string]interface{}); ok {
		updatePluginConfig(&stagedRuntime.Plugin, pluginUpdates)
		stagedConfig.Plugin = stagedRuntime.Plugin
	}
	if summaryLLMUpdates, ok := updates["summary_llm"].(map[string]interface{}); ok {
		updateLLMConfig(&stagedRuntime.SummaryLLM, summaryLLMUpdates)
		stagedConfig.SummaryLLM = stagedRuntime.SummaryLLM
	}
	if minioUpdates, ok := updates["minio"].(map[string]interface{}); ok {
		updateMinIOConfig(&stagedRuntime.MinIO, minioUpdates)
		stagedConfig.MinIO = stagedRuntime.MinIO
		minioChanged = true
	}
	if apmPlusUpdates, ok := updates["apmplus"].(map[string]interface{}); ok {
		updateAPMPlusConfig(&stagedRuntime.APMPlus, apmPlusUpdates)
		stagedConfig.APMPlus = stagedRuntime.APMPlus
	}
	if err := ValidateContextCompressionConfig(stagedRuntime.ContextCompression); err != nil {
		return err
	}

	// MinIO 变更先做运行时刷新校验，避免把不可用配置写入磁盘。
	if minioChanged {
		if err := refreshMinIORuntimeLocked(stagedConfig.MinIO); err != nil {
			return err
		}
	}
	// 落盘失败时回滚已刷新成功的 MinIO 运行时，避免运行态和持久化态分叉。
	if err := saveConfigToFileWith(&stagedConfig); err != nil {
		if minioChanged {
			if rollbackErr := refreshMinIORuntimeLocked(previousMinIO); rollbackErr != nil {
				return fmt.Errorf("failed to save config: %w; additionally failed to rollback minio runtime: %v", err, rollbackErr)
			}
		}
		return err
	}
	globalConfig = &stagedConfig
	globalRuntime = &stagedRuntime
	return nil
}

// saveConfigToFile 保存配置到文件
func saveConfigToFile() error {
	return saveConfigToFileWith(globalConfig)
}

func saveConfigToFileWith(cfg *Config) error {
	if configFilePath == "" {
		return nil
	}
	if cfg == nil {
		return nil
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(configFilePath, data, 0644)
}

// ReloadConfig 重新加载配置
func ReloadConfig() error {
	configMu.Lock()
	defer configMu.Unlock()

	// 重新严格加载配置文件，避免读取失败时静默回退为空配置。
	cfg, err := LoadConfigStrict(configFilePath)
	if err != nil {
		return err
	}
	// 先验证运行时存储客户端可重建，再替换全局配置，避免半更新状态。
	if err := refreshMinIORuntimeLocked(cfg.MinIO); err != nil {
		return err
	}
	globalConfig = cfg

	globalRuntime = &RuntimeConfig{
		Agent:              cfg.Agent,
		Tools:              cfg.Tools,
		ContextCompression: cfg.ContextCompression,
		Plugin:             cfg.Plugin,
		SummaryLLM:         cfg.SummaryLLM,
		MinIO:              cfg.MinIO,
		APMPlus:            cfg.APMPlus,
	}

	return nil
}

func refreshMinIORuntimeLocked(minioCfg MinIOConfig) error {
	if minioRuntimeRefresher == nil {
		return nil
	}
	return minioRuntimeRefresher(minioCfg)
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
		MinIO: MinIOConfigResponse{
			Endpoint:      r.MinIO.Endpoint,
			AccessKey:     maskAPIKey(r.MinIO.AccessKey),
			SecretKey:     maskAPIKey(r.MinIO.SecretKey),
			Bucket:        r.MinIO.Bucket,
			UseSSL:        r.MinIO.UseSSL,
			PublicBaseURL: r.MinIO.PublicBaseURL,
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

// EffectiveContextCompressionEnabled returns the runtime behavior for enabled when config omits it.
func EffectiveContextCompressionEnabled(cfg ContextCompressionConfig) bool {
	if cfg.Enabled != nil {
		return *cfg.Enabled
	}
	return true
}

// ValidateContextCompressionConfig validates context compression invariants before committing config changes.
func ValidateContextCompressionConfig(cfg ContextCompressionConfig) error {
	if cfg.HardLimitTokens <= 0 {
		return fmt.Errorf("context_compression.hard_limit_tokens must be greater than 0")
	}
	if cfg.SoftLimitTokens <= 0 || cfg.SoftLimitTokens >= cfg.HardLimitTokens {
		return fmt.Errorf("context_compression.soft_limit_tokens must be greater than 0 and less than hard_limit_tokens")
	}
	if cfg.OutputReserveTokens <= 0 {
		return fmt.Errorf("context_compression.output_reserve_tokens must be greater than 0")
	}
	if cfg.MaxRecentMessages <= 0 {
		return fmt.Errorf("context_compression.max_recent_messages must be greater than 0")
	}
	if cfg.EstimatedCharsPerToken <= 0 {
		return fmt.Errorf("context_compression.estimated_chars_per_token must be greater than 0")
	}
	return nil
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

func updateMinIOConfig(cfg *MinIOConfig, updates map[string]interface{}) {
	if v, ok := updates["endpoint"].(string); ok {
		cfg.Endpoint = v
	}
	if v, ok := updates["access_key"].(string); ok {
		cfg.AccessKey = v
	}
	if v, ok := updates["secret_key"].(string); ok {
		cfg.SecretKey = v
	}
	if v, ok := updates["bucket"].(string); ok {
		cfg.Bucket = v
	}
	if v, ok := updates["use_ssl"].(bool); ok {
		cfg.UseSSL = v
	}
	if v, ok := updates["public_base_url"].(string); ok {
		cfg.PublicBaseURL = v
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
