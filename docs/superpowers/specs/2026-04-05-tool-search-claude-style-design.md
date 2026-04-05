# Tool Search 改为 Claude Code 风格并移除 OpenViking 工具索引设计

## 1. 背景与目标

当前 openIntern 的 `tool_search` 依赖 OpenViking 工具索引：

- 插件和工具变更后会写入 OpenViking `resources/tools`
- `tool_search` 调用时通过 OpenViking `/api/v1/search/find` 召回候选工具
- 再回 MySQL 过滤当前用户启用态工具

这套实现的问题是：

- 工具检索依赖额外外部索引服务，链路长，调试成本高
- 插件写入、删除、MCP 同步都要额外维护一条 OpenViking 对账流水线
- `tool_search` 的查询协议与 Claude Code 的使用方式不一致，模型迁移成本高

本次改造目标如下：

- 将 `tool_search` 的输入协议改为尽量贴合 Claude Code
- 将工具检索改为本地关键词检索，不再依赖 OpenViking
- 删除“工具同步到 OpenViking”的整条索引链路
- 保留 OpenViking 在 memory 与 skills 相关能力中的现有用途，不扩散影响范围

## 2. 当前现状

### 2.1 当前 `tool_search` 运行方式

当前运行时行为位于：

- `openIntern_backend/internal/services/middlewares/toolsearch/tool.go`
- `openIntern_backend/internal/services/plugin/plugin_search.go`
- `openIntern_backend/internal/dao/plugin_search.go`

模型可见的 `tool_search` 入参为：

- `intent`
- `keywords`
- `top_k`

执行流程为：

1. 将 `intent` 和 `keywords` 拼成查询文本
2. 调用 OpenViking `/api/v1/search/find`
3. 将命中的 URI 解析为 `tool_id`
4. 回 MySQL 过滤用户下启用态工具
5. 返回 `selected_tool_names`

### 2.2 当前 OpenViking 工具索引链路

工具索引相关代码主要位于：

- `openIntern_backend/internal/services/plugin/plugin_openviking_sync.go`
- `openIntern_backend/internal/dao/plugin_store.go`
- `openIntern_backend/internal/dao/plugin_search.go`

当前插件生命周期和 MCP 同步会触发以下行为：

- 创建插件时入队 OpenViking reconcile
- 更新插件时入队 OpenViking reconcile
- 删除插件时入队 OpenViking cleanup
- MCP 同步后根据变更结果直接同步或入队同步

这条链路只服务于工具检索，不为实际工具执行提供能力。

## 3. 范围与非目标

### 3.1 本次范围

- 调整 `tool_search` 的输入协议
- 新增 Claude Code 风格的本地关键词检索实现
- 删除 OpenViking 工具索引写入、删除、队列和召回逻辑
- 更新相关配置与文档

### 3.2 非目标

- 不改动 memory 对 OpenViking 的使用方式
- 不改动 skills 相关 OpenViking 配置
- 不重构动态工具可见性中间件整体结构
- 不引入新的搜索引擎或全文检索中间件

## 4. 总体方案

### 4.1 `tool_search` 协议改为 Claude Code 风格

`tool_search` 的模型入参改为：

- `query`: 用于工具检索或直接选择工具
- `max_results`: 返回结果上限，默认值保持小范围

查询语义对齐 Claude Code：

- `select:tool_a,tool_b`：按工具名精确选择
- 普通关键词：按工具名、描述等字段做本地匹配
- `+required optional`：带 `+` 的词为必选词，其余词参与排序

不再保留：

- `intent`
- `keywords`
- `top_k`

### 4.2 `tool_search` 输出保持现有后端约束

为了最小化对运行时可见性中间件的改动，`tool_search` 的执行结果仍返回：

- `selected_tool_names`

也就是说：

- 输入协议贴近 Claude Code
- 输出仍复用 openIntern 当前的“按工具名解锁动态工具”的机制

这样可以避免重写 `toolsearch` middleware 的状态提取逻辑。

### 4.3 本地关键词检索替代 OpenViking 召回

新的检索链路直接基于 MySQL 当前启用态工具快照执行：

1. 读取当前用户全部启用态 runtime tool
2. 将每个工具构造成本地候选项
3. 对查询词进行规范化、拆词、必选词过滤
4. 参考 Claude Code 的规则对每个工具打分
5. 按得分排序后返回前 `max_results`

候选项字段至少包括：

- `tool_id`
- `tool_name`
- `description`
- `plugin_name`
- `plugin_description`
- `runtime_type`

## 5. 检索算法设计

### 5.1 工具名解析

参考 Claude Code 的 `parseToolName` 思路，对工具名建立可搜索片段：

- 下划线拆词
- CamelCase 拆词
- 全量小写归一化
- 保留完整归一化名称作为兜底匹配文本

openIntern 当前工具名主要为普通工具名，不强制引入 MCP 前缀特判；但实现层保留对多段下划线工具名的良好支持。

### 5.2 查询词解析

查询文本统一进行：

- trim
- lower-case
- 按空白切词
- 将 `+term` 识别为必选词

若查询是 `select:` 前缀：

- 按逗号切出工具名
- 先在候选工具中精确匹配
- 找不到则忽略，不报错

### 5.3 评分规则

评分规则借鉴 Claude Code，但结合 openIntern 当前数据做简化：

- 工具名精确片段命中：高权重
- 工具名片段包含命中：中高权重
- 工具完整归一化名包含命中：中权重
- 工具描述命中：中权重
- 插件名命中：中权重
- 插件描述命中：低中权重

必选词规则：

- 任何必选词未命中，则该工具直接过滤

排序规则：

- 先按总分降序
- 分数相同时按工具名升序，保证结果稳定

### 5.4 结果数量与类型控制

继续保留运行时类型过滤能力：

- `api`
- `mcp`
- `code`

同时保留现有 MCP 数量限制能力，避免单次结果被 MCP 工具完全占满。

## 6. 代码改造设计

### 6.1 `tool_search` 元工具

修改：

- `openIntern_backend/internal/services/middlewares/toolsearch/tool.go`

改造内容：

- 输入 schema 改为 `query/max_results`
- 描述文案改为 Claude Code 风格
- 支持 `select:` 和关键词搜索
- 调用新的本地搜索服务方法

### 6.2 搜索服务层

修改：

- `openIntern_backend/internal/services/plugin/plugin_search.go`

新增职责：

- 本地候选工具加载
- 查询解析
- 打分与排序
- 结果截断

删除职责：

- 依赖 OpenViking 召回候选 URI
- 解析 OpenViking 返回结果
- 处理 `target_uri`

### 6.3 DAO 层

删除：

- `openIntern_backend/internal/dao/plugin_search.go`

新增或调整：

- 在现有 `plugin` DAO 中补充“查询当前用户启用态 runtime tool 搜索视图”的方法
- 返回工具与插件联合后的最小搜索字段集合

### 6.4 OpenViking 工具索引链路

删除：

- `openIntern_backend/internal/services/plugin/plugin_openviking_sync.go`
- `openIntern_backend/internal/dao/plugin_store.go`

并同步移除以下调用点中的 OpenViking 逻辑：

- 插件创建
- 插件更新
- 插件删除
- MCP 插件同步

### 6.5 配置与初始化

修改：

- `openIntern_backend/internal/config/config.go`
- `openIntern_backend/main.go`

调整内容：

- 删除插件侧 `openviking_sync_*` 配置
- 删除 `tools.openviking.tools_root`
- 保留 memory/skills 仍需的 OpenViking 基础配置

如果 `ContextStore` 中 `ToolsRoot()` 已无调用方，则一并删除该字段和方法。

## 7. 文档与兼容性

需要同步更新：

- `README.md`
- 可能涉及工具检索说明的注释

兼容性说明：

- 这次是明确的不兼容变更
- 任何依赖旧 `intent/keywords/top_k` 协议的 agent 配置都需要更新
- OpenViking 不再承担工具检索职责

## 8. 风险与控制

### 8.1 结果质量回退风险

从向量召回切到关键词匹配后，召回质量可能下降。控制方式：

- 尽量贴近 Claude Code 的打分规则
- 将工具名命中置于最高权重
- 把插件名和插件描述作为补充信号而不是主信号

### 8.2 动态工具可见性回归风险

如果 `tool_search` 输出结构被改坏，动态工具不会被正确放开。控制方式：

- 保持 `selected_tool_names` 输出不变
- 不改 `middleware.go` 的状态提取接口

### 8.3 删除链路残留风险

如果 OpenViking 队列、配置、调用点删不干净，会留下死代码或启动噪音。控制方式：

- 通过全局搜索清理 `queueOpenViking`、`ToolStoreConfigured`、`ToolsRoot`
- 编译验证后再补 README

## 9. 验证方案

本次验证只使用允许的轻量命令，不运行 `go test`。

需要验证：

- 后端代码可以通过 `go build` 编译
- `tool_search` 相关包编译通过
- OpenViking 工具索引相关引用已全部移除
- README 与配置说明不再描述工具检索依赖 OpenViking
