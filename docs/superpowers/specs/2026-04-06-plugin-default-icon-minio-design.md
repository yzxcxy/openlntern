# Plugin Default Icon MinIO Design

## Goal

将项目根目录提供的插件默认图片收敛到后端资源目录，并让所有新建插件默认使用 MinIO 中的固定公共对象。

## Current State

- 新建插件默认图标来自 [openIntern_backend/config.yaml](/Users/fqc/project/agent/openIntern/openIntern_backend/config.yaml) 的 `plugin.default_icon_url`。
- 现有配置仍指向外部 COS 链接，不属于仓库可控资源。
- 新环境初始化脚本 [init-dev-data.sh](/Users/fqc/project/agent/openIntern/scripts/init-dev-data.sh) 只负责建 bucket 和默认账号，不会预热默认插件图标对象。

## Decision

1. 将默认图片移入后端资源目录 `openIntern_backend/assets/plugin/default-icon.jpg`。
2. 将默认插件图标配置改为稳定对象 key `public/plugin/icon/default-plugin.jpg`。
3. 在新环境初始化脚本中，确保该对象不存在时才上传到 MinIO，保持幂等。
4. 插件与 Agent 的 `public/...` 资源访问统一走后端资产代理 `/v1/assets/...`，不再直连 MinIO URL，避免私有 bucket 下出现 `403`。

## Why This Design

- 复用现有插件图标解析逻辑，不改接口协议。
- 避免默认图标依赖仓库外部对象存储地址。
- 新项目初始化时一次性写入，排障路径清晰，不把副作用塞进服务启动过程。
- 对前端暴露稳定应用地址，不依赖 MinIO bucket 的匿名读策略。

## Files

- 新增资源文件：`openIntern_backend/assets/plugin/default-icon.jpg`
- 修改配置：`openIntern_backend/config.yaml`
- 修改初始化脚本：`scripts/init-dev-data.sh`
- 修改说明文档：`docs/local-development.md`

## Verification

- `bash -n scripts/init-dev-data.sh`
- 检查 `openIntern_backend/config.yaml` 中 `plugin.default_icon_url` 是否为 `public/plugin/icon/default-plugin.jpg`
- 新环境执行 `./scripts/init-dev-data.sh` 后，通过 MinIO 或 `mc stat` 确认对象 `public/plugin/icon/default-plugin.jpg` 已存在
