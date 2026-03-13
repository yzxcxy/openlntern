# 启动All in sandbox

项目位置：[sandbox](https://github.com/agent-infra/sandbox)

```bash
docker run -d --security-opt seccomp=unconfined --name=sandbox  -p 8081:8080 enterprise-public-cn-beijing.cr.volces.com/vefaas-public/all-in-one-sandbox:latest
```

MCP endpoint 固定为：

```text
http://127.0.0.1:8081/mcp
```
