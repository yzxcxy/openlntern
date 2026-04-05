# Promptfoo Evals

这个目录提供 openIntern 的最小 `promptfoo` 评测接入，评测层与前后端运行时隔离，不改业务代码。

## 目录

- `promptfooconfig.yaml`：Promptfoo 入口配置
- `providers/openintern-chat.mjs`：自定义 Promptfoo provider，负责登录和请求 `/v1/chat/sse`
- `src/`：可复用的登录、请求体构造、AG-UI SSE 解析模块
- `tests/`：Node 内置测试，覆盖核心评测辅助逻辑

## 使用

1. 启动 openIntern 后端，确保 `POST /v1/auth/login` 和 `POST /v1/chat/sse` 可访问。
2. 复制 `.env.promptfoo.example` 为 `.env.promptfoo.local` 并填入本地登录凭证、模型提供商 ID、模型 ID。
3. 运行本地测试：

```bash
cd /Users/fqc/project/agent/openIntern/evals/promptfoo
npm run test
```

4. 运行评测：

```bash
cd /Users/fqc/project/agent/openIntern/evals/promptfoo
npm run eval
```

5. 查看结果：

```bash
cd /Users/fqc/project/agent/openIntern/evals/promptfoo
npm run view
```

## 可选配置

- `OPENINTERN_CONVERSATION_MODE=chat|agent`
- `OPENINTERN_SELECTED_AGENT_ID=<enabled-agent-id>`：仅在 `agent` 模式下需要
- 在 `promptfooconfig.yaml` 的 `tests` 中继续追加更多 case

## 说明

- 当前默认使用自定义 JS provider，而不是直接写 HTTP target 配置。这样可以复用本地登录逻辑，并稳定解析 AG-UI SSE 文本输出。
- `npx promptfoo@latest` 首次执行可能会下载依赖，因此需要可用网络。
