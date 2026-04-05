# Multi-User Infra Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a unified external-dependencies `compose.yaml`, cold-start against a new MySQL instance, and refactor backend/frontend data access from single-user/global resources to user-scoped resources.

**Architecture:** Keep `openIntern_backend` and `openIntern_forentend` running locally while Docker Compose manages only MySQL, Redis, and MinIO. Refactor persistence so all business resources carry `user_id`, DAO methods require `user_id`, JWT no longer carries `role`, and `/v1/users` becomes a current-user self-service surface instead of an admin-style management surface.

**Tech Stack:** Docker Compose, MySQL, Redis, MinIO, Go, Gin, GORM, Next.js, TypeScript

---

## Repo Constraints

- This repo explicitly forbids adding Go test files casually.
- Do not run `go test`.
- Do not run `pnpm lint` or other long lint-like commands.
- Verification for this plan uses `docker compose`, `go build`, and targeted manual smoke checks.

## File Structure

### New files

- Create: `compose.yaml`
- Create: `docs/local-development.md`
- Create: `docs/superpowers/plans/2026-04-05-multi-user-infra.md`

### Backend files to modify

- Modify: `openIntern_backend/config.yaml`
- Modify: `openIntern_backend/internal/database/database.go`
- Modify: `openIntern_backend/internal/middleware/auth.go`
- Modify: `openIntern_backend/internal/routers/router.go`
- Modify: `openIntern_backend/internal/services/account/auth.go`
- Modify: `openIntern_backend/internal/services/account/user.go`
- Modify: `openIntern_backend/internal/controllers/user.go`
- Modify: `openIntern_backend/internal/models/user.go`
- Modify: `openIntern_backend/internal/models/thread.go`
- Modify: `openIntern_backend/internal/models/message.go`
- Modify: `openIntern_backend/internal/models/a2ui.go`
- Modify: `openIntern_backend/internal/models/agent.go`
- Modify: `openIntern_backend/internal/models/agent_binding.go`
- Modify: `openIntern_backend/internal/models/plugin.go`
- Modify: `openIntern_backend/internal/models/tool.go`
- Modify: `openIntern_backend/internal/models/plugin_default.go`
- Modify: `openIntern_backend/internal/models/model_provider.go`
- Modify: `openIntern_backend/internal/models/model_catalog.go`
- Modify: `openIntern_backend/internal/models/default_model_config.go`
- Modify: `openIntern_backend/internal/models/memory_sync_state.go`
- Modify: `openIntern_backend/internal/models/memory_usage_log.go`
- Modify: `openIntern_backend/internal/models/thread_context_snapshot.go`
- Modify: `openIntern_backend/internal/dao/thread.go`
- Modify: `openIntern_backend/internal/dao/message.go`
- Modify: `openIntern_backend/internal/dao/a2ui.go`
- Modify: `openIntern_backend/internal/dao/agent.go`
- Modify: `openIntern_backend/internal/dao/agent_binding.go`
- Modify: `openIntern_backend/internal/dao/plugin.go`
- Modify: `openIntern_backend/internal/dao/model_provider.go`
- Modify: `openIntern_backend/internal/dao/model_catalog.go`
- Modify: `openIntern_backend/internal/dao/memory_sync_state.go`
- Modify: `openIntern_backend/internal/dao/memory_usage_log.go`
- Modify: `openIntern_backend/internal/dao/thread_context_snapshot.go`
- Modify: `openIntern_backend/internal/services/chat/thread.go`
- Modify: `openIntern_backend/internal/services/chat/message.go`
- Modify: `openIntern_backend/internal/services/chat/context_snapshot.go`
- Modify: `openIntern_backend/internal/services/a2ui/service.go`
- Modify: `openIntern_backend/internal/services/agent/agent_definition_service.go`
- Modify: `openIntern_backend/internal/services/agent/agent_runtime_compile.go`
- Modify: `openIntern_backend/internal/services/agent/agent_runtime_context.go`
- Modify: `openIntern_backend/internal/services/plugin/plugin.go`
- Modify: `openIntern_backend/internal/services/plugin/plugin_search.go`
- Modify: `openIntern_backend/internal/services/model/provider.go`
- Modify: `openIntern_backend/internal/services/model/catalog.go`
- Modify: `openIntern_backend/internal/services/memory/sync_state.go`
- Modify: `openIntern_backend/internal/controllers/thread.go`
- Modify: `openIntern_backend/internal/controllers/chat.go`
- Modify: `openIntern_backend/internal/controllers/a2ui.go`
- Modify: `openIntern_backend/internal/controllers/agent.go`
- Modify: `openIntern_backend/internal/controllers/plugin.go`
- Modify: `openIntern_backend/internal/controllers/model_provider.go`
- Modify: `openIntern_backend/internal/controllers/model_catalog.go`

### Frontend files to modify

- Modify: `openIntern_forentend/app/(workspace)/auth.ts`
- Modify: `openIntern_forentend/app/(workspace)/user/page.tsx`
- Modify: `openIntern_forentend/app/(workspace)/models/page.tsx`
- Modify: `openIntern_forentend/app/(workspace)/a2ui/page.tsx`
- Modify: `openIntern_forentend/app/(workspace)/layout.tsx`

### Optional follow-up inspection files during execution

- Inspect: `openIntern_backend/internal/controllers/config.go`
- Inspect: `openIntern_backend/internal/services/plugin/plugin_openviking_sync.go`
- Inspect: `openIntern_backend/internal/services/memory/sync_worker.go`
- Inspect: `openIntern_forentend/app/(workspace)/plugins/page.tsx`

### Task 1: Add Compose-Managed External Dependencies

**Files:**
- Create: `compose.yaml`
- Modify: `openIntern_backend/config.yaml`
- Create: `docs/local-development.md`

- [ ] **Step 1: Add the root Compose file for MySQL, Redis, and MinIO**

```yaml
services:
  mysql:
    image: mysql:8.4
    container_name: openintern-mysql
    restart: unless-stopped
    environment:
      MYSQL_ROOT_PASSWORD: root
      MYSQL_DATABASE: open_intern
      TZ: Asia/Shanghai
    ports:
      - "3306:3306"
    volumes:
      - openintern_mysql_data:/var/lib/mysql
    command:
      - --character-set-server=utf8mb4
      - --collation-server=utf8mb4_unicode_ci
      - --default-time-zone=+08:00
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "127.0.0.1", "-proot"]
      interval: 10s
      timeout: 5s
      retries: 10

  redis:
    image: redis:7.4
    container_name: openintern-redis
    restart: unless-stopped
    ports:
      - "6379:6379"
    volumes:
      - openintern_redis_data:/data
    command: ["redis-server", "--appendonly", "yes"]

  minio:
    image: minio/minio:RELEASE.2025-02-28T09-55-16Z
    container_name: openintern-minio
    restart: unless-stopped
    environment:
      MINIO_ROOT_USER: minioadmin
      MINIO_ROOT_PASSWORD: minioadmin
    ports:
      - "9000:9000"
      - "9001:9001"
    volumes:
      - openintern_minio_data:/data
    command: server /data --console-address ":9001"

volumes:
  openintern_mysql_data:
  openintern_redis_data:
  openintern_minio_data:
```

- [ ] **Step 2: Point backend defaults at the new local dependency endpoints**

```yaml
mysql:
  dsn: root:root@tcp(127.0.0.1:3306)/open_intern?charset=utf8mb4&parseTime=True&loc=Local
redis:
  addr: 127.0.0.1:6379
  password: ""
  db: 0
```

- [ ] **Step 3: Document the new startup order and clarify that MinIO is not yet wired into COS uploads**

~~~md
# Local Development

## Start external dependencies

```bash
docker compose up -d
docker compose ps
```

## Start backend and frontend

1. Start `openIntern_backend` locally.
2. Start `openIntern_forentend` locally.
3. Keep `openviking` local; it is not managed by Compose yet.

## MinIO

- API: `http://127.0.0.1:9000`
- Console: `http://127.0.0.1:9001`
- This round only deploys MinIO and Console.
- Existing COS upload logic remains unchanged.
~~~

- [ ] **Step 4: Validate the Compose file before moving on**

Run: `docker compose config`
Expected: renders the merged Compose configuration without schema errors

- [ ] **Step 5: Commit the infra scaffold**

```bash
git add compose.yaml openIntern_backend/config.yaml docs/local-development.md
git commit -m "chore: add compose-managed local dependencies"
```

### Task 2: Replace Legacy-Compatible DB Bootstrap With Cold-Start Schema Definitions

**Files:**
- Modify: `openIntern_backend/internal/database/database.go`
- Modify: `openIntern_backend/internal/models/user.go`
- Modify: `openIntern_backend/internal/models/thread.go`
- Modify: `openIntern_backend/internal/models/message.go`
- Modify: `openIntern_backend/internal/models/a2ui.go`
- Modify: `openIntern_backend/internal/models/agent.go`
- Modify: `openIntern_backend/internal/models/agent_binding.go`
- Modify: `openIntern_backend/internal/models/plugin.go`
- Modify: `openIntern_backend/internal/models/tool.go`
- Modify: `openIntern_backend/internal/models/plugin_default.go`
- Modify: `openIntern_backend/internal/models/model_provider.go`
- Modify: `openIntern_backend/internal/models/model_catalog.go`
- Modify: `openIntern_backend/internal/models/default_model_config.go`
- Modify: `openIntern_backend/internal/models/memory_sync_state.go`
- Modify: `openIntern_backend/internal/models/memory_usage_log.go`
- Modify: `openIntern_backend/internal/models/thread_context_snapshot.go`

- [ ] **Step 1: Remove old-column compatibility branches from the database initializer**

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
); err != nil {
    return fmt.Errorf("migrate database: %w", err)
}

if err := DB.Migrator().AlterColumn(&models.Message{}, "Content"); err != nil {
    return fmt.Errorf("alter message.content: %w", err)
}
if err := DB.Migrator().AlterColumn(&models.Message{}, "Metadata"); err != nil {
    return fmt.Errorf("alter message.metadata: %w", err)
}
```

- [ ] **Step 2: Add `user_id` to every user-owned model and convert global unique keys to user-scoped composite keys**

```go
type Thread struct {
    ID       uint   `gorm:"primarykey" json:"-"`
    UserID   string `gorm:"column:user_id;index;not null;size:36" json:"user_id"`
    ThreadID string `gorm:"column:thread_id;uniqueIndex:ux_thread_user_thread,priority:2;not null;size:36" json:"thread_id"`
    Title    string `gorm:"size:200" json:"title"`
}

type Plugin struct {
    ID       uint   `gorm:"primarykey" json:"-"`
    UserID   string `gorm:"column:user_id;index;not null;size:36" json:"user_id"`
    PluginID string `gorm:"column:plugin_id;uniqueIndex:ux_plugin_user_plugin,priority:2;not null;size:36" json:"plugin_id"`
    Name     string `gorm:"uniqueIndex:ux_plugin_user_name,priority:2;not null;size:120" json:"name"`
}
```

- [ ] **Step 3: Rename `agent.owner_id` to `agent.user_id` and remove user-role semantics from the user model**

```go
type User struct {
    ID       uint   `gorm:"primarykey" json:"-"`
    UserID   string `gorm:"uniqueIndex;not null;size:36" json:"user_id"`
    Username string `gorm:"uniqueIndex;not null;size:50" json:"username"`
    Email    string `gorm:"uniqueIndex;not null;size:100" json:"email"`
    Password string `gorm:"not null" json:"-"`
    Avatar   string `gorm:"size:255" json:"avatar"`
    Phone    string `gorm:"size:50" json:"phone"`
}

type Agent struct {
    ID     uint   `gorm:"primarykey" json:"-"`
    AgentID string `gorm:"column:agent_id;uniqueIndex:ux_agent_user_agent,priority:2;not null;size:36" json:"agent_id"`
    UserID string `gorm:"column:user_id;index;not null;size:36" json:"user_id"`
}
```

- [ ] **Step 4: Add user-scoped indexes for sequence, defaults, and runtime lookup tables**

```go
type Message struct {
    UserID   string `gorm:"column:user_id;index;index:idx_message_user_thread_sequence,priority:1;not null;size:36" json:"user_id"`
    ThreadID string `gorm:"index;index:idx_message_user_thread_sequence,priority:2;not null;size:64" json:"thread_id"`
    Sequence int64  `gorm:"index:idx_message_user_thread_sequence,priority:3;not null;default:0" json:"sequence"`
}

type DefaultModelConfig struct {
    UserID    string `gorm:"column:user_id;uniqueIndex:ux_default_model_user_key,priority:1;not null;size:36" json:"user_id"`
    ConfigKey string `gorm:"column:config_key;uniqueIndex:ux_default_model_user_key,priority:2;not null;size:80" json:"config_key"`
}
```

- [ ] **Step 5: Build the backend to verify the schema compiles before touching DAO logic**

Run: `cd openIntern_backend && go build ./...`
Expected: build exits successfully with no compile errors

- [ ] **Step 6: Commit the schema pass**

```bash
git add openIntern_backend/internal/database/database.go openIntern_backend/internal/models
git commit -m "refactor: rebuild schema for user-scoped resources"
```

### Task 3: Remove Role-Based Auth Semantics And Convert User APIs To Self-Service

**Files:**
- Modify: `openIntern_backend/internal/services/account/auth.go`
- Modify: `openIntern_backend/internal/middleware/auth.go`
- Modify: `openIntern_backend/internal/services/account/user.go`
- Modify: `openIntern_backend/internal/controllers/user.go`
- Modify: `openIntern_backend/internal/routers/router.go`
- Modify: `openIntern_forentend/app/(workspace)/auth.ts`
- Modify: `openIntern_forentend/app/(workspace)/user/page.tsx`

- [ ] **Step 1: Shrink JWT claims and token generation to `user_id` only**

```go
type TokenClaims struct {
    UserID string `json:"user_id"`
    jwt.RegisteredClaims
}

func GenerateToken(userID string) (string, int64, error) {
    claims := TokenClaims{
        UserID: userID,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(expiresAt),
            IssuedAt:  jwt.NewNumericDate(now),
        },
    }
}
```

- [ ] **Step 2: Stop writing `role` into Gin context and stop refreshing tokens with a role argument**

```go
refreshedToken, expiresAt, err := accountsvc.GenerateToken(claims.UserID)
if err == nil {
    c.Header("X-Access-Token", refreshedToken)
    c.Header("X-Token-Expires", strconv.FormatInt(expiresAt, 10))
}
c.Set("user_id", claims.UserID)
```

- [ ] **Step 3: Replace admin-style `/v1/users/:id` handlers with current-user handlers**

```go
userGroup := r.Group("/v1/users", middleware.AuthRequired())
{
    userGroup.GET("/me", controllers.GetCurrentUser)
    userGroup.PUT("/me", controllers.UpdateCurrentUser)
    userGroup.POST("/me/avatar", controllers.UploadCurrentUserAvatar)
}
```

- [ ] **Step 4: Make the user controller read the active user from context instead of path params**

```go
func GetCurrentUser(c *gin.Context) {
    userID := strings.TrimSpace(c.GetString("user_id"))
    user, err := accountsvc.User.GetUserByUserID(userID)
    if err != nil {
        response.NotFound(c, "user not found")
        return
    }
    response.JSONSuccess(c, http.StatusOK, user)
}
```

- [ ] **Step 5: Update frontend auth and profile pages to stop reading/displaying `role` and to call `/v1/users/me`**

```ts
export type StoredUser = {
  user_id?: string | number;
  username?: string;
  email?: string;
  phone?: string;
  avatar?: string;
  created_at?: string;
  updated_at?: string;
};
```

```ts
const data = await requestBackend<UserInfo>("/v1/users/me", {
  fallbackMessage: "获取用户信息失败",
  router,
});
```

- [ ] **Step 6: Build the backend and manually smoke the auth flow**

Run: `cd openIntern_backend && go build ./...`
Expected: build exits successfully

Run: `curl -s -X POST http://127.0.0.1:8080/v1/auth/login -H 'Content-Type: application/json' -d '{"identifier":"demo","password":"demo"}'`
Expected: response contains `token`, `expires_at`, and `user`, but no role-dependent behavior is required

- [ ] **Step 7: Commit the auth and self-service route changes**

```bash
git add openIntern_backend/internal/services/account/auth.go openIntern_backend/internal/middleware/auth.go openIntern_backend/internal/services/account/user.go openIntern_backend/internal/controllers/user.go openIntern_backend/internal/routers/router.go 'openIntern_forentend/app/(workspace)/auth.ts' 'openIntern_forentend/app/(workspace)/user/page.tsx'
git commit -m "refactor: convert auth and user APIs to self-service"
```

### Task 4: Scope Threads, Messages, And Memory State By User

**Files:**
- Modify: `openIntern_backend/internal/dao/thread.go`
- Modify: `openIntern_backend/internal/dao/message.go`
- Modify: `openIntern_backend/internal/dao/memory_sync_state.go`
- Modify: `openIntern_backend/internal/dao/memory_usage_log.go`
- Modify: `openIntern_backend/internal/dao/thread_context_snapshot.go`
- Modify: `openIntern_backend/internal/services/chat/thread.go`
- Modify: `openIntern_backend/internal/services/chat/message.go`
- Modify: `openIntern_backend/internal/services/chat/context_snapshot.go`
- Modify: `openIntern_backend/internal/controllers/thread.go`
- Modify: `openIntern_backend/internal/controllers/chat.go`
- Modify: `openIntern_backend/internal/services/memory/sync_state.go`

- [ ] **Step 1: Change DAO signatures so every thread/message lookup requires `userID`**

```go
func (d *ThreadDAO) List(userID string, page, pageSize int) ([]models.Thread, int64, error) {
    query := database.DB.Model(&models.Thread{}).Where("user_id = ?", userID)
}

func (d *ThreadDAO) GetByThreadID(userID, threadID string) (*models.Thread, error) {
    var item models.Thread
    if err := database.DB.Where("user_id = ? AND thread_id = ?", userID, threadID).First(&item).Error; err != nil {
        return nil, err
    }
    return &item, nil
}
```

- [ ] **Step 2: Lock sequences and delete thread-scoped records inside a user-scoped transaction**

```go
var thread models.Thread
if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
    Where("user_id = ? AND thread_id = ?", userID, threadID).
    First(&thread).Error; err != nil {
    return err
}

if err := tx.Where("user_id = ? AND thread_id = ?", userID, threadID).Delete(&models.Message{}).Error; err != nil {
    return err
}
```

- [ ] **Step 3: Thread the active `user_id` through chat services instead of relying on global caches**

```go
func (s *ThreadService) EnsureThread(userID, threadID, title string) (*models.Thread, error) {
    thread, err := dao.Thread.GetByThreadID(userID, threadID)
    if errors.Is(err, gorm.ErrRecordNotFound) {
        thread = &models.Thread{
            UserID:   userID,
            ThreadID: threadID,
            Title:    title,
        }
        return thread, dao.Thread.Create(thread)
    }
    return thread, err
}
```

- [ ] **Step 4: Update controllers to pass the current user into thread and message operations**

```go
func ListThreads(c *gin.Context) {
    userID := strings.TrimSpace(c.GetString("user_id"))
    threads, total, err := chatsvc.Thread.ListThreads(userID, page, pageSize)
}
```

- [ ] **Step 5: Build and smoke-test thread isolation with two different user tokens**

Run: `cd openIntern_backend && go build ./...`
Expected: build exits successfully

Run: create one thread with token A, then call `GET /v1/threads/:thread_id` with token B
Expected: token B gets not found or unauthorized access to that thread

- [ ] **Step 6: Commit the thread/message/memory isolation changes**

```bash
git add openIntern_backend/internal/dao/thread.go openIntern_backend/internal/dao/message.go openIntern_backend/internal/dao/memory_sync_state.go openIntern_backend/internal/dao/memory_usage_log.go openIntern_backend/internal/dao/thread_context_snapshot.go openIntern_backend/internal/services/chat/thread.go openIntern_backend/internal/services/chat/message.go openIntern_backend/internal/services/chat/context_snapshot.go openIntern_backend/internal/services/memory/sync_state.go openIntern_backend/internal/controllers/thread.go openIntern_backend/internal/controllers/chat.go
git commit -m "refactor: scope threads and memory state by user"
```

### Task 5: Scope A2UI And Agents By User

**Files:**
- Modify: `openIntern_backend/internal/dao/a2ui.go`
- Modify: `openIntern_backend/internal/dao/agent.go`
- Modify: `openIntern_backend/internal/dao/agent_binding.go`
- Modify: `openIntern_backend/internal/services/a2ui/service.go`
- Modify: `openIntern_backend/internal/services/agent/agent_definition_service.go`
- Modify: `openIntern_backend/internal/services/agent/agent_runtime_compile.go`
- Modify: `openIntern_backend/internal/services/agent/agent_runtime_context.go`
- Modify: `openIntern_backend/internal/controllers/a2ui.go`
- Modify: `openIntern_backend/internal/controllers/agent.go`

- [ ] **Step 1: Convert A2UI DAO methods to user-scoped filters and user-scoped name uniqueness**

```go
func (d *A2UIDAO) GetByA2UIID(userID, a2uiID string) (*models.A2UI, error) {
    var item models.A2UI
    if err := database.DB.Where("user_id = ? AND a2ui_id = ?", userID, a2uiID).First(&item).Error; err != nil {
        return nil, err
    }
    return &item, nil
}
```

- [ ] **Step 2: Update agent persistence and bindings to use `user_id` consistently**

```go
func (d *AgentDAO) GetByAgentID(userID, agentID string) (*models.Agent, error) {
    var item models.Agent
    if err := database.DB.Where("agent_id = ? AND user_id = ?", agentID, userID).First(&item).Error; err != nil {
        return nil, err
    }
    return &item, nil
}
```

```go
Where("agent.user_id = ? AND agent_binding.binding_type = ? AND agent_binding.binding_target_id = ? AND agent_binding.deleted_at IS NULL", userID, "sub_agent", targetAgentID)
```

- [ ] **Step 3: Pass the current user into agent runtime compilation and resource resolution**

```go
const userIDRuntimeContextKey runtimeContextKey = "openintern_agent_user_id"

func withUserID(ctx context.Context, userID string) context.Context {
    return context.WithValue(ctx, userIDRuntimeContextKey, userID)
}
```

- [ ] **Step 4: Update A2UI and agent controllers to source IDs from context and reject cross-user access**

```go
ownerID := strings.TrimSpace(c.GetString("user_id"))
item, err := agentsvc.Agent.GetByAgentID(ownerID, c.Param("id"))
```

- [ ] **Step 5: Build the backend and smoke-test cross-user agent reads**

Run: `cd openIntern_backend && go build ./...`
Expected: build exits successfully

Run: create an agent under token A, then request that agent under token B
Expected: token B cannot read or mutate the agent

- [ ] **Step 6: Commit the A2UI and agent scoping work**

```bash
git add openIntern_backend/internal/dao/a2ui.go openIntern_backend/internal/dao/agent.go openIntern_backend/internal/dao/agent_binding.go openIntern_backend/internal/services/a2ui/service.go openIntern_backend/internal/services/agent/agent_definition_service.go openIntern_backend/internal/services/agent/agent_runtime_compile.go openIntern_backend/internal/services/agent/agent_runtime_context.go openIntern_backend/internal/controllers/a2ui.go openIntern_backend/internal/controllers/agent.go
git commit -m "refactor: scope a2ui and agents by user"
```

### Task 6: Scope Plugins, Tools, Model Providers, Models, And Defaults By User

**Files:**
- Modify: `openIntern_backend/internal/dao/plugin.go`
- Modify: `openIntern_backend/internal/dao/model_provider.go`
- Modify: `openIntern_backend/internal/dao/model_catalog.go`
- Modify: `openIntern_backend/internal/services/plugin/plugin.go`
- Modify: `openIntern_backend/internal/services/plugin/plugin_search.go`
- Modify: `openIntern_backend/internal/services/model/provider.go`
- Modify: `openIntern_backend/internal/services/model/catalog.go`
- Modify: `openIntern_backend/internal/controllers/plugin.go`
- Modify: `openIntern_backend/internal/controllers/model_provider.go`
- Modify: `openIntern_backend/internal/controllers/model_catalog.go`

- [ ] **Step 1: Require `userID` in plugin DAO CRUD and tool joins**

```go
func (d *PluginDAO) GetByPluginID(userID, pluginID string) (*models.Plugin, error) {
    var item models.Plugin
    if err := database.DB.Where("user_id = ? AND plugin_id = ?", userID, pluginID).First(&item).Error; err != nil {
        return nil, err
    }
    return &item, nil
}

func (d *PluginDAO) ListRuntimeTools(userID, runtimeType, status string, toolIDs []string) ([]models.Tool, error) {
    query := database.DB.Model(&models.Tool{}).
        Joins("JOIN plugin ON plugin.plugin_id = tool.plugin_id AND plugin.user_id = tool.user_id").
        Where("plugin.user_id = ? AND plugin.runtime_type = ? AND plugin.status = ? AND tool.enabled = ?", userID, runtimeType, status, true)
}
```

- [ ] **Step 2: Persist plugin tools and defaults with the same `user_id` as the owning plugin**

```go
item := &models.Plugin{
    UserID:      userID,
    Name:        name,
    RuntimeType: runtimeType,
}

tool := models.Tool{
    UserID:   userID,
    PluginID: item.PluginID,
    ToolName: normalizedToolName,
}
```

- [ ] **Step 3: Scope model provider/model/default-model services by current user**

```go
func (s *ModelProviderService) Create(userID string, input CreateModelProviderInput) (*models.ModelProvider, error) {
    item := &models.ModelProvider{
        UserID:           userID,
        Name:             name,
        APIType:          apiType,
        APIKeyCiphertext: ciphertext,
    }
    return item, dao.ModelProvider.Create(item)
}
```

```go
func (d *DefaultModelConfigDAO) UpsertByConfigKey(userID, configKey, modelID string) (*models.DefaultModelConfig, error) {
    item, err := d.GetByConfigKey(userID, configKey)
    // create or update only inside the active user's scope
}
```

- [ ] **Step 4: Update controllers to pass `user_id` into plugins and models**

```go
userID := strings.TrimSpace(c.GetString("user_id"))
item, err := modelsvc.ModelProvider.Create(userID, input)
```

```go
userID := strings.TrimSpace(c.GetString("user_id"))
items, total, err := pluginsvc.Plugin.List(userID, page, pageSize, filter)
```

- [ ] **Step 5: Align the frontend model-management and plugin-management pages with user-scoped backend responses**

```ts
const [providersRes, modelsRes, defaultModelRes] = await Promise.all([
  requestBackend<BackendPage<ModelProviderItem>>("/v1/model-providers?page=1&page_size=100", {
    fallbackMessage: "获取模型提供商失败",
    router,
  }),
  requestBackend<BackendPage<ModelItem>>("/v1/models?page=1&page_size=200", {
    fallbackMessage: "获取模型失败",
    router,
  }),
  requestBackend<DefaultModelResponse>("/v1/models/default", {
    fallbackMessage: "获取默认模型失败",
    router,
  }),
]);
```

- [ ] **Step 6: Build the backend and manually verify plugin/model isolation**

Run: `cd openIntern_backend && go build ./...`
Expected: build exits successfully

Run: create plugin/model under token A, then list `/v1/plugins` and `/v1/models` under token B
Expected: token B does not see token A resources

- [ ] **Step 7: Commit the plugin/model isolation changes**

```bash
git add openIntern_backend/internal/dao/plugin.go openIntern_backend/internal/dao/model_provider.go openIntern_backend/internal/dao/model_catalog.go openIntern_backend/internal/services/plugin/plugin.go openIntern_backend/internal/services/plugin/plugin_search.go openIntern_backend/internal/services/model/provider.go openIntern_backend/internal/services/model/catalog.go openIntern_backend/internal/controllers/plugin.go openIntern_backend/internal/controllers/model_provider.go openIntern_backend/internal/controllers/model_catalog.go 'openIntern_forentend/app/(workspace)/models/page.tsx'
git commit -m "refactor: scope plugins and models by user"
```

### Task 7: Final Verification, Docs Sync, And Cleanup

**Files:**
- Modify: `docs/local-development.md`
- Inspect: `compose.yaml`
- Inspect: `openIntern_backend/config.yaml`
- Inspect: `openIntern_forentend/app/(workspace)/layout.tsx`
- Inspect: `openIntern_forentend/app/(workspace)/a2ui/page.tsx`

- [ ] **Step 1: Start the external dependencies and verify all services are healthy**

Run: `docker compose up -d`
Expected: three services start successfully

Run: `docker compose ps`
Expected: `mysql`, `redis`, and `minio` are in `running` or healthy state

- [ ] **Step 2: Verify service logs for obvious startup failures**

Run: `docker compose logs --tail=100 mysql redis minio`
Expected: no crash loop, auth failure, or port-binding errors

- [ ] **Step 3: Rebuild the backend after the full refactor**

Run: `cd openIntern_backend && go build ./...`
Expected: build exits successfully with no compile errors

- [ ] **Step 4: Run a targeted manual smoke checklist**

```text
1. Register user A and log in.
2. Register user B and log in.
3. Create thread, agent, A2UI, plugin, provider, and model under user A.
4. Confirm user B cannot read, list, update, or delete those resources.
5. Confirm `/v1/users/me` returns only the current user profile.
6. Confirm MinIO Console opens at `http://127.0.0.1:9001`.
7. Confirm existing COS-backed upload flows are untouched by this round.
```

- [ ] **Step 5: Update docs with any verification caveats discovered during smoke checks**

```md
## Verification

- `docker compose config`
- `docker compose up -d`
- `docker compose ps`
- `docker compose logs --tail=100 mysql redis minio`
- `cd openIntern_backend && go build ./...`
```

- [ ] **Step 6: Commit the final verification/doc sync**

```bash
git add docs/local-development.md
git commit -m "docs: document multi-user local verification flow"
```
