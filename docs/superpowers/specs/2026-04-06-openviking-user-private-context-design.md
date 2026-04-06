# OpenViking User-Private Context Design

## Goal

将 OpenViking 中的 `memory`、`skills`、`知识库` 从当前共享命名空间改为按 openIntern 登录用户私有隔离，直接使用 openIntern `user_id` 作为 OpenViking `{user_space}`，不兼容旧 `default` 数据。

## Scope

- 包含：
  - 用户长期记忆检索路径
  - skill 文件存储路径与 skill frontmatter 元数据
  - 知识库目录、导入、枚举与内容访问路径
- 不包含：
  - agent 命名空间重构
  - OpenViking session 结构调整
  - 基于 session API 的 memory sync 写入隔离
  - 旧 `default` 数据迁移或兼容读取
  - OpenViking 原生 account/user key 多租户鉴权接入

## Current State

- 用户长期记忆根路径写死为 `viking://user/default/memories/`，agent 记忆根路径写死为 `viking://agent/default/memories/`。
- skill 根路径来自 [openIntern_backend/config.yaml](/Users/fqc/project/agent/openIntern/openIntern_backend/config.yaml) 的 `tools.openviking.skills_root`，当前配置为 `viking://agent/skills`。
- skill 元数据表 [skill_frontmatter.go](/Users/fqc/project/agent/openIntern/openIntern_backend/internal/models/skill_frontmatter.go) 只有 `skill_name/raw`，没有 `user_id`。
- 知识库根路径固定为 `viking://resources/`，所有用户共享同一层目录。
- 后端接口做了登录校验，但 OpenViking 路径没有绑定当前登录用户，因此隔离只存在于 MySQL 线程/消息表，不存在于 OpenViking 数据层。

## Decision

### 1. 用户私有路径规则

- 用户记忆根改为：`viking://user/{user_id}/memories/`
- 用户 skills 根改为：`viking://user/{user_id}/skills/`
- 用户知识库根改为：`viking://resources/users/{user_id}/kbs/{kb_name}/`

其中 `{user_id}` 直接使用 openIntern 当前登录用户 UUID，不引入别名或 slug。

### 2. 统一 URI 构造层

新增一层集中路径解析能力，所有 OpenViking 用户私有路径都从同一处构造，禁止在业务代码中继续散落拼接：

- `UserMemoryRoot(userID string) string`
- `UserSkillsRoot(userID string) string`
- `UserKnowledgeBaseRoot(userID string) string`
- `UserKnowledgeBaseURI(userID, kbName string) string`

这层只负责路径规则，不承担 OpenViking 调用逻辑。

### 3. Memory 设计

- 用户长期记忆检索只查当前用户的 `viking://user/{user_id}/memories/`
- 当前范围内不重构 agent 记忆逻辑，也不把 agent 记忆改成多用户
- 当前范围内不修改基于 OpenViking session API 的 memory sync 写路径，因为这会引入 session 结构调整，超出本次边界
- 不再保留 `default` 兜底检索，不做双读

### 4. Skills 设计

- skill 文件目录改为当前用户的 `viking://user/{user_id}/skills/`
- `skills_root` 不再作为全局固定配置根使用，而是保留为可选默认值或直接废弃，由代码按 user space 生成根路径
- skill 元数据表增加 `user_id` 字段，并将查询条件统一改为 `(user_id, skill_name)`
- skill 导入时，先写当前用户的 frontmatter，再把技能包导入当前用户的 skills 根
- skill 列表、摘要读取、文件读取都只允许访问当前用户 skills 根

### 5. 知识库设计

- 顶层知识库列表不再直接列举 `viking://resources/`
- 列表接口只枚举 `viking://resources/users/{user_id}/kbs/` 下的一级目录名作为 KB 名称
- 导入 ZIP、上传单文件、读取树、移动、删除、读取内容都必须限定在当前用户 KB 根前缀下
- 外部传入的 URI 若不在当前用户 KB 前缀内，直接判为非法输入

## Why This Design

- 与 OpenViking 官方 `user/{user_space}` 模型一致，但不引入当前项目暂时不需要的原生多租户鉴权体系。
- 直接使用 openIntern `user_id` 作为 `{user_space}`，消除映射表、重命名和迁移问题。
- 统一路径构造层可以把用户隔离规则从业务逻辑里抽出来，避免未来再次出现硬编码 `default`。
- 不兼容旧数据可以显著降低分支复杂度，适合未上线项目。

## Data Model Changes

- 修改 [skill_frontmatter.go](/Users/fqc/project/agent/openIntern/openIntern_backend/internal/models/skill_frontmatter.go)
  - 新增 `UserID string`
  - 为 `(user_id, skill_name)` 建立查询索引
- 相关 DAO 与 service 全部改为显式传入 `userID`

## File Areas

- OpenViking 路径与 DAO：
  - `openIntern_backend/internal/dao/memory_search.go`
  - `openIntern_backend/internal/dao/skill_store.go`
  - `openIntern_backend/internal/dao/knowledge_base.go`
  - 新增一个集中 URI 构造文件
- Memory：
  - `openIntern_backend/internal/services/memory/openviking/retriever.go`
  - `openIntern_backend/internal/services/memory/openviking/sync_backend.go`
- Skills：
  - `openIntern_backend/internal/models/skill_frontmatter.go`
  - `openIntern_backend/internal/dao/skill_frontmatter.go`
  - `openIntern_backend/internal/services/skill/frontmatter.go`
  - `openIntern_backend/internal/services/middlewares/skill/backend.go`
  - `openIntern_backend/internal/controllers/skill.go`
- 知识库：
  - `openIntern_backend/internal/services/kb/service.go`
  - `openIntern_backend/internal/controllers/kb.go`

## Non-Goals

- 不尝试保留旧 `default` 数据可见性
- 不增加后台迁移任务
- 不在前端页面展示“为什么这样做”的实现说明
- 不在本轮通过改造 session id 或 session user space 来解决 memory sync 写入隔离

## Verification

- 使用当前登录用户 A 导入 skill、创建知识库、触发记忆检索后，确认 OpenViking 读路径只落在 A 的命名空间下
- 使用另一用户 B 登录后，列表接口看不到 A 的 skills 与知识库
- B 侧检索 memory 时不会命中 A 的用户记忆
- `go build ./...` 通过
- 不执行 `go test`
