# Tool Search 模式开发方案（调研版）

> 状态：仅方案设计，不包含本次代码实现。
> 日期：2026-03-08

## 1. 目标与范围

目标：将聊天中的 `plugins.mode=search` 从“占位模式”升级为“按用户问题自动检索并装配工具”的真实运行模式。

范围：

1. 仅覆盖 `conversation.mode=chat`。
2. 仅处理插件工具（`api`/`mcp`/`code`），不影响内建工具（A2UI/COS/Skill）。
3. Tool Search 的存储和检索主链路统一走 OpenViking。
4. 本文只给出可落地实施方案，不做代码开发。

## 2. 现状调研结论

### 2.1 前端现状

1. 聊天页已有模式切换：`pluginMode: "select" | "search"`。
2. 发送请求时已透传：

```json
{
  "agentConfig": {
    "plugins": {
      "mode": "select|search",
      "selectedToolIds": []
    }
  }
}
```

3. 当 `pluginMode=search` 时，前端会发送空的 `selectedToolIds`。

关键位置：

- `openIntern/openIntern_forentend/app/(workspace)/chat/ChatModeConfigureArea.tsx`
- `openIntern/openIntern_forentend/app/(workspace)/chat/page.semi.tsx`

### 2.2 后端现状

1. 后端已经解析 `agentConfig.plugins.mode` 与 `selectedToolIds`。
2. 运行时工具装配逻辑中，`mode=search` 会直接跳过插件工具装载。

当前逻辑等价于：

- `select`：按 `selectedToolIds` 装配插件工具。
- `search`：只保留内建工具，不装配插件工具。

关键位置：

- `openIntern/openIntern_backend/internal/services/agent/agent_forwarded_props.go`
- `openIntern/openIntern_backend/internal/services/agent/agent_runner.go`
- `openIntern/openIntern_backend/internal/services/plugin/*.go`
- `openIntern/openIntern_backend/internal/dao/plugin.go`

## 3. 设计原则

1. 请求级生效：工具搜索结果只影响当前请求，不改全局状态。
2. 可回退：搜索失败时不报错中断，对话可继续（仅内建工具）。
3. 可观测：记录搜索 query、命中工具、分数与截断原因。
4. 双存储职责明确：MySQL 是配置真源，OpenViking 是检索真源。
5. 一致性可恢复：允许短暂最终一致，但必须提供同步状态与回补机制。

## 4. 目标链路（Search 模式）

1. 前端发送 `plugins.mode=search`。
2. 后端读取本轮用户问题文本（最后一条 user message）。
3. 在 OpenViking 中按 `target_uri` 执行工具检索（`find`）。
4. 将命中 URI 解析为 `tool_id`，回查 MySQL 做启用态与运行时过滤。
5. 对候选结果做截断和类型限流（TopK、maxMCPTools）。
6. 用命中的 `toolIds` 复用现有 `BuildRuntimeCodeTools/BuildRuntimeAPITools/BuildRuntimeMCPTools` 装配。
7. 与内建工具合并后交给 runner。

## 5. 详细方案

### 5.1 协议设计（向前兼容）

保持现有字段可用，并扩展可选 `search` 配置：

```json
{
  "agentConfig": {
    "plugins": {
      "mode": "search",
      "selectedToolIds": [],
      "search": {
        "topK": 8,
        "runtimeTypes": ["api", "mcp", "code"],
        "minScore": 0,
        "maxMCPTools": 3
      }
    }
  }
}
```

约束：

1. `select` 模式忽略 `search`。
2. `search` 模式忽略 `selectedToolIds`（避免双来源冲突）。
3. 未传 `search` 时使用后端默认值。

### 5.2 查询文本提取

搜索 query 来源：

1. 仅使用本轮最后一条 user message 的文本内容。
2. 若该文本为空：不做插件搜索，回退为仅内建工具。

建议新增内部 helper（agent 包内）：

- `ExtractSearchQueryFromRunInput(input *types.RunAgentInput) string`

### 5.3 数据访问与候选召回

采用 OpenViking 作为 Tool Search 的检索入口。

索引组织建议：

1. 每个工具在 OpenViking 对应一个资源文档（tool profile）。
2. URI 规范建议：`viking://resources/openintern/tools/{tenant_or_user}/{tool_id}.md`。
3. 文档内容建议包含：
   - `tool_id`、`tool_name`
   - `plugin_id`、`plugin_name`
   - `runtime_type`
   - `description`
   - 关键入参字段（从 `input_schema_json` 提取）
   - 业务标签（可选）

检索请求建议：

1. 统一使用 `POST /api/v1/search/find` 做召回。
2. 不使用 `search` 方法，不传 `session_id`。
3. 请求带 `target_uri`，限定到工具索引根目录，避免召回无关资源。

### 5.4 MySQL 与 OpenViking 同步机制（重点）

职责分层：

1. MySQL：插件/工具配置真源（启用态、运行时类型、执行配置）。
2. OpenViking：工具检索索引真源（用于 find 召回）。

同步策略（建议事件驱动 + 回补）：

1. 插件/工具创建或更新：发布 `UPSERT_TOOL_DOC` 同步事件。
2. 工具禁用/删除、插件禁用/删除：发布 `DELETE_TOOL_DOC` 同步事件。
3. 同步事件与 MySQL 变更在同一事务提交（outbox 模式），避免丢事件。
4. 异步 worker 消费 outbox，调用 OpenViking：
   - upsert：生成 tool profile 文件并 `POST /api/v1/resources`
   - delete：`DELETE /api/v1/fs?uri=...`
5. 记录同步状态：`pending/success/failed`、`retry_count`、`last_error`、`synced_at`。
6. 增加定时 reconciliation：按天全量校验 MySQL 与 OpenViking 差异并自动回补。

查询时一致性保护：

1. OpenViking 返回命中后，必须回查 MySQL（`tool_id IN (...)`）。
2. 仅保留 `plugin.status=enabled AND tool.enabled=true` 的工具。
3. 对已失效或不同步记录直接丢弃，防止脏召回进入 runner。

### 5.5 排序与截断策略

优先使用 OpenViking 返回分数，并做轻量业务重排：

1. 基础排序：`ov_score DESC`。
2. 业务校正：对 `runtime_type` 做权重或限流（尤其 `mcp`）。
3. 结果截断：`TopK` + `maxMCPTools`。
4. 命中数量为 0 时，不装载插件工具。

### 5.6 运行时装配改造

改造 `resolveRuntimeTools` 分支：

1. `mode=select`：保持现状（按 `selectedToolIds`）。
2. `mode=search`：
   - 通过 OpenViking 检索 + MySQL 二次过滤得到 `toolIds`；
   - 复用现有三类工具构建方法进行装配；
   - 未命中则不装插件工具。

注意：

1. `runtimeConfig=nil` 时维持当前行为（仅内建工具）。
2. 不修改 `state.agentTools` 全局切片。

### 5.7 可观测性

建议新增日志字段：

1. `thread_id`, `run_id`
2. `plugin_mode`
3. `search_query`
4. `openviking_target_uri`
5. `matched_tool_ids`
6. `candidate_count`, `top_k`, `mcp_truncated`
7. `sync_version`（可选）
8. `sync_lag_ms`（可选）

可选：向前端发送一条 activity，展示“本轮自动选择了哪些工具”。

### 5.8 前端改造建议

默认可不改 UI，仅保持现有 `Search` 开关。

可选增强：

1. 在搜索模式显示“自动检索工具中”。
2. 在回答顶部展示“本轮命中工具标签”。
3. 提供高级参数（TopK/类型过滤）并写入 `plugins.search`。

## 6. 代码改造清单（建议）

后端：

1. `internal/services/agent/agent_forwarded_props.go`
2. `internal/services/agent/agent_runner.go`
3. `internal/services/agent/*_test.go`（补充分支测试）
4. `internal/services/plugin/`（新增 openviking 检索与重排 service）
5. `internal/dao/plugin.go`（新增按 tool_id 批量过滤查询）
6. `internal/dao/context_store.go`（复用/扩展 OpenViking 调用封装）
7. `internal/database/context_store.go`（必要时补充请求封装）
8. `internal/services/plugin/*sync*.go`（新增 outbox worker 与重试）
9. `internal/controllers/plugin.go`（可选：新增调试查询与重建索引 API）
10. `internal/routers/router.go`（可选：注册调试 API）

前端（可选）：

1. `openIntern_forentend/app/(workspace)/chat/page.semi.tsx`
2. `openIntern_forentend/app/(workspace)/chat/ChatModeConfigureArea.tsx`

## 7. 测试与验收

### 7.1 单元测试

1. `plugins.search` 配置解析：默认值、边界值、非法值。
2. 查询提取：仅最后一条 user message 文本、空文本。
3. OpenViking 返回解析与 URI -> `tool_id` 映射。
4. `resolveRuntimeTools`：`select/search/empty` 三分支。
5. MySQL 二次过滤（禁用工具、禁用插件、脏数据）。
6. outbox 同步状态机（成功、失败、重试、幂等）。

### 7.2 集成测试

1. `search` 模式请求能命中插件工具并实际调用。
2. 无命中时系统仍能完成回复（仅内建工具）。
3. `mcp` 上限生效，避免连接过多。
4. OpenViking 不可用时系统可降级（不装载插件工具但不阻断主对话）。
5. MySQL 与 OpenViking 人工制造不一致时可被查询时过滤与回补任务修复。

### 7.3 DoD

1. `search` 模式下，插件工具可被自动装配。
2. 工具召回由 OpenViking 完成，且查询时通过 MySQL 二次校验。
3. 工具命中与同步状态可在日志中追踪。
4. 不影响 `select` 模式现有行为。
5. 对话链路无新增阻断性错误。

## 8. 实施建议

1. 后端接入 OpenViking `find` 检索 + TopK 装配。
2. 打通 MySQL -> OpenViking 的基础同步链路（outbox + worker）。
3. 前端保持当前 UI，不新增复杂交互。
4. 补齐核心单测与最小集成测试。
5. 优化 OpenViking `find` 召回质量（query 扩展、阈值与重排策略）。
6. 增加前端“命中工具可视化”。
7. 增加配置项（按模型/租户/场景配置 TopK 与类型权重）。
8. 完善索引回补与审计任务（全量巡检、报警）。

## 9. 风险与对策

1. 索引不同步导致误召回：查询时强制 MySQL 二次过滤 + 定时回补。
2. OpenViking 暂时不可用：降级为不装插件工具，保证主对话可用。
3. MCP 连接开销：增加 `maxMCPTools` 限制与超时控制。
4. 工具过多导致上下文膨胀：强制 `TopK` 上限。
5. 模式行为混淆：明确 `search` 忽略 `selectedToolIds`，并在日志打印最终生效工具。

---

本方案可直接作为 `tool search` 模式开发基线。确认后可进入实现。
