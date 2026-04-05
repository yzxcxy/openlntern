# MinIO 统一对象存储改造设计

## 1. 背景与目标

当前 openIntern 的对象存储能力仍然以 COS 为中心组织：

- 后端启动通过 `cfg.COS` 初始化文件服务
- 聊天附件、用户头像、插件图标上传都直接依赖现有文件服务
- 运行时配置接口和前端设置页仍暴露 `cos` 字段
- 内建工具仍提供 `upload_to_cos`
- 默认头像、插件图标等公共资源 URL 也直接写死为 COS 地址

这套模型存在三个核心问题：

- 基础设施命名已经与目标方案不一致，继续叠加改造会让 MinIO 落地后仍保留大量 COS 残留语义
- 上层业务直接拼对象 key，存储命名规则分散在控制器和业务代码中，边界不清晰
- 缺少统一的“用户私有对象”和“系统公共对象”模型，不利于多用户隔离和后续扩展

本次改造目标如下：

- 全面移除 COS 语义，统一切换为 MinIO
- 使用单桶模型，并采用前缀隔离：
  - `users/<user_id>/...`
  - `public/...`
- `users/...` 用于用户私有对象写入与隔离
- `public/...` 用于所有用户可直接访问的公共对象，只允许受控后端场景写入
- 新增统一对象存储领域服务，所有上层服务通过它访问 MinIO
- 让聊天附件、头像、插件图标、默认公共资源、内建工具上传全部走统一对象存储边界

本次改造接受不兼容重构，不保留对 COS 的兼容逻辑。

## 2. 范围与非目标

### 2.1 本次范围

- 将后端配置、运行时配置接口、前端设置页从 `cos` 全量改为 `minio`
- 用 MinIO SDK 重建对象存储基础设施层
- 新增对象存储领域服务，统一管理对象 key、访问级别和 URL 生成
- 将以下上传链路接入统一对象存储领域服务：
  - 聊天附件
  - 用户头像
  - 插件图标
  - 内建工具上传
- 将默认头像、内建插件图标等公共静态资源统一迁移到 `public/...` 命名空间

### 2.2 非目标

- 不保留 COS 配置兼容字段
- 不做 COS 到 MinIO 的在线迁移逻辑
- 不实现多桶模型
- 不实现普通用户写入 `public/...`
- 不在本轮引入签名 URL、临时凭证或细粒度 bucket policy 管理界面
- 不顺手重构与对象存储无关的模块

## 3. 约束与设计原则

本次改造遵循以下原则：

- 统一边界：所有业务只通过对象存储领域服务访问对象存储
- 命名集中：对象 key 的生成规则只能存在于对象存储层
- 默认私有：普通业务上传默认进入 `users/<user_id>/...`
- 公共只读：`public/...` 对普通用户只读不可写
- 基础设施透明：上层业务不关心 MinIO SDK、bucket 结构、URL 拼接细节
- 明确失败：配置不完整、前缀越权、对象 key 非法时直接失败，不做静默降级或兼容兜底

## 4. 当前现状

### 4.1 文件服务现状

当前文件服务位于 `openIntern_backend/internal/services/storage/file.go`，直接持有 COS client，并暴露：

- `InitFile`
- `UploadWithKey`
- `UploadPath`
- `UploadReader`
- `Delete`

这一层同时承担了基础设施访问和业务对象 key 接收两类职责，导致上层能够直接传入任意 key。

### 4.2 上层上传现状

当前存在以下典型接入点：

- 聊天上传在 `internal/services/chat/upload.go` 中直接构造 key
- 用户头像上传在 `internal/controllers/user.go` 中直接构造 key
- 插件图标上传在 `internal/controllers/plugin.go` 中直接构造 key
- 内建工具在 `internal/services/builtin_tool/cos.go` 中直接接收 `cos_key`

这些调用链路没有统一的访问级别约束，公共对象与私有对象的模型也未显式表达。

### 4.3 配置与前端现状

当前以下位置仍然是 COS 语义：

- `internal/config/config.go`
- `internal/config/runtime_config.go`
- `internal/controllers/config.go`
- 前端设置页
- `builtin_plugins.yaml`
- 默认头像 URL 常量
- 后端 agent 默认头像常量

因此本轮必须从配置模型到业务层做一次完整切换，而不是仅替换 SDK。

## 5. 总体架构

### 5.1 架构分层

对象存储能力拆成两层：

1. `MinIOStore`
   基础设施层，负责和 MinIO SDK 直接交互

2. `ObjectStorageService`
   领域层，负责对象归属、对象命名、访问级别约束和 URL 生成

上层控制器和业务服务只依赖 `ObjectStorageService`。

### 5.2 依赖方向

依赖关系固定如下：

- 控制器 / 业务服务 -> `ObjectStorageService`
- `ObjectStorageService` -> `MinIOStore`
- `MinIOStore` -> MinIO SDK

禁止以下依赖方式：

- 控制器直接依赖 MinIO SDK
- 业务代码直接拼接完整对象 key 后调用底层 SDK
- 前端或配置层保留 COS 命名作为兼容字段

### 5.3 可见性模型

对象统一分为两类：

- 用户对象
  - 前缀：`users/<user_id>/...`
  - 写入主体：普通业务接口或受控后端逻辑
  - 读取方式：当前版本允许返回对象 URL；后续如需改为签名 URL，只改对象存储层

- 公共对象
  - 前缀：`public/...`
  - 写入主体：仅受控后端逻辑
  - 读取方式：所有用户直接访问公开 URL

## 6. 对象命名空间设计

### 6.1 前缀模型

单桶下固定只允许两类根前缀：

- `users`
- `public`

任何对象 key 都必须落在这两个前缀之一，禁止其他根前缀。

### 6.2 用户对象路径规则

用户对象统一使用以下结构：

```text
users/<user_id>/<purpose>/<scope...>/<date>/<uuid>.<ext>
```

说明如下：

- `user_id`：当前登录用户 ID，经过路径段清洗
- `purpose`：对象用途，受枚举约束
- `scope`：按用途补充的上下文路径，可为空
- `date`：`YYYYMMDD`
- `uuid`：随机对象名，避免覆盖冲突
- `ext`：基于原文件名或 MIME type 推导出的扩展名

### 6.3 公共对象路径规则

公共对象统一使用以下结构：

```text
public/<domain>/<resource...>/<file>
```

建议第一版公共对象按以下命名：

- `public/system/avatar/openintern-default.jpg`
- `public/plugin/icon/<name>.png`

公共对象不按用户维度分片，不引入日期目录，要求路径稳定，便于直接引用和缓存。

### 6.4 `purpose` 枚举

第一版建议只允许以下 `purpose`：

- `chat`
- `avatar`
- `plugin`

后续新增用途时，必须在对象存储层显式扩展，而不是由上层随意传字符串。

### 6.5 典型对象 key 示例

- 聊天附件：
  `users/u_123/chat/thread_abc/20260406/550e8400-e29b-41d4-a716-446655440000.pdf`
- 用户头像：
  `users/u_123/avatar/20260406/550e8400-e29b-41d4-a716-446655440000.png`
- 插件图标：
  `users/u_123/plugin/plugin_456/20260406/550e8400-e29b-41d4-a716-446655440000.jpg`
- 系统默认头像：
  `public/system/avatar/openintern-default.jpg`

## 7. 服务设计

### 7.1 `MinIOStore`

`MinIOStore` 只处理 MinIO 本身，不承载业务规则。

建议职责如下：

- 初始化 MinIO client
- 校验 bucket 是否存在
- 执行对象上传
- 执行对象删除
- 生成公开 URL
- 进行基础 key 合法性校验

建议接口如下：

```go
type MinIOStore interface {
    PutObject(ctx context.Context, key string, reader io.Reader, size int64, opts PutObjectOptions) (*StoredObject, error)
    RemoveObject(ctx context.Context, key string) error
    BuildPublicURL(key string) (string, error)
}
```

其中：

- `PutObjectOptions` 至少包含 `ContentType`
- `StoredObject` 包含 `Bucket`、`Key`、`Size`、`ContentType`

### 7.2 `ObjectStorageService`

`ObjectStorageService` 负责所有业务规则。

建议职责如下：

- 根据对象归属和用途生成规范对象 key
- 统一执行对象 key 路径清洗
- 统一限制用户对象与公共对象的写入权限
- 统一为上层返回对象描述和可访问 URL
- 统一删除时的归属校验

建议接口如下：

```go
type ObjectStorageService interface {
    UploadUserObject(ctx context.Context, userID string, spec UploadUserObjectSpec, reader io.Reader, size int64) (*ObjectDescriptor, error)
    UploadSystemPublicObject(ctx context.Context, spec UploadPublicObjectSpec, reader io.Reader, size int64) (*ObjectDescriptor, error)
    DeleteUserObject(ctx context.Context, userID string, objectKey string) error
    BuildPublicURL(objectKey string) (string, error)
}
```

### 7.3 上传参数设计

不允许上层直接上传“完整对象 key”，而是使用结构化参数。

建议的 `UploadUserObjectSpec`：

```go
type UploadUserObjectSpec struct {
    Purpose          string
    ScopeSegments    []string
    OriginalFileName string
    ContentType      string
}
```

建议的 `UploadPublicObjectSpec`：

```go
type UploadPublicObjectSpec struct {
    Domain           string
    ResourceSegments []string
    FileName         string
    ContentType      string
}
```

### 7.4 返回值设计

对象存储层统一返回以下结构：

```go
type ObjectDescriptor struct {
    Bucket      string
    Key         string
    URL         string
    Visibility  string
    ContentType string
    Size        int64
}
```

这样上层不需要自己拼 bucket、key 和 URL，也不需要知道 URL 是否公开。

## 8. 安全与访问控制

### 8.1 用户对象写入约束

普通业务上传只能写入 `users/<user_id>/...`。

具体要求：

- `user_id` 必须来自已认证上下文
- `ObjectStorageService` 必须自己把 `user_id` 写入对象 key
- 上层不得传入任意完整 key 企图覆盖其他用户前缀

### 8.2 公共对象写入约束

`public/...` 只允许受控系统逻辑写入。

本轮定义如下：

- 普通用户接口不能写 `public/...`
- 内建工具不能写 `public/...`
- 前端上传入口不能写 `public/...`
- 仅后端受控初始化或系统管理型逻辑允许调用 `UploadSystemPublicObject`

### 8.3 删除约束

`DeleteUserObject` 必须验证待删除对象 key 是否属于当前用户前缀：

```text
users/<current_user_id>/...
```

若对象 key 不属于当前用户，则直接报错，禁止删除。

### 8.4 对象 key 校验规则

所有对象 key 和路径段必须经过统一校验：

- 禁止空字符串
- 禁止以 `/` 开头
- 禁止包含 `..`
- 禁止出现重复分隔导致的空路径段
- 禁止未落在 `users/...` 或 `public/...` 根前缀

路径段清洗规则与当前聊天上传中的 `sanitizePathSegment` 保持一致，但应上移为对象存储层公共能力。

## 9. URL 策略

### 9.1 公共对象 URL

`public/...` 对象的 URL 由 `public_base_url + "/" + objectKey` 生成。

例如：

```text
https://assets.example.com/public/system/avatar/openintern-default.jpg
```

`public_base_url` 必须由配置显式提供，禁止在代码中拼接 MinIO 内部 endpoint 作为公共访问地址。

### 9.2 用户对象 URL

第一版用户对象 URL 仍由对象存储层统一返回。

当前策略如下：

- 对象存储层统一构建并返回可访问 URL
- 上层仅消费 `ObjectDescriptor.URL`
- 后续如果用户对象需要切到签名 URL 或经网关访问，只改对象存储层

## 10. 配置模型设计

### 10.1 后端配置字段

`config.yaml` 中对象存储配置统一改为：

```yaml
minio:
  endpoint: 127.0.0.1:9000
  access_key: xxx
  secret_key: xxx
  bucket: openintern
  use_ssl: false
  public_base_url: http://127.0.0.1:9000
```

其中：

- `endpoint`：MinIO API endpoint
- `access_key`：访问 key
- `secret_key`：访问密钥
- `bucket`：单桶名称
- `use_ssl`：是否启用 HTTPS
- `public_base_url`：公共 URL 基地址

### 10.2 运行时配置与前端设置页

运行时配置接口和前端设置页统一改为 `minio` 字段。

要求如下：

- 后端 `RuntimeConfig`、`RuntimeConfigResponse` 改为 `MinIO`
- 配置更新请求从 `cos` 改为 `minio`
- 前端设置页的类型、表单状态、保存逻辑、展示文案全部切换为 MinIO
- 不保留 COS 兼容字段或文案

## 11. 现有模块接入方案

### 11.1 聊天附件上传

`internal/services/chat/upload.go` 不再自行拼接完整对象 key，而是改为：

- 传入 `userID`
- 使用 `purpose = chat`
- `ScopeSegments` 包含线程 ID
- 由对象存储层统一生成：
  `users/<user_id>/chat/<thread_id>/<date>/<uuid>.<ext>`

聊天上传保留现有 MIME type 检测、大小限制和媒体类型分类逻辑。

### 11.2 用户头像上传

`internal/controllers/user.go` 改为通过对象存储层上传头像对象：

- `purpose = avatar`
- 无额外 scope
- 对象 key 统一生成：
  `users/<user_id>/avatar/<date>/<uuid>.<ext>`

上传成功后继续把返回的 URL 写入用户资料。

### 11.3 插件图标上传

`internal/controllers/plugin.go` 改为通过对象存储层上传插件图标：

- `purpose = plugin`
- `ScopeSegments` 包含 `plugin_id`

对象 key 统一生成：

```text
users/<user_id>/plugin/<plugin_id>/<date>/<uuid>.<ext>
```

### 11.4 内建工具上传

现有 `upload_to_cos` 不再保留。

建议改为新的 MinIO 语义工具，例如：

- `upload_to_object_storage`

参数不再使用 `cos_key`，而改为结构化字段，例如：

- `purpose`
- `scope_segments`
- `sandbox_path`
- `content_type`（可选）

工具执行约束如下：

- 只能写当前用户的 `users/...` 前缀
- 不允许调用方直接写 `public/...`
- 调用链路内部仍通过统一对象存储领域服务完成上传

### 11.5 默认公共资源

以下静态默认资源统一迁移到 `public/...`：

- 前端默认头像常量
- 后端默认 agent 头像常量
- 内建插件默认图标或图标 URL

代码中不再保留 COS 地址常量。

## 12. 初始化与运行时行为

### 12.1 启动阶段

后端启动时对象存储初始化流程如下：

1. 读取 `cfg.MinIO`
2. 校验配置完整性
3. 初始化 MinIO client
4. 校验 bucket 存在
5. 构建 `MinIOStore`
6. 构建 `ObjectStorageService`

若配置缺失或 bucket 不存在，启动直接失败。

### 12.2 运行时配置更新

运行时更新 MinIO 配置后，本轮建议保持与现有运行时配置模型一致：

- 配置写回 `config.yaml`
- 重新加载配置后刷新内存中的 MinIO 配置对象

若当前实现中对象存储服务在启动时静态初始化，则本轮应保证重新加载后可重新初始化对应服务实例，而不是只改配置值不改实际 client。

## 13. 错误处理与日志

### 13.1 错误分类

对象存储层错误分为三类：

- 配置错误
  - MinIO 配置缺失
  - bucket 不存在
  - endpoint 无法连接

- 业务错误
  - `user_id` 为空
  - `purpose` 非法
  - 非法写入 `public/...`
  - 删除时对象归属不匹配
  - 路径段非法

- 基础设施错误
  - 上传失败
  - 删除失败
  - URL 构建失败

### 13.2 日志要求

日志应记录：

- userID
- purpose
- object key
- size
- content type
- 错误阶段

日志不得记录：

- `access_key`
- `secret_key`
- 原始敏感配置

## 14. 实现边界建议

建议按以下边界组织后端代码：

- `internal/config`
  - MinIO 配置模型
- `internal/services/storage`
  - `minio_store.go`
  - `object_storage.go`
  - 公共路径清洗与 key 生成辅助函数
- `internal/services/chat/upload.go`
  - 改为调用对象存储服务
- `internal/controllers/user.go`
  - 改为调用对象存储服务
- `internal/controllers/plugin.go`
  - 改为调用对象存储服务
- `internal/services/builtin_tool`
  - 改造内建上传工具名称、参数和描述
- `internal/config/runtime_config.go`
  - 运行时配置响应与更新逻辑改为 MinIO
- 前端设置页
  - 改为 MinIO 表单

该结构保证：

- 基础设施层与业务规则层分离
- 对象 key 逻辑只存在一份
- 后续新增业务上传点时只需新增 `purpose` 或 scope 规则

## 15. 验证方案

由于当前协作约束不允许直接执行 `go test`，本轮验证方案定义如下：

### 15.1 编译验证

- 在 `openIntern_backend` 执行 `go build ./...`
- 在 `openIntern_forentend` 执行轻量级 TypeScript 编译检查，前提是项目已有对应命令且耗时可控

### 15.2 手工链路验证

- 配置页能够保存 MinIO 配置
- 聊天附件上传后对象落到：
  `users/<user_id>/chat/...`
- 用户头像上传后对象落到：
  `users/<user_id>/avatar/...`
- 插件图标上传后对象落到：
  `users/<user_id>/plugin/...`
- 默认头像和内建公共资源 URL 指向：
  `public/...`
- 普通业务路径尝试写 `public/...` 时明确失败

### 15.3 结构验证

- 后端不再依赖 COS SDK
- 代码中不再残留 `cos` 配置字段和 `upload_to_cos`
- 默认公共资源常量不再引用 COS URL

## 16. 风险与后续扩展

### 16.1 当前风险

- 若 `public_base_url` 配置不正确，公共资源 URL 会可写不可读
- 若用户对象当前直接返回原始对象 URL，未来切换到签名 URL 时需要保证对象存储层接口保持稳定
- 运行时动态修改 MinIO 配置时，必须保证底层 client 同步刷新，否则配置更新会与实际上传行为不一致

### 16.2 后续可扩展方向

- 为用户对象补充签名 URL
- 为不同 `purpose` 增加生命周期策略
- 为公共资源补充初始化同步工具
- 为对象删除、替换和垃圾清理增加统一审计能力

## 17. 结论

本次改造应将对象存储从“上传文件工具能力”升级为“统一对象命名与访问控制域服务”。

最终结果应满足：

- MinIO 成为唯一对象存储实现
- 单桶下通过 `users/...` 与 `public/...` 完成隔离
- 普通业务只能写用户对象
- 公共对象只读且路径稳定
- 上层服务不再自己拼对象 key
- 配置、后端、前端、内建工具、默认资源全部切换到统一 MinIO 语义
