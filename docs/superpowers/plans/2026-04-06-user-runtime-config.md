# User Runtime Config Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将设置页中的 `agent` 和 `context_compression` 改为按登录用户隔离，同时保持 `/v1/config` 的接口形态和系统级配置的全局行为不变。

**Architecture:** 新增一层 `user_runtime_config` 数据库存储，按 `(user_id, config_key)` 保存用户配置块覆盖值；后端在读取 `/v1/config` 和执行 agent 时，统一用“全局默认值 + 用户覆盖值”解析出当前用户的有效配置。由于仓库约束禁止随意新增 Go test 文件和执行 `go test`，本计划使用 `go build`、接口读写回归和双用户手工验证替代 TDD 中的自动化测试步骤。

**Tech Stack:** Go, Gin, GORM, MySQL, Next.js, TypeScript

---

## File Map

**Create:**

- `openIntern_backend/internal/models/user_runtime_config.go`
- `openIntern_backend/internal/dao/user_runtime_config.go`
- `openIntern_backend/internal/services/config/user_runtime_config.go`

**Modify:**

- `openIntern_backend/internal/database/database.go`
- `openIntern_backend/internal/controllers/config.go`
- `openIntern_backend/internal/config/runtime_config.go`
- `openIntern_backend/internal/services/agent/agent_service.go`
- `openIntern_backend/internal/services/agent/agent_init.go`
- `openIntern_backend/internal/services/agent/agent_entry.go`
- `openIntern_backend/internal/services/agent/context_compression.go`
- `openIntern_backend/main.go`
- `openIntern_forentend/app/(workspace)/settings/page.tsx`

**Verification targets:**

- `go build ./...` from `openIntern_backend`
- authenticated `GET /v1/config`
- authenticated `PUT /v1/config` for two different users
- manual chat run to confirm user-specific `max_iterations` and compression settings take effect

### Task 1: Add The User Runtime Config Persistence Layer

**Files:**

- Create: `openIntern_backend/internal/models/user_runtime_config.go`
- Create: `openIntern_backend/internal/dao/user_runtime_config.go`
- Modify: `openIntern_backend/internal/database/database.go`

- [ ] **Step 1: Add the GORM model**

```go
package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type UserRuntimeConfig struct {
	ID          uint           `gorm:"primarykey" json:"-"`
	UserID      string         `gorm:"column:user_id;uniqueIndex:ux_user_runtime_config_user_key,priority:1;index;not null;size:36" json:"user_id"`
	ConfigKey   string         `gorm:"column:config_key;uniqueIndex:ux_user_runtime_config_user_key,priority:2;not null;size:80" json:"config_key"`
	ConfigValue datatypes.JSON `gorm:"column:config_value;type:json;not null" json:"config_value"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}
```

- [ ] **Step 2: Register the model in database migration**

```go
if err := DB.AutoMigrate(
	&models.User{},
	&models.A2UI{},
	&models.Thread{},
	&models.Message{},
	&models.Agent{},
	&models.AgentBinding{},
	&models.MemorySyncState{},
	&models.MemoryUsageLog{},
	&models.SkillFrontmatter{},
	&models.Plugin{},
	&models.Tool{},
	&models.PluginDefault{},
	&models.ModelProvider{},
	&models.ModelCatalog{},
	&models.DefaultModelConfig{},
	&models.ThreadContextSnapshot{},
	&models.SandboxInstance{},
	&models.UserRuntimeConfig{},
); err != nil {
	return fmt.Errorf("migrate database: %w", err)
}
```

- [ ] **Step 3: Add a focused DAO for user config blocks**

```go
type UserRuntimeConfigDAO struct{}

var UserRuntimeConfig = new(UserRuntimeConfigDAO)

func (d *UserRuntimeConfigDAO) GetByUserIDAndKey(userID string, configKey string) (*models.UserRuntimeConfig, error)
func (d *UserRuntimeConfigDAO) ListByUserID(userID string, configKeys []string) ([]models.UserRuntimeConfig, error)
func (d *UserRuntimeConfigDAO) Upsert(userID string, configKey string, configValue []byte) (*models.UserRuntimeConfig, error)
```
```
func (d *UserRuntimeConfigDAO) Upsert(userID string, configKey string, configValue []byte) (*models.UserRuntimeConfig, error) {
	item, err := d.GetByUserIDAndKey(userID, configKey)
	if err != nil {
		return nil, err
	}
	if item == nil {
		created := models.UserRuntimeConfig{
			UserID:      strings.TrimSpace(userID),
			ConfigKey:   strings.TrimSpace(configKey),
			ConfigValue: datatypes.JSON(configValue),
		}
		if err := database.DB.Create(&created).Error; err != nil {
			return nil, err
		}
		return &created, nil
	}
	item.ConfigValue = datatypes.JSON(configValue)
	if err := database.DB.Save(item).Error; err != nil {
		return nil, err
	}
	return item, nil
}
```

- [ ] **Step 4: Verify schema and DAO compile**

Run:

```bash
cd /Users/fqc/project/agent/openIntern/openIntern_backend
go build ./...
```

Expected: build succeeds without output.

- [ ] **Step 5: Commit the persistence layer**

```bash
git -C /Users/fqc/project/agent/openIntern add \
  openIntern_backend/internal/models/user_runtime_config.go \
  openIntern_backend/internal/dao/user_runtime_config.go \
  openIntern_backend/internal/database/database.go
git -C /Users/fqc/project/agent/openIntern commit -m "feat: add user runtime config persistence"
```

### Task 2: Add User Config Validation And Merge Logic

**Files:**

- Create: `openIntern_backend/internal/services/config/user_runtime_config.go`
- Modify: `openIntern_backend/internal/config/runtime_config.go`

- [ ] **Step 1: Define user-config payload types and the whitelist**

```go
package configsvc

import "openIntern/internal/config"

const (
	UserRuntimeConfigKeyAgent              = "agent"
	UserRuntimeConfigKeyContextCompression = "context_compression"
)

type ResolvedUserRuntimeConfig struct {
	Agent              config.AgentConfig
	ContextCompression config.ContextCompressionConfig
}

type AgentConfigPatch struct {
	MaxIterations int `json:"max_iterations"`
}

type ContextCompressionPatch struct {
	Enabled                *bool `json:"enabled"`
	SoftLimitTokens        int   `json:"soft_limit_tokens"`
	HardLimitTokens        int   `json:"hard_limit_tokens"`
	OutputReserveTokens    int   `json:"output_reserve_tokens"`
	MaxRecentMessages      int   `json:"max_recent_messages"`
	EstimatedCharsPerToken int   `json:"estimated_chars_per_token"`
}
```

- [ ] **Step 2: Add validation helpers for `agent` and `context_compression`**

```go
func ValidateAgentPatch(input map[string]any) (*AgentConfigPatch, error) {
	normalized, err := decodeStrict[AgentConfigPatch](input)
	if err != nil {
		return nil, err
	}
	if normalized.MaxIterations < 1 {
		return nil, errors.New("agent.max_iterations must be greater than 0")
	}
	return normalized, nil
}

func ValidateContextCompressionPatch(input map[string]any) (*ContextCompressionPatch, error) {
	normalized, err := decodeStrict[ContextCompressionPatch](input)
	if err != nil {
		return nil, err
	}
	if normalized.HardLimitTokens <= 0 {
		return nil, errors.New("context_compression.hard_limit_tokens must be greater than 0")
	}
	if normalized.SoftLimitTokens <= 0 || normalized.SoftLimitTokens >= normalized.HardLimitTokens {
		return nil, errors.New("context_compression.soft_limit_tokens must be greater than 0 and less than hard_limit_tokens")
	}
	if normalized.OutputReserveTokens <= 0 || normalized.MaxRecentMessages <= 0 || normalized.EstimatedCharsPerToken <= 0 {
		return nil, errors.New("context_compression numeric values must be greater than 0")
	}
	return normalized, nil
}
```

- [ ] **Step 3: Add the resolver that merges defaults with user overrides**

```go
type UserRuntimeConfigService struct{}

var UserRuntimeConfig = new(UserRuntimeConfigService)

func (s *UserRuntimeConfigService) Resolve(userID string) (*ResolvedUserRuntimeConfig, error) {
	defaults := config.GetRuntimeConfig()
	resolved := &ResolvedUserRuntimeConfig{
		Agent:              defaults.Agent,
		ContextCompression: defaults.ContextCompression,
	}
	items, err := dao.UserRuntimeConfig.ListByUserID(userID, []string{
		UserRuntimeConfigKeyAgent,
		UserRuntimeConfigKeyContextCompression,
	})
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		switch item.ConfigKey {
		case UserRuntimeConfigKeyAgent:
			var patch AgentConfigPatch
			if err := json.Unmarshal(item.ConfigValue, &patch); err != nil {
				return nil, err
			}
			resolved.Agent.MaxIterations = patch.MaxIterations
		case UserRuntimeConfigKeyContextCompression:
			var patch ContextCompressionPatch
			if err := json.Unmarshal(item.ConfigValue, &patch); err != nil {
				return nil, err
			}
			resolved.ContextCompression = config.ContextCompressionConfig{
				Enabled:                patch.Enabled,
				SoftLimitTokens:        patch.SoftLimitTokens,
				HardLimitTokens:        patch.HardLimitTokens,
				OutputReserveTokens:    patch.OutputReserveTokens,
				MaxRecentMessages:      patch.MaxRecentMessages,
				EstimatedCharsPerToken: patch.EstimatedCharsPerToken,
			}
		}
	}
	return resolved, nil
}
```

- [ ] **Step 4: Add save helpers that only write allowed config blocks**

```go
func (s *UserRuntimeConfigService) SaveAgent(userID string, input map[string]any) error {
	patch, err := ValidateAgentPatch(input)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(patch)
	if err != nil {
		return err
	}
	_, err = dao.UserRuntimeConfig.Upsert(userID, UserRuntimeConfigKeyAgent, payload)
	return err
}

func (s *UserRuntimeConfigService) SaveContextCompression(userID string, input map[string]any) error {
	patch, err := ValidateContextCompressionPatch(input)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(patch)
	if err != nil {
		return err
	}
	_, err = dao.UserRuntimeConfig.Upsert(userID, UserRuntimeConfigKeyContextCompression, payload)
	return err
}
```

- [ ] **Step 5: Verify the merge layer builds**

Run:

```bash
cd /Users/fqc/project/agent/openIntern/openIntern_backend
go build ./...
```

Expected: build succeeds and the new service has no import cycle.

- [ ] **Step 6: Commit the merge/validation layer**

```bash
git -C /Users/fqc/project/agent/openIntern add \
  openIntern_backend/internal/services/config/user_runtime_config.go \
  openIntern_backend/internal/config/runtime_config.go
git -C /Users/fqc/project/agent/openIntern commit -m "feat: add user runtime config resolver"
```

### Task 3: Split `/v1/config` Into User-Scoped And Global Sections

**Files:**

- Modify: `openIntern_backend/internal/controllers/config.go`

- [ ] **Step 1: Change `GetConfig` to read the authenticated user**

```go
func GetConfig(c *gin.Context) {
	userID := strings.TrimSpace(c.GetString("user_id"))
	globalRuntime := config.GetRuntimeConfig()
	userRuntime, err := configsvc.UserRuntimeConfig.Resolve(userID)
	if err != nil {
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, "failed to load config: "+err.Error())
		return
	}
	runtimeCfg := &config.RuntimeConfig{
		Agent:              userRuntime.Agent,
		Tools:              globalRuntime.Tools,
		ContextCompression: userRuntime.ContextCompression,
		Plugin:             globalRuntime.Plugin,
		SummaryLLM:         globalRuntime.SummaryLLM,
		MinIO:              globalRuntime.MinIO,
		APMPlus:            globalRuntime.APMPlus,
	}
	response.JSONSuccess(c, http.StatusOK, runtimeCfg.ToResponse())
}
```

- [ ] **Step 2: Route user sections to the new persistence service**

```go
func UpdateConfig(c *gin.Context) {
	userID := strings.TrimSpace(c.GetString("user_id"))
	var req UpdateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JSONError(c, http.StatusBadRequest, response.CodeBadRequest, "invalid request: "+err.Error())
		return
	}
	if req.Agent != nil {
		if err := configsvc.UserRuntimeConfig.SaveAgent(userID, req.Agent); err != nil {
			response.JSONError(c, http.StatusBadRequest, response.CodeBadRequest, "failed to update agent config: "+err.Error())
			return
		}
		response.JSONMessage(c, http.StatusOK, "config updated successfully")
		return
	}
	if req.ContextCompression != nil {
		if err := configsvc.UserRuntimeConfig.SaveContextCompression(userID, req.ContextCompression); err != nil {
			response.JSONError(c, http.StatusBadRequest, response.CodeBadRequest, "failed to update context_compression config: "+err.Error())
			return
		}
		response.JSONMessage(c, http.StatusOK, "config updated successfully")
		return
	}
```
```go
	updates := make(map[string]interface{})
	if req.Tools != nil {
		updates["tools"] = req.Tools
	}
	if req.Plugin != nil {
		updates["plugin"] = req.Plugin
	}
	if req.SummaryLLM != nil {
		updates["summary_llm"] = req.SummaryLLM
	}
	if req.MinIO != nil {
		updates["minio"] = req.MinIO
	}
	if req.APMPlus != nil {
		updates["apmplus"] = req.APMPlus
	}
	if len(updates) == 0 {
		response.JSONError(c, http.StatusBadRequest, response.CodeBadRequest, "no config updates provided")
		return
	}
	if err := config.UpdateRuntimeConfig(updates); err != nil {
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, "failed to update config: "+err.Error())
		return
	}
	response.JSONMessage(c, http.StatusOK, "config updated successfully")
}
```

- [ ] **Step 3: Make the controller reject mixed user/global updates**

```go
if req.Agent != nil && (req.Tools != nil || req.Plugin != nil || req.SummaryLLM != nil || req.MinIO != nil || req.APMPlus != nil || req.ContextCompression != nil) {
	response.JSONError(c, http.StatusBadRequest, response.CodeBadRequest, "agent config must be updated separately")
	return
}
if req.ContextCompression != nil && (req.Tools != nil || req.Plugin != nil || req.SummaryLLM != nil || req.MinIO != nil || req.APMPlus != nil) {
	response.JSONError(c, http.StatusBadRequest, response.CodeBadRequest, "context_compression config must be updated separately")
	return
}
```

- [ ] **Step 4: Verify config handlers compile**

Run:

```bash
cd /Users/fqc/project/agent/openIntern/openIntern_backend
go build ./...
```

Expected: build succeeds with the new controller imports and branches.

- [ ] **Step 5: Commit the controller changes**

```bash
git -C /Users/fqc/project/agent/openIntern add \
  openIntern_backend/internal/controllers/config.go
git -C /Users/fqc/project/agent/openIntern commit -m "feat: split user and global config updates"
```

### Task 4: Make Agent Execution Read User-Specific Runtime Config

**Files:**

- Modify: `openIntern_backend/internal/services/agent/agent_service.go`
- Modify: `openIntern_backend/internal/services/agent/agent_init.go`
- Modify: `openIntern_backend/internal/services/agent/agent_entry.go`
- Modify: `openIntern_backend/internal/services/agent/context_compression.go`
- Modify: `openIntern_backend/main.go`

- [ ] **Step 1: Stop storing per-user values inside global agent state**

```go
type UserRuntimeConfigResolver interface {
	Resolve(userID string) (*configsvc.ResolvedUserRuntimeConfig, error)
}

type Dependencies struct {
	A2UIService                builtinTool.A2UIServiceInterface
	FileUploader               builtinTool.FileUploader
	MessageStore               MessageStore
	MemoryRetriever            MemoryRetriever
	MemorySyncStateStore       MemorySyncStateStore
	ThreadContextSnapshotStore ThreadContextSnapshotStore
	ThreadStore                ThreadStore
	ModelCatalogResolver       ModelCatalogResolver
	ModelProviderResolver      ModelProviderKeyResolver
	SkillFrontmatterStore      skillmiddleware.SkillFrontmatterStore
	UserRuntimeConfigResolver  UserRuntimeConfigResolver
}

type runtimeState struct {
	apmplusShutdown  func(context.Context) error
	summaryModel     einoModel.ToolCallingChatModel
	staticAgentTools []einoTool.BaseTool
	agentHandlers    []adk.ChatModelAgentMiddleware
}
```

- [ ] **Step 2: Inject the resolver into the default service**

```go
var defaultService = NewService(Dependencies{
	A2UIService:                a2uisvc.A2UI,
	FileUploader:               storagesvc.File,
	MessageStore:               chatsvc.Message,
	MemoryRetriever:            memorysvc.MemoryRetriever,
	MemorySyncStateStore:       memorysvc.MemorySyncState,
	ThreadContextSnapshotStore: chatsvc.ThreadContextSnapshot,
	ThreadStore:                chatsvc.Thread,
	ModelCatalogResolver:       modelCatalogResolverAdapter{},
	ModelProviderResolver:      modelsvc.ModelProvider,
	SkillFrontmatterStore:      skillsvc.FrontmatterStoreAdapter{Store: skillsvc.SkillFrontmatter},
	UserRuntimeConfigResolver:  configsvc.UserRuntimeConfig,
})
```

- [ ] **Step 3: Resolve user config at run time and use it for max iterations**

```go
userRuntime, err := s.deps.UserRuntimeConfigResolver.Resolve(ownerID)
if err != nil {
	return err
}
maxIterations := userRuntime.Agent.MaxIterations
if maxIterations <= 0 {
	maxIterations = 10
}
```

- [ ] **Step 4: Build compression settings from the resolved user config instead of global state**

```go
compressionSettings := newContextCompressionSettings(userRuntime.ContextCompression)
preparedInput, compressionStats, err := s.compressInputContext(ctx, mergedInput, runtimeConfig, state, compressionSettings)
```
```go
func (s *Service) compressInputContext(
	ctx context.Context,
	input *types.RunAgentInput,
	runtimeConfig *AgentRuntimeConfig,
	state runtimeState,
	settings contextCompressionSettings,
) (*types.RunAgentInput, *contextCompressionStats, error) {
```

- [ ] **Step 5: Remove the now-unused compression and max-iteration initialization wiring**

```go
s.setState(runtimeState{
	apmplusShutdown: shutdown,
	summaryModel:    runtimeSummaryModel,
	staticAgentTools: allTools,
	agentHandlers:   []adk.ChatModelAgentMiddleware{patchToolCallsMiddleware, skillMiddleware},
})
```

- [ ] **Step 6: Verify backend build after runtime wiring changes**

Run:

```bash
cd /Users/fqc/project/agent/openIntern/openIntern_backend
go build ./...
```

Expected: build succeeds and no agent file still references `state.maxIterations` or `state.contextCompression`.

- [ ] **Step 7: Commit the runtime changes**

```bash
git -C /Users/fqc/project/agent/openIntern add \
  openIntern_backend/internal/services/agent/agent_service.go \
  openIntern_backend/internal/services/agent/agent_init.go \
  openIntern_backend/internal/services/agent/agent_entry.go \
  openIntern_backend/internal/services/agent/context_compression.go \
  openIntern_backend/main.go
git -C /Users/fqc/project/agent/openIntern commit -m "feat: resolve user runtime config during agent runs"
```

### Task 5: Keep The Settings Page Behavior Stable And Verify With Two Users

**Files:**

- Modify: `openIntern_forentend/app/(workspace)/settings/page.tsx`

- [ ] **Step 1: Keep the frontend contract unchanged, but make success messages explicit per section**

```tsx
const handleSave = async (section: string, updates: Record<string, unknown>) => {
  if (!getValidToken()) return;
  setSaving(true);
  setError("");
  setSuccess("");
  try {
    await requestBackend("/v1/config", {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ [section]: updates }),
      fallbackMessage: "更新配置失败",
      router,
    });
    setSuccess(
      section === "agent"
        ? "Agent 设置保存成功"
        : section === "context_compression"
          ? "高级设置保存成功"
          : "配置保存成功"
    );
    await fetchConfig();
  } catch (err) {
```

- [ ] **Step 2: Manually verify user isolation through the API**

Run user A flow:

```bash
curl -s http://127.0.0.1:8080/v1/config \
  -H "Authorization: Bearer <USER_A_TOKEN>"
curl -s http://127.0.0.1:8080/v1/config \
  -X PUT \
  -H "Authorization: Bearer <USER_A_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"agent":{"max_iterations":3}}'
curl -s http://127.0.0.1:8080/v1/config \
  -H "Authorization: Bearer <USER_A_TOKEN>"
```

Expected: the final response for user A shows `agent.max_iterations = 3`.

- [ ] **Step 3: Verify user B is unaffected**

```bash
curl -s http://127.0.0.1:8080/v1/config \
  -H "Authorization: Bearer <USER_B_TOKEN>"
```

Expected: user B does not see user A’s `agent.max_iterations = 3` unless B saved the same override.

- [ ] **Step 4: Verify global sections still work**

```bash
curl -s http://127.0.0.1:8080/v1/config \
  -X PUT \
  -H "Authorization: Bearer <ADMIN_OR_ANY_VALID_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"summary_llm":{"provider":"deepseek","model":"test-model","base_url":"http://example.com","api_key":"test-key"}}'
```

Expected: request succeeds and the updated global section is visible to both users on subsequent `GET /v1/config`.

- [ ] **Step 5: Run the final compile check**

```bash
cd /Users/fqc/project/agent/openIntern/openIntern_backend
go build ./...
```

Expected: build succeeds without output.

- [ ] **Step 6: Commit the final pass**

```bash
git -C /Users/fqc/project/agent/openIntern add \
  openIntern_forentend/app/\(workspace\)/settings/page.tsx
git -C /Users/fqc/project/agent/openIntern commit -m "feat: isolate user settings from global config"
```

## Self-Review

- Spec coverage:
  - 数据模型与可扩展 key 设计由 Task 1 和 Task 2 覆盖。
  - `/v1/config` 读写分流由 Task 3 覆盖。
  - agent 运行时按 `user_id` 生效由 Task 4 覆盖。
  - 双用户隔离与系统配置回归由 Task 5 覆盖。
- Placeholder scan:
  - 计划中未使用未定义步骤或留空实现说明。
  - 每个任务都给出了明确文件、代码骨架和验证命令。
- Type consistency:
  - 统一使用 `UserRuntimeConfig`、`ResolvedUserRuntimeConfig`、`UserRuntimeConfigResolver`。
  - `agent` 与 `context_compression` 作为唯一允许的用户级 key，前后文命名一致。
