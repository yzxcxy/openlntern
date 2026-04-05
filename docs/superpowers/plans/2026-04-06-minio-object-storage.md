# MinIO 统一对象存储实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将系统对象存储从 COS 全量切换到 MinIO，落地单桶双前缀模型 `users/...` 与 `public/...`，并通过统一对象存储领域服务收敛所有上传和 URL 生成逻辑。

**Architecture:** 保留 `internal/services/storage` 作为对象存储模块目录，但将其拆成 MinIO 基础设施层与对象存储领域层。上层控制器、聊天上传和内建工具只依赖领域层，不再直接拼接对象 key 或依赖 SDK。配置、运行时配置、前端设置页和默认公共资源一并切到 MinIO 语义。

**Tech Stack:** Go 1.25, Gin, GORM, MinIO Go SDK, Next.js, TypeScript, YAML runtime config

---

## 实施前约束

- 当前用户要求所有操作都在当前分支进行，不创建 worktree 或新分支。
- 当前仓库约束禁止未经允许执行 `go test`。
- 当前仓库约束不建议随意新增 Go test 文件。
- 因此本计划使用以下验证方式：
  - 后端：`go build ./...`
  - 前端：可控耗时的 TypeScript 编译检查
  - 手工对象路径与接口联调验证

## 文件结构与职责

### 后端对象存储模块

- Modify: `openIntern_backend/internal/services/storage/file.go`
  - 将现有 COS 客户端实现替换为 MinIO 基础设施实现
- Create: `openIntern_backend/internal/services/storage/object_storage.go`
  - 定义 `ObjectStorageService`、对象 spec、对象描述、路径生成和访问约束
- Create: `openIntern_backend/internal/services/storage/path_policy.go`
  - 承载路径清洗、前缀校验、用途枚举和 key 构造辅助函数

### 后端配置与启动

- Modify: `openIntern_backend/internal/config/config.go`
  - 将 `COSConfig` 改为 `MinIOConfig`
- Modify: `openIntern_backend/internal/config/runtime_config.go`
  - 将运行时配置与更新逻辑从 `cos` 改为 `minio`
- Modify: `openIntern_backend/internal/controllers/config.go`
  - 更新配置更新请求体字段
- Modify: `openIntern_backend/main.go`
  - 启动时初始化 MinIO 对象存储
- Modify: `openIntern_backend/config.yaml`
  - 切换到 MinIO 配置字段
- Modify: `openIntern_backend/go.mod`
  - 引入 MinIO SDK，移除 COS SDK

### 后端业务接入

- Modify: `openIntern_backend/internal/services/chat/upload.go`
  - 改为通过对象存储领域服务上传聊天附件
- Modify: `openIntern_backend/internal/controllers/user.go`
  - 改为上传用户头像对象
- Modify: `openIntern_backend/internal/controllers/plugin.go`
  - 改为上传用户插件图标对象
- Modify: `openIntern_backend/internal/services/builtin_tool/cos.go`
  - 改为对象存储内建工具
- Modify: `openIntern_backend/builtin_plugins.yaml`
  - 改名并更新工具 schema
- Modify: `openIntern_backend/internal/services/agent/agent_definition_service.go`
  - 切换默认头像常量

### 前端与公共资源

- Modify: `openIntern_forentend/app/(workspace)/settings/page.tsx`
  - 配置类型切到 `minio`
- Modify: `openIntern_forentend/app/(workspace)/settings/components/SystemSettings.tsx`
  - 设置页对象存储表单切到 MinIO
- Modify: `openIntern_forentend/app/shared/avatar.ts`
  - 切换默认头像 URL 到 `public/...`

## Task 1: 切换配置模型与启动初始化

**Files:**
- Modify: `openIntern_backend/internal/config/config.go`
- Modify: `openIntern_backend/internal/config/runtime_config.go`
- Modify: `openIntern_backend/internal/controllers/config.go`
- Modify: `openIntern_backend/main.go`
- Modify: `openIntern_backend/config.yaml`

- [ ] **Step 1: 将静态配置模型改为 MinIO**

```go
type Config struct {
    Port               string                   `yaml:"port"`
    MySQL              MySQLConfig              `yaml:"mysql"`
    Redis              RedisConfig              `yaml:"redis"`
    JWT                JWTConfig                `yaml:"jwt"`
    MinIO              MinIOConfig              `yaml:"minio"`
    Plugin             PluginConfig             `yaml:"plugin"`
    SummaryLLM         LLMConfig                `yaml:"summary_llm"`
    Tools              ToolsConfig              `yaml:"tools"`
    Agent              AgentConfig              `yaml:"agent"`
    ContextCompression ContextCompressionConfig `yaml:"context_compression"`
    APMPlus            APMPlusConfig            `yaml:"apmplus"`
}

type MinIOConfig struct {
    Endpoint      string `yaml:"endpoint" json:"endpoint"`
    AccessKey     string `yaml:"access_key" json:"access_key,omitempty"`
    SecretKey     string `yaml:"secret_key" json:"secret_key,omitempty"`
    Bucket        string `yaml:"bucket" json:"bucket"`
    UseSSL        bool   `yaml:"use_ssl" json:"use_ssl"`
    PublicBaseURL string `yaml:"public_base_url" json:"public_base_url"`
}
```

- [ ] **Step 2: 将运行时配置响应和更新逻辑从 `cos` 切到 `minio`**

```go
type RuntimeConfig struct {
    Agent              AgentConfig              `json:"agent" yaml:"agent"`
    Tools              ToolsConfig              `json:"tools" yaml:"tools"`
    ContextCompression ContextCompressionConfig `json:"context_compression" yaml:"context_compression"`
    Plugin             PluginConfig             `json:"plugin" yaml:"plugin"`
    SummaryLLM         LLMConfig                `json:"summary_llm" yaml:"summary_llm"`
    MinIO              MinIOConfig              `json:"minio" yaml:"minio"`
    APMPlus            APMPlusConfig            `json:"apmplus" yaml:"apmplus"`
}

type RuntimeConfigResponse struct {
    Agent              AgentConfig              `json:"agent"`
    Tools              ToolsConfigResponse      `json:"tools"`
    ContextCompression ContextCompressionConfig `json:"context_compression"`
    Plugin             PluginConfig             `json:"plugin"`
    SummaryLLM         LLMConfigResponse        `json:"summary_llm"`
    MinIO              MinIOConfigResponse      `json:"minio"`
    APMPlus            APMPlusConfigResponse    `json:"apmplus"`
}

func updateMinIOConfig(cfg *MinIOConfig, updates map[string]interface{}) {
    if v, ok := updates["endpoint"].(string); ok {
        cfg.Endpoint = v
    }
    if v, ok := updates["access_key"].(string); ok && v != "" {
        cfg.AccessKey = v
    }
    if v, ok := updates["secret_key"].(string); ok && v != "" {
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
```

- [ ] **Step 3: 更新配置控制器请求体字段**

```go
type UpdateConfigRequest struct {
    Agent              map[string]interface{} `json:"agent,omitempty"`
    Tools              map[string]interface{} `json:"tools,omitempty"`
    ContextCompression map[string]interface{} `json:"context_compression,omitempty"`
    Plugin             map[string]interface{} `json:"plugin,omitempty"`
    SummaryLLM         map[string]interface{} `json:"summary_llm,omitempty"`
    MinIO              map[string]interface{} `json:"minio,omitempty"`
    APMPlus            map[string]interface{} `json:"apmplus,omitempty"`
}
```

- [ ] **Step 4: 将启动初始化切到 MinIO**

```go
accountsvc.InitAuth(cfg.JWT.Secret, cfg.JWT.ExpireMinutes)
if err := storagesvc.InitObjectStorage(cfg.MinIO); err != nil {
    log.Fatalf("failed to init object storage: %v", err)
}
```

- [ ] **Step 5: 切换 `config.yaml` 示例配置**

```yaml
minio:
  endpoint: 127.0.0.1:9000
  access_key: minioadmin
  secret_key: minioadmin
  bucket: openintern
  use_ssl: false
  public_base_url: http://127.0.0.1:9000/openintern
```

- [ ] **Step 6: 运行后端编译检查**

Run: `cd /Users/fqc/project/agent/openIntern/openIntern_backend && go build ./...`

Expected: 编译通过；如失败，错误应集中在后续尚未改造的对象存储调用点或依赖未替换处。

- [ ] **Step 7: 提交当前任务**

```bash
git -C /Users/fqc/project/agent/openIntern add \
  openIntern_backend/internal/config/config.go \
  openIntern_backend/internal/config/runtime_config.go \
  openIntern_backend/internal/controllers/config.go \
  openIntern_backend/main.go \
  openIntern_backend/config.yaml
git -C /Users/fqc/project/agent/openIntern commit -m "refactor: switch config model to minio"
```

## Task 2: 建立 MinIO 基础设施层与对象存储领域层

**Files:**
- Modify: `openIntern_backend/internal/services/storage/file.go`
- Create: `openIntern_backend/internal/services/storage/object_storage.go`
- Create: `openIntern_backend/internal/services/storage/path_policy.go`
- Modify: `openIntern_backend/go.mod`

- [ ] **Step 1: 将 `go.mod` 依赖切换到 MinIO SDK**

```go
require (
    github.com/minio/minio-go/v7 v7.0.95
)
```

并移除：

```go
github.com/tencentyun/cos-go-sdk-v5 v0.7.72
```

- [ ] **Step 2: 在 `path_policy.go` 中定义用途枚举和路径清洗能力**

```go
const (
    ObjectPurposeChat   = "chat"
    ObjectPurposeAvatar = "avatar"
    ObjectPurposePlugin = "plugin"
)

func sanitizePathSegment(value string, fallback string) string

func validateRelativeSegments(segments []string) error

func buildUserObjectKey(userID string, purpose string, scope []string, fileName string, contentType string, now time.Time) (string, error)

func buildPublicObjectKey(domain string, resource []string, fileName string) (string, error)
```

- [ ] **Step 3: 将 `file.go` 重写为 MinIO 基础设施实现**

```go
type MinIOStore struct {
    client        *minio.Client
    bucket        string
    publicBaseURL string
}

func InitObjectStorage(cfg config.MinIOConfig) error

func (s *MinIOStore) PutObject(ctx context.Context, key string, reader io.Reader, size int64, opts PutObjectOptions) (*StoredObject, error)

func (s *MinIOStore) RemoveObject(ctx context.Context, key string) error

func (s *MinIOStore) BuildObjectURL(key string) (string, error)
```

`PutObject` 内部最少应包含：

```go
info, err := s.client.PutObject(ctx, s.bucket, key, reader, size, minio.PutObjectOptions{
    ContentType: opts.ContentType,
})
```

- [ ] **Step 4: 新增对象存储领域服务**

```go
type UploadUserObjectSpec struct {
    Purpose          string
    ScopeSegments    []string
    OriginalFileName string
    ContentType      string
}

type UploadPublicObjectSpec struct {
    Domain           string
    ResourceSegments []string
    FileName         string
    ContentType      string
}

type ObjectDescriptor struct {
    Bucket      string `json:"bucket"`
    Key         string `json:"key"`
    URL         string `json:"url"`
    Visibility  string `json:"visibility"`
    ContentType string `json:"content_type"`
    Size        int64  `json:"size"`
}

type ObjectStorageService struct {
    store *MinIOStore
}

var ObjectStorage = new(ObjectStorageService)
```

- [ ] **Step 5: 实现用户对象上传与公共对象上传约束**

```go
func (s *ObjectStorageService) UploadUserObject(ctx context.Context, userID string, spec UploadUserObjectSpec, reader io.Reader, size int64) (*ObjectDescriptor, error) {
    key, err := buildUserObjectKey(userID, spec.Purpose, spec.ScopeSegments, spec.OriginalFileName, spec.ContentType, time.Now())
    if err != nil {
        return nil, err
    }
    stored, err := s.store.PutObject(ctx, key, reader, size, PutObjectOptions{
        ContentType: spec.ContentType,
    })
    if err != nil {
        return nil, err
    }
    url, err := s.store.BuildObjectURL(stored.Key)
    if err != nil {
        return nil, err
    }
    return &ObjectDescriptor{
        Bucket:      stored.Bucket,
        Key:         stored.Key,
        URL:         url,
        Visibility:  "user",
        ContentType: stored.ContentType,
        Size:        stored.Size,
    }, nil
}
```

`UploadSystemPublicObject` 必须显式限制只生成 `public/...` 前缀对象，且不向普通业务暴露。

- [ ] **Step 6: 实现删除约束**

```go
func (s *ObjectStorageService) DeleteUserObject(ctx context.Context, userID string, objectKey string) error {
    normalizedUserID := sanitizePathSegment(userID, "")
    if normalizedUserID == "" {
        return errors.New("user_id is required")
    }
    expectedPrefix := "users/" + normalizedUserID + "/"
    if !strings.HasPrefix(strings.TrimSpace(objectKey), expectedPrefix) {
        return errors.New("object key does not belong to current user")
    }
    return s.store.RemoveObject(ctx, objectKey)
}
```

- [ ] **Step 7: 运行后端编译检查**

Run: `cd /Users/fqc/project/agent/openIntern/openIntern_backend && go mod tidy && go build ./...`

Expected: MinIO SDK 可解析；若报错，应集中在旧 `storagesvc.File` 调用点尚未迁移。

- [ ] **Step 8: 提交当前任务**

```bash
git -C /Users/fqc/project/agent/openIntern add \
  openIntern_backend/go.mod \
  openIntern_backend/go.sum \
  openIntern_backend/internal/services/storage/file.go \
  openIntern_backend/internal/services/storage/object_storage.go \
  openIntern_backend/internal/services/storage/path_policy.go
git -C /Users/fqc/project/agent/openIntern commit -m "feat: add minio object storage service"
```

## Task 3: 将聊天附件上传接入对象存储领域服务

**Files:**
- Modify: `openIntern_backend/internal/services/chat/upload.go`

- [ ] **Step 1: 用对象存储 spec 替换手写 key**

```go
uploaded, err := storagesvc.ObjectStorage.UploadUserObject(ctx, ownerID, storagesvc.UploadUserObjectSpec{
    Purpose:          storagesvc.ObjectPurposeChat,
    ScopeSegments:    []string{threadID},
    OriginalFileName: fileHeader.Filename,
    ContentType:      mimeType,
}, file, fileHeader.Size)
if err != nil {
    return nil, err
}
```

- [ ] **Step 2: 用对象存储返回值填充响应**

```go
return &ChatUploadAsset{
    Key:       uploaded.Key,
    URL:       uploaded.URL,
    MIMEType:  mimeType,
    FileName:  normalizeFileName(fileHeader.Filename, mimeType),
    Size:      fileHeader.Size,
    MediaKind: classifyMediaKind(mimeType),
}, nil
```

- [ ] **Step 3: 删除聊天服务中不再需要的 key 拼接函数**

删除：

```go
func buildChatUploadObjectKey(ownerID string, threadID string, fileName string, mimeType string) string
```

保留：

- MIME 检测
- 文件大小校验
- 扩展名推导
- 媒体类型分类

- [ ] **Step 4: 运行后端编译检查**

Run: `cd /Users/fqc/project/agent/openIntern/openIntern_backend && go build ./...`

Expected: 聊天上传模块不再引用旧的 key 构造函数，编译通过。

- [ ] **Step 5: 提交当前任务**

```bash
git -C /Users/fqc/project/agent/openIntern add \
  openIntern_backend/internal/services/chat/upload.go
git -C /Users/fqc/project/agent/openIntern commit -m "refactor: route chat uploads through object storage"
```

## Task 4: 将头像与插件图标上传接入对象存储领域服务

**Files:**
- Modify: `openIntern_backend/internal/controllers/user.go`
- Modify: `openIntern_backend/internal/controllers/plugin.go`

- [ ] **Step 1: 改造用户头像上传**

```go
uploaded, err := storagesvc.ObjectStorage.UploadUserObject(c.Request.Context(), userID, storagesvc.UploadUserObjectSpec{
    Purpose:          storagesvc.ObjectPurposeAvatar,
    OriginalFileName: fileHeader.Filename,
    ContentType:      strings.TrimSpace(fileHeader.Header.Get("Content-Type")),
}, file, fileHeader.Size)
if err != nil {
    response.InternalError(c)
    return
}
```

返回值改为：

```go
response.JSONSuccess(c, http.StatusOK, gin.H{
    "key": uploaded.Key,
    "url": uploaded.URL,
})
```

- [ ] **Step 2: 改造插件图标上传**

```go
uploaded, err := storagesvc.ObjectStorage.UploadUserObject(c.Request.Context(), c.GetString("user_id"), storagesvc.UploadUserObjectSpec{
    Purpose:          storagesvc.ObjectPurposePlugin,
    ScopeSegments:    []string{"icon"},
    OriginalFileName: fileHeader.Filename,
    ContentType:      contentType,
}, file, fileHeader.Size)
```

- [ ] **Step 3: 清理旧的 `path/filepath/uuid` 直拼 key 逻辑**

删除控制器中这类代码：

```go
ext := filepath.Ext(fileHeader.Filename)
key := path.Join("plugin", "icon", uuid.NewString()+ext)
```

- [ ] **Step 4: 运行后端编译检查**

Run: `cd /Users/fqc/project/agent/openIntern/openIntern_backend && go build ./...`

Expected: 控制器不再依赖旧的 `UploadWithKey` 签名。

- [ ] **Step 5: 提交当前任务**

```bash
git -C /Users/fqc/project/agent/openIntern add \
  openIntern_backend/internal/controllers/user.go \
  openIntern_backend/internal/controllers/plugin.go
git -C /Users/fqc/project/agent/openIntern commit -m "refactor: use object storage for user assets"
```

## Task 5: 改造内建工具与默认公共资源

**Files:**
- Modify: `openIntern_backend/internal/services/builtin_tool/cos.go`
- Modify: `openIntern_backend/builtin_plugins.yaml`
- Modify: `openIntern_backend/internal/services/agent/agent_definition_service.go`
- Modify: `openIntern_forentend/app/shared/avatar.ts`

- [ ] **Step 1: 将内建工具改为对象存储语义**

重命名输入结构：

```go
type UploadToObjectStorageInput struct {
    Purpose       string   `json:"purpose"`
    ScopeSegments []string `json:"scope_segments"`
    SandboxPath   string   `json:"sandbox_path"`
    ContentType   string   `json:"content_type,omitempty"`
}
```

核心上传逻辑改为：

```go
uploaded, err := storagesvc.ObjectStorage.UploadUserObject(ctx, userID, storagesvc.UploadUserObjectSpec{
    Purpose:          input.Purpose,
    ScopeSegments:    input.ScopeSegments,
    OriginalFileName: path.Base(sandboxPath),
    ContentType:      strings.TrimSpace(input.ContentType),
}, bytes.NewReader(decoded), int64(len(decoded)))
```

- [ ] **Step 2: 更新工具定义与 manifest**

YAML 中替换为：

```yaml
name: "Object Storage"
description: "内建对象存储工具集合，用于把沙箱文件上传到 MinIO 用户空间。"
tool_name: "upload_to_object_storage"
```

输入 schema 替换为：

```json
{
  "type": "object",
  "properties": {
    "purpose": { "type": "string" },
    "scope_segments": {
      "type": "array",
      "items": { "type": "string" }
    },
    "sandbox_path": { "type": "string" },
    "content_type": { "type": "string" }
  },
  "required": ["purpose", "sandbox_path"],
  "additionalProperties": false
}
```

- [ ] **Step 3: 切换默认头像常量到 `public/...`**

后端与前端统一使用稳定公共路径：

```go
const defaultAgentAvatarURL = "http://127.0.0.1:9000/openintern/public/system/avatar/openintern-default.jpg"
```

```ts
export const OPENINTERN_DEFAULT_AVATAR_URL =
  "http://127.0.0.1:9000/openintern/public/system/avatar/openintern-default.jpg";
```

前后端先统一落同一个稳定常量，避免默认资源路径再次分叉。

- [ ] **Step 4: 运行后端与前端编译检查**

Run: `cd /Users/fqc/project/agent/openIntern/openIntern_backend && go build ./...`

Run: `cd /Users/fqc/project/agent/openIntern/openIntern_forentend && pnpm exec tsc --noEmit`

Expected: 后端内建工具、常量与前端默认头像引用全部通过编译。

- [ ] **Step 5: 提交当前任务**

```bash
git -C /Users/fqc/project/agent/openIntern add \
  openIntern_backend/internal/services/builtin_tool/cos.go \
  openIntern_backend/builtin_plugins.yaml \
  openIntern_backend/internal/services/agent/agent_definition_service.go \
  openIntern_forentend/app/shared/avatar.ts
git -C /Users/fqc/project/agent/openIntern commit -m "refactor: switch builtin uploads and defaults to minio"
```

## Task 6: 切换前端设置页到 MinIO 配置

**Files:**
- Modify: `openIntern_forentend/app/(workspace)/settings/page.tsx`
- Modify: `openIntern_forentend/app/(workspace)/settings/components/SystemSettings.tsx`

- [ ] **Step 1: 更新设置页配置类型**

```ts
type ConfigResponse = {
  summary_llm?: {
    model?: string;
    api_key?: string;
    base_url?: string;
    provider?: string;
  };
  minio?: {
    endpoint?: string;
    access_key?: string;
    secret_key?: string;
    bucket?: string;
    use_ssl?: boolean;
    public_base_url?: string;
  };
  apmplus?: {
    host?: string;
    app_key?: string;
    service_name?: string;
    release?: string;
  };
};
```

并将：

```tsx
cos={config?.cos}
```

改为：

```tsx
minio={config?.minio}
```

- [ ] **Step 2: 更新系统设置组件 props 和状态**

```ts
type MinIOConfig = {
  endpoint?: string;
  access_key?: string;
  secret_key?: string;
  bucket?: string;
  use_ssl?: boolean;
  public_base_url?: string;
};
```

状态字段改为：

```ts
const [minioEndpoint, setMinioEndpoint] = useState("");
const [minioAccessKey, setMinioAccessKey] = useState("");
const [minioSecretKey, setMinioSecretKey] = useState("");
const [minioBucket, setMinioBucket] = useState("");
const [minioUseSSL, setMinioUseSSL] = useState(false);
const [minioPublicBaseUrl, setMinioPublicBaseUrl] = useState("");
```

- [ ] **Step 3: 更新保存逻辑**

```ts
const handleSaveMinIO = () => {
  const updates: Record<string, unknown> = {
    endpoint: minioEndpoint,
    bucket: minioBucket,
    use_ssl: minioUseSSL,
    public_base_url: minioPublicBaseUrl,
  };
  if (minioAccessKey.trim()) {
    updates.access_key = minioAccessKey.trim();
  }
  if (minioSecretKey.trim()) {
    updates.secret_key = minioSecretKey.trim();
  }
  onSave("minio", updates);
};
```

- [ ] **Step 4: 更新页面文案**

将：

```tsx
对象存储配置 (COS)
腾讯云对象存储服务配置，用于文件上传和存储
```

改为：

```tsx
对象存储配置 (MinIO)
MinIO 对象存储配置，用于用户私有对象与公共资源管理
```

- [ ] **Step 5: 运行前端编译检查**

Run: `cd /Users/fqc/project/agent/openIntern/openIntern_forentend && pnpm exec tsc --noEmit`

Expected: 设置页不再引用 `cos` 字段和类型。

- [ ] **Step 6: 提交当前任务**

```bash
git -C /Users/fqc/project/agent/openIntern add \
  openIntern_forentend/app/\(workspace\)/settings/page.tsx \
  openIntern_forentend/app/\(workspace\)/settings/components/SystemSettings.tsx
git -C /Users/fqc/project/agent/openIntern commit -m "refactor: switch settings ui to minio"
```

## Task 7: 收尾清理与手工联调验证

**Files:**
- Modify: `openIntern_backend/internal/services/storage/file.go`
- Modify: `openIntern_backend/internal/services/builtin_tool/cos.go`
- Modify: `openIntern_backend/internal/config/config.go`
- Modify: `openIntern_backend/internal/config/runtime_config.go`
- Modify: `openIntern_forentend/app/(workspace)/settings/page.tsx`
- Modify: `openIntern_forentend/app/(workspace)/settings/components/SystemSettings.tsx`
- Modify: `openIntern_forentend/app/shared/avatar.ts`

- [ ] **Step 1: 清理残留 COS 命名**

运行以下搜索并清理业务残留：

```bash
rg -n "(?i)\bcos\b|upload_to_cos|cos_key|Secret ID|Secret Key|腾讯云对象存储" \
  /Users/fqc/project/agent/openIntern/openIntern_backend \
  /Users/fqc/project/agent/openIntern/openIntern_forentend
```

保留项仅限历史设计文档或提交信息，不允许留在运行时代码和配置里。

- [ ] **Step 2: 运行最终编译检查**

Run: `cd /Users/fqc/project/agent/openIntern/openIntern_backend && go build ./...`

Run: `cd /Users/fqc/project/agent/openIntern/openIntern_forentend && pnpm exec tsc --noEmit`

Expected: 全量编译通过。

- [ ] **Step 3: 执行手工联调验证**

验证项：

```text
1. 在设置页保存 MinIO 配置成功
2. 聊天附件上传后对象路径位于 users/<user_id>/chat/...
3. 用户头像上传后对象路径位于 users/<user_id>/avatar/...
4. 插件图标上传后对象路径位于 users/<user_id>/plugin/...
5. 默认头像 URL 指向 public/system/avatar/openintern-default.jpg
6. 普通业务接口无法写入 public/...
```

- [ ] **Step 4: 记录最终验证结果**

在实现总结中必须包含：

```text
- 改了什么
- 为什么这么改
- 如何验证（命令与结果）
```

- [ ] **Step 5: 提交最终清理**

```bash
git -C /Users/fqc/project/agent/openIntern add -A
git -C /Users/fqc/project/agent/openIntern commit -m "refactor: migrate object storage to minio"
```
