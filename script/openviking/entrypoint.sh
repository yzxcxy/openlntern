#!/bin/bash
set -e

# 生成配置文件
CONFIG_FILE="/root/.openviking/ov.conf"

if [ ! -f "$CONFIG_FILE" ]; then
    echo "正在生成 OpenViking 配置文件..."
    
    # 如果环境变量未设置，使用默认配置（本地模式）
    if [ -z "$VLM_API_KEY" ]; then
        echo "警告: 未设置 VLM_API_KEY，将使用默认本地配置"
    fi

    cat > "$CONFIG_FILE" <<EOF
{
  "embedding": {
    "dense": {
      "api_base": "${EMBEDDING_API_BASE:-https://ark.cn-beijing.volces.com/api/v3}",
      "api_key": "${EMBEDDING_API_KEY:-}",
      "provider": "${EMBEDDING_PROVIDER:-volcengine}",
      "dimension": ${EMBEDDING_DIMENSION:-1024},
      "model": "${EMBEDDING_MODEL:-doubao-embedding-vision-250615}"
    }
  },
  "vlm": {
    "api_base": "${VLM_API_BASE:-https://ark.cn-beijing.volces.com/api/v3}",
    "api_key": "${VLM_API_KEY:-}",
    "provider": "${VLM_PROVIDER:-volcengine}",
    "model": "${VLM_MODEL:-doubao-seed-1-8-251228}"
  },
  "server": {
    "host": "${OV_HOST:-0.0.0.0}",
    "port": ${OV_PORT:-8080},
    "data_path": "${OV_DATA_PATH:-/app/data}"
  }
}
EOF
    echo "配置文件已生成: $CONFIG_FILE"
fi

# 验证配置
echo "当前配置:"
cat "$CONFIG_FILE"

# 启动 OpenViking HTTP 服务
echo "正在启动 OpenViking Server..."
exec python -c "
import openviking as ov
import os

# 创建并启动 HTTP 服务
# 注意：这里假设 OpenViking 提供了 HTTP server 启动方式
# 实际命令可能需要根据官方文档调整
from openviking.server import start_server

start_server(
    config_path='$CONFIG_FILE',
    data_path='${OV_DATA_PATH:-/app/data}',
    host='${OV_HOST:-0.0.0.0}',
    port=${OV_PORT:-8080}
)
"