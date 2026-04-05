# AIO Sandbox User Instance Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the global AIO sandbox URL model with a per-user Docker-managed sandbox instance lifecycle that lazily creates, reuses, and recycles containers during local development.

**Architecture:** Introduce a new sandbox lifecycle module under `internal/services/sandbox` that owns instance metadata, Docker CLI provider logic, health checks, idle recycling, and user-level endpoint resolution. Remove package-level sandbox URL state from agent, builtin tool, and plugin code paths; instead, resolve the current user's sandbox endpoint at tool execution time.

**Tech Stack:** Go, Gin, GORM, MySQL, Docker CLI, Eino tools, existing runtime config system

---

## File Map

**Create:**

- `openIntern_backend/internal/models/sandbox_instance.go`
- `openIntern_backend/internal/dao/sandbox_instance.go`
- `openIntern_backend/internal/services/sandbox/types.go`
- `openIntern_backend/internal/services/sandbox/store.go`
- `openIntern_backend/internal/services/sandbox/provider.go`
- `openIntern_backend/internal/services/sandbox/provider_docker.go`
- `openIntern_backend/internal/services/sandbox/manager.go`
- `openIntern_backend/internal/services/sandbox/client.go`
- `openIntern_backend/internal/services/builtin_tool/sandbox_execute_bash.go`

**Modify:**

- `openIntern_backend/internal/config/config.go`
- `openIntern_backend/internal/config/runtime_config.go`
- `openIntern_backend/internal/database/database.go`
- `openIntern_backend/main.go`
- `openIntern_backend/internal/services/agent/agent_service.go`
- `openIntern_backend/internal/services/agent/agent_init.go`
- `openIntern_backend/internal/services/agent/agent_runner.go`
- `openIntern_backend/internal/services/agent/agent_runtime_compile.go`
- `openIntern_backend/internal/services/agent/agent_entry.go`
- `openIntern_backend/internal/services/agent/agent_debug_entry.go`
- `openIntern_backend/internal/services/plugin/plugin.go`
- `openIntern_backend/internal/services/plugin/plugin_code.go`
- `openIntern_backend/internal/services/plugin/plugin_code_debug.go`
- `openIntern_backend/internal/services/builtin_tool/cos.go`
- `openIntern_backend/internal/controllers/config.go`
- `openIntern_backend/config.yaml`
- `README.md`
- `openIntern_backend/script/setup_sandbox.md`

**Remove or stop using:**

- `openIntern_backend/internal/services/builtin_tool/sandbox_mcp.go`

**Verification targets:**

- `go build ./...` from `openIntern_backend`
- manual Docker lifecycle checks via `docker ps`, `docker inspect`, and chat/code-path smoke tests

### Task 1: Add Sandbox Config and Persistence

**Files:**

- Create: `openIntern_backend/internal/models/sandbox_instance.go`
- Create: `openIntern_backend/internal/dao/sandbox_instance.go`
- Modify: `openIntern_backend/internal/config/config.go`
- Modify: `openIntern_backend/internal/config/runtime_config.go`
- Modify: `openIntern_backend/internal/database/database.go`
- Modify: `openIntern_backend/internal/controllers/config.go`

- [ ] **Step 1: Define the sandbox instance model**

Add a new GORM model with one row per user:

```go
package models

import "time"

type SandboxInstance struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	UserID         string    `gorm:"type:varchar(191);not null;uniqueIndex:uk_sandbox_instance_user" json:"user_id"`
	Provider       string    `gorm:"type:varchar(32);not null;index:idx_sandbox_instance_status_lease,priority:1" json:"provider"`
	Status         string    `gorm:"type:varchar(32);not null;index:idx_sandbox_instance_status_lease,priority:2" json:"status"`
	InstanceID     string    `gorm:"type:varchar(191);not null" json:"instance_id"`
	Endpoint       string    `gorm:"type:varchar(255);not null" json:"endpoint"`
	LastActiveAt   time.Time `gorm:"not null" json:"last_active_at"`
	LeaseExpiresAt time.Time `gorm:"not null;index:idx_sandbox_instance_status_lease,priority:3" json:"lease_expires_at"`
	LastError      string    `gorm:"type:text" json:"last_error"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
```

- [ ] **Step 2: Register the model in database migration**

Extend `database.Init` so the new table is created with the rest of the schema:

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
); err != nil {
	return fmt.Errorf("migrate database: %w", err)
}
```

- [ ] **Step 3: Add DAO methods for user-scoped lifecycle operations**

Create DAO methods that the lifecycle manager can call directly:

```go
type SandboxInstanceDAO struct{}

func (SandboxInstanceDAO) GetByUserID(userID string) (*models.SandboxInstance, error)
func (SandboxInstanceDAO) UpsertProvisioning(userID, provider string, leaseExpiresAt time.Time) (*models.SandboxInstance, error)
func (SandboxInstanceDAO) MarkReady(userID string, updates map[string]any) error
func (SandboxInstanceDAO) MarkFailed(userID, message string) error
func (SandboxInstanceDAO) MarkRecyclingIfExpired(userID string, now time.Time) (bool, error)
func (SandboxInstanceDAO) TouchReady(userID string, lastActiveAt, leaseExpiresAt time.Time) error
func (SandboxInstanceDAO) DeleteByUserID(userID string) error
func (SandboxInstanceDAO) ListExpiredReady(now time.Time, limit int) ([]models.SandboxInstance, error)
```

- [ ] **Step 4: Replace URL-based sandbox config with lifecycle config**

Change `SandboxConfig` in `config.go` to control creation rather than connection:

```go
type SandboxDockerConfig struct {
	Image   string `yaml:"image" json:"image"`
	Host    string `yaml:"host" json:"host"`
	Network string `yaml:"network" json:"network"`
}

type SandboxConfig struct {
	Enabled                  *bool               `yaml:"enabled" json:"enabled"`
	Provider                 string              `yaml:"provider" json:"provider"`
	IdleTTLSeconds           int                 `yaml:"idle_ttl_seconds" json:"idle_ttl_seconds"`
	CreateTimeoutSeconds     int                 `yaml:"create_timeout_seconds" json:"create_timeout_seconds"`
	RecycleIntervalSeconds   int                 `yaml:"recycle_interval_seconds" json:"recycle_interval_seconds"`
	HealthcheckTimeoutSeconds int                `yaml:"healthcheck_timeout_seconds" json:"healthcheck_timeout_seconds"`
	Docker                   SandboxDockerConfig `yaml:"docker" json:"docker"`
}
```

- [ ] **Step 5: Update runtime config read/write behavior**

Change runtime config updates so the new sandbox fields round-trip cleanly:

```go
func updateToolsConfig(cfg *ToolsConfig, updates map[string]interface{}) {
	if sandboxUpdates, ok := updates["sandbox"].(map[string]interface{}); ok {
		if provider, ok := sandboxUpdates["provider"].(string); ok {
			cfg.Sandbox.Provider = provider
		}
		if ttl, ok := sandboxUpdates["idle_ttl_seconds"].(float64); ok {
			cfg.Sandbox.IdleTTLSeconds = int(ttl)
		}
		if dockerUpdates, ok := sandboxUpdates["docker"].(map[string]interface{}); ok {
			if image, ok := dockerUpdates["image"].(string); ok {
				cfg.Sandbox.Docker.Image = image
			}
		}
	}
}
```

- [ ] **Step 6: Verify the backend still builds after schema and config changes**

Run:

```bash
cd /Users/fqc/project/agent/openIntern/openIntern_backend
go build ./...
```

Expected: build succeeds with the new sandbox model and config types wired in.

- [ ] **Step 7: Commit the schema/config foundation**

Run:

```bash
git -C /Users/fqc/project/agent/openIntern add \
  openIntern_backend/internal/models/sandbox_instance.go \
  openIntern_backend/internal/dao/sandbox_instance.go \
  openIntern_backend/internal/config/config.go \
  openIntern_backend/internal/config/runtime_config.go \
  openIntern_backend/internal/database/database.go \
  openIntern_backend/internal/controllers/config.go
git -C /Users/fqc/project/agent/openIntern commit -m "feat: add sandbox lifecycle schema and config"
```

### Task 2: Implement the Sandbox Lifecycle Module and Docker Provider

**Files:**

- Create: `openIntern_backend/internal/services/sandbox/types.go`
- Create: `openIntern_backend/internal/services/sandbox/store.go`
- Create: `openIntern_backend/internal/services/sandbox/provider.go`
- Create: `openIntern_backend/internal/services/sandbox/provider_docker.go`
- Create: `openIntern_backend/internal/services/sandbox/manager.go`
- Create: `openIntern_backend/internal/services/sandbox/client.go`

- [ ] **Step 1: Define shared sandbox lifecycle types**

Create a focused type layer for lifecycle state and provider wiring:

```go
package sandbox

import "time"

const (
	StatusProvisioning = "provisioning"
	StatusReady        = "ready"
	StatusFailed       = "failed"
	StatusRecycling    = "recycling"
	ProviderDocker     = "docker"
	ProviderSCF        = "scf"
)

type Instance struct {
	UserID         string
	Provider       string
	Status         string
	InstanceID     string
	Endpoint       string
	LastActiveAt   time.Time
	LeaseExpiresAt time.Time
	LastError      string
}

type CreateRequest struct {
	UserID string
}

type CreateResult struct {
	InstanceID string
	Endpoint   string
}
```

- [ ] **Step 2: Add the provider interface and a Docker CLI implementation**

Define the provider API and implement Docker-only behavior:

```go
type Provider interface {
	Name() string
	Create(ctx context.Context, req CreateRequest) (*CreateResult, error)
	FindExisting(ctx context.Context, userID string) (*CreateResult, error)
	Destroy(ctx context.Context, instance Instance) error
	HealthCheck(ctx context.Context, instance Instance) error
}
```

The Docker provider should internally call:

```bash
docker run -d --security-opt seccomp=unconfined --name <name> \
  --label openintern.managed=true \
  --label openintern.user_id=<user_id> \
  -p 0:8080 <image>
```

Then inspect the container:

```bash
docker inspect <container_id>
```

- [ ] **Step 3: Implement stable container naming and discovery**

Use stable naming plus labels so restart recovery works:

```go
func containerNameForUser(userID string) string {
	sum := sha1.Sum([]byte(strings.TrimSpace(userID)))
	return "openintern-sandbox-" + hex.EncodeToString(sum[:])[:12]
}
```

Discovery should first use labels, then container name as a fallback.

- [ ] **Step 4: Implement the lifecycle manager with singleflight**

Build `Manager` around DAO + provider + TTL:

```go
type Manager struct {
	store            Store
	provider         Provider
	idleTTL          time.Duration
	createTimeout    time.Duration
	healthcheckTimeout time.Duration
	sf               singleflight.Group
}

func (m *Manager) GetOrCreate(ctx context.Context, userID string) (*Instance, error)
func (m *Manager) Touch(ctx context.Context, userID string) error
func (m *Manager) Destroy(ctx context.Context, userID string) error
func (m *Manager) RecycleIdle(ctx context.Context, limit int) (int, error)
```

`GetOrCreate` must:

- trust Docker as the final truth
- reuse healthy ready containers
- rebind orphaned containers after restart
- mark provisioning before `docker run`
- mark failed on provider errors

- [ ] **Step 5: Add a small HTTP client layer for endpoint-based calls**

Create a small client helper so plugin/builtin code stops duplicating request assembly:

```go
type Client struct {
	httpClient *http.Client
}

func (c *Client) ExecuteCode(ctx context.Context, endpoint string, payload any) ([]byte, error)
func (c *Client) ReadFile(ctx context.Context, endpoint, sandboxPath string) ([]byte, error)
```

- [ ] **Step 6: Verify the lifecycle package compiles**

Run:

```bash
cd /Users/fqc/project/agent/openIntern/openIntern_backend
go build ./...
```

Expected: the new lifecycle package compiles without touching runtime call sites yet.

- [ ] **Step 7: Commit the lifecycle module**

Run:

```bash
git -C /Users/fqc/project/agent/openIntern add \
  openIntern_backend/internal/services/sandbox/types.go \
  openIntern_backend/internal/services/sandbox/store.go \
  openIntern_backend/internal/services/sandbox/provider.go \
  openIntern_backend/internal/services/sandbox/provider_docker.go \
  openIntern_backend/internal/services/sandbox/manager.go \
  openIntern_backend/internal/services/sandbox/client.go
git -C /Users/fqc/project/agent/openIntern commit -m "feat: add sandbox lifecycle manager and docker provider"
```

### Task 3: Bootstrap the Sandbox Manager and Background Recycler

**Files:**

- Modify: `openIntern_backend/main.go`
- Modify: `openIntern_backend/internal/services/agent/agent_service.go`
- Modify: `openIntern_backend/internal/services/agent/agent_init.go`
- Modify: `openIntern_backend/internal/services/plugin/plugin.go`

- [ ] **Step 1: Add sandbox dependencies to the agent runtime**

Extend `agent.Dependencies` and `runtimeState` to carry the manager instead of a base URL:

```go
type SandboxManager interface {
	GetOrCreate(ctx context.Context, userID string) (*sandbox.Instance, error)
	Touch(ctx context.Context, userID string) error
}

type Dependencies struct {
	A2UIService                builtinTool.A2UIServiceInterface
	FileUploader               builtinTool.FileUploader
	SandboxManager             SandboxManager
	// ...
}

type runtimeState struct {
	apmplusShutdown    func(context.Context) error
	summaryModel       einoModel.ToolCallingChatModel
	sandboxManager     SandboxManager
	// ...
}
```

- [ ] **Step 2: Initialize the sandbox manager at process startup**

Create the provider and manager from config in `main.go` before agent init:

```go
sandboxManager, sandboxShutdown, err := sandboxsvc.InitManager(cfg.Tools.Sandbox)
if err != nil {
	log.Fatalf("failed to init sandbox manager: %v", err)
}
if sandboxShutdown != nil {
	defer sandboxShutdown()
}
```

Pass it into `agentsvc.NewService(...)` dependencies and into plugin/builtin layers that need user-scoped resolution.

- [ ] **Step 3: Remove the global sandbox URL state**

Delete package-level URL state and its setter:

```go
var sandboxBaseURL string

func SetSandboxBaseURL(baseURL string) {
	sandboxBaseURL = strings.TrimSpace(baseURL)
}
```

Replace init validation in `agent_init.go` with provider-based validation:

```go
if !sandboxEnabled(toolsCfg.Sandbox) {
	return nil, fmt.Errorf("tools.sandbox.enabled is required")
}
if strings.TrimSpace(toolsCfg.Sandbox.Provider) != sandbox.ProviderDocker {
	return nil, fmt.Errorf("tools.sandbox.provider must be docker in local development")
}
```

- [ ] **Step 4: Start the idle recycler ticker**

Add startup wiring for a process-local recycler:

```go
func InitManager(cfg config.SandboxConfig) (*Manager, func(), error) {
	manager := NewManager(...)
	stop := manager.StartRecycler()
	return manager, stop, nil
}
```

The ticker interval must come from `recycle_interval_seconds`.

- [ ] **Step 5: Verify bootstrap compiles after removing the URL model**

Run:

```bash
cd /Users/fqc/project/agent/openIntern/openIntern_backend
go build ./...
```

Expected: all references to `sandboxBaseURL` are either removed or fail loudly and are ready for the next tasks.

- [ ] **Step 6: Commit bootstrap wiring**

Run:

```bash
git -C /Users/fqc/project/agent/openIntern add \
  openIntern_backend/main.go \
  openIntern_backend/internal/services/agent/agent_service.go \
  openIntern_backend/internal/services/agent/agent_init.go \
  openIntern_backend/internal/services/plugin/plugin.go
git -C /Users/fqc/project/agent/openIntern commit -m "feat: bootstrap sandbox manager and recycler"
```

### Task 4: Switch Code Execution and COS File Reads to User-Resolved Instances

**Files:**

- Modify: `openIntern_backend/internal/services/plugin/plugin_code.go`
- Modify: `openIntern_backend/internal/services/plugin/plugin_code_debug.go`
- Modify: `openIntern_backend/internal/services/builtin_tool/cos.go`
- Modify: `openIntern_backend/internal/services/agent/agent_entry.go`
- Modify: `openIntern_backend/internal/services/agent/agent_debug_entry.go`

- [ ] **Step 1: Stop passing sandbox base URL through context**

Delete `ContextKeySandboxBaseURL` usage and keep only authenticated `user_id` in runtime context:

```go
ctx = context.WithValue(ctx, builtinTool.ContextKeyUserID, ownerID)
```

Remove:

```go
ctx = context.WithValue(ctx, builtinTool.ContextKeySandboxBaseURL, state.sandboxBaseURL)
```

- [ ] **Step 2: Resolve user instances inside code execution**

Refactor `RunCodeInSandbox` call sites so they first resolve the current user's endpoint:

```go
userID := userIDFromContext(ctx)
instance, err := sandboxManager.GetOrCreate(ctx, userID)
if err != nil {
	return "", err
}
output, err := sandboxClient.ExecuteCode(ctx, instance.Endpoint, payload)
if err == nil {
	_ = sandboxManager.Touch(ctx, userID)
}
```

- [ ] **Step 3: Refactor code debug path to use the same flow**

Use identical instance resolution for debug execution:

```go
instance, err := sandboxManager.GetOrCreate(ctx, userID)
if err != nil {
	return nil, err
}
output, err := sandboxClient.ExecuteCode(ctx, instance.Endpoint, payload)
```

- [ ] **Step 4: Refactor COS file reads to use the manager and client**

Replace the old base URL lookup in `cos.go` with user-based resolution:

```go
userID, _ := ctx.Value(ContextKeyUserID).(string)
instance, err := manager.GetOrCreate(ctx, strings.TrimSpace(userID))
if err != nil {
	return "", err
}
decoded, err := sandboxClient.ReadFile(ctx, instance.Endpoint, sandboxPath)
if err == nil {
	_ = manager.Touch(ctx, userID)
}
```

- [ ] **Step 5: Build the backend and perform manual smoke verification**

Run:

```bash
cd /Users/fqc/project/agent/openIntern/openIntern_backend
go build ./...
docker ps --format '{{.ID}} {{.Names}} {{.Ports}}'
```

Expected:

- backend builds
- after a manual code-execution smoke call, one per-user sandbox container appears
- repeated calls reuse the same container name

- [ ] **Step 6: Commit the code/COS integration**

Run:

```bash
git -C /Users/fqc/project/agent/openIntern add \
  openIntern_backend/internal/services/plugin/plugin_code.go \
  openIntern_backend/internal/services/plugin/plugin_code_debug.go \
  openIntern_backend/internal/services/builtin_tool/cos.go \
  openIntern_backend/internal/services/agent/agent_entry.go \
  openIntern_backend/internal/services/agent/agent_debug_entry.go
git -C /Users/fqc/project/agent/openIntern commit -m "feat: route code and cos sandbox access through lifecycle manager"
```

### Task 5: Replace Dynamic MCP Injection with a Local Bash Proxy Tool

**Files:**

- Create: `openIntern_backend/internal/services/builtin_tool/sandbox_execute_bash.go`
- Modify: `openIntern_backend/internal/services/agent/agent_runner.go`
- Modify: `openIntern_backend/internal/services/agent/agent_runtime_compile.go`
- Remove or stop using: `openIntern_backend/internal/services/builtin_tool/sandbox_mcp.go`

- [ ] **Step 1: Define a local builtin tool for bash execution**

Create a static tool definition instead of discovering it from a remote MCP server at runner-build time:

```go
type SandboxExecuteBashInput struct {
	Command string `json:"command"`
}

func GetSandboxExecuteBashTool(manager sandbox.Manager, client *sandbox.Client) (einoTool.BaseTool, error) {
	return utils.InferTool[SandboxExecuteBashInput, string](
		"sandbox_execute_bash",
		"在当前用户的 AIO sandbox 中执行 bash 命令。",
		func(ctx context.Context, input SandboxExecuteBashInput) (string, error) {
			userID, _ := ctx.Value(ContextKeyUserID).(string)
			instance, err := manager.GetOrCreate(ctx, strings.TrimSpace(userID))
			if err != nil {
				return "", err
			}
			return client.ExecuteBashViaMCP(ctx, instance.Endpoint, input.Command)
		},
	)
}
```

- [ ] **Step 2: Remove runner-time sandbox MCP discovery**

Delete this pattern from runner setup:

```go
sandboxTools, sandboxCleanup, err := builtinTool.GetSandboxMCPTools(ctx, state.sandboxBaseURL)
```

Replace it with a prebuilt local tool:

```go
sandboxTool, err := builtinTool.GetSandboxExecuteBashTool(state.sandboxManager, state.sandboxClient)
resolved.staticTools = append(resolved.staticTools, sandboxTool)
```

- [ ] **Step 3: Do the same replacement in agent mode compile**

Mirror the same local-tool injection in `agent_runtime_compile.go` so the custom agent mode and the default runner behave identically.

- [ ] **Step 4: Verify that the backend builds with the old MCP loader removed**

Run:

```bash
cd /Users/fqc/project/agent/openIntern/openIntern_backend
go build ./...
```

Expected: no remaining call sites depend on `GetSandboxMCPTools(ctx, baseURL)`.

- [ ] **Step 5: Perform a manual bash-tool smoke check**

Run:

```bash
docker ps --format '{{.Names}}'
docker inspect <container_id> --format '{{json .Config.Labels}}'
```

Expected:

- the user's sandbox container exists
- labels include `openintern.managed=true` and `openintern.user_id=<user_id>`
- a repeated bash tool call does not create a second container for the same user

- [ ] **Step 6: Commit the local proxy tool migration**

Run:

```bash
git -C /Users/fqc/project/agent/openIntern add \
  openIntern_backend/internal/services/builtin_tool/sandbox_execute_bash.go \
  openIntern_backend/internal/services/agent/agent_runner.go \
  openIntern_backend/internal/services/agent/agent_runtime_compile.go
git -C /Users/fqc/project/agent/openIntern commit -m "feat: replace sandbox mcp loader with local bash proxy tool"
```

### Task 6: Update Local Config and Documentation

**Files:**

- Modify: `openIntern_backend/config.yaml`
- Modify: `README.md`
- Modify: `openIntern_backend/script/setup_sandbox.md`

- [ ] **Step 1: Replace the old URL config example with lifecycle config**

Update the local config sample:

```yaml
tools:
  sandbox:
    enabled: true
    provider: docker
    idle_ttl_seconds: 1800
    create_timeout_seconds: 30
    recycle_interval_seconds: 30
    healthcheck_timeout_seconds: 10
    docker:
      image: enterprise-public-cn-beijing.cr.volces.com/vefaas-public/all-in-one-sandbox:latest
      host: "127.0.0.1"
      network: ""
```

- [ ] **Step 2: Rewrite the local sandbox setup guide around Docker-managed instances**

Document that:

- openIntern now creates one container per user on demand
- developers no longer manually bind a single fixed `8081`
- idle containers are removed automatically
- backend restart reuses managed containers when possible

- [ ] **Step 3: Add an explicit manual verification section**

Document these checks:

```bash
docker ps --format 'table {{.Names}}\t{{.Ports}}\t{{.Status}}'
docker inspect <container_name> --format '{{json .Config.Labels}}'
docker rm -f <container_name>
```

Expected:

- first sandbox use creates a container
- second sandbox use by the same user reuses it
- deleting the container manually causes the next request to recreate it

- [ ] **Step 4: Build the backend one final time**

Run:

```bash
cd /Users/fqc/project/agent/openIntern/openIntern_backend
go build ./...
```

Expected: the project builds with the new config fields and no stale references to `tools.sandbox.url`.

- [ ] **Step 5: Commit docs and config updates**

Run:

```bash
git -C /Users/fqc/project/agent/openIntern add \
  openIntern_backend/config.yaml \
  README.md \
  openIntern_backend/script/setup_sandbox.md
git -C /Users/fqc/project/agent/openIntern commit -m "docs: document per-user sandbox lifecycle setup"
```

## Self-Review

### Spec Coverage

- User-level lifecycle module: covered by Tasks 1-3
- Docker CLI provider: covered by Task 2
- Lazy create / reuse / touch / recycle: covered by Tasks 2-4
- Restart recovery / container claim: covered by Task 2
- Removing global URL model: covered by Tasks 3-5
- Code / COS / bash runtime resolution: covered by Tasks 4-5
- Config and docs update: covered by Task 6

### Placeholder Scan

- No `TBD`, `TODO`, or “implement later” placeholders remain.
- All verification steps use `go build` and manual Docker checks to respect the current repo rule of not running `go test` without explicit approval.

### Type Consistency

- Lifecycle provider constants use `docker` / `scf`.
- Runtime status constants use `provisioning` / `ready` / `failed` / `recycling`.
- All runtime call sites refer to a single `SandboxManager` abstraction and a single `Instance.Endpoint` field.
