# AIO Sandbox 用户级实例生命周期改造设计

## 1. 背景与目标

当前 openIntern 对 AIO sandbox 的接入方式是“全局固定 base URL”模型：

- 后端启动时强依赖 `tools.sandbox.url`
- Agent 运行时从该固定 URL 拉取 `sandbox_execute_bash`
- Code 调试、Code 插件执行、COS 上传前的文件读取也都直接调用该固定 URL

这套模型只适合单实例共享，不支持“每个用户一个 sandbox 实例”的运行方式。当前需要将 sandbox 改造成用户级常驻实例模型，并满足以下约束：

- 每个用户最多绑定一个 sandbox 实例
- 首次真正使用 sandbox 相关能力时才懒创建实例
- 实例在用户空闲后由后台自动回收
- 实例回收后允许文件系统和进程状态丢失，不做恢复
- 当前仍处于本地开发阶段，仅实现 Docker CLI 方案
- 暂不实现 SCF，只在模型和接口中保留扩展位
- 暂不处理实例内并发锁、文件路径隔离、浏览器/Jupyter 会话隔离

本次改造接受不兼容重构，不保留旧的全局 `tools.sandbox.url` 接入模式。

## 2. 当前现状

### 2.1 全局 URL 注入

当前后端初始化时会读取 `tools.sandbox.url`，如果为空则直接启动失败。随后该 URL 会被放入全局运行时状态，并传给插件服务中的 package 级变量。

这意味着：

- 所有用户共享同一个 sandbox endpoint
- sandbox 不能按用户维度动态切换
- sandbox 生命周期无法与用户行为绑定

### 2.2 工具接入现状

当前 sandbox 相关能力分为两类：

- MCP 工具：
  - Agent 运行时固定连接 `baseURL + /mcp`
  - 拉取 `sandbox_execute_bash` 工具并注入运行时
- HTTP 直连能力：
  - Code 执行使用 `POST /v1/code/execute`
  - COS 上传前的读文件使用 `POST /v1/file/read`

这三类能力当前都直接依赖固定 base URL。

### 2.3 当前文档和本地运行方式

仓库内当前已有本地 sandbox 启动说明：

- 使用 Docker 运行 AIO sandbox 镜像
- 宿主机固定映射到 `127.0.0.1:8081`
- MCP endpoint 固定为 `/mcp`

这套说明仍然是单容器共享模式，不满足用户级实例管理需求。

## 3. 范围与非目标

### 3.1 本次范围

- 将 sandbox 接入方式改为用户级实例生命周期管理
- 新增本地 Docker CLI provider
- 新增实例元数据表与 DAO
- 实现懒创建、复用、续期、自动回收
- 支持后端重启后自动接管已有容器
- 将 Code 执行、COS 文件读取、bash 执行改为运行时按用户解析实例 endpoint
- 删除旧的全局 sandbox base URL 模型

### 3.2 非目标

- 不实现 SCF provider
- 不考虑线上多副本部署
- 不做 Redis 分布式锁
- 不做实例内并发任务锁
- 不做工作目录隔离
- 不做 Jupyter、Browser、Shell 的会话级治理
- 不做实例状态恢复或持久化文件系统
- 不将 sandbox 暴露为普通插件或参与插件同步体系

## 4. 总体方案

### 4.1 sandbox 作为系统内建能力

本次改造后，AIO sandbox 不再属于插件系统，不再同步到插件表，也不作为用户可配置插件存在。

它改为后端内部的一类系统能力：

- Agent 在需要 sandbox 能力时，调用本地定义的内建工具
- 内建工具执行时，再按当前 `user_id` 解析或创建用户实例
- 得到该用户的 endpoint 后，再调用对应的 sandbox HTTP 或 MCP 接口

也就是说：

- “是否支持 sandbox”是系统能力
- “当前用户对应哪个 sandbox 实例”是运行时资源问题

二者不再混用。

### 4.2 生命周期模型

生命周期采用“用户级常驻实例”模型：

- 同一用户最多有一个 sandbox 实例
- 第一次调用 sandbox 能力时懒创建
- 后续请求复用该实例
- 空闲超时后由后台回收
- 回收后下次再重新创建空实例

本次明确接受“实例被回收后状态丢失”。

### 4.3 开发阶段约束

由于当前系统仍处于初期开发阶段，本次设计刻意收缩：

- 只支持单机开发态
- 只实现 Docker CLI provider
- 并发创建只使用进程内 `singleflight`
- 后台回收只使用后端进程内 ticker

这保证第一版先把正确的资源模型跑通，而不是提前引入上线复杂度。

## 5. 模块设计

### 5.1 SandboxManager

`SandboxManager` 负责生命周期控制，是唯一对上层暴露的入口。

职责如下：

- `GetOrCreate(ctx, userID)`
- `Touch(ctx, userID)`
- `Destroy(ctx, userID)`
- `RecycleIdle(ctx)`

它负责：

- 查询实例记录
- 决定是否复用或创建
- 调用 provider 创建和销毁
- 处理过期和状态切换
- 对接单机进程内并发创建合并

### 5.2 SandboxProvider

`SandboxProvider` 负责真正的资源创建和销毁。

本次先保留抽象接口，但只实现 `DockerSandboxProvider`。

建议接口：

```go
type SandboxProvider interface {
    Name() string
    Create(ctx context.Context, req CreateSandboxInstanceRequest) (*CreateSandboxInstanceResult, error)
    Destroy(ctx context.Context, instance SandboxInstance) error
    HealthCheck(ctx context.Context, instance SandboxInstance) error
    FindExisting(ctx context.Context, userID string) (*CreateSandboxInstanceResult, error)
}
```

其中：

- `Create` 创建新容器并返回 `instance_id` 与 `endpoint`
- `Destroy` 销毁容器
- `HealthCheck` 校验实例可用
- `FindExisting` 允许在后端重启后认领已存在容器

### 5.3 SandboxInstanceStore

`SandboxInstanceStore` 负责实例元数据的持久化，包括：

- 按用户查询当前实例
- Upsert `provisioning`
- 更新为 `ready`
- 更新为 `failed`
- 将 `ready` 更新为 `recycling`
- 删除实例记录
- 扫描过期实例

### 5.4 SandboxClient

`SandboxClient` 负责实际调用 sandbox endpoint，屏蔽协议细节。

本次至少覆盖：

- `POST /v1/code/execute`
- `POST /v1/file/read`
- MCP `/mcp` 下的 bash 执行调用

上层业务不再自己拼装全局 base URL。

## 6. 数据模型设计

建议新增 `sandbox_instances` 表，字段如下：

- `id`
- `user_id`
- `provider`
- `status`
- `instance_id`
- `endpoint`
- `last_active_at`
- `lease_expires_at`
- `last_error`
- `created_at`
- `updated_at`

字段说明：

- `user_id`：用户业务 ID，第一期设置唯一索引
- `provider`：当前写入 `docker`，未来可扩展 `scf`
- `status`：实例状态
- `instance_id`：provider 内部实例标识；Docker 场景下为 `container_id`
- `endpoint`：实例对外访问地址，例如 `http://127.0.0.1:32768`
- `last_active_at`：最近一次成功使用时间
- `lease_expires_at`：当前租约过期时间，用于后台回收
- `last_error`：最近一次创建或销毁失败摘要

建议索引：

- `UNIQUE(user_id)`
- `INDEX(status, lease_expires_at)`

## 7. 状态机设计

实例状态只保留四种：

- `provisioning`
- `ready`
- `failed`
- `recycling`

状态含义如下：

- `provisioning`：当前正在创建实例
- `ready`：实例可用
- `failed`：最近一次创建或销毁失败，允许下次重新创建
- `recycling`：后台正在回收，不允许普通续期逻辑抢回

### 7.1 GetOrCreate 主流程

`GetOrCreate(ctx, userID)` 的流程如下：

1. 校验 `userID` 非空
2. 查询实例表中的当前记录
3. 若命中 `ready`，进一步校验：
   - `lease_expires_at` 是否未过期
   - 对应 Docker 容器是否还存在
   - 健康检查是否通过
4. 若以上都成立，直接返回并更新续期时间
5. 若记录不存在、已过期、容器丢失、健康检查失败、状态为 `failed`，进入创建流程
6. 创建流程使用进程内 `singleflight` 按 `user_id` 合并并发请求
7. 创建者先写入或覆盖为 `provisioning`
8. 调 provider 创建容器
9. 创建成功后更新为 `ready`
10. 创建失败后更新为 `failed` 与 `last_error`

### 7.2 后端重启后的自修复

数据库记录不是唯一真相，Docker 容器状态才是最终真相。

因此 `GetOrCreate` 需要支持自修复：

- 若 DB 有 `ready` 记录，但容器已不存在，则视为失效，走重建
- 若 DB 无记录，但 Docker 中存在该用户容器，则认领该容器并补写记录
- 若 DB 与 Docker 都不存在，则创建新容器

这样即使后端重启、DB 被清空、容器被手工删除，也能自动恢复正确状态。

### 7.3 Touch

`Touch(ctx, userID)` 仅对 `ready` 状态生效：

- 更新 `last_active_at`
- 更新 `lease_expires_at = now + idle_ttl`

它不应对 `provisioning`、`failed`、`recycling` 生效。

## 8. Docker CLI Provider 设计

### 8.1 基本策略

本次仅使用 `docker` CLI，不引入 Docker SDK。

原因：

- 当前仅面向本地开发期
- CLI 更容易调试和排障
- 无需额外维护 Docker API 连接逻辑

### 8.2 容器命名与标识

建议为每个用户生成稳定容器名：

- `openintern-sandbox-<stable_short_hash(user_id)>`

同时写入 Docker label：

- `openintern.managed=true`
- `openintern.user_id=<user_id>`

其中：

- `container_id` 作为实例主标识写入实例表
- 容器名和 label 用于后端重启后的认领和排障

### 8.3 创建方式

建议创建命令形态如下：

```bash
docker run -d \
  --security-opt seccomp=unconfined \
  --name <container_name> \
  --label openintern.managed=true \
  --label openintern.user_id=<user_id> \
  -p 0:8080 \
  <image>
```

设计要点：

- 容器镜像来自配置
- 宿主机端口使用 `-p 0:8080` 由 Docker 自动分配
- 创建后通过 `docker inspect` 解析映射端口
- endpoint 统一写成 `http://127.0.0.1:<host_port>`

不自行维护端口池，避免第一期引入额外状态管理。

### 8.4 接管已有容器

`FindExisting` 需要支持：

- 通过 label 查询已有容器
- 通过容器名兜底查询
- 返回 `container_id` 与解析后的 endpoint

后端重启后，如果容器还在，应直接复用，而不是清空重建。

### 8.5 销毁方式

销毁优先级如下：

1. 优先按 `instance_id` 执行 `docker rm -f`
2. 若 `instance_id` 不可用，则按稳定容器名兜底删除

这样可降低 DB 记录不一致时的回收失败率。

## 9. 健康检查设计

健康检查的目标不是做完整能力验证，而是确认实例已经可以接收请求。

第一期建议策略：

- 优先使用最轻量的 HTTP 探活方式
- 如果 AIO sandbox 没有明确 health endpoint，则使用一个最小可接受的接口探测
- 不在健康检查阶段执行复杂 MCP 初始化或长耗时命令

健康检查成功后，才能将状态推进为 `ready`。

## 10. 自动回收设计

### 10.1 回收 worker

后台回收使用后端进程内 ticker 周期触发。

建议配置项：

- `idle_ttl_seconds`
- `recycle_interval_seconds`

worker 流程：

1. 周期扫描 `status = ready AND lease_expires_at < now`
2. 对候选记录尝试推进为 `recycling`
3. 条件更新时再次校验：
   - 当前状态仍然是 `ready`
   - `lease_expires_at` 仍然过期
4. 调 provider 销毁容器
5. 成功则删除实例记录
6. 失败则写回 `failed` 与 `last_error`

### 10.2 回收与前台请求的关系

本次不做“回收中中断取消”。

也就是说：

- 一旦记录进入 `recycling`
- 普通续期逻辑不再尝试把它抢回 `ready`
- 若此时用户又发起新请求，则等回收结束后重新创建

这会牺牲少量边界时延，但能显著简化第一期控制面逻辑。

## 11. 配置改造设计

旧配置中的：

```yaml
tools:
  sandbox:
    url: http://127.0.0.1:8081
```

需要直接废弃。

新配置改为控制面配置，例如：

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
      host: 127.0.0.1
      network: ""
```

说明：

- `provider` 第一版只允许 `docker`
- 如果配置成 `scf`，直接返回明确的 `not implemented`
- 配置表达的是“如何调度实例”，而不是“连接哪个固定实例”

## 12. 代码改造设计

### 12.1 删除全局 sandboxBaseURL

以下模型需要删除：

- 启动期强依赖 `tools.sandbox.url`
- package 级 `sandboxBaseURL`
- 上下文中的 `ContextKeySandboxBaseURL`

新的运行时依赖应改为：

- `user_id`
- `SandboxManager`

### 12.2 改造 Agent 层 sandbox 工具接入

当前 `GetSandboxMCPTools(ctx, baseURL)` 是“在 runner 构建时连接远端 MCP，再把远端工具对象注入运行时”。

这不适合用户级动态实例，因为 runner 构建时不应要求实例已经存在。

因此需要重构为：

- 在本地定义固定的 `sandbox_execute_bash` 内建工具
- `Info()` 返回本地静态 schema
- `Run()` 时：
  - 从上下文拿 `user_id`
  - 调用 `SandboxManager.GetOrCreate`
  - 拿到 endpoint
  - 再通过该 endpoint 调对应实例

也就是说，工具定义本地化，实例解析延迟到执行时。

### 12.3 改造 Code 执行链路

以下能力需要统一改成运行时解析用户实例：

- Code 插件执行
- Code 调试接口

新的执行路径：

1. 从上下文拿 `user_id`
2. `GetOrCreate`
3. 获得 endpoint
4. `POST /v1/code/execute`

### 12.4 改造 COS 文件读取链路

`upload_to_cos` 调用前也要先解析用户实例：

1. 从上下文拿 `user_id`
2. `GetOrCreate`
3. 调该实例的 `POST /v1/file/read`
4. 读取文件后上传 COS

### 12.5 初始化逻辑改造

系统启动时不再检查“是否存在固定 sandbox URL”，而改为检查：

- sandbox 是否启用
- provider 是否支持
- Docker 相关配置是否完整

系统启动阶段验证的是“能否创建实例”，不是“能否连接某个现成实例”。

## 13. 建议的实施顺序

建议分五步落地，降低排障难度：

1. 先新增 sandbox lifecycle 模块、实例表、DAO，但暂不接业务调用
2. 实现 `DockerSandboxProvider` 与 `SandboxManager.GetOrCreate/RecycleIdle`
3. 删除全局 `sandboxBaseURL` 与旧配置校验
4. 优先改造 Code 执行和 COS 读文件链路
5. 最后重做 `sandbox_execute_bash`，从远端 MCP 工具注入改为本地代理工具

采用这个顺序的原因是：

- 先把资源编排独立打通
- 再替换最直接的 HTTP 依赖
- 最后处理 Agent 运行时工具装配

这样遇到问题时更容易判断是“容器生命周期问题”还是“工具运行时问题”。

## 14. 风险与取舍

### 14.1 当前明确接受的取舍

- 不兼容旧的固定 URL 接入方式
- 不保留 sandbox 作为插件系统的一部分
- 不实现上线能力
- 不做实例内并发治理
- 不做状态恢复

### 14.2 当前主要风险

- Docker CLI 返回信息解析不稳定，需要统一封装命令调用和错误格式
- 健康检查接口如果选得过重，会拉长首次创建时延
- 后端进程内回收 worker 仅适合单机开发态，后续上线前必须重做

这些风险在当前阶段是可接受的，因为本次目标是先完成正确的资源模型切换。

## 15. 验收标准

本次设计完成后，至少应满足以下行为：

- 用户 A 第一次调用 sandbox 相关能力时自动创建 Docker 容器
- 用户 A 后续调用复用同一容器
- 用户 B 调用时创建另一个独立容器
- 用户空闲超时后，后台自动删除对应容器与实例记录
- 后端重启后，如果用户容器仍在，后续请求能够自动复用
- 若 DB 记录存在但容器已丢失，系统能够自动重建
- Code 执行、读 sandbox 文件、bash 执行均不再依赖全局固定 sandbox URL

## 16. 后续扩展点

以下内容明确留到后续版本：

- SCF provider 实现
- 多进程 / 多实例环境下的分布式锁
- 实例内工作目录隔离
- Browser / Shell / Jupyter 的细粒度并发治理
- 更完善的健康检查与观测
- 用户实例状态的管理接口或调试页面
