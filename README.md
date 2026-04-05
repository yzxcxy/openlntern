# openIntern

openIntern 是一个面向智能体工作流的全栈项目，当前仓库包含：

- `openIntern_backend`：Go 后端，负责认证、会话、聊天流式输出、插件、Skill、知识库、模型配置等能力。
- `openIntern_forentend`：Next.js 前端，负责管理后台与聊天工作台界面。
- `go`：本地 AG-UI Go SDK，后端通过 `replace` 指向该目录。
- `scripts`：本地开发辅助脚本。
- `开发文档`：设计、链路梳理和协议文档，内容仅供参考，未必与当前实现完全一致。

## 当前能力

从现有路由和页面实现看，项目已覆盖以下核心模块：

- 用户注册、登录与 JWT 鉴权
- 流式聊天接口 `POST /v1/chat/sse`
- 会话与消息管理
- A2UI 配置管理
- Skill 元数据管理与 `.zip` 导入
- 插件管理，支持 API / MCP / Code 三种运行时
- 知识库导入、文件上传与树形管理
- 模型提供商、模型列表与默认模型配置

## 技术栈

后端：

- Go 1.25.x
- Gin
- GORM
- Redis
- MySQL
- Eino / AG-UI 相关集成

前端：

- Next.js 16
- React 19
- TypeScript
- Tailwind CSS 4
- Semi UI
- CopilotKit / AG-UI Client

## 目录说明

```text
openIntern/
├── openIntern_backend/      # Go 服务
├── openIntern_forentend/    # Next.js 前端（目录名保持现状拼写）
├── go/                      # 本地 AG-UI Go SDK
├── scripts/                 # 启动和辅助脚本
└── 开发文档/                 # 设计和实现文档
```

## 运行前准备

本地建议先准备这些依赖：

- Go `1.25.5` 或兼容版本
- Node.js `20+`
- `pnpm`
- MySQL
- Redis
- Docker / Docker Compose（用于托管 OpenViking）
- 可选：Sandbox 服务，默认读取 `http://127.0.0.1:8081`

## 配置

### 后端

后端默认从 [openIntern_backend/config.yaml](/Users/fqc/project/agent/openIntern/openIntern_backend/config.yaml) 读取配置，当前代码会初始化：

- MySQL
- Redis
- JWT
- COS 文件存储
- LLM / Summary LLM
- 上下文压缩
- APMPlus

常用配置项包括：

```yaml
port: ":8080"
mysql:
  dsn: "root:root@tcp(127.0.0.1:3306)/open_intern?charset=utf8mb4&parseTime=True&loc=Local"
redis:
  addr: "127.0.0.1:6379"
tools:
  sandbox:
    url: "http://127.0.0.1:8081"
  openviking:
    base_url: "http://127.0.0.1:1933"
```

说明：

- OpenViking 的连接参数仍然由 [openIntern_backend/config.yaml](/Users/fqc/project/agent/openIntern/openIntern_backend/config.yaml) 中的 `tools.openviking` 提供，供后端业务能力调用。
- OpenViking 的服务启动、停止和内部参数管理不再由 openIntern 前后端负责。

安全说明：

- 仓库协作约定要求 `config.yaml` 中的密钥和凭证禁止提交。
- 当前 [openIntern_backend/config.yaml](/Users/fqc/project/agent/openIntern/openIntern_backend/config.yaml) 含有敏感字段，继续协作前应尽快改为本地私有配置并轮换已有密钥。

### 前端

前端本地环境变量文件为 [openIntern_forentend/.env.local](/Users/fqc/project/agent/openIntern/openIntern_forentend/.env.local)。

当前代码至少依赖以下变量：

```bash
API_BASE_URL=http://127.0.0.1:8080
NEXT_PUBLIC_API_BASE_URL=http://127.0.0.1:8080
```

说明：

- `/api/backend/*` 会代理到 `API_BASE_URL`
- `/api/copilotkit/*` 会使用 `API_BASE_URL` 或 `NEXT_PUBLIC_API_BASE_URL`

## 本地启动

1. 启动后端

```bash
cd openIntern_backend
go run main.go
```

2. 启动前端

```bash
cd openIntern_forentend
pnpm install
pnpm dev
```

3. 如需 OpenViking，请先通过仓库根目录的 Docker Compose 启动对应容器

```bash
docker compose up -d openviking
```

默认访问地址：

- 前端：`http://127.0.0.1:3000`
- 后端：`http://127.0.0.1:8080`
- OpenViking：`http://127.0.0.1:1933`
- Sandbox：`http://127.0.0.1:8081`

## 开发说明

- 前端首页会重定向到 `/chat`
- 前端通过本地存储保存 `token` 和 `user`
- 后端 API 主要位于 `/v1/*`
- 后端 `go.mod` 中将 AG-UI Go SDK `replace` 到仓库内的 [go](/Users/fqc/project/agent/openIntern/go)

## 常用开发命令

前端：

```bash
cd openIntern_forentend
pnpm dev
pnpm build
```

后端：

```bash
cd openIntern_backend
go run main.go
```

Promptfoo 评测：

```bash
cd evals/promptfoo
npm run test
npm run eval
```

评测接入说明见 [evals/promptfoo/README.md](/Users/fqc/project/agent/openIntern/evals/promptfoo/README.md)。

## 协作约定摘录

结合仓库内 [AGENTS.md](/Users/fqc/project/agent/openIntern/AGENTS.md)，开发时建议遵守：

- 仅修改与任务直接相关的文件
- 涉及接口、数据结构或行为变化时，同步更新文档或注释
- 新增代码需要添加注释
- 不要将密钥、凭证提交到仓库
- 未经允许，不执行耗时较长的 `pnpm lint`
- 未经允许，不执行 `go test`

## 备注

- `开发文档` 目录中部分设计文档可能已过期，排查问题时请以当前代码实现为准。
- `openIntern_forentend` 目录名中的 `forentend` 拼写为仓库现状，README 保持一致，不做额外修正。
