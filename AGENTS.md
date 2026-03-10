# openIntern 协作说明

## 1) 仓库结构
- `openIntern_backend`：Go 后端服务（Gin + GORM + Redis + Milvus）。
- `openIntern_forentend`：Next.js 前端（TypeScript，目录名按仓库现状保留 `forentend` 拼写）。
- `go`：本地 AG-UI Go SDK，后端 `go.mod` 里通过 `replace` 指向该目录。
- `scripts`：项目脚本。
- `开发文档`：设计与实现文档（可能包含错误的设计）。

## 2) 环境与配置
- 后端默认读取 `openIntern_backend/config.yaml`。
- 前端本地环境变量使用 `openIntern_forentend/.env.local`。
- `config.yaml`、`.env.local` 中的密钥和凭证禁止提交到仓库。

## 3) 协作约定
- 仅修改与任务直接相关的文件，避免顺手重构无关模块。
- 变更涉及接口、数据结构或行为时，同步更新文档或注释。
- 新增代码需要添加注释
- 提交前至少说明：
  - 改了什么；
  - 为什么这么改；
  - 如何验证（命令与结果）。
- 不要随便加go 的test文件
- 出现问题的时候，不要直接使用一种兼容错误的方式去修改，这样子只会隐藏问题

## 测试约定

- 不要执行`pnpm lint` 之类耗时很长的命令
- 没有允许，不允许执行go test