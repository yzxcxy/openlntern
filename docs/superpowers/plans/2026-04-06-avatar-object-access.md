# Avatar Object Access Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将头像存储从“数据库存完整 MinIO URL”改为“数据库存对象 key、接口返回可访问地址”，修复前端头像访问失败与 `127.0.0.1` 暴露问题。

**Architecture:** 后端对象存储层新增“对象 key 解析为签名资源地址”和“按 key 读取对象流”能力；用户登录与用户信息接口改为返回响应 DTO；启动阶段执行一次已知旧头像 URL 到对象 key 的显式迁移；前端统一把后端返回的相对资源路径映射到 `/api/backend` 代理地址。

**Tech Stack:** Go 1.25, Gin, GORM, MinIO Go SDK, Next.js, TypeScript

---

## 实施前约束

- 当前仓库明确禁止未经允许执行 `go test`。
- 当前仓库明确禁止执行 `pnpm lint` 之类高耗时命令。
- 当前任务不新增 Go test 文件。
- 因此本计划使用以下验证方式：
  - 后端：`go build ./...`
  - 前端：`npx tsc --noEmit`
  - 手工请求验证：上传头像、读取用户信息、请求头像资源地址

## 文件结构与职责

- Modify: `openIntern_backend/internal/services/storage/file.go`
  - 增加对象读取与对象 key 提取能力
- Modify: `openIntern_backend/internal/services/storage/object_storage.go`
  - 增加对象访问 URL 解析与对象读取封装
- Modify: `openIntern_backend/internal/services/storage/path_policy.go`
  - 保留对象 key 路径辅助能力
- Modify: `openIntern_backend/internal/controllers/user.go`
  - 上传头像时只存 key；登录和当前用户接口返回 DTO
- Create: `openIntern_backend/internal/controllers/asset.go`
  - 新增受控对象读取接口
- Modify: `openIntern_backend/internal/services/account/user.go`
  - 增加头像字段迁移入口
- Modify: `openIntern_backend/internal/dao/user.go`
  - 增加头像迁移所需的数据访问方法
- Modify: `openIntern_backend/internal/models/user.go`
  - 修正 `avatar` 字段注释语义
- Modify: `openIntern_backend/internal/routers/router.go`
  - 注册资源读取路由
- Modify: `openIntern_backend/main.go`
  - 在启动阶段执行头像字段迁移
- Create: `openIntern_forentend/app/shared/backend-url.ts`
  - 统一把后端相对资源路径映射到 `/api/backend`
- Modify: `openIntern_forentend/app/(workspace)/user/page.tsx`
  - 使用统一头像 URL 归一化
- Modify: `openIntern_forentend/app/(workspace)/layout.tsx`
  - 使用统一头像 URL 归一化
- Modify: `openIntern_forentend/app/(workspace)/agents/editor/page.tsx`
  - 使用统一头像 URL 归一化
- Modify: `openIntern_forentend/app/(workspace)/chat/page.semi.tsx`
  - 使用统一头像 URL 归一化

### Task 1: 扩展后端对象存储边界

**Files:**
- Modify: `openIntern_backend/internal/services/storage/file.go`
- Modify: `openIntern_backend/internal/services/storage/object_storage.go`
- Modify: `openIntern_backend/internal/services/storage/path_policy.go`

- [ ] **Step 1: 保持对象 key 路径辅助能力可复用**

实现要求：

- 继续集中维护 `users/...` 与 `public/...` 路径约束
- 不把对象 key 规则散落到控制器中

- [ ] **Step 2: 为 MinIO 基础设施层增加对象读取能力**

```go
type ReadObjectResult struct {
	Reader      io.ReadCloser
	ContentType string
	Size        int64
}

func (s *MinIOStore) GetObject(ctx context.Context, key string) (*ReadObjectResult, error)
```

实现要求：

- 对 key 做 `normalizeObjectKey`
- 调用 MinIO `GetObject`
- 显式 `Stat` 一次，确保对象真实存在且能拿到 `ContentType` / `Size`

- [ ] **Step 3: 为对象存储领域层增加访问地址解析**

```go
func (s *ObjectStorageService) ResolveObjectAccessURL(storedValue string) (string, error)
```

实现要求：

- `public/...` 返回 `/v1/assets/public/...`
- `users/...` 返回 `/v1/assets/users/...?...` 形式的签名资源地址
- 兼容将已知旧 MinIO URL 解析回对象 key，供迁移与响应转换复用

- [ ] **Step 4: 为对象存储领域层增加对象读取封装**

```go
type ObjectReadResult struct {
	Reader      io.ReadCloser
	ContentType string
	Size        int64
}

func (s *ObjectStorageService) ReadObject(ctx context.Context, objectKey string) (*ObjectReadResult, error)
```

- [ ] **Step 5: 运行后端编译检查**

Run: `cd /Users/fqc/project/agent/openIntern/openIntern_backend && go build ./...`

Expected: 编译通过；如失败，应集中在尚未接通的新接口调用点。

### Task 2: 调整用户接口与头像资源读取

**Files:**
- Modify: `openIntern_backend/internal/controllers/user.go`
- Create: `openIntern_backend/internal/controllers/asset.go`
- Modify: `openIntern_backend/internal/routers/router.go`
- Modify: `openIntern_backend/internal/models/user.go`

- [ ] **Step 1: 定义用户响应 DTO 和转换函数**

```go
type userResponse struct {
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone"`
	Avatar    string    `json:"avatar"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
```

转换要求：

- 读取数据库中的 `user.Avatar` 对象 key
- 若非空，则调用 `storagesvc.ObjectStorage.ResolveObjectAccessURL`
- 若为空，则保持空字符串

- [ ] **Step 2: 修改登录与当前用户接口返回 DTO**

实现要求：

- `Login` 中返回的 `user` 使用 DTO
- `GetCurrentUser` 返回 DTO
- 不再把 GORM `models.User` 原始结构直接暴露给前端

- [ ] **Step 3: 修改头像上传逻辑为只存 key**

```go
if err := accountsvc.User.UpdateUser(userID, map[string]interface{}{"avatar": uploaded.Key}); err != nil {
	return
}

resolvedURL, err := storagesvc.ObjectStorage.ResolveObjectAccessURL(uploaded.Key)
```

返回值要求：

- `key` 返回对象 key
- `url` 返回解析后的资源地址

- [ ] **Step 4: 新增头像/对象代理读取控制器**

```go
func GetObjectAsset(c *gin.Context) {
	objectKey := strings.TrimPrefix(c.Param("objectKey"), "/")
	if strings.HasPrefix(objectKey, "users/") {
		expiresAt, _ := strconv.ParseInt(c.Query("expires"), 10, 64)
		if !storagesvc.ObjectStorage.VerifyObjectAccessSignature(objectKey, expiresAt, c.Query("signature")) {
			response.Forbidden(c)
			return
		}
	}
	result, err := storagesvc.ObjectStorage.ReadObject(c.Request.Context(), objectKey)
	if err != nil {
		response.NotFound(c, "object not found")
		return
	}
	defer result.Reader.Close()
	c.DataFromReader(http.StatusOK, result.Size, result.ContentType, result.Reader, nil)
}
```

实现要求：

- 接口路径：`GET /v1/assets/*objectKey`
- 不依赖 `Authorization` 头
- `users/...` 必须带有效签名
- `public/...` 允许直接读取

- [ ] **Step 5: 修正用户模型注释**

```go
Avatar string `gorm:"size:255" json:"avatar"` // 头像对象 key
```

- [ ] **Step 6: 运行后端编译检查**

Run: `cd /Users/fqc/project/agent/openIntern/openIntern_backend && go build ./...`

Expected: 编译通过。

### Task 3: 增加头像字段显式迁移

**Files:**
- Modify: `openIntern_backend/internal/services/account/user.go`
- Modify: `openIntern_backend/internal/dao/user.go`
- Modify: `openIntern_backend/main.go`

- [ ] **Step 1: 增加头像迁移所需 DAO 方法**

```go
func (d *UserDAO) ListUsersWithAvatar() ([]models.User, error)

func (d *UserDAO) UpdateAvatarByUserID(userID string, avatar string) error
```

- [ ] **Step 2: 增加旧头像 URL 到对象 key 的显式迁移逻辑**

```go
func (s *UserService) MigrateAvatarObjectKeys() error
```

迁移规则：

- 空值直接跳过
- 已经是 `users/` 或 `public/` key 的值直接跳过
- 若值匹配当前 MinIO `public_base_url + "/"` 前缀，则裁剪为对象 key 并回写数据库
- 若值匹配 `scheme://endpoint/bucket/` 前缀，则裁剪为对象 key 并回写数据库
- 其他值只记录日志，不做静默兼容

- [ ] **Step 3: 在应用启动阶段执行迁移**

```go
if err := accountsvc.User.MigrateAvatarObjectKeys(); err != nil {
	log.Fatalf("failed to migrate avatar object keys: %v", err)
}
```

位置要求：

- 在 `InitObjectStorage(cfg.MinIO)` 成功之后
- 在路由启动之前

- [ ] **Step 4: 运行后端编译检查**

Run: `cd /Users/fqc/project/agent/openIntern/openIntern_backend && go build ./...`

Expected: 编译通过。

### Task 4: 统一前端头像资源地址

**Files:**
- Create: `openIntern_forentend/app/shared/backend-url.ts`
- Modify: `openIntern_forentend/app/(workspace)/user/page.tsx`
- Modify: `openIntern_forentend/app/(workspace)/layout.tsx`
- Modify: `openIntern_forentend/app/(workspace)/agents/editor/page.tsx`
- Modify: `openIntern_forentend/app/(workspace)/chat/page.semi.tsx`

- [ ] **Step 1: 新增后端资源地址归一化函数**

```ts
const BACKEND_API_BASE = "/api/backend";

export const resolveBackendAssetUrl = (value?: string) => {
  const trimmed = (value ?? "").trim();
  if (!trimmed) return "";
  if (/^https?:\/\//i.test(trimmed) || trimmed.startsWith("data:")) {
    return trimmed;
  }
  if (trimmed.startsWith("/")) {
    return `${BACKEND_API_BASE}${trimmed}`;
  }
  return trimmed;
};
```

- [ ] **Step 2: 用户页头像渲染接入归一化函数**

替换：

```tsx
src={userInfo?.avatar || OPENINTERN_DEFAULT_AVATAR_URL}
```

为：

```tsx
src={resolveBackendAssetUrl(userInfo?.avatar) || OPENINTERN_DEFAULT_AVATAR_URL}
```

- [ ] **Step 3: 工作区布局、Agent 编辑器、聊天页的用户头像渲染接入同一函数**

实现要求：

- 仅替换用户头像来源
- 不改动 agent 默认头像、系统默认头像等无关逻辑

- [ ] **Step 4: 运行前端类型检查**

Run: `cd /Users/fqc/project/agent/openIntern/openIntern_forentend && npx tsc --noEmit`

Expected: 类型检查通过。

### Task 5: 手工验证链路

**Files:**
- No code changes

- [ ] **Step 1: 启动后端并上传头像**

验证点：

- 上传接口返回 `key` 与 `url`
- `url` 以 `/v1/assets/` 开头，而不是 `http://127.0.0.1:9000`
- `users/...` 资源地址带签名参数

- [ ] **Step 2: 检查数据库中的 `user.avatar`**

Expected:

- 存的是 `users/<user_id>/avatar/...` 对象 key
- 不再是完整 URL

- [ ] **Step 3: 请求用户信息接口**

Run: `curl -s -H 'Authorization: Bearer <token>' http://127.0.0.1:8080/v1/users/me`

Expected:

- `data.avatar` 是 `/v1/assets/users/...?...` 路径

- [ ] **Step 4: 通过前端代理地址请求头像资源**

Run: `curl -I 'http://localhost:3000/api/backend/v1/assets/users/<user_id>/avatar/<date>/<file>.png?expires=<ts>&signature=<sig>'`

Expected:

- 返回 `200 OK`
- `Content-Type` 为图片类型

- [ ] **Step 5: 提交实现**

```bash
git -C /Users/fqc/project/agent/openIntern add \
  openIntern_backend/internal/services/storage/file.go \
  openIntern_backend/internal/services/storage/object_storage.go \
  openIntern_backend/internal/services/storage/path_policy.go \
  openIntern_backend/internal/controllers/user.go \
  openIntern_backend/internal/controllers/asset.go \
  openIntern_backend/internal/routers/router.go \
  openIntern_backend/internal/services/account/user.go \
  openIntern_backend/internal/dao/user.go \
  openIntern_backend/internal/models/user.go \
  openIntern_backend/main.go \
  openIntern_forentend/app/shared/backend-url.ts \
  'openIntern_forentend/app/(workspace)/user/page.tsx' \
  'openIntern_forentend/app/(workspace)/layout.tsx' \
  'openIntern_forentend/app/(workspace)/agents/editor/page.tsx' \
  'openIntern_forentend/app/(workspace)/chat/page.semi.tsx'
git -C /Users/fqc/project/agent/openIntern commit -m "fix: serve user avatars through app asset URLs"
```
