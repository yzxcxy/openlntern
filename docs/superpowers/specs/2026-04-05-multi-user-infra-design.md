# 多用户模式与统一外部依赖编排改造设计

## 1. 背景与目标

当前项目以单用户模式为主，部分核心资源默认按全局共享处理，数据库初始化中也保留了针对旧表结构的兼容迁移逻辑。现阶段需要进行一次明确的不兼容改造，直接切换到新的数据库实例，并将核心业务资源统一改造为用户级隔离模型。

本次改造目标如下：

- 支持多用户模式，核心业务数据按用户隔离。
- 使用新的 MySQL 实例冷启动空库，库名继续保持 `open_intern`。
- 在仓库根目录提供统一的 `compose.yaml`，托管外部依赖。
- 新增 MinIO 及其可视化管理入口，但本轮不替换现有 COS 上传链路。
- `openviking` 继续由本地进程管理，暂不纳入 Docker Compose。
- 删除为兼容旧表结构而存在的列存在性判断与旧列清理逻辑。

## 2. 当前现状

### 2.1 配置与启动现状

- 后端默认读取 `openIntern_backend/config.yaml`。
- 前端本地环境变量使用 `openIntern_forentend/.env.local`。
- 后端当前直接初始化 MySQL、Redis、文件存储、OpenViking 和路由服务。
- 项目内目前没有统一的 `docker-compose` 或 `compose.yaml` 来编排 MySQL、Redis 等外部依赖。

### 2.2 数据库初始化现状

当前数据库初始化逻辑位于 `openIntern_backend/internal/database/database.go`，行为包括：

- 使用 `AutoMigrate` 自动建表。
- 对 `message.content`、`message.metadata` 进行显式 `AlterColumn`。
- 使用 `HasColumn` + `DropColumn` 删除历史遗留字段，例如：
  - `a2ui.type`
  - `a2ui.user_id`
  - `thread.owner_id`
  - `memory_sync_state.openviking_session_id`

这些逻辑是为兼容旧数据库实例存在的。由于本次直接切换到新数据库实例并冷启动空库，这类兼容逻辑不再需要，继续保留只会增加维护噪音并掩盖真实模型定义。

### 2.3 用户模型与资源归属现状

当前代码中仅部分资源显式具备用户归属：

- `agent` 使用 `owner_id`
- 控制器中很多接口通过中间件读取 `user_id`

但仍有大量资源仍偏全局模型，例如：

- `thread`
- `message`
- `a2ui`
- `plugin`
- `tool`
- `plugin_default`
- `model_provider`
- `model_catalog`
- `default_model_config`
- 多个 memory 相关表

同时，认证 token 和中间件仍保留 `role` 语义，与“不再区分管理员和普通账号”的新要求不一致。

## 3. 范围与非目标

### 3.1 本次范围

- 新增统一 `compose.yaml`，仅编排外部依赖。
- 使用新 MySQL 实例冷启动空库。
- 保留库名 `open_intern`。
- 将核心业务资源统一改造成用户隔离模型。
- 去掉用户角色区分带来的后台管理语义。
- 新增 MinIO 和 MinIO Console。
- 更新启动文档和配置说明。

### 3.2 非目标

- 不做旧数据库数据迁移。
- 不做 COS 到 MinIO 的对象存储切换。
- 不把 `openIntern_backend` 或 `openIntern_forentend` 放进 Compose。
- 不把 `openviking` 纳入 Compose。
- 不引入组织、租户、团队等更高层级的多租户模型。
- 不保留管理员查看或管理所有用户资源的能力。

## 4. 总体方案

### 4.1 依赖编排方案

在仓库根目录新增统一 `compose.yaml`，只管理外部依赖：

- `mysql`
- `redis`
- `minio`

Compose 负责：

- 提供一致的本地依赖启动入口。
- 为 MySQL、Redis、MinIO 提供持久化数据卷。
- 暴露 MinIO 的对象服务端口与 Console 管理端口。

本轮不纳入 Compose 的服务：

- `openIntern_backend`
- `openIntern_forentend`
- `openviking`
- sandbox

其中 sandbox 继续按现有单独运行说明处理；`openviking` 继续保持本地进程管理模式。

### 4.2 数据库切换方案

- 使用新的 MySQL 实例。
- 数据库名保持为 `open_intern`。
- 不兼容旧实例，不复用旧库，不写迁移脚本。
- 后端启动时直接连接新实例并执行建表逻辑。

数据库初始化策略调整为：

- 保留 `AutoMigrate` 作为冷启动建表手段。
- 保留必要的显式列类型修正，例如 `LONGTEXT` 放大。
- 删除所有基于旧库兼容的 `HasColumn`/`DropColumn` 分支。

### 4.3 多用户隔离方案

统一规则如下：

- 除用户认证本身外，所有核心业务资源默认属于某个用户。
- 所有业务查询必须显式带 `user_id`。
- 所有资源唯一性约束从全局唯一改为用户维度复合唯一。
- 所有跨资源关联关系必须在用户维度内闭合。

本次不引入 `tenant_id` 或组织概念，只采用单层 `user_id` 隔离模型。

## 5. 数据模型改造设计

### 5.1 用户模型

`user` 表保留 `user_id` 作为业务主键。

角色处理策略：

- 不再区分 `admin` 和 `user`。
- 认证 token 不再承载角色授权语义。
- 后端接口不再保留后台管理型权限分层。

实现建议：

- 优先删除 `role` 字段及其使用链路。
- 如果需要短期兼容前端字段展示，可以保留数据库列但在业务层停止使用。

由于本次是新库冷启动，推荐直接移除 `role` 的业务依赖并同步清理接口返回中的角色语义。

### 5.2 核心资源新增 `user_id`

以下表统一新增 `user_id` 字段，并建立索引：

- `thread`
- `message`
- `a2ui`
- `agent`
- `agent_binding`
- `plugin`
- `tool`
- `plugin_default`
- `model_provider`
- `model_catalog`
- `default_model_config`
- `memory_sync_state`
- `memory_usage_log`
- `thread_context_snapshot`

`agent` 当前使用 `owner_id`，本次统一重命名为 `user_id`，避免同一代码库中并存多种归属字段命名。

### 5.3 唯一索引与查询键设计

统一索引策略分为两类：

#### 详情查询键

对按业务 ID 查询详情的资源，使用 `user_id + business_id` 作为核心查询键。例如：

- `thread`: `user_id + thread_id`
- `agent`: `user_id + agent_id`
- `plugin`: `user_id + plugin_id`
- `tool`: `user_id + tool_id`
- `model_provider`: `user_id + provider_id`
- `model_catalog`: `user_id + model_id`

#### 用户内唯一键

原先全局唯一或默认全局去重的资源，改为用户维度复合唯一。例如：

- `a2ui`: `user_id + name`
- `agent_binding`: `user_id + agent_id + binding_type + binding_target_id`
- `tool`: `user_id + plugin_id + tool_name`
- `plugin_default`: `user_id + model_id + tool_id`
- `default_model_config`: `user_id + config_key`
- `model_catalog`: `user_id + provider_id + model_key`

对于 `message`，保留线程内顺序设计，并将业务查询统一为：

- `user_id + thread_id`
- `user_id + thread_id + sequence`

这样可以在不依赖额外线程查询的情况下直接完成用户级消息读取与校验。

### 5.4 关联关系规则

所有跨表读取和写入必须满足以下规则：

- `message` 必须属于某个 `user_id` 下的 `thread`
- `tool` 必须属于某个 `user_id` 下的 `plugin`
- `agent_binding` 的来源 agent 与绑定目标都必须在同一个 `user_id` 范围内
- `model_catalog` 必须属于某个 `user_id` 下的 `model_provider`
- `plugin_default` 只能引用当前用户自己的 model 与 tool
- memory 相关状态表必须只追踪当前用户自己的线程上下文

数据库层面仍保持“不创建物理外键约束”的现有策略，但业务层查询必须显式带 `user_id` 保证关联闭合。

## 6. 后端代码层改造设计

### 6.1 认证与中间件

当前 JWT claims 中包含：

- `user_id`
- `role`

本次改造后：

- token 继续承载 `user_id`
- 去掉 `role` claim
- 中间件不再向上下文写入 `role`
- token 刷新逻辑仅依赖 `user_id`

涉及范围：

- `openIntern_backend/internal/services/account/auth.go`
- `openIntern_backend/internal/middleware/auth.go`
- 所有读取 `role` 的控制器与业务代码

### 6.2 DAO 层改造原则

所有 DAO 接口签名统一改造为显式接收 `userID`，避免依赖上层控制器自行拼接过滤条件。

约束要求：

- 不允许在 DAO 内提供“无 user_id 过滤”的普通业务查询入口。
- 列表、详情、更新、删除都必须强制使用 `user_id`。
- 涉及跨表 join 的查询必须同时校验关联表用户归属。

这意味着多用户隔离必须沉到 DAO 层，而不是停留在 Controller 层约定。

### 6.3 Service 层改造原则

Service 层统一按“当前登录用户只能访问自己的资源”收口：

- 创建时自动写入 `user_id`
- 读取时只查当前 `user_id`
- 更新和删除时必须以 `user_id + business_id` 做条件
- 跨资源组合操作时校验所有参与资源均归属于同一 `user_id`

### 6.4 Controller 与路由语义调整

当前 `/v1/users` 路由组仍具备后台管理色彩，例如：

- 创建任意用户
- 列出所有用户
- 获取指定用户
- 修改指定用户
- 删除指定用户

这与“不再区分管理和普通账号”冲突。

推荐改造为自服务语义：

- 保留 `register`
- 保留 `login`
- 新增或保留“当前用户信息读取”
- 保留“当前用户资料更新”
- 保留“当前用户头像上传”
- 移除“列出所有用户”和“任意删除其他用户”的通用能力

如果必须保留部分接口路径，则其语义也应收缩为仅允许访问当前登录用户自己的数据。

### 6.5 配置接口策略

`/v1/config` 当前表现更接近系统级全局配置入口。由于模型已经切为全用户隔离，本次需要对配置语义做明确区分：

- 运行时系统配置仍属于服务实例级配置，继续由本地配置文件承载。
- 业务资源配置，例如默认模型、插件默认关系等，全部改为用户级存储。

本次优先保证数据库内的“用户业务配置”完成隔离；文件级运行配置仍保持单实例配置，不做每用户一份配置文件。

## 7. 基础设施与配置设计

### 7.1 Compose 管理对象

根目录新增 `compose.yaml`，编排以下服务：

- MySQL
- Redis
- MinIO

每个服务要求：

- 明确容器名或服务名，便于排障。
- 使用持久化卷，避免每次重启丢失数据。
- 提供健康检查或至少具备可观察的启动日志。

### 7.2 MinIO 设计

本轮 MinIO 的定位是“基础设施预埋”，不是立即替代 COS。

要求：

- 暴露 S3 兼容对象服务端口。
- 暴露 MinIO Console 端口作为可视化管理入口。
- 提供根账号与密码配置入口。
- 持久化对象数据。

本轮不做以下事项：

- 不修改 `openIntern_backend/internal/services/storage/file.go`
- 不新增 MinIO 上传适配器
- 不切换插件图标、头像、聊天附件、知识库文件上传链路

### 7.3 后端默认连接配置

后端仍通过 `openIntern_backend/config.yaml` 读取运行配置。

本轮需要调整默认连接配置，使其能够直连 Compose 暴露的依赖，例如：

- MySQL 连接到新实例地址
- Redis 连接到 Compose 暴露端口

由于后端本身不运行在容器内，因此默认连接地址应继续以本机可访问地址为主，而不是 Compose 内网服务名。

### 7.4 OpenViking 处理策略

`openviking` 本轮保持本地进程模式，不进入 Compose。

后端继续：

- 读取 OpenViking 本地配置
- 本地拉起 OpenViking
- 响应启动、停止、重启控制请求

这保证本轮只解决“外部依赖统一编排”，不混入额外的运行时迁移风险。

## 8. 数据库初始化策略

`openIntern_backend/internal/database/database.go` 调整目标如下：

- 保留模型注册与 `AutoMigrate`
- 保留必要的显式 `AlterColumn`
- 删除所有旧字段兼容分支

具体原则：

- 新库只接受当前模型定义，不对历史脏结构做容错。
- 如果建表失败，应直接报错并阻止服务启动。
- 不再使用“列存在则删除”的兼容写法，因为这会掩盖模型与数据库真实结构的不一致。

## 9. 文档与开发流程调整

需要补充或更新的文档内容：

- 根目录或后端启动文档新增 Compose 启动步骤。
- 明确本地开发顺序：
  - 先启动 Compose 依赖
  - 再启动后端
  - 再启动前端
  - 如需 memory/openviking 相关能力，再由后端本地管理 OpenViking
- 明确说明 MinIO 本轮仅部署，不接入现有 COS 上传链路。

## 10. 验证策略

遵循仓库约束：

- 不执行 `go test`
- 不执行 `pnpm lint` 等长耗时检查

本次推荐验证方式如下：

### 10.1 基础设施验证

- `docker compose config`
- `docker compose up -d`
- `docker compose ps`
- `docker compose logs --tail=100 mysql`
- `docker compose logs --tail=100 redis`
- `docker compose logs --tail=100 minio`

验证目标：

- MySQL 可启动并初始化新库。
- Redis 可接受连接。
- MinIO API 与 Console 均可访问。

### 10.2 后端验证

优先使用定向编译或最小启动验证，例如：

- `go build ./...` 或更小范围的定向构建
- 启动后端并观察是否能完成数据库初始化与 Redis 初始化

验证目标：

- 新模型可在新库中冷启动建表成功。
- 认证链路不再依赖角色字段。
- 用户资源查询均按 `user_id` 隔离。

### 10.3 联调验证

进行必要的最小接口 smoke test，验证：

- 注册、登录正常
- 当前用户只能看到自己的 agent、thread、plugin、model 数据
- 默认模型、插件默认关系等配置已用户隔离

## 11. 实施顺序

建议按以下顺序实施：

1. 新增根目录 `compose.yaml`，完成 MySQL、Redis、MinIO 编排。
2. 更新依赖启动文档，明确本地开发流程。
3. 修改后端默认配置，指向新的 MySQL/Redis 依赖。
4. 精简数据库初始化逻辑，删除旧列兼容分支。
5. 修改模型定义，为核心资源补齐 `user_id` 并统一索引。
6. 将 `agent.owner_id` 统一改为 `user_id`。
7. 调整 DAO 层接口签名和查询条件，强制带 `user_id`。
8. 调整 Service 层创建、查询、更新、删除逻辑。
9. 调整 Controller 与 `/v1/users` 路由语义，去除后台管理式接口能力。
10. 做 Compose、后端启动和最小联调验证。

## 12. 风险与控制

### 12.1 风险

- 用户隔离改造范围较广，容易遗漏某些全局查询。
- 复合唯一索引调整后，部分现有业务去重逻辑可能需要同步调整。
- `/v1/users` 语义改变会影响前端现有调用方式。
- MinIO 虽未接入业务链路，但引入后需要避免误导为“已完成对象存储切换”。

### 12.2 控制措施

- 统一从模型、DAO、Service、Controller 四层逐层收口，不只改单层。
- 所有业务列表和详情查询逐个补 `user_id` 过滤。
- 文档中明确 MinIO 仅部署，不承担当前上传流量。
- 使用新数据库实例冷启动，避免旧库脏结构干扰本轮改造判断。

## 13. 结论

本次采用“方案 1”执行：

- 使用新的 MySQL 实例冷启动空库，库名保持 `open_intern`
- 使用统一 Compose 只编排外部依赖
- 新增 MinIO 与可视化管理入口
- `openviking` 继续本地管理
- 全量核心业务资源改为用户级隔离
- 不再保留角色区分和旧库兼容迁移逻辑

该方案与当前需求最一致，且在不承担旧数据迁移成本的前提下，能够最大程度降低后续维护复杂度。
