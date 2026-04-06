# 用户级运行时配置覆盖层设计

## 1. 背景与目标

当前设置页的配置入口绑定的是后端全局运行时配置：

- `GET /v1/config` 直接返回进程级 `globalRuntime`
- `PUT /v1/config` 直接修改 `globalRuntime/globalConfig`
- 更新后会写回 `openIntern_backend/config.yaml`

这导致当前登录用户在设置页保存 `Agent 设置` 或 `高级设置` 时，会直接影响所有用户。

本次目标是：

- 将 `Agent 设置` 与 `高级设置` 改为按用户隔离
- `系统配置` 仍然保持全局配置
- 保持前端设置页入口不变，继续通过 `/v1/config` 读写
- 为未来新增更多“用户自己的配置”预留扩展能力

## 2. 范围与非目标

### 2.1 本次范围

- 设置页中的 `agent`
- 设置页中的 `context_compression`
- Agent 运行时对这两类配置的读取路径
- 后端新增用户配置覆盖层的数据模型、DAO、service 与接口合并逻辑

### 2.2 非目标

- 不将 `summary_llm`、`minio`、`apmplus` 改为用户级
- 不将 `tools`、`plugin` 改为用户级
- 不移除 `config.yaml`；它仍然是全局默认值来源
- 不引入整套“每用户完整 RuntimeConfig 缓存”
- 不修改前端设置页的信息架构

## 3. 当前问题

当前 [config.go](/Users/fqc/project/agent/openIntern/openIntern_backend/internal/controllers/config.go) 中：

- `GetConfig` 不读取 `user_id`
- `UpdateConfig` 不读取 `user_id`
- `UpdateConfig` 直接调用全局 `config.UpdateRuntimeConfig(updates)`

当前 [runtime_config.go](/Users/fqc/project/agent/openIntern/openIntern_backend/internal/config/runtime_config.go) 中：

- `UpdateRuntimeConfig` 会修改进程级 `globalRuntime/globalConfig`
- 同时写回 `config.yaml`
- 因此任何一个用户的配置保存都会成为全局配置

## 4. 总体方案

本次采用“全局默认值 + 用户覆盖值”的双层模型：

- `config.yaml` 与进程级运行时仍负责全局默认值
- 数据库新增用户运行时配置覆盖层
- 请求 `/v1/config` 时，按当前登录用户返回合并后的有效配置
- Agent 实际执行时，按当前 `user_id` 解析有效配置，而不是只依赖全局运行态

### 4.1 设计原则

- 全局配置和用户配置职责分离
- 仅允许白名单配置项进入用户覆盖层
- 每个配置块独立存储，避免一个大 JSON 导致并发覆盖
- 增加新用户配置时尽量不改表结构

## 5. 数据模型设计

建议新增 `user_runtime_config` 表。

字段：

- `id`
- `user_id`
- `config_key`
- `config_value`
- `created_at`
- `updated_at`
- `deleted_at`

字段约束：

- `user_id`：openIntern 业务 `user_id`
- `config_key`：配置块名称，例如 `agent`、`context_compression`
- `config_value`：JSON 文本，存储该配置块的结构化内容

索引设计：

- `UNIQUE(user_id, config_key)`
- `INDEX(user_id)`

### 5.1 为什么不做“一用户一整份完整配置 JSON”

不采用单条大 JSON 的原因：

- 不同设置块保存时容易互相覆盖
- 并发更新难以保证局部修改安全
- 后续排查与回滚粒度过粗
- 后面增加用户配置时，局部演进不够清晰

### 5.2 为什么不做“每种配置一张表”

不采用多张表的原因：

- 后续每增加一种用户配置都要改表结构
- 用户配置项可能持续扩张，表会快速碎片化
- 这次需求只需要“用户级覆盖层”，不需要过早做强 schema 拆分

## 6. 可覆盖配置白名单

本次只允许以下 `config_key` 进入用户覆盖层：

- `agent`
- `context_compression`

对应规则：

- `agent`：允许覆盖 `max_iterations`
- `context_compression`：允许覆盖 `enabled`、`soft_limit_tokens`、`hard_limit_tokens`、`output_reserve_tokens`、`max_recent_messages`、`estimated_chars_per_token`

其余配置块：

- `tools`
- `plugin`
- `summary_llm`
- `minio`
- `apmplus`

继续只走全局配置，不允许写入用户配置表。

## 7. 读接口设计

`GET /v1/config` 的返回结构保持不变，但数据来源改为：

1. 读取全局默认配置
2. 读取当前 `user_id` 的用户配置覆盖值
3. 仅对白名单配置块执行覆盖合并
4. 返回最终有效配置

合并规则：

- `agent`：用用户配置完整覆盖默认 `agent` 块中的可覆盖字段
- `context_compression`：用用户配置完整覆盖默认 `context_compression` 块中的可覆盖字段
- `system` 相关块继续直接返回全局配置

如果某用户没有保存过配置：

- 返回纯全局默认值

## 8. 写接口设计

`PUT /v1/config` 保持接口路径不变，但行为调整如下：

- 读取当前登录用户 `user_id`
- 校验请求体里是否只包含允许用户覆盖的配置块
- 对每个配置块做结构校验与字段白名单过滤
- 将校验后的结果 Upsert 到 `user_runtime_config`

关键变化：

- `agent` 与 `context_compression` 的更新不再写回 `config.yaml`
- 不再调用全局 `config.UpdateRuntimeConfig` 处理这两类配置
- 若请求中包含全局块，例如 `summary_llm`，仍继续走现有全局更新逻辑

### 8.1 混合更新规则

为兼容当前前端入口，后端允许一次请求只更新一个 section。

当前页面行为本身就是：

- `AgentSettings` 单独提交 `agent`
- `AdvancedSettings` 单独提交 `context_compression`
- `SystemSettings` 单独提交系统级配置

因此后端无需支持复杂的“多 section 混合事务提交”。

## 9. 运行时读取设计

### 9.1 Agent 运行时

当前 agent 初始化时，会将以下配置固化进进程级 `runtimeState`：

- `contextCompression`
- `maxIterations`
- `summaryModel`

这里的 `summaryModel` 仍然保持全局；但 `contextCompression` 与 `maxIterations` 需要支持按用户生效。

因此建议改为：

- 启动期仍初始化全局依赖，例如 `summaryModel`、全局工具、中间件
- 在真正执行 `RunAgent` 时，根据当前 `user_id` 动态解析用户有效配置
- 运行时状态里不再把 `maxIterations` 和 `contextCompression` 当作全局固定值使用

### 9.2 推荐实现方式

新增一层用户配置解析 service，例如：

```go
type RuntimeConfigResolver interface {
    ResolveUserConfig(userID string) (*ResolvedUserRuntimeConfig, error)
}
```

解析结果至少包含：

- `Agent config.AgentConfig`
- `ContextCompression config.ContextCompressionConfig`

`ResolveUserConfig` 的流程：

1. 读取全局默认值
2. 加载当前用户的覆盖项
3. 做白名单字段合并
4. 返回当前用户的有效配置

Agent 运行入口只依赖这份解析结果，不直接依赖全局 `maxIterations/contextCompression`。

## 10. 配置校验设计

虽然 `config_value` 存在 JSON 字段中，但不能直接透传。

每个 `config_key` 都必须经过显式校验：

- `agent.max_iterations` 必须是正整数
- `context_compression.soft_limit_tokens`、`hard_limit_tokens`、`output_reserve_tokens`、`max_recent_messages`、`estimated_chars_per_token` 必须是有效整数
- `soft_limit_tokens` 与 `hard_limit_tokens` 的关系要保持合法

后端必须拒绝：

- 未知 `config_key`
- 未知字段
- 非法类型
- 明显矛盾的数值组合

这样可以保证虽然底层用 JSON 存储，但行为仍然是强约束的。

## 11. 文件改动范围

### 11.1 数据层

- 新增用户配置 model
- 新增用户配置 DAO
- 在数据库初始化里加入 `AutoMigrate`

### 11.2 配置层

- 新增“读取用户有效配置”的解析逻辑
- 保留现有全局配置能力，用于系统级配置和默认值

### 11.3 接口层

- 修改 [config.go](/Users/fqc/project/agent/openIntern/openIntern_backend/internal/controllers/config.go)
- `GET /v1/config` 改为按 `user_id` 返回合并结果
- `PUT /v1/config` 改为区分“用户级 section”与“全局 section”

### 11.4 Agent 层

- 调整 agent 运行时读取 `maxIterations` 和 `contextCompression` 的方式
- 保持 `summary_llm` 等系统级依赖仍在启动期初始化

## 12. 扩展策略

未来如果继续增加用户自己的配置，不新增表结构，按以下方式扩展：

1. 增加新的 `config_key`
2. 定义该 key 对应的数据结构
3. 增加校验器
4. 增加与全局默认值的合并逻辑
5. 在 `/v1/config` 响应中决定是否暴露

这样可以避免每次新需求都去重新设计一套用户配置存储模型。

## 13. 非目标与约束

- 不执行 `go test`
- 不新增与本次任务无关的重构
- 不在前端页面中写入“为什么这样做”的实现说明
- 不把当前系统级配置误改成用户级配置

## 14. 验证方式

- 用户 A 保存 `agent.max_iterations` 后，再次读取 `/v1/config` 能看到 A 自己的值
- 用户 B 登录读取 `/v1/config` 时，`agent` 与 `context_compression` 仍是自己的值或全局默认值，不会看到 A 的覆盖值
- 用户 A 与 B 分别运行 agent 时，最大迭代次数与上下文压缩参数按各自配置生效
- `system` 相关配置更新后仍对所有用户一致
- 只执行与本次改动相关的轻量验证命令，不执行 `go test`
