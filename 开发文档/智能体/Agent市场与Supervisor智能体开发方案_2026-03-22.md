# Agent 市场与 Supervisor 智能体开发方案（调研版）

> 状态：仅方案设计，不包含本次代码实现。
> 日期：2026-03-22

## 1. 目标与范围

目标：在当前项目中补齐“Agent 配置实体 + Agent 市场 + Agent 调试测试 + Supervisor 组合编排”这一整条链路，并尽量复用现有聊天、模型、工具、Skill、知识库、长期记忆能力。

本次方案覆盖：

1. Agent 市场列表页。
2. Agent 创建、编辑、启用、停用、测试。
3. Agent 类型：
   - `single`
   - `supervisor`
4. Agent 可配置资源：
   - 系统提示词
   - 智能体名字
   - 智能体描述
   - 示例问法
   - 聊天背景
   - Agent 头像
   - 默认模型
   - 工具
   - Skill
   - 知识库
   - Agent 级长期记忆开关
5. `supervisor` 额外支持绑定已启用的 subagent。
6. 创建页、编辑页内嵌测试能力。

明确不在本次 v1 强行一次做完的内容：

1. 公开共享、多租户广场、点赞收藏等“真正 marketplace 社区能力”。
2. Agent 版本发布流、版本回滚、灰度发布。
3. 多级 supervisor 递归编排的复杂治理。
4. Agent 级长期记忆的精细化观测、回放、手动清理后台。

## 2. 现状调研结论

## 2.1 已有能力可以直接复用

后端现有运行链已经具备这些基础能力：

1. 模型目录与默认模型：
   - `openIntern_backend/internal/controllers/model_catalog.go`
   - `openIntern_backend/internal/services/model/catalog.go`
2. 插件工具运行时装配：
   - `openIntern_backend/internal/controllers/plugin.go`
   - `openIntern_backend/internal/services/plugin/*.go`
   - `openIntern_backend/internal/services/agent/agent_runner.go`
3. Skill 中间件与 Skill 存储：
   - `openIntern_backend/internal/services/agent/agent_init.go`
   - `openIntern_backend/internal/services/middlewares/skill/*.go`
   - `openIntern_backend/internal/controllers/skill.go`
4. 知识库管理与聊天期上下文注入：
   - `openIntern_backend/internal/controllers/kb.go`
   - `openIntern_backend/internal/services/kb/*.go`
   - `openIntern_backend/internal/services/agent/agent_forwarded_props.go`
5. 长期记忆读写链路：
   - `openIntern_backend/internal/services/agent/memory_retriever.go`
   - `openIntern_backend/internal/services/memory/*.go`
   - `开发文档/3.13链路梳理/长期记忆链路梳理.md`
6. 聊天 SSE 与 Runner：
   - `openIntern_backend/internal/controllers/chat.go`
   - `openIntern_backend/internal/services/agent/agent_entry.go`
   - `openIntern_backend/internal/services/agent/agent_runner.go`

从实现角度看，项目并不是“没有 agent 能力”，而是“已经有 agent runtime 的零散能力，但缺失 Agent 作为业务实体的配置中心与编排入口”。

## 2.2 当前缺口

当前代码还缺这些关键部分：

1. 没有 `agent` 业务表，无法持久化 Agent 配置。
2. 没有 Agent 资源绑定关系，工具/Skill/知识库/子 Agent 只能临时从前端透传。
3. 当前聊天页虽然有 `conversationMode = "agent"` 开关，但后端在 `RunAgent` 中直接返回：

```go
if runtimeConfig != nil && strings.EqualFold(runtimeConfig.Conversation.Mode, "agent") {
	err := fmt.Errorf("agent mode is not available yet")
	_ = sender.Error(err.Error(), "agent_mode_not_available")
	return err
}
```

4. `buildEinoRunner(...)` 目前只会构建一个普通 `ChatModelAgent`，没有根据已保存 Agent 定义编译 `single/supervisor`。
5. 当前长期记忆中的 agent memory 根是固定的 `viking://agent/default/memories/`，还不是“某个 Agent 自己的长期记忆空间”。
6. 现有聊天历史一定会落 `thread/message` 表，不适合创建/编辑页内嵌测试这种“临时调试会话”。
7. 当前 chat 模式下，Skill middleware 是全局挂载的，模型理论上可以看到整个 Skill 仓库；前端 mention 选择的 skill 只是在 `forwardedProps` 里额外插入一条“优先参考这些技能”的约束消息，并没有真正把 Skill 可见范围限制到所选集合。

结论：这不是简单加一个前端页面即可，必须同时补“数据模型、运行时编译器、测试入口、依赖校验”。

## 3. 设计原则

1. Agent 配置与聊天运行解耦：
   Agent 市场管理的是“配置实体”，聊天运行拿的是“编译后的运行时定义”。
2. 单 Agent 与 Supervisor 统一走同一套编译入口：
   避免 single 和 supervisor 各写一套执行逻辑。
3. 测试链路与正式会话隔离：
   创建/编辑页里的测试不能污染正式线程和正式长期记忆。
4. 资源引用必须显式校验：
   不接受“资源丢了就静默忽略”的兼容方式，避免把错误藏起来。
5. 前端页面要围绕“配置 + 测试”闭环设计：
   创建和编辑页不是纯表单，必须能在当前草稿上即时测试。
6. Supervisor v1 控制复杂度：
   先把一层编排做稳，再考虑递归 supervisor。

## 4. 推荐范围边界

## 4.1 Agent 状态

建议 Agent 状态使用三态：

1. `draft`
   - 新建默认状态
   - 可编辑、可测试
   - 不可作为 subagent 被绑定
2. `enabled`
   - 已启用
   - 会出现在 Agent 市场主列表
   - 可被 supervisor 绑定
   - 可用于正式运行
3. `disabled`
   - 历史 Agent 暂停使用
   - 可编辑、可测试
   - 不可被新的 supervisor 绑定

这样比只有“启用/未启用”更稳，因为用户通常需要先配置和测试，再启用。

## 4.2 Supervisor 的子 Agent 范围

按当前需求，`supervisor` 需要支持绑定任意类型的已启用 Agent：

1. `single`
2. `supervisor`

这意味着运行时必须支持递归编译，而不能再按“只支持一层 subagent”设计。

因此需要在方案里同时落这几个硬约束：

1. 必须做依赖环检测。
2. 必须设置最大编排深度，建议 v1 先限制为 `3`。
3. 必须限制单个 supervisor 的直接 subagent 数量，避免运行时膨胀。
4. 必须明确 parent supervisor 与 child supervisor 的上下文、工具、知识库、长期记忆边界。

结论：

1. 允许 supervisor 绑定任意类型 Agent。
2. 但实现上必须把 Agent 依赖图当成 DAG 处理。
3. 一旦检测到环或超深度，创建、更新、启用、调试都要直接报错。

## 4.3 聊天背景与示例问法的语义

这两个字段建议明确为“前端展示属性”，不参与模型 prompt：

1. `聊天背景`
   - 用于 Agent 对话页/测试页 UI 背景
   - 不下发给模型
2. `示例问法`
   - 用于市场卡片、Agent 详情页的快捷提问
   - 不下发给模型

否则只会浪费 token，并造成展示配置和行为配置耦合。

## 5. 核心实现思路

整体建议拆成三层：

1. Agent 定义层：
   保存 Agent 元数据与资源绑定关系。
2. Agent 编译层：
   把保存的 Agent 定义编译成运行时 `CompiledAgentDefinition`。
3. Agent 运行层：
   单 Agent 或 supervisor 最终都走统一 Runner 执行。

运行链路建议如下：

1. 前端进入 Agent 市场，创建或编辑 Agent 草稿。
2. 点击测试时，把“当前草稿”直接发给调试接口。
3. 后端校验草稿，编译为运行时 Agent。
4. `single`：
   - 构建一个 `ChatModelAgent`
5. `supervisor`：
   - 先把每个子 Agent 编译出来
   - 再用 `adk.NewAgentTool(...)` 包装成父 Agent 可调用工具
   - 父 Agent 仍然是 `ChatModelAgent`
6. 正式启用后，市场或详情页通过 Agent 正式聊天入口运行已启用 Agent。

这里和 Eino ADK 的契合点很明确：

1. `single` 直接对应 `ChatModelAgent`
2. `supervisor` 对应“主 Agent + AgentAsTool 子 Agent”
3. 正式执行继续使用 `Runner`

## 6. 数据模型设计

建议新增两张核心表，避免把各种资源绑死成多张关联表。

## 6.1 `agent`

建议字段：

```text
id
agent_id                string(36)   业务 ID
owner_id                string(36)   创建人
name                    string(120)
description             text
agent_type              string(20)   single | supervisor
status                  string(20)   draft | enabled | disabled
system_prompt           longtext
avatar_url              string(255)
chat_background_json    longtext
example_questions_json  longtext
default_model_id        string(36)
agent_memory_enabled    bool
sort                    int          预留市场排序
created_at
updated_at
deleted_at
```

说明：

1. `default_model_id` 直接关联现有 `model_catalog.model_id`。
2. `chat_background_json` 建议存 JSON，而不是只存字符串，方便后续支持：
   - 图片
   - 渐变
   - 纯色
3. `example_questions_json` 建议存字符串数组。
4. `owner_id` 必须补上，后续多用户隔离靠它。
5. `name` 建议在 `(owner_id, name, deleted_at)` 维度保持逻辑唯一，减少市场列表歧义。

## 6.2 `agent_binding`

建议用统一绑定表：

```text
id
agent_id                string(36)
binding_type            string(20)   tool | skill | kb | sub_agent
binding_target_id       string(255)  tool_id / skill_name / kb_name / agent_id
sort                    int
metadata_json           longtext
created_at
updated_at
```

唯一索引建议：

```text
unique(agent_id, binding_type, binding_target_id)
```

选择统一绑定表而不是多张表的原因：

1. `tool` 是 MySQL 表。
2. `skill` 元数据在 MySQL，但实际内容在 OpenViking。
3. `kb` 本体不在 MySQL，名字就是业务主键。
4. `sub_agent` 是新的 Agent 实体。

资源来源本来就不统一，统一绑定表更符合当前项目事实。

## 6.3 可选增强表

如果本轮只做市场 + 测试，可以先不加。

如果要同时支持“正式 Agent 对话页”，建议再补：

### `thread` 扩展字段

```text
thread_type             string(20)   chat | agent
agent_id                string(36)
```

这样线程列表才知道某个会话属于哪个 Agent。

## 7. 后端接口设计

## 7.1 Agent 市场管理接口

建议新增路由组：

```text
/v1/agents
```

接口列表：

1. `POST /v1/agents`
   - 创建 Agent
2. `GET /v1/agents`
   - Agent 市场列表
   - 支持 `keyword/type/status/page/page_size`
3. `GET /v1/agents/:id`
   - Agent 详情
4. `PUT /v1/agents/:id`
   - 更新 Agent
5. `POST /v1/agents/:id/enable`
   - 启用 Agent
6. `POST /v1/agents/:id/disable`
   - 停用 Agent
7. `GET /v1/agents/enabled-options`
   - 返回可绑定 subagent 候选
   - 返回所有已启用 Agent，并带上 `agent_type`

建议单独补一个资源上传接口：

8. `POST /v1/agents/assets/image`
   - 上传 Agent 头像或聊天背景图

这部分可以直接复用插件图标上传链路和 `services/storage/file.go`。

## 7.2 Agent 调试测试接口

建议不要复用现有 `/v1/chat/sse`，而是新增：

```text
POST /v1/agents/debug/sse
```

请求体建议：

```json
{
  "draft": {
    "agent_id": "optional-saved-id",
    "name": "客服分流 Agent",
    "agent_type": "supervisor",
    "description": "负责把任务分发给不同专家",
    "system_prompt": "你是一个分流型 supervisor...",
    "avatar_url": "",
    "chat_background_json": {},
    "example_questions": ["帮我分析这个需求"],
    "default_model_id": "xxx",
    "agent_memory_enabled": true,
    "tool_ids": ["tool_a", "tool_b"],
    "skill_names": ["spec-writer"],
    "knowledge_base_names": ["产品文档"],
    "sub_agent_ids": ["agent_1", "agent_2"]
  },
  "runtime_override": {
    "model_id": "optional-model-id"
  },
  "messages": [],
  "debug_session_id": "optional"
}
```

调试接口要求：

1. 支持“未保存草稿”直接测试。
2. 不写 `thread` / `message` 表。
3. 不触发正式长期记忆写入。
4. 校验失败直接报错，不做静默降级。
5. 支持运行时改模型：
   - `single`：直接覆盖该 Agent 本轮使用模型
   - `supervisor`：只覆盖主 Agent 模型，不覆盖 subagent 自己配置的模型

这样创建页和编辑页才能真正做到“边改边测”。

## 7.3 正式 Agent 运行接口

建议在启用 Agent 后提供正式入口：

```text
POST /v1/agents/:id/chat/sse
```

建议请求体也支持：

```json
{
  "messages": [],
  "runtime_override": {
    "model_id": "optional-model-id"
  }
}
```

正式入口与调试入口区别：

1. 只接受已启用 Agent。
2. 允许正式线程落库。
3. 允许正式 Agent 级长期记忆读写。
4. 后续可挂到独立的 Agent 对话页。
5. 支持运行时改模型：
   - `single`：覆盖本轮 Agent 模型
   - `supervisor`：只覆盖 supervisor 主 Agent 模型
   - subagent 始终使用它们各自配置的模型，不跟随父 Agent 覆盖

同时建议保留“聊天页里的 Agent 模式”，但行为改成“选择已启用 Agent 后进入 Agent 对话”。

## 8. Agent 编译与运行时设计

建议在现有 `internal/services/agent` 包内继续扩展，不新开一套运行包。

## 8.1 新增运行时定义

建议新增内部结构：

```go
type AgentDefinition struct {
    AgentID              string
    Name                 string
    Description          string
    Type                 string
    SystemPrompt         string
    DefaultModelID       string
    AgentMemoryEnabled   bool
    ToolIDs              []string
    SkillNames           []string
    KnowledgeBaseNames   []string
    SubAgentIDs          []string
}

type RuntimeOverride struct {
    ModelID string
}

type CompileMode string

const (
    CompileModeDebug   CompileMode = "debug"
    CompileModeRuntime CompileMode = "runtime"
)

type CompileOptions struct {
    Mode                  CompileMode
    PersistMessages       bool
    EnableMemoryWriteback bool
    RuntimeOverride       RuntimeOverride
}
```

## 8.2 single 的编译方式

`single` 编译结果：

1. 模型：
   - 若本轮传入 `runtime_override.model_id`，优先使用该模型
   - 否则使用 Agent 自己的 `default_model_id`
   - 仍为空时回退系统默认模型
2. 指令：
   - `ChatModelAgentConfig.Instruction = system_prompt`
3. 工具：
   - 绑定的 `tool_ids`
   - 复用现有插件工具构建逻辑
4. Skill：
   - 复用现有 Skill middleware
   - 运行前把绑定 Skill 转成固定约束消息注入
5. 知识库：
   - 运行前按最后一条 user 消息检索绑定知识库
6. 长期记忆：
   - 若 `agent_memory_enabled=true`，注入该 Agent 自己的 memory root

## 8.3 supervisor 的编译方式

`supervisor` 编译结果：

1. 先编译自己的主 Agent 资源：
   - 自己的模型
   - 自己的提示词
   - 自己的工具/Skill/知识库/长期记忆
2. 再编译每个 subagent
3. 用 `adk.NewAgentTool(...)` 把 subagent 暴露为主 Agent 可调用工具
4. subagent tool 的显示名和描述来自子 Agent 的 `name/description`

模型覆盖规则需要明确：

1. 若本轮传入 `runtime_override.model_id`，只覆盖 supervisor 主 Agent 自己的模型。
2. subagent 不继承父 supervisor 的运行时模型覆盖。
3. 每个 subagent 始终按它自己的配置编译：
   - 优先它自己的 `default_model_id`
   - 为空时回退系统默认模型

这样可以保证：

1. 主 Agent 可以在运行时切换“大脑”。
2. 子 Agent 仍然保持自己的专业模型配置。
3. 不会因为父 Agent 临时换模型，把整棵 Agent 树的模型全改掉。

因为现在允许绑定任意类型 Agent，所以这里的“编译 subagent”本身也可能再次进入 supervisor 编译流程。

因此编译器需要带上依赖路径信息，例如：

```go
type CompileStack struct {
    Path []string
    Seen map[string]int
}
```

编译规则建议：

1. 若当前 `agent_id` 已存在于 `Seen`，直接报“检测到 Agent 循环依赖”。
2. 若 `Path` 深度超过上限，直接报“Agent 编排深度超限”。
3. 每次递归编译子 Agent 时复制一份新的栈上下文。

推荐策略：

1. subagent 运行时上下文和 supervisor 隔离。
2. supervisor 只拿到 subagent 输出结果，不共享完整聊天历史。
3. 这正好符合 `AgentAsTool` 的设计。

这样做的好处：

1. 子 Agent 有自己的工具与知识边界。
2. 子 Agent 可以有自己单独的模型。
3. 子 Agent 可以拥有自己的 Agent 级长期记忆。
4. child supervisor 也可以继续调度它自己的 subagent。

## 8.4 为什么仍然建议 supervisor 继续走 AgentAsTool

因为当前仓库已经采用 Eino ADK，最省成本的路径仍然是：

1. 让每个 supervisor 仍然是一个 `ChatModelAgent`
2. 让它把 subagent 当成工具调用
3. 递归 supervisor 只是“子工具里再包一层 Agent”，而不是另起一套运行框架

这比自己维护“分发器 + 子线程 + 汇总器”要稳很多，也更贴合现有代码结构。

## 9. 资源装配策略

## 9.1 工具

工具只绑定“启用态工具”。

启用/测试时都需要校验：

1. `tool_id` 存在
2. `tool.enabled = true`
3. `plugin.status = enabled`

不满足时直接报错。

运行时建议复用：

1. `Plugin.BuildAllRuntimeCodeTools(...)`
2. `Plugin.BuildAllRuntimeAPITools(...)`
3. `Plugin.BuildAllRuntimeMCPTools(...)`

但要补一个“按指定 `tool_id` 精确构建”的入口，避免 supervisor 的每个子 Agent都去全量装配。

## 9.2 Skill

这里必须明确区分 chat 模式和 agent 模式。

### chat 模式现状

当前 chat 模式的 Skill 行为是：

1. 后端全局挂载 Skill middleware。
2. `list_skill_files` / `read_skill_file` 这类 Skill 文件工具也是全局可用。
3. 前端通过 mention 选择的 skill，只会在 prompt 里插入一条“请优先参考并执行以下技能约束”消息。
4. 它不是 Skill 白名单，也不是强隔离。

也就是说，当前 chat 更接近“全局 Skill 仓库 + 用户提示优先使用哪些 Skill”。

### agent 模式建议

Agent 中的 Skill 是绑定资源，不能沿用 chat 这套“全局可见”的策略。

建议 Agent 模式改成：

1. 只暴露该 Agent 绑定的 Skill。
2. 绑定 Skill 继续通过“约束提示 + Skill middleware”组合生效。
3. 但 Skill middleware 背后的 backend 需要加一层白名单过滤，而不是直接把整个 `dao.SkillStore` 暴露给 Agent。

建议新增一个受限 backend，例如：

```go
type FilteredSkillBackend struct {
    Base          *RemoteBackend
    AllowedSkills map[string]struct{}
}
```

行为要求：

1. `List()` 只返回绑定的 Skill frontmatter。
2. `Get(name)` 只允许读取绑定 Skill。
3. `list_skill_files` / `read_skill_file` 也要复用同一份白名单，而不是继续直接对全 Skill 仓库开放。

这样才能保证：

1. chat 模式继续保持现在的宽松策略。
2. agent 模式真正具备“Skill 绑定”语义。
3. supervisor/子 Agent 之间的 Skill 边界也能真正隔离。

Skill 绑定本身仍然不建议把 Skill 全量内容预先塞进系统提示词。

原因：

1. 现有 Skill middleware 已支持按需加载。
2. 可以继续利用 progressive disclosure。
3. 避免 system prompt 膨胀。

## 9.3 知识库

知识库继续复用当前链路：

1. 取最后一条 user 文本
2. 对绑定知识库逐个检索
3. 拼接临时 system 消息注入

只是来源不再是前端 mention 选择，而是 Agent 固定绑定配置。

## 9.4 长期记忆

这是本需求里最需要改干净的一点。

当前实现里的 agent memory 根是固定值：

```text
viking://agent/default/memories/
```

要支持“Agent 级长期记忆”，建议改成：

```text
viking://agent/{agent_id}/memories/
```

规则建议：

1. 用户级长期记忆：
   - 维持现状
   - 继续走 `viking://user/default/memories/`
2. Agent 级长期记忆：
   - 只有当 `agent_memory_enabled=true` 时才参与检索和写入
   - 目标根改为 `viking://agent/{agent_id}/memories/`
3. 调试接口：
   - v1 默认不写 Agent 长期记忆
   - 避免创建/编辑页测试污染正式记忆

这比“测试也写入正式记忆”更安全。  
如果后续确实要测试记忆，再单独做“调试记忆沙箱”。

## 10. 校验与约束

## 10.1 创建/更新时校验

1. `name` 必填
2. `agent_type` 只能是 `single/supervisor`
3. `single` 不允许提交 `sub_agent_ids`
4. `supervisor` 的 `sub_agent_ids` 必须来自已启用 Agent
5. `default_model_id` 非空时必须存在且可用
6. `tool_ids/skill_names/kb_names` 必须都能解析到真实资源
7. `example_questions_json` 必须是字符串数组
8. `chat_background_json` 必须是合法 JSON
9. 不允许形成 Agent 依赖环
10. 不允许超过最大编排深度

## 10.2 启用时强化校验

启用比保存更严格，建议额外校验：

1. `system_prompt` 不能为空
2. `description` 不能为空
3. `supervisor` 至少绑定 1 个 subagent
4. subagent 全部为 `enabled`
5. 不允许引用自己
6. 不允许出现循环依赖
7. 不允许超过最大编排深度
8. 不允许被禁用的模型/工具

## 10.3 停用与依赖校验

如果某个 Agent 已被启用中的 supervisor 绑定，建议：

1. 禁止直接停用
2. 禁止直接删除
3. 返回依赖列表给前端展示

这样可以避免 supervisor 运行时才暴露配置缺陷。

## 11. 前端页面方案

## 11.1 Agent 市场页

建议新增页面：

```text
openIntern_forentend/app/(workspace)/agents/page.tsx
```

页面职责：

1. 列表展示 Agent 卡片
2. 支持按：
   - 名字搜索
   - 类型过滤
   - 状态过滤
3. 卡片动作：
   - 创建
   - 编辑
   - 启用
   - 停用
   - 测试

卡片展示字段建议：

1. 头像
2. 名字
3. 描述
4. 类型
5. 状态
6. 默认模型
7. 绑定资源数量摘要：
   - 工具数
   - Skill 数
   - 知识库数
   - subagent 数

## 11.2 创建/编辑页

建议做成同一个编辑器组件：

```text
openIntern_forentend/app/(workspace)/agents/agent-editor.tsx
```

布局建议左右分栏：

1. 左侧：配置表单
2. 右侧：测试面板

表单区块建议：

1. 基础信息
   - 类型
   - 名字
   - 描述
   - 头像
   - 聊天背景
   - 示例问法
2. 运行配置
   - 系统提示词
   - 默认模型
   - Agent 长期记忆开关
3. 资源绑定
   - 工具
   - Skill
   - 知识库
4. Supervisor 专属
   - subagent 选择器

行为要求：

1. 当类型切到 `single` 时，隐藏并清空 subagent。
2. 当类型切到 `supervisor` 时，展示 subagent 选择器。
3. subagent 选择器展示“所有已启用 Agent”。
4. 选择器里需要显示 Agent 类型标签，帮助区分 `single/supervisor`。

## 11.3 测试面板

测试面板是这次需求的关键，不建议做成单独弹窗之后再跳别处。

建议直接内嵌在编辑页右侧，包含：

1. 聊天气泡区
2. 输入框
3. 清空会话按钮
4. 重新编译并测试按钮
5. 当前草稿资源摘要

测试时前端发送“当前草稿 + 当前测试会话消息”到 `/v1/agents/debug/sse`。

这样用户能在未保存草稿时直接验证：

1. 系统提示词是否生效
2. 工具是否能调用
3. Skill 是否能触发
4. 知识库是否能命中
5. supervisor 是否会正确调度子 Agent

## 11.4 现有聊天页的处理建议

当前聊天页有一个尚未落地的 `Agent` 模式开关。  
结合新需求，建议不要移除，而是把它真正落成“Agent 对话模式”。

建议交互：

1. 当 `conversationMode = "chat"` 时，维持现有普通聊天模式：
   - 可以自由选模型
   - 可以自由选工具模式
   - mention 选择 Skill / 知识库
2. 当 `conversationMode = "agent"` 时：
   - 展示“Agent 选择器”
   - 候选只展示已启用 Agent
   - 选择 Agent 后，将该 Agent 的 `default_model_id` 回填到当前模型选择器
   - 若该 Agent 未配置默认模型，则回填系统默认模型
   - 回填之后，用户仍然可以继续手动改模型
3. 用户在 Agent 模式下手动改模型时，本质上就是设置本轮 `runtime_override.model_id`
4. 若切换到另一个 Agent，则再次按新 Agent 的默认模型重新回填

这套交互和前面的运行时规则一致：

1. 选中 Agent 时先用 Agent 默认模型初始化当前对话配置。
2. 若用户不再修改，则直接按该默认模型运行。
3. 若用户手动修改，则以手动选择为准。
4. 对 `supervisor` 来说，聊天页里手动改模型也只影响主 Agent，不影响 subagent。

前端建议新增状态：

1. `selectedAgentId`
2. `selectedAgentOption`
3. `agentModelBackfilled`

其中 `agentModelBackfilled` 用来区分：

1. 当前模型值是否来自“刚选中 Agent 时的自动回填”
2. 用户是否已经手动改过模型

这样可以避免出现：

1. 用户手动改过模型后，普通刷新又被错误覆盖
2. 同一个 Agent 再次选中时是否需要重新回填语义不清

对应接口建议：

1. 聊天页初始化时拉取 `/v1/agents?status=enabled`
2. Agent 模式发送消息时走 `/v1/agents/:id/chat/sse`
3. 若当前模型选择器值和 Agent 默认模型不同，则把差异作为 `runtime_override.model_id` 传给后端

## 12. 代码拆分原则

实现时需要明确按“共用层”和“业务专属层”拆文件，避免把 chat 和 agent 的差异继续堆进同一个大文件里。

### 12.1 应放在共用层的内容

这些能力本质上是运行时基础设施，应抽成共用文件或共用子目录：

1. 模型解析与实例化
2. 工具按 ID 装配
3. 知识库检索与上下文注入
4. 长期记忆 root 解析与注入
5. Skill backend 过滤能力
6. Runner 构建公共逻辑
7. AG-UI 消息流转与 sender/accumulator 适配

建议位置：

1. `openIntern_backend/internal/services/agent/runtime/`
2. `openIntern_backend/internal/services/agent/resources/`
3. `openIntern_backend/internal/services/agent/memory/`

### 12.2 应和 chat 分开的内容

这些逻辑明显只属于原有 chat 模式，不应该继续和 Agent 市场逻辑耦合：

1. `forwardedProps` 解析
2. mention 选择的 Skill / 知识库临时注入
3. chat 页面自己的模型/工具临时选择
4. 普通线程标题生成与普通聊天线程落库入口

建议位置：

1. `openIntern_backend/internal/services/chat_runtime/`
2. 或保留在现有 `internal/services/agent/agent_entry.go` 附近，但拆成 `chat_*` 文件

### 12.3 应和 agent 分开的内容

这些逻辑只属于 Agent 市场与 Agent 运行：

1. Agent CRUD
2. Agent 绑定关系解析
3. Agent 编译器
4. supervisor 依赖图与递归编译
5. Agent 调试测试入口
6. Agent 正式运行入口

建议位置：

1. `openIntern_backend/internal/services/agent_definition/`
2. `openIntern_backend/internal/services/agent_runtime/`

如果不想新建顶层目录，也至少要在 `internal/services/agent/` 下拆成：

1. `chat_*`
2. `definition_*`
3. `runtime_*`
4. `resource_*`

### 12.4 文件拆分的具体约束

1. 不要把 chat 的 `forwardedProps` 解析继续塞进 Agent 编译器。
2. 不要把 Agent 市场 CRUD 和运行时 Runner 装配写在同一个文件。
3. 不要把 single/supervisor 递归编译和普通 chat SSE 入口写在同一个文件。
4. Skill 的白名单过滤层要做成可复用组件，供 Agent 模式和后续其他模式共用。

### 12.5 推荐目录结构草案

下面是结合当前仓库结构后，建议新增或拆分的目录草案。目标不是“为了好看重构目录”，而是把 chat 和 agent 的边界拆清楚。

#### 后端目录草案

```text
openIntern_backend/internal/
├── controllers/
│   ├── chat.go
│   └── agent.go                         # 新增：Agent CRUD / enable / disable / debug / runtime chat
├── models/
│   ├── thread.go
│   ├── message.go
│   ├── agent.go                        # 新增：Agent 主表
│   └── agent_binding.go                # 新增：Agent 资源绑定表
├── dao/
│   ├── thread.go
│   ├── message.go
│   ├── agent.go                        # 新增：Agent DAO
│   └── agent_binding.go                # 新增：Agent 绑定 DAO
├── services/
│   ├── agent/                          # 保留为“运行时编排总入口”
│   │   ├── default_service.go
│   │   ├── agent_service.go
│   │   ├── runtime/
│   │   │   ├── runner_builder.go       # 共用：构建 Runner、Sender、流式输出桥接
│   │   │   ├── model_resolver.go       # 共用：按 model_id/provider 解析模型
│   │   │   ├── tool_resolver.go        # 共用：按 tool_id 精确装配工具
│   │   │   ├── kb_context.go           # 共用：知识库上下文注入
│   │   │   ├── memory_scope.go         # 共用：用户级/Agent级 memory root 解析
│   │   │   ├── memory_context.go       # 共用：长期记忆注入
│   │   │   ├── skill_backend_filter.go # 共用：Skill 白名单过滤 backend
│   │   │   └── message_stream.go       # 共用：AG-UI message/sender 适配
│   │   ├── chatmode/
│   │   │   ├── entry.go                # chat 模式入口，承接原 RunAgent 主链路
│   │   │   ├── forwarded_props.go      # chat 专属：agentConfig/contextSelections 解析
│   │   │   ├── context_compression.go  # chat 专属：上下文压缩接线
│   │   │   └── persistence.go          # chat 专属：线程/消息持久化
│   │   ├── agentmode/
│   │   │   ├── definition.go           # AgentDefinition / DTO
│   │   │   ├── definition_service.go   # Agent CRUD / 启停 / 校验
│   │   │   ├── compiler.go             # single/supervisor 编译总入口
│   │   │   ├── compiler_single.go      # single 编译
│   │   │   ├── compiler_supervisor.go  # supervisor 编译
│   │   │   ├── dependency_graph.go     # 环检测 / 深度限制 / 依赖查询
│   │   │   ├── resource_resolver.go    # 从 binding 解析 tools/skills/kbs/subagents
│   │   │   ├── debug_entry.go          # /v1/agents/debug/sse
│   │   │   ├── runtime_entry.go        # /v1/agents/:id/chat/sse
│   │   │   ├── debug_session.go        # 调试会话隔离，不落正式线程
│   │   │   └── persistence.go          # Agent 正式线程落库（如与 chat 存在差异）
│   │   └── shared/
│   │       ├── types.go                # 运行时公共类型
│   │       ├── errors.go               # 运行时公共错误
│   │       └── validate.go             # 通用校验 helper
│   ├── middlewares/
│   │   └── skill/
│   │       ├── backend.go
│   │       ├── tools.go
│   │       └── filtered_backend.go     # 新增：只暴露绑定 Skill 的 backend
│   └── memory/
│       └── ...                         # 保持现有 provider/sync 能力
└── routers/
    └── router.go
```

#### 后端拆分说明

1. `services/agent/runtime/`
   - 放 chat 和 agent 都会用到的运行时能力。
   - 这里不能依赖 `forwardedProps`，也不能依赖 `agent_binding` 表。
2. `services/agent/chatmode/`
   - 保留原 chat 模式的临时配置行为。
   - 继续服务普通聊天页。
3. `services/agent/agentmode/`
   - 只负责 Agent 业务实体、编译和运行。
   - 不应该反向依赖 chat 页的 mention 选择逻辑。
4. `middlewares/skill/filtered_backend.go`
   - 用来补齐“agent 绑定 Skill”和“chat 全局 Skill”之间的能力差异。

#### 前端目录草案

```text
openIntern_forentend/app/
├── (workspace)/
│   ├── chat/
│   │   ├── page.semi.tsx
│   │   ├── ChatModeConfigureArea.tsx
│   │   ├── chat-helpers.ts
│   │   └── ...                         # 保持普通 chat 页，不直接塞 Agent 市场逻辑
│   ├── agents/
│   │   ├── page.tsx                    # 新增：Agent 市场列表页
│   │   ├── [agentId]/
│   │   │   └── page.tsx                # 新增：Agent 编辑页
│   │   ├── new/
│   │   │   └── page.tsx                # 新增：Agent 创建页
│   │   ├── components/
│   │   │   ├── AgentCard.tsx
│   │   │   ├── AgentStatusTag.tsx
│   │   │   ├── AgentEditor.tsx
│   │   │   ├── AgentBasicForm.tsx
│   │   │   ├── AgentPromptForm.tsx
│   │   │   ├── AgentResourceBindingForm.tsx
│   │   │   ├── AgentSubagentSelector.tsx
│   │   │   ├── AgentDebugPanel.tsx
│   │   │   ├── AgentDebugMessageList.tsx
│   │   │   └── AgentDebugComposer.tsx
│   │   ├── hooks/
│   │   │   ├── useAgentCatalog.ts
│   │   │   ├── useAgentEditorState.ts
│   │   │   ├── useAgentDebugSession.ts
│   │   │   └── useAgentResourceOptions.ts
│   │   ├── services/
│   │   │   ├── agent-api.ts           # Agent CRUD / enable / disable / debug / runtime chat
│   │   │   └── agent-mappers.ts
│   │   ├── types/
│   │   │   └── agent.ts
│   │   └── constants/
│   │       └── agent.ts
│   └── ...
├── components/
│   └── ui/
│       └── ...                         # 继续复用通用 UI 组件
└── ...
```

#### 前端拆分说明

1. `chat/`
   - 继续只负责普通聊天页。
   - 不要把 Agent 市场的表单状态、debug 会话状态塞进这里。
2. `agents/components/`
   - 页面型组件和业务组件拆开，避免一个超大 `page.tsx`。
3. `agents/hooks/`
   - 把“表单状态管理”“调试会话状态管理”“资源选项拉取”拆开。
4. `agents/services/agent-api.ts`
   - 统一封装 Agent 相关接口，避免页面里散落 `requestBackend(...)`。

#### 最小可接受拆分版本

如果本轮不想一次建太多目录，至少做到下面这个粒度：

```text
openIntern_backend/internal/services/agent/
├── chat_*.go
├── runtime_*.go
├── agent_definition_*.go
├── agent_compile_*.go
└── agent_debug_*.go
```

```text
openIntern_forentend/app/(workspace)/agents/
├── page.tsx
├── new/page.tsx
├── [agentId]/page.tsx
├── agent-editor.tsx
├── agent-debug-panel.tsx
└── agent-api.ts
```

这已经能避免最糟糕的耦合：不会把普通 chat 和 Agent 市场继续写进同一个页面文件，也不会把 chat 运行入口和 agent 编译器堆在同一个 Go 文件里。

## 13. 后端代码改造清单（建议）

## 13.1 数据层

新增：

1. `openIntern_backend/internal/models/agent.go`
2. `openIntern_backend/internal/models/agent_binding.go`
3. `openIntern_backend/internal/dao/agent.go`
4. `openIntern_backend/internal/dao/agent_binding.go`

修改：

5. `openIntern_backend/internal/database/database.go`
   - AutoMigrate 新表

## 13.2 控制器与路由

新增：

1. `openIntern_backend/internal/controllers/agent.go`

修改：

2. `openIntern_backend/internal/routers/router.go`
   - 注册 `/v1/agents/*`

## 13.3 服务层

建议在现有 `internal/services/agent` 包中补这些文件：

1. `agent_definition.go`
   - Agent 定义与 DTO
2. `agent_definition_service.go`
   - CRUD / enable / disable / 校验
3. `agent_compile.go`
   - single/supervisor 编译器
4. `agent_debug_entry.go`
   - 调试测试 SSE 入口
5. `agent_runtime_entry.go`
   - 正式 Agent 运行入口
6. `agent_memory_scope.go`
   - Agent 级长期记忆 root 解析
7. `agent_resource_resolver.go`
   - 工具/Skill/知识库/subagent 解析
8. `agent_dependency_graph.go`
   - 依赖图、环检测、深度校验

同时需要调整现有：

9. `agent_runner.go`
   - 抽出“按定义编译 Agent”的公共逻辑
10. `agent_entry.go`
   - 继续承载普通聊天执行
11. `memory_retriever.go`
   - 支持 agent-specific memory root

## 14. 分阶段落地建议

建议不要一口气把所有东西一起改完，按四步推进更稳。

### 第一阶段：Agent 市场基础 CRUD

目标：

1. 建表
2. 列表/详情/创建/编辑/启停接口
3. 前端市场页和编辑页
4. 资源绑定和启用校验

此阶段先不接正式运行，只把配置中心建起来。

### 第二阶段：single Agent 调试链路

目标：

1. `/v1/agents/debug/sse`
2. 按草稿编译 single Agent
3. 复用模型/工具/Skill/知识库
4. 调试会话不落库

做到这一步后，创建页和编辑页已经可用。

### 第三阶段：supervisor 编排

目标：

1. subagent 绑定
2. `AgentAsTool` 编译 supervisor
3. supervisor 调试测试
4. 依赖与启用校验
5. 环检测与深度限制

### 第四阶段：正式 Agent 对话入口

目标：

1. `/v1/agents/:id/chat/sse`
2. Agent 级正式线程落库
3. Agent 级长期记忆正式启用
4. 视情况补 Agent 对话页

## 15. 风险与关键决策点

## 15.1 最大风险：测试链路污染正式链路

如果测试直接复用 `/v1/chat/sse`：

1. 会写正式线程
2. 会触发标题生成
3. 会触发长期记忆同步
4. 很难和正式聊天隔离

所以调试接口必须单独做。

## 15.2 第二风险：supervisor 递归复杂度

现在需求已经明确允许 supervisor 绑定任意类型 Agent，因此这个风险不能通过“限制类型”绕开，只能正面处理：

1. 会出现依赖环
2. 资源递归膨胀
3. 编译和排错都更难
4. supervisor 链路更容易放大模型与工具初始化成本

所以必须把以下机制作为必做项，而不是优化项：

1. 依赖图环检测
2. 最大深度限制
3. 最大 subagent 数量限制
4. 编译过程的结构化日志

## 15.3 第三风险：工具构建成本过高

当前运行时偏向“全量构建工具 + 中间件过滤”。  
对于 supervisor 来说，如果每个 subagent 都这么做，成本会明显变高。

建议补“按 `tool_id` 精确构建工具”能力，减少不必要的 MCP/API/code tool 初始化。

## 15.4 第四风险：长期记忆语义不清

如果不把“用户长期记忆”和“Agent 级长期记忆”拆开，后续很容易出现：

1. 同一个 Agent 和另一个 Agent 互相污染记忆
2. 测试会话误写入正式记忆
3. supervisor 和 subagent 使用同一记忆空间

所以必须在本次设计里把 memory root 规则一次定清楚。

## 16. 验收标准（DoD）

满足以下条件即可认为需求主链路完成：

1. Agent 市场可展示 `draft/enabled/disabled` 三态 Agent。
2. 用户可创建并编辑 `single` 与 `supervisor` Agent。
3. `single` 不允许绑定 subagent。
4. `supervisor` 可绑定任意类型的已启用 Agent。
5. 创建页和编辑页均可对当前草稿直接测试。
6. single 测试时可正确使用：
   - 系统提示词
   - 默认模型
   - 工具
   - Skill
   - 知识库
7. supervisor 测试时可正确递归调用 subagent。
8. Agent 启用时会做强校验，错误不会被静默吞掉。
9. 若存在环依赖或超深度，保存、启用、调试都会被拒绝。
10. 正式运行时，Agent 级长期记忆能按 `agent_id` 隔离。

## 17. 最终建议

综合当前仓库结构，最稳的实现路径是：

1. 先把 Agent 当成“可持久化配置实体”补出来。
2. 在现有 `services/agent` 里新增“按定义编译运行时 Agent”的能力。
3. `single` 直接落到 `ChatModelAgent`。
4. `supervisor` 通过 `AgentAsTool` 包装 subagent。
5. 调试测试单独走 `/v1/agents/debug/sse`，不要污染正式线程和正式长期记忆。
6. supervisor 允许绑定任意类型 Agent，但必须同步实现 DAG 校验、深度上限和递归编译能力。

如果按这个方案推进，当前仓库已有的模型、工具、Skill、知识库、长期记忆能力都能最大程度复用，真正新增的核心只有三块：

1. Agent 配置数据模型
2. Agent 编译器
3. Agent 调试测试入口

这三块补齐后，后续再扩“正式 Agent 对话页”会自然很多，而不是继续在现有聊天页的临时 `forwardedProps` 上堆逻辑。
