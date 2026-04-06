# OpenViking 容器化远程导入改造设计

## 1. 背景与目标

当前 openIntern 已将 OpenViking 部署到 Docker 容器内运行，`openIntern_backend` 与 `openIntern_forentend` 也按容器化方式部署。

现有后端接入里，知识库导入与 skill 导入仍然把后端本地文件路径直接传给 OpenViking：

- 知识库导入调用 `POST /api/v1/resources`，请求体里的 `path` 指向后端本地临时目录或文件
- skill 导入调用 `POST /api/v1/skills`，请求体里的 `data` 指向后端本地解压目录

这套方式隐含前提是“后端进程与 OpenViking 共享同一套可见文件系统”。当 OpenViking 运行在独立容器内，且不允许依赖共享卷时，这个前提不成立，导入会失败。

本次改造目标：

- 去掉 OpenViking 导入链路对宿主机路径和共享卷的依赖
- 保持前端接口与主要业务流程不变
- 让知识库导入、知识库单文件上传、skill zip 导入都改为真正的远程 HTTP 导入
- 不引入兼容性“兜底 hack”，避免继续隐藏部署边界问题

## 2. 当前问题定位

### 2.1 受影响链路

当前直接依赖本地路径的逻辑有两条：

- `openIntern_backend/internal/services/kb/service.go`
  - `Import`
  - `UploadFile`
- `openIntern_backend/internal/controllers/skill.go`
  - `ImportSkill`

它们最终分别落到：

- `openIntern_backend/internal/dao/context_store.go:addResourceWithRootURI`
- `openIntern_backend/internal/dao/context_store.go:importSkill`

### 2.2 问题本质

问题不在于 OpenViking 是否容器化，而在于导入协议仍以“调用方本地文件路径”作为输入。

只要后端与 OpenViking 不共享文件系统，下列写法都会失效：

- `{"path":"/tmp/kb-import-123/docs","target":"viking://resources/xxx/"}`
- `{"data":"/tmp/skill-import-123/unzipped/web-search"}`

因此这次需要改造的是“导入协议适配层”，而不是 search、session、read、list 这类纯 HTTP 能力。

## 3. 范围与非目标

### 3.1 本次范围

- 改造 OpenViking 资源导入与 skill 导入的后端实现
- 抽象统一的“本地内容 -> OpenViking 临时上传 -> OpenViking 导入”流程
- 更新相关文档与配置说明
- 为核心路径补充单元测试

### 3.2 非目标

- 不改造 OpenViking 的搜索、会话、记忆检索接口
- 不改造前端知识库与 skill 管理接口形态
- 不引入 Docker API、`docker cp` 或进入容器执行命令的旁路方案
- 不依赖共享卷或宿主机固定目录
- 不新增集成测试或执行 `go test`

## 4. 方案比较

### 4.1 方案 A：远程临时上传后再导入

做法：

- 后端把待导入文件或目录打包为上传载荷
- 先通过 OpenViking 的临时上传接口把内容上传到服务端
- 再调用资源导入或 skill 导入接口，用 `temp_file_id` 触发真正入库

优点：

- 完全符合容器隔离边界
- 不依赖共享卷
- 后端与 OpenViking 只通过 HTTP 交互
- 能统一知识库与 skill 导入模型

缺点：

- 需要补齐 OpenViking `temp_upload` 请求封装
- 目录导入需要在后端重新打包

### 4.2 方案 B：逐文件远程写入 OpenViking 文件系统

做法：

- 逐级创建目录
- 逐文件写入内容
- 再触发索引或解析

问题：

- 当前公开文档对稳定的写内容接口描述不足
- skill 目录和资源解析完成态不容易统一
- 实现复杂度高，且行为更依赖 OpenViking 内部细节

### 4.3 方案 C：通过 Docker API 或 `docker cp` 向容器送文件

做法：

- 后端识别 OpenViking 容器
- 把文件复制进容器内部临时目录
- 再调用现有 path 导入接口

问题：

- 仍然绕开业务 API
- 运维权限要求更高
- 与“不依赖共享卷、真正对接 Docker 内部 OpenViking”的方向不一致

### 4.4 结论

本次采用方案 A。

## 5. 总体设计

### 5.1 核心原则

- OpenViking 对 openIntern 来说是远程 HTTP 服务，不假设共享文件系统
- 所有需要导入的本地内容，必须先转换为 HTTP 可上传载荷
- “目录导入”和“单文件导入”都统一走远程上传入口
- 上层业务接口尽量不变，改动收敛在 DAO 与 service/controller 层

### 5.2 统一导入模型

导入统一拆成两步：

1. 上传临时文件
2. 使用 `temp_file_id` 触发 OpenViking 导入

抽象后链路如下：

```text
backend local temp dir/file
  -> package as upload payload
  -> POST /api/v1/resources/temp_upload
  -> get temp_file_id
  -> POST /api/v1/resources or POST /api/v1/skills
  -> OpenViking imports and indexes content
```

### 5.3 目录导入策略

由于知识库 zip 导入与 skill zip 导入本质都是“一个目录树”的导入，本次统一处理为：

- 后端先在本容器内临时解压、校验、整理目录
- 再把目录重新打包为单个归档文件
- 通过远程上传接口上传归档文件
- 再通过导入接口让 OpenViking在服务端解包并导入

这里允许后端使用自身临时目录做安全校验和重打包，但这些路径不会暴露给 OpenViking。

## 6. 模块改造设计

### 6.1 `internal/database/context_store.go`

新增 OpenViking 远程上传能力：

- `UploadTempFile(ctx, localPath string) (*TempUploadResult, error)`
- `UploadTempArchive(ctx, rootDir string, archiveBaseName string) (*TempUploadResult, error)`

其中：

- `UploadTempFile` 负责上传单个文件
- `UploadTempArchive` 负责把目录打包成 zip 或 tar.gz 后再上传

新增导入请求模型：

- `AddResourceFromTempFile`
- `AddSkillFromTempFile`

请求体不再使用本地 `path` 或本地目录 `data`，而是使用远程上传返回的 `temp_file_id`。

### 6.2 `internal/dao/context_store.go`

当前：

- `addResourceWithRootURI(ctx, resourcePath, targetURI, wait, timeoutSeconds)`
- `importSkill(ctx, rootDir string)`

改造后：

- 保留原有接口名的上层语义，但内部不再把 `resourcePath`、`rootDir` 原样传给 OpenViking
- 对目录场景先远程上传归档，再发起导入
- 对单文件场景先远程上传文件，再发起导入

建议拆成更明确的内部函数：

- `uploadTempFile`
- `uploadTempArchive`
- `addResourceFromTempFileID`
- `addSkillFromTempFileID`

这样可以把“上传”和“导入”错误分开，日志也更清楚。

### 6.3 `internal/services/kb/service.go`

保留现有业务接口：

- `Import`
- `UploadFile`

行为变化：

- `Import` 仍然可以在后端临时目录解压 zip 以做路径校验
- 解压后不再把 `rootDir` 传给 OpenViking
- `UploadFile` 仍然先把 multipart 文件落到后端临时目录，但后续改为远程上传到 OpenViking

### 6.4 `internal/controllers/skill.go`

保留现有 zip 校验与 frontmatter 校验逻辑：

- 仍然检查 `.zip`
- 仍然解析 `SKILL.md`
- 仍然写入 skill frontmatter 表

行为变化：

- 校验通过后，不再把 `rootDir` 当作 OpenViking 可见目录
- 改为将 skill 根目录打包并上传，然后调用 skill 导入接口

### 6.5 配置层

本次不新增必须配置项。

继续使用：

- `tools.openviking.base_url`
- `tools.openviking.api_key`
- `tools.openviking.timeout_seconds`

文档中需要明确：

- `tools.openviking` 不再假设本地路径可见
- 若部署为独立容器，只需要网络可达即可

## 7. 请求流设计

### 7.1 知识库 zip 导入

```text
前端上传 zip
  -> backend 解压到自身临时目录
  -> 校验 zip entry 路径安全
  -> backend 重新打包目录
  -> POST /api/v1/resources/temp_upload
  -> 获取 temp_file_id
  -> POST /api/v1/resources
     { temp_file_id, target, wait, timeout }
  -> 返回 accepted
```

### 7.2 知识库单文件上传

```text
前端上传文件
  -> backend 落盘到自身临时文件
  -> POST /api/v1/resources/temp_upload
  -> 获取 temp_file_id
  -> POST /api/v1/resources
     { temp_file_id, target, wait, timeout }
  -> 返回 accepted
```

### 7.3 skill zip 导入

```text
前端上传 skill zip
  -> backend 解压并定位 skill root
  -> 校验 SKILL.md 与 frontmatter
  -> 写 skill_frontmatters
  -> backend 重新打包 skill root
  -> POST /api/v1/resources/temp_upload
  -> 获取 temp_file_id
  -> POST /api/v1/skills
     { temp_file_id, wait, timeout }
  -> 返回 skill_uri
```

## 8. 错误处理设计

### 8.1 上传阶段错误

上传失败时应直接返回错误，不进入导入阶段。

日志中至少记录：

- OpenViking endpoint
- 上传文件名或归档名
- HTTP 状态码
- 截断后的响应体

### 8.2 导入阶段错误

导入失败时保留 OpenViking 返回的错误信息，不做模糊包装。

目标是让调用方直接看到：

- 参数错误
- 不支持的上传格式
- OpenViking 服务端导入失败

### 8.3 skill frontmatter 与 OpenViking 导入顺序

当前 skill 导入流程中，frontmatter 数据库存储发生在 OpenViking 导入前。

本次先保持这个顺序，避免扩大改动范围，但要补充说明：

- 如果 OpenViking 导入失败，会留下仅存在于业务库、但未成功导入 OpenViking 的 skill 元数据

这属于现存行为的延续，不在本次顺手修复。

## 9. 测试设计

本次只补充轻量单元测试，不执行 `go test`。

建议覆盖：

- 目录打包时忽略非法路径
- 根据文件还是目录选择上传策略
- 上传成功后正确构造 `temp_file_id` 导入请求
- skill 导入与资源导入的请求体字段差异
- OpenViking 返回错误时是否原样透传

优先增加 DAO 或小型辅助函数测试，避免引入大规模集成测试。

## 10. 文档改造

需要同步更新：

- `README.md`
- `docs/local-development.md`

更新点：

- 删除或修正“OpenViking 可以读取 backend 本地路径”的隐含表述
- 明确导入改为通过 HTTP 上传，不依赖共享卷
- 保留 OpenViking 独立容器部署方式说明

## 11. 风险与约束

- 本设计依赖 OpenViking 服务端已支持 `temp_upload -> temp_file_id` 导入链路
- 若部署的 OpenViking 版本过旧，不支持该链路，则需要先升级 OpenViking
- 目录重新打包会带来额外 I/O，但当前导入频率低，优先保证部署正确性

## 12. 实施计划

1. 在 context store 层增加远程临时上传和基于 `temp_file_id` 的导入封装
2. 改造知识库导入与单文件上传逻辑
3. 改造 skill zip 导入逻辑
4. 补充单元测试
5. 更新 README 与本地开发文档
