# openIntern

openIntern 是一个面向智能体工作流的全栈项目，包含：
- `openIntern_backend`：Go 后端（Gin + GORM + Redis + Milvus）
- `openIntern_forentend`：Next.js 前端（目录名按仓库现状保留 `forentend` 拼写）
- `go`：本地 AG-UI Go SDK（后端通过 `replace` 引用）

## 仓库结构

```text
.
├── openIntern_backend      # 后端服务
├── openIntern_forentend    # 前端应用
├── go                      # 本地 AG-UI Go SDK
├── scripts                 # 启动与辅助脚本
└── 开发文档                # 设计/实现文档
```

## 技术栈

- 后端：Go 1.25.5、Gin、GORM、MySQL、Redis、Milvus
- 前端：Next.js 16、React 19、TypeScript、pnpm
- 其他：OpenViking（可选，和长期记忆/技能上下文相关）

## 环境准备

建议本地准备：
- Go `1.25.5`（与 `openIntern_backend/go.mod` 一致）
- Node.js `>=20`
- pnpm
- MySQL（建议 8.x）
- Redis
- Milvus
- Docker（可选，用于 sandbox）

## 配置说明

### 1) 后端配置 `openIntern_backend/config.yaml`

后端默认从 `openIntern_backend/config.yaml` 读取配置。

关键项说明：
- 必需：
  - `port`
  - `mysql.dsn`
  - `jwt.secret`
  - `embedding_llm.model` / `embedding_llm.api_key`（启动时会初始化）
  - `milvus.address` / `milvus.dimension` / `milvus.metric_type`（启动时会初始化）
  - `tools.sandbox.url`（Agent 初始化要求）
- 按需：
  - `cos`：四项都为空时视为未启用；若只填部分会报配置不完整
  - `tools.openviking`：不配置可启动，但长期记忆同步相关能力会降级
  - `llm` / `summary_llm`：聊天与标题等能力依赖

### 2) 前端配置 `openIntern_forentend/.env.local`

至少配置：

```bash
API_BASE_URL=http://localhost:8080
```

说明：
- 前端通过 `/api/backend/*` 代理到 `API_BASE_URL`
- CopilotKit 路由会优先读取 `API_BASE_URL`，其次 `NEXT_PUBLIC_API_BASE_URL`

### 3) 安全要求

- 禁止提交真实密钥、Token、凭证到仓库
- `config.yaml`、`.env.local` 应仅保留本地开发配置

## 启动方式

### 方式 A：手动启动

1. 启动依赖服务（MySQL / Redis / Milvus）
2. （可选）启动 sandbox：

```bash
docker run -d --security-opt seccomp=unconfined --name=sandbox -p 8081:8080 enterprise-public-cn-beijing.cr.volces.com/vefaas-public/all-in-one-sandbox:latest
```

3. 启动后端：

```bash
cd openIntern_backend
go run main.go
```

4. 启动前端：

```bash
cd openIntern_forentend
pnpm dev
```

5. 浏览器访问：
- 前端：`http://localhost:3000`
- 后端：`http://localhost:8080`

### 方式 B：使用仓库脚本一键启动

```bash
./scripts/start-dev-services.sh auto
```

常用参数：
- `auto|terminal|tmux`：启动模式
- `--with-openviking`：强制启动 `openviking-server`
- `--without-openviking`：不启动 `openviking-server`

示例：

```bash
./scripts/start-dev-services.sh tmux --without-openviking
```

## 快速验证

### 1) 后端注册/登录

```bash
# 注册
curl -X POST http://localhost:8080/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"username":"demo","email":"demo@example.com","password":"123456"}'

# 登录
curl -X POST http://localhost:8080/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"identifier":"demo@example.com","password":"123456"}'
```

返回 `token` 即表示鉴权链路可用。

### 2) 前端联通验证

- 打开 `http://localhost:3000/login`
- 完成登录后进入工作台页面

## 开发约定（简要）

- 仅修改与任务直接相关文件，避免无关重构
- 接口/数据结构/行为变更时同步更新文档
- 不提交密钥与凭证
- 默认不执行耗时检查命令（如 `pnpm lint`）与 `go test`

## 相关文档

- 项目协作说明：`AGENTS.md`
- 详细设计资料：`开发文档/`
- sandbox 说明：`openIntern_backend/script/setup_sandbox.md`

