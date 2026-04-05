# AIO Sandbox 本地调试说明

项目位置：[sandbox](https://github.com/agent-infra/sandbox)

当前 openIntern 已改成“每个用户一个 sandbox 实例”的本地开发模式：

- 不再手工启动一个固定名、固定端口的共享 sandbox 容器
- 后端会在用户首次调用 sandbox 相关能力时，自动执行 `docker run`
- 同一用户后续请求会复用同一个容器
- 空闲超时后，后端后台会自动回收该容器
- 后端重启后，如容器仍在，会自动认领并复用它

镜像默认仍使用：

```bash
enterprise-public-cn-beijing.cr.volces.com/vefaas-public/all-in-one-sandbox:latest
```

常用排查命令：

```bash
docker ps --format 'table {{.Names}}\t{{.Ports}}\t{{.Status}}'
docker inspect <container_name> --format '{{json .Config.Labels}}'
docker rm -f <container_name>
```

手工删除某个用户容器后，下一次相关请求会自动创建新的 sandbox 实例。
