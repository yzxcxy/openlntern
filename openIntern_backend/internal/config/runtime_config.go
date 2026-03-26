package config

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
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

// OpenVikingServiceConfig OpenViking服务配置（来自 ov.conf）
type OpenVikingServiceConfig struct {
	Storage  OpenVikingStorageConfig  `json:"storage,omitempty" yaml:"storage,omitempty"`
	Log      OpenVikingLogConfig      `json:"log,omitempty" yaml:"log,omitempty"`
	Embedding OpenVikingEmbeddingConfig `json:"embedding,omitempty" yaml:"embedding,omitempty"`
	VLM      OpenVikingVLMConfig      `json:"vlm,omitempty" yaml:"vlm,omitempty"`
	Parsers  OpenVikingParsersConfig  `json:"parsers,omitempty" yaml:"parsers,omitempty"`
	Feishu   OpenVikingFeishuConfig   `json:"feishu,omitempty" yaml:"feishu,omitempty"`
	Rerank   OpenVikingRerankConfig   `json:"rerank,omitempty" yaml:"rerank,omitempty"`
}

type OpenVikingStorageConfig struct {
	Workspace string                    `json:"workspace,omitempty" yaml:"workspace,omitempty"`
	VectorDB  OpenVikingVectorDBConfig  `json:"vectordb,omitempty" yaml:"vectordb,omitempty"`
	AGFS      OpenVikingAGFSConfig      `json:"agfs,omitempty" yaml:"agfs,omitempty"`
}

type OpenVikingVectorDBConfig struct {
	Name    string `json:"name,omitempty" yaml:"name,omitempty"`
	Backend string `json:"backend,omitempty" yaml:"backend,omitempty"`
}

type OpenVikingAGFSConfig struct {
	Port     int    `json:"port,omitempty" yaml:"port,omitempty"`
	LogLevel string `json:"log_level,omitempty" yaml:"log_level,omitempty"`
	Backend  string `json:"backend,omitempty" yaml:"backend,omitempty"`
}

type OpenVikingLogConfig struct {
	Level  string `json:"level,omitempty" yaml:"level,omitempty"`
	Output string `json:"output,omitempty" yaml:"output,omitempty"`
}

type OpenVikingEmbeddingConfig struct {
	Dense        OpenVikingDenseEmbeddingConfig  `json:"dense,omitempty" yaml:"dense,omitempty"`
	Sparse       OpenVikingSparseEmbeddingConfig `json:"sparse,omitempty" yaml:"sparse,omitempty"`
	Hybrid       OpenVikingHybridEmbeddingConfig `json:"hybrid,omitempty" yaml:"hybrid,omitempty"`
	MaxConcurrent int                             `json:"max_concurrent,omitempty" yaml:"max_concurrent,omitempty"`
}

type OpenVikingDenseEmbeddingConfig struct {
	APIBase       string            `json:"api_base,omitempty" yaml:"api_base,omitempty"`
	APIKey        string            `json:"api_key,omitempty" yaml:"api_key,omitempty"`
	Provider      string            `json:"provider,omitempty" yaml:"provider,omitempty"`
	Dimension     int               `json:"dimension,omitempty" yaml:"dimension,omitempty"`
	Model         string            `json:"model,omitempty" yaml:"model,omitempty"`
	Input         string            `json:"input,omitempty" yaml:"input,omitempty"`
	BatchSize     int               `json:"batch_size,omitempty" yaml:"batch_size,omitempty"`
	QueryParam    string            `json:"query_param,omitempty" yaml:"query_param,omitempty"`
	DocumentParam string            `json:"document_param,omitempty" yaml:"document_param,omitempty"`
	ExtraHeaders  map[string]string `json:"extra_headers,omitempty" yaml:"extra_headers,omitempty"`
	AK            string            `json:"ak,omitempty" yaml:"ak,omitempty"` // vikingdb provider
	SK            string            `json:"sk,omitempty" yaml:"sk,omitempty"` // vikingdb provider
	Region        string            `json:"region,omitempty" yaml:"region,omitempty"` // vikingdb provider
}

type OpenVikingSparseEmbeddingConfig struct {
	APIBase  string `json:"api_base,omitempty" yaml:"api_base,omitempty"`
	APIKey   string `json:"api_key,omitempty" yaml:"api_key,omitempty"`
	Provider string `json:"provider,omitempty" yaml:"provider,omitempty"`
	Model    string `json:"model,omitempty" yaml:"model,omitempty"`
}

type OpenVikingHybridEmbeddingConfig struct {
	APIBase   string `json:"api_base,omitempty" yaml:"api_base,omitempty"`
	APIKey    string `json:"api_key,omitempty" yaml:"api_key,omitempty"`
	Provider  string `json:"provider,omitempty" yaml:"provider,omitempty"`
	Model     string `json:"model,omitempty" yaml:"model,omitempty"`
	Dimension int    `json:"dimension,omitempty" yaml:"dimension,omitempty"`
}

type OpenVikingVLMConfig struct {
	APIBase      string            `json:"api_base,omitempty" yaml:"api_base,omitempty"`
	APIKey       string            `json:"api_key,omitempty" yaml:"api_key,omitempty"`
	Provider     string            `json:"provider,omitempty" yaml:"provider,omitempty"`
	Model        string            `json:"model,omitempty" yaml:"model,omitempty"`
	MaxConcurrent int               `json:"max_concurrent,omitempty" yaml:"max_concurrent,omitempty"`
	Thinking     bool              `json:"thinking,omitempty" yaml:"thinking,omitempty"`
	Stream       bool              `json:"stream,omitempty" yaml:"stream,omitempty"`
	ExtraHeaders map[string]string `json:"extra_headers,omitempty" yaml:"extra_headers,omitempty"`
}

type OpenVikingParsersConfig struct {
	Code OpenVikingCodeParserConfig `json:"code,omitempty" yaml:"code,omitempty"`
}

type OpenVikingCodeParserConfig struct {
	CodeSummaryMode string `json:"code_summary_mode,omitempty" yaml:"code_summary_mode,omitempty"`
}

type OpenVikingFeishuConfig struct {
	AppID              string `json:"app_id,omitempty" yaml:"app_id,omitempty"`
	AppSecret          string `json:"app_secret,omitempty" yaml:"app_secret,omitempty"`
	Domain             string `json:"domain,omitempty" yaml:"domain,omitempty"`
	MaxRowsPerSheet    int    `json:"max_rows_per_sheet,omitempty" yaml:"max_rows_per_sheet,omitempty"`
	MaxRecordsPerTable int    `json:"max_records_per_table,omitempty" yaml:"max_records_per_table,omitempty"`
}

type OpenVikingRerankConfig struct {
	APIBase   string `json:"api_base,omitempty" yaml:"api_base,omitempty"`
	APIKey    string `json:"api_key,omitempty" yaml:"api_key,omitempty"`
	Provider  string `json:"provider,omitempty" yaml:"provider,omitempty"`
	Model     string `json:"model,omitempty" yaml:"model,omitempty"`
	Threshold float64 `json:"threshold,omitempty" yaml:"threshold,omitempty"`
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
	OpenVikingService  OpenVikingServiceConfigResponse `json:"openviking_service"`
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
	Sandbox    SandboxConfig        `json:"sandbox"`
	Memory     MemoryProviderConfig `json:"memory"`
	OpenViking OpenVikingConfigResponse `json:"openviking"`
}

// OpenVikingConfigResponse OpenViking配置响应（敏感字段脱敏）
type OpenVikingConfigResponse struct {
	BaseURL                    string `json:"base_url"`
	APIKey                     string `json:"api_key,omitempty"`
	SkillsRoot                 string `json:"skills_root"`
	ToolsRoot                  string `json:"tools_root"`
	TimeoutSeconds             int    `json:"timeout_seconds"`
	MemorySearchTimeoutSeconds int    `json:"memory_search_timeout_seconds"`
	MemorySyncDelaySeconds     int    `json:"memory_sync_delay_seconds"`
	MemorySyncPollSeconds      int    `json:"memory_sync_poll_seconds"`
	MemorySyncTimeoutSeconds   int    `json:"memory_sync_timeout_seconds"`
	MemorySyncRetrySeconds     int    `json:"memory_sync_retry_seconds"`
}

// OpenVikingServiceConfigResponse OpenViking服务配置响应（敏感字段脱敏）
type OpenVikingServiceConfigResponse struct {
	Storage  OpenVikingStorageConfigResponse  `json:"storage"`
	Log      OpenVikingLogConfig              `json:"log"`
	Embedding OpenVikingEmbeddingConfigResponse `json:"embedding"`
	VLM      OpenVikingVLMConfigResponse      `json:"vlm"`
	Parsers  OpenVikingParsersConfig          `json:"parsers"`
	Feishu   OpenVikingFeishuConfigResponse   `json:"feishu"`
	Rerank   OpenVikingRerankConfigResponse   `json:"rerank"`
}

type OpenVikingStorageConfigResponse struct {
	Workspace string                   `json:"workspace"`
	VectorDB  OpenVikingVectorDBConfig `json:"vectordb"`
	AGFS      OpenVikingAGFSConfig     `json:"agfs"`
}

type OpenVikingEmbeddingConfigResponse struct {
	Dense        OpenVikingDenseEmbeddingConfigResponse  `json:"dense"`
	Sparse       OpenVikingSparseEmbeddingConfigResponse `json:"sparse"`
	Hybrid       OpenVikingHybridEmbeddingConfigResponse `json:"hybrid"`
	MaxConcurrent int                                    `json:"max_concurrent"`
}

type OpenVikingDenseEmbeddingConfigResponse struct {
	APIBase       string            `json:"api_base"`
	APIKey        string            `json:"api_key,omitempty"`
	Provider      string            `json:"provider"`
	Dimension     int               `json:"dimension"`
	Model         string            `json:"model"`
	Input         string            `json:"input"`
	BatchSize     int               `json:"batch_size"`
	QueryParam    string            `json:"query_param"`
	DocumentParam string            `json:"document_param"`
	ExtraHeaders  map[string]string `json:"extra_headers"`
	AK            string            `json:"ak,omitempty"` // vikingdb - 脱敏
	SK            string            `json:"sk,omitempty"` // vikingdb - 脱敏
	Region        string            `json:"region"`
}

type OpenVikingSparseEmbeddingConfigResponse struct {
	APIBase  string `json:"api_base"`
	APIKey   string `json:"api_key,omitempty"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

type OpenVikingHybridEmbeddingConfigResponse struct {
	APIBase   string `json:"api_base"`
	APIKey    string `json:"api_key,omitempty"`
	Provider  string `json:"provider"`
	Model     string `json:"model"`
	Dimension int    `json:"dimension"`
}

type OpenVikingVLMConfigResponse struct {
	APIBase      string            `json:"api_base"`
	APIKey       string            `json:"api_key,omitempty"`
	Provider     string            `json:"provider"`
	Model        string            `json:"model"`
	MaxConcurrent int               `json:"max_concurrent"`
	Thinking     bool              `json:"thinking"`
	Stream       bool              `json:"stream"`
	ExtraHeaders map[string]string `json:"extra_headers"`
}

type OpenVikingFeishuConfigResponse struct {
	AppID              string `json:"app_id"`
	AppSecret          string `json:"app_secret,omitempty"`
	Domain             string `json:"domain"`
	MaxRowsPerSheet    int    `json:"max_rows_per_sheet"`
	MaxRecordsPerTable int    `json:"max_records_per_table"`
}

type OpenVikingRerankConfigResponse struct {
	APIBase   string  `json:"api_base"`
	APIKey    string  `json:"api_key,omitempty"`
	Provider  string  `json:"provider"`
	Model     string  `json:"model"`
	Threshold float64 `json:"threshold"`
}

// 全局配置管理
var (
	globalConfig      *Config
	globalRuntime     *RuntimeConfig
	globalOpenViking  *OpenVikingServiceConfig
	configMu          sync.RWMutex
	configFilePath    string
	openVikingConfigPath string
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

	// 设置 OpenViking 配置路径（和 config.yaml 同级目录）
	openVikingConfigPath = filepath.Join(filepath.Dir(cfgPath), "ov.conf")

	// 加载 OpenViking 配置
	globalOpenViking = loadOpenVikingConfig(openVikingConfigPath)
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

// GetOpenVikingServiceConfig 获取 OpenViking 服务配置
func GetOpenVikingServiceConfig() *OpenVikingServiceConfig {
	configMu.RLock()
	defer configMu.RUnlock()
	if globalOpenViking == nil {
		return &OpenVikingServiceConfig{}
	}
	return globalOpenViking
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

// UpdateOpenVikingServiceConfig 更新 OpenViking 服务配置
func UpdateOpenVikingServiceConfig(updates map[string]interface{}) error {
	configMu.Lock()
	defer configMu.Unlock()

	// 更新配置
	if storageUpdates, ok := updates["storage"].(map[string]interface{}); ok {
		updateStorageConfig(globalOpenViking, storageUpdates)
	}
	if logUpdates, ok := updates["log"].(map[string]interface{}); ok {
		if level, ok := logUpdates["level"].(string); ok {
			globalOpenViking.Log.Level = level
		}
		if output, ok := logUpdates["output"].(string); ok {
			globalOpenViking.Log.Output = output
		}
	}
	if embeddingUpdates, ok := updates["embedding"].(map[string]interface{}); ok {
		updateEmbeddingConfig(globalOpenViking, embeddingUpdates)
	}
	if vlmUpdates, ok := updates["vlm"].(map[string]interface{}); ok {
		updateVLMConfig(globalOpenViking, vlmUpdates)
	}
	if parsersUpdates, ok := updates["parsers"].(map[string]interface{}); ok {
		if codeUpdates, ok := parsersUpdates["code"].(map[string]interface{}); ok {
			if mode, ok := codeUpdates["code_summary_mode"].(string); ok {
				globalOpenViking.Parsers.Code.CodeSummaryMode = mode
			}
		}
	}
	if rerankUpdates, ok := updates["rerank"].(map[string]interface{}); ok {
		updateRerankConfig(globalOpenViking, rerankUpdates)
	}

	// 写回 OpenViking 配置文件
	return saveOpenVikingConfigToFile()
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

// saveOpenVikingConfigToFile 保存 OpenViking 配置到文件
func saveOpenVikingConfigToFile() error {
	if openVikingConfigPath == "" {
		return nil
	}

	// 确保目录存在
	dir := filepath.Dir(openVikingConfigPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// 构建只包含有效配置的 map，避免序列化空结构体
	configMap := make(map[string]interface{})

	// Storage
	if globalOpenViking.Storage.Workspace != "" || globalOpenViking.Storage.VectorDB.Name != "" || globalOpenViking.Storage.AGFS.Backend != "" {
		storageMap := make(map[string]interface{})
		if globalOpenViking.Storage.Workspace != "" {
			storageMap["workspace"] = globalOpenViking.Storage.Workspace
		}
		if globalOpenViking.Storage.VectorDB.Name != "" || globalOpenViking.Storage.VectorDB.Backend != "" {
			vdbMap := make(map[string]interface{})
			if globalOpenViking.Storage.VectorDB.Name != "" {
				vdbMap["name"] = globalOpenViking.Storage.VectorDB.Name
			}
			if globalOpenViking.Storage.VectorDB.Backend != "" {
				vdbMap["backend"] = globalOpenViking.Storage.VectorDB.Backend
			}
			storageMap["vectordb"] = vdbMap
		}
		if globalOpenViking.Storage.AGFS.Backend != "" || globalOpenViking.Storage.AGFS.Port != 0 || globalOpenViking.Storage.AGFS.LogLevel != "" {
			agfsMap := make(map[string]interface{})
			if globalOpenViking.Storage.AGFS.Port != 0 {
				agfsMap["port"] = globalOpenViking.Storage.AGFS.Port
			}
			if globalOpenViking.Storage.AGFS.LogLevel != "" {
				agfsMap["log_level"] = globalOpenViking.Storage.AGFS.LogLevel
			}
			if globalOpenViking.Storage.AGFS.Backend != "" {
				agfsMap["backend"] = globalOpenViking.Storage.AGFS.Backend
			}
			storageMap["agfs"] = agfsMap
		}
		configMap["storage"] = storageMap
	}

	// Log
	if globalOpenViking.Log.Level != "" || globalOpenViking.Log.Output != "" {
		logMap := make(map[string]interface{})
		if globalOpenViking.Log.Level != "" {
			logMap["level"] = globalOpenViking.Log.Level
		}
		if globalOpenViking.Log.Output != "" {
			logMap["output"] = globalOpenViking.Log.Output
		}
		configMap["log"] = logMap
	}

	// Embedding - 只保存 dense，跳过空的 sparse/hybrid
	if globalOpenViking.Embedding.Dense.Model != "" || globalOpenViking.Embedding.Dense.APIBase != "" || globalOpenViking.Embedding.MaxConcurrent != 0 {
		embeddingMap := make(map[string]interface{})
		if globalOpenViking.Embedding.Dense.Model != "" || globalOpenViking.Embedding.Dense.APIBase != "" {
			denseMap := make(map[string]interface{})
			if globalOpenViking.Embedding.Dense.APIBase != "" {
				denseMap["api_base"] = globalOpenViking.Embedding.Dense.APIBase
			}
			if globalOpenViking.Embedding.Dense.APIKey != "" {
				denseMap["api_key"] = globalOpenViking.Embedding.Dense.APIKey
			}
			if globalOpenViking.Embedding.Dense.Provider != "" {
				denseMap["provider"] = globalOpenViking.Embedding.Dense.Provider
			}
			if globalOpenViking.Embedding.Dense.Dimension != 0 {
				denseMap["dimension"] = globalOpenViking.Embedding.Dense.Dimension
			}
			if globalOpenViking.Embedding.Dense.Model != "" {
				denseMap["model"] = globalOpenViking.Embedding.Dense.Model
			}
			if globalOpenViking.Embedding.Dense.Input != "" {
				denseMap["input"] = globalOpenViking.Embedding.Dense.Input
			}
			if globalOpenViking.Embedding.Dense.BatchSize != 0 {
				denseMap["batch_size"] = globalOpenViking.Embedding.Dense.BatchSize
			}
			if globalOpenViking.Embedding.Dense.QueryParam != "" {
				denseMap["query_param"] = globalOpenViking.Embedding.Dense.QueryParam
			}
			if globalOpenViking.Embedding.Dense.DocumentParam != "" {
				denseMap["document_param"] = globalOpenViking.Embedding.Dense.DocumentParam
			}
			if len(globalOpenViking.Embedding.Dense.ExtraHeaders) > 0 {
				denseMap["extra_headers"] = globalOpenViking.Embedding.Dense.ExtraHeaders
			}
			if globalOpenViking.Embedding.Dense.AK != "" {
				denseMap["ak"] = globalOpenViking.Embedding.Dense.AK
			}
			if globalOpenViking.Embedding.Dense.SK != "" {
				denseMap["sk"] = globalOpenViking.Embedding.Dense.SK
			}
			if globalOpenViking.Embedding.Dense.Region != "" {
				denseMap["region"] = globalOpenViking.Embedding.Dense.Region
			}
			embeddingMap["dense"] = denseMap
		}
		// 只有当 sparse 有 model 时才保存
		if globalOpenViking.Embedding.Sparse.Model != "" {
			sparseMap := make(map[string]interface{})
			if globalOpenViking.Embedding.Sparse.APIBase != "" {
				sparseMap["api_base"] = globalOpenViking.Embedding.Sparse.APIBase
			}
			if globalOpenViking.Embedding.Sparse.APIKey != "" {
				sparseMap["api_key"] = globalOpenViking.Embedding.Sparse.APIKey
			}
			if globalOpenViking.Embedding.Sparse.Provider != "" {
				sparseMap["provider"] = globalOpenViking.Embedding.Sparse.Provider
			}
			sparseMap["model"] = globalOpenViking.Embedding.Sparse.Model
			embeddingMap["sparse"] = sparseMap
		}
		// 只有当 hybrid 有 model 时才保存
		if globalOpenViking.Embedding.Hybrid.Model != "" {
			hybridMap := make(map[string]interface{})
			if globalOpenViking.Embedding.Hybrid.APIBase != "" {
				hybridMap["api_base"] = globalOpenViking.Embedding.Hybrid.APIBase
			}
			if globalOpenViking.Embedding.Hybrid.APIKey != "" {
				hybridMap["api_key"] = globalOpenViking.Embedding.Hybrid.APIKey
			}
			if globalOpenViking.Embedding.Hybrid.Provider != "" {
				hybridMap["provider"] = globalOpenViking.Embedding.Hybrid.Provider
			}
			hybridMap["model"] = globalOpenViking.Embedding.Hybrid.Model
			if globalOpenViking.Embedding.Hybrid.Dimension != 0 {
				hybridMap["dimension"] = globalOpenViking.Embedding.Hybrid.Dimension
			}
			embeddingMap["hybrid"] = hybridMap
		}
		if globalOpenViking.Embedding.MaxConcurrent != 0 {
			embeddingMap["max_concurrent"] = globalOpenViking.Embedding.MaxConcurrent
		}
		configMap["embedding"] = embeddingMap
	}

	// VLM
	if globalOpenViking.VLM.Model != "" || globalOpenViking.VLM.APIBase != "" {
		vlmMap := make(map[string]interface{})
		if globalOpenViking.VLM.APIBase != "" {
			vlmMap["api_base"] = globalOpenViking.VLM.APIBase
		}
		if globalOpenViking.VLM.APIKey != "" {
			vlmMap["api_key"] = globalOpenViking.VLM.APIKey
		}
		if globalOpenViking.VLM.Provider != "" {
			vlmMap["provider"] = globalOpenViking.VLM.Provider
		}
		if globalOpenViking.VLM.Model != "" {
			vlmMap["model"] = globalOpenViking.VLM.Model
		}
		if globalOpenViking.VLM.MaxConcurrent != 0 {
			vlmMap["max_concurrent"] = globalOpenViking.VLM.MaxConcurrent
		}
		if globalOpenViking.VLM.Thinking {
			vlmMap["thinking"] = true
		}
		if globalOpenViking.VLM.Stream {
			vlmMap["stream"] = true
		}
		if len(globalOpenViking.VLM.ExtraHeaders) > 0 {
			vlmMap["extra_headers"] = globalOpenViking.VLM.ExtraHeaders
		}
		configMap["vlm"] = vlmMap
	}

	// Parsers
	if globalOpenViking.Parsers.Code.CodeSummaryMode != "" {
		parsersMap := make(map[string]interface{})
		codeMap := make(map[string]interface{})
		codeMap["code_summary_mode"] = globalOpenViking.Parsers.Code.CodeSummaryMode
		parsersMap["code"] = codeMap
		configMap["parsers"] = parsersMap
	}

	// Rerank - 只有当有 model 时才保存
	if globalOpenViking.Rerank.Model != "" {
		rerankMap := make(map[string]interface{})
		if globalOpenViking.Rerank.APIBase != "" {
			rerankMap["api_base"] = globalOpenViking.Rerank.APIBase
		}
		if globalOpenViking.Rerank.APIKey != "" {
			rerankMap["api_key"] = globalOpenViking.Rerank.APIKey
		}
		if globalOpenViking.Rerank.Provider != "" {
			rerankMap["provider"] = globalOpenViking.Rerank.Provider
		}
		rerankMap["model"] = globalOpenViking.Rerank.Model
		if globalOpenViking.Rerank.Threshold != 0 {
			rerankMap["threshold"] = globalOpenViking.Rerank.Threshold
		}
		configMap["rerank"] = rerankMap
	}

	data, err := json.MarshalIndent(configMap, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(openVikingConfigPath, data, 0644)
}

// loadOpenVikingConfig 加载 OpenViking 配置
func loadOpenVikingConfig(path string) *OpenVikingServiceConfig {
	cfg := &OpenVikingServiceConfig{
		Log: OpenVikingLogConfig{
			Level:  "INFO",
			Output: "stdout",
		},
	}

	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("warning: failed to load openviking config from %s: %v", path, err)
		return cfg
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		log.Printf("warning: failed to parse openviking config: %v", err)
	}

	return cfg
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

	// 重新加载 OpenViking 配置
	globalOpenViking = loadOpenVikingConfig(openVikingConfigPath)

	return nil
}

// ToResponse 转换为前端响应格式（脱敏敏感字段）
func (r *RuntimeConfig) ToResponse() RuntimeConfigResponse {
	return RuntimeConfigResponse{
		Agent: r.Agent,
		Tools: ToolsConfigResponse{
			Sandbox: r.Tools.Sandbox,
			Memory:  r.Tools.Memory,
			OpenViking: OpenVikingConfigResponse{
				BaseURL:                    r.Tools.OpenViking.BaseURL,
				APIKey:                     maskAPIKey(r.Tools.OpenViking.APIKey),
				SkillsRoot:                 r.Tools.OpenViking.SkillsRoot,
				ToolsRoot:                  r.Tools.OpenViking.ToolsRoot,
				TimeoutSeconds:             r.Tools.OpenViking.TimeoutSeconds,
				MemorySearchTimeoutSeconds: r.Tools.OpenViking.MemorySearchTimeoutSeconds,
				MemorySyncDelaySeconds:     r.Tools.OpenViking.MemorySyncDelaySeconds,
				MemorySyncPollSeconds:      r.Tools.OpenViking.MemorySyncPollSeconds,
				MemorySyncTimeoutSeconds:   r.Tools.OpenViking.MemorySyncTimeoutSeconds,
				MemorySyncRetrySeconds:     r.Tools.OpenViking.MemorySyncRetrySeconds,
			},
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

// ToResponse 转换为前端响应格式（脱敏敏感字段）
func (c *OpenVikingServiceConfig) ToResponse() OpenVikingServiceConfigResponse {
	return OpenVikingServiceConfigResponse{
		Storage: OpenVikingStorageConfigResponse{
			Workspace: c.Storage.Workspace,
			VectorDB:  c.Storage.VectorDB,
			AGFS:      c.Storage.AGFS,
		},
		Log: c.Log,
		Embedding: OpenVikingEmbeddingConfigResponse{
			Dense: OpenVikingDenseEmbeddingConfigResponse{
				APIBase:       c.Embedding.Dense.APIBase,
				APIKey:        maskAPIKey(c.Embedding.Dense.APIKey),
				Provider:      c.Embedding.Dense.Provider,
				Dimension:     c.Embedding.Dense.Dimension,
				Model:         c.Embedding.Dense.Model,
				Input:         c.Embedding.Dense.Input,
				BatchSize:     c.Embedding.Dense.BatchSize,
				QueryParam:    c.Embedding.Dense.QueryParam,
				DocumentParam: c.Embedding.Dense.DocumentParam,
				ExtraHeaders:  c.Embedding.Dense.ExtraHeaders,
				AK:            maskAPIKey(c.Embedding.Dense.AK),
				SK:            maskAPIKey(c.Embedding.Dense.SK),
				Region:        c.Embedding.Dense.Region,
			},
			Sparse: OpenVikingSparseEmbeddingConfigResponse{
				APIBase:  c.Embedding.Sparse.APIBase,
				APIKey:   maskAPIKey(c.Embedding.Sparse.APIKey),
				Provider: c.Embedding.Sparse.Provider,
				Model:    c.Embedding.Sparse.Model,
			},
			Hybrid: OpenVikingHybridEmbeddingConfigResponse{
				APIBase:   c.Embedding.Hybrid.APIBase,
				APIKey:    maskAPIKey(c.Embedding.Hybrid.APIKey),
				Provider:  c.Embedding.Hybrid.Provider,
				Model:     c.Embedding.Hybrid.Model,
				Dimension: c.Embedding.Hybrid.Dimension,
			},
			MaxConcurrent: c.Embedding.MaxConcurrent,
		},
		VLM: OpenVikingVLMConfigResponse{
			APIBase:       c.VLM.APIBase,
			APIKey:        maskAPIKey(c.VLM.APIKey),
			Provider:      c.VLM.Provider,
			Model:         c.VLM.Model,
			MaxConcurrent: c.VLM.MaxConcurrent,
			Thinking:      c.VLM.Thinking,
			Stream:        c.VLM.Stream,
			ExtraHeaders:  c.VLM.ExtraHeaders,
		},
		Parsers: c.Parsers,
		Feishu: OpenVikingFeishuConfigResponse{
			AppID:              c.Feishu.AppID,
			AppSecret:          maskAPIKey(c.Feishu.AppSecret),
			Domain:             c.Feishu.Domain,
			MaxRowsPerSheet:    c.Feishu.MaxRowsPerSheet,
			MaxRecordsPerTable: c.Feishu.MaxRecordsPerTable,
		},
		Rerank: OpenVikingRerankConfigResponse{
			APIBase:   c.Rerank.APIBase,
			APIKey:    maskAPIKey(c.Rerank.APIKey),
			Provider:  c.Rerank.Provider,
			Model:     c.Rerank.Model,
			Threshold: c.Rerank.Threshold,
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
		if url, ok := sandboxUpdates["url"].(string); ok {
			cfg.Sandbox.Url = url
		}
	}
	if memoryUpdates, ok := updates["memory"].(map[string]interface{}); ok {
		if provider, ok := memoryUpdates["provider"].(string); ok {
			cfg.Memory.Provider = provider
		}
	}
	if ovUpdates, ok := updates["openviking"].(map[string]interface{}); ok {
		updateOpenVikingConfig(&cfg.OpenViking, ovUpdates)
	}
}

func updateOpenVikingConfig(cfg *OpenVikingConfig, updates map[string]interface{}) {
	if v, ok := updates["base_url"].(string); ok {
		cfg.BaseURL = v
	}
	if v, ok := updates["api_key"].(string); ok {
		cfg.APIKey = v
	}
	if v, ok := updates["skills_root"].(string); ok {
		cfg.SkillsRoot = v
	}
	if v, ok := updates["tools_root"].(string); ok {
		cfg.ToolsRoot = v
	}
	if v, ok := updates["timeout_seconds"].(float64); ok {
		cfg.TimeoutSeconds = int(v)
	}
	if v, ok := updates["memory_search_timeout_seconds"].(float64); ok {
		cfg.MemorySearchTimeoutSeconds = int(v)
	}
	if v, ok := updates["memory_sync_delay_seconds"].(float64); ok {
		cfg.MemorySyncDelaySeconds = int(v)
	}
	if v, ok := updates["memory_sync_poll_seconds"].(float64); ok {
		cfg.MemorySyncPollSeconds = int(v)
	}
	if v, ok := updates["memory_sync_timeout_seconds"].(float64); ok {
		cfg.MemorySyncTimeoutSeconds = int(v)
	}
	if v, ok := updates["memory_sync_retry_seconds"].(float64); ok {
		cfg.MemorySyncRetrySeconds = int(v)
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

func updateEmbeddingConfig(cfg *OpenVikingServiceConfig, updates map[string]interface{}) {
	if denseUpdates, ok := updates["dense"].(map[string]interface{}); ok {
		if v, ok := denseUpdates["api_base"].(string); ok {
			cfg.Embedding.Dense.APIBase = v
		}
		if v, ok := denseUpdates["api_key"].(string); ok && v != "" {
			cfg.Embedding.Dense.APIKey = v
		}
		if v, ok := denseUpdates["provider"].(string); ok {
			cfg.Embedding.Dense.Provider = v
		}
		if v, ok := denseUpdates["dimension"].(float64); ok {
			cfg.Embedding.Dense.Dimension = int(v)
		}
		if v, ok := denseUpdates["model"].(string); ok {
			cfg.Embedding.Dense.Model = v
		}
		if v, ok := denseUpdates["input"].(string); ok {
			cfg.Embedding.Dense.Input = v
		}
		if v, ok := denseUpdates["batch_size"].(float64); ok {
			cfg.Embedding.Dense.BatchSize = int(v)
		}
		if v, ok := denseUpdates["query_param"].(string); ok {
			cfg.Embedding.Dense.QueryParam = v
		}
		if v, ok := denseUpdates["document_param"].(string); ok {
			cfg.Embedding.Dense.DocumentParam = v
		}
		if v, ok := denseUpdates["extra_headers"].(map[string]interface{}); ok {
			if cfg.Embedding.Dense.ExtraHeaders == nil {
				cfg.Embedding.Dense.ExtraHeaders = make(map[string]string)
			}
			for key, val := range v {
				if s, ok := val.(string); ok {
					cfg.Embedding.Dense.ExtraHeaders[key] = s
				}
			}
		}
		if v, ok := denseUpdates["ak"].(string); ok && v != "" {
			cfg.Embedding.Dense.AK = v
		}
		if v, ok := denseUpdates["sk"].(string); ok && v != "" {
			cfg.Embedding.Dense.SK = v
		}
		if v, ok := denseUpdates["region"].(string); ok {
			cfg.Embedding.Dense.Region = v
		}
	}
	if sparseUpdates, ok := updates["sparse"].(map[string]interface{}); ok {
		if v, ok := sparseUpdates["api_base"].(string); ok {
			cfg.Embedding.Sparse.APIBase = v
		}
		if v, ok := sparseUpdates["api_key"].(string); ok && v != "" {
			cfg.Embedding.Sparse.APIKey = v
		}
		if v, ok := sparseUpdates["provider"].(string); ok {
			cfg.Embedding.Sparse.Provider = v
		}
		if v, ok := sparseUpdates["model"].(string); ok {
			cfg.Embedding.Sparse.Model = v
		}
	}
	if hybridUpdates, ok := updates["hybrid"].(map[string]interface{}); ok {
		if v, ok := hybridUpdates["api_base"].(string); ok {
			cfg.Embedding.Hybrid.APIBase = v
		}
		if v, ok := hybridUpdates["api_key"].(string); ok && v != "" {
			cfg.Embedding.Hybrid.APIKey = v
		}
		if v, ok := hybridUpdates["provider"].(string); ok {
			cfg.Embedding.Hybrid.Provider = v
		}
		if v, ok := hybridUpdates["model"].(string); ok {
			cfg.Embedding.Hybrid.Model = v
		}
		if v, ok := hybridUpdates["dimension"].(float64); ok {
			cfg.Embedding.Hybrid.Dimension = int(v)
		}
	}
	if v, ok := updates["max_concurrent"].(float64); ok {
		cfg.Embedding.MaxConcurrent = int(v)
	}
}

func updateVLMConfig(cfg *OpenVikingServiceConfig, updates map[string]interface{}) {
	if v, ok := updates["api_base"].(string); ok {
		cfg.VLM.APIBase = v
	}
	if v, ok := updates["api_key"].(string); ok && v != "" {
		cfg.VLM.APIKey = v
	}
	if v, ok := updates["provider"].(string); ok {
		cfg.VLM.Provider = v
	}
	if v, ok := updates["model"].(string); ok {
		cfg.VLM.Model = v
	}
	if v, ok := updates["max_concurrent"].(float64); ok {
		cfg.VLM.MaxConcurrent = int(v)
	}
	if v, ok := updates["thinking"].(bool); ok {
		cfg.VLM.Thinking = v
	}
	if v, ok := updates["stream"].(bool); ok {
		cfg.VLM.Stream = v
	}
	if v, ok := updates["extra_headers"].(map[string]interface{}); ok {
		if cfg.VLM.ExtraHeaders == nil {
			cfg.VLM.ExtraHeaders = make(map[string]string)
		}
		for key, val := range v {
			if s, ok := val.(string); ok {
				cfg.VLM.ExtraHeaders[key] = s
			}
		}
	}
}

func updateStorageConfig(cfg *OpenVikingServiceConfig, updates map[string]interface{}) {
	if v, ok := updates["workspace"].(string); ok {
		cfg.Storage.Workspace = v
	}
	if vectordbUpdates, ok := updates["vectordb"].(map[string]interface{}); ok {
		if v, ok := vectordbUpdates["name"].(string); ok {
			cfg.Storage.VectorDB.Name = v
		}
		if v, ok := vectordbUpdates["backend"].(string); ok {
			cfg.Storage.VectorDB.Backend = v
		}
	}
	if agfsUpdates, ok := updates["agfs"].(map[string]interface{}); ok {
		if v, ok := agfsUpdates["port"].(float64); ok {
			cfg.Storage.AGFS.Port = int(v)
		}
		if v, ok := agfsUpdates["log_level"].(string); ok {
			cfg.Storage.AGFS.LogLevel = v
		}
		if v, ok := agfsUpdates["backend"].(string); ok {
			cfg.Storage.AGFS.Backend = v
		}
	}
}

func updateRerankConfig(cfg *OpenVikingServiceConfig, updates map[string]interface{}) {
	if v, ok := updates["api_base"].(string); ok {
		cfg.Rerank.APIBase = v
	}
	if v, ok := updates["api_key"].(string); ok && v != "" {
		cfg.Rerank.APIKey = v
	}
	if v, ok := updates["provider"].(string); ok {
		cfg.Rerank.Provider = v
	}
	if v, ok := updates["model"].(string); ok {
		cfg.Rerank.Model = v
	}
	if v, ok := updates["threshold"].(float64); ok {
		cfg.Rerank.Threshold = v
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

// GetOpenVikingConfigPath 获取 OpenViking 配置文件路径
func GetOpenVikingConfigPath() string {
	return openVikingConfigPath
}

// GetConfigFilePath 获取配置文件路径
func GetConfigFilePath() string {
	return configFilePath
}