#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKEND_DIR="${ROOT_DIR}/openIntern_backend"

DEFAULT_USERNAME="${OPENINTERN_INIT_USERNAME:-admin}"
DEFAULT_EMAIL="${OPENINTERN_INIT_EMAIL:-admin@example.com}"
DEFAULT_PASSWORD="${OPENINTERN_INIT_PASSWORD:-admin123456}"
DEFAULT_PHONE="${OPENINTERN_INIT_PHONE:-}"
MINIO_ENDPOINT="${OPENINTERN_MINIO_ENDPOINT:-http://127.0.0.1:9000}"
MINIO_ACCESS_KEY="${OPENINTERN_MINIO_ACCESS_KEY:-minioadmin}"
MINIO_SECRET_KEY="${OPENINTERN_MINIO_SECRET_KEY:-minioadmin}"
MINIO_BUCKET="${OPENINTERN_MINIO_BUCKET:-open-intern}"
DEFAULT_PLUGIN_ICON_PATH="${ROOT_DIR}/openIntern_backend/assets/plugin/default-icon.jpg"
DEFAULT_PLUGIN_ICON_OBJECT_KEY="${OPENINTERN_DEFAULT_PLUGIN_ICON_OBJECT_KEY:-public/plugin/icon/default-plugin.jpg}"

ensure_container_running() {
  local service="$1"
  local container_name="$2"

  # 已存在的固定命名容器直接启动，避免 compose 在脏环境里因重名容器失败。
  if docker inspect "${container_name}" >/dev/null 2>&1; then
    docker start "${container_name}" >/dev/null 2>&1 || true
    return 0
  fi

  docker compose up -d "${service}"
}

wait_for_http_ok() {
  local url="$1"
  local name="$2"
  local retries="${3:-60}"

  # 启动依赖后轮询健康接口，避免初始化命令在服务尚未可用时直接失败。
  for ((i = 1; i <= retries; i++)); do
    if curl -fsS "${url}" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done

  echo "等待 ${name} 就绪超时: ${url}" >&2
  return 1
}

wait_for_mysql_ready() {
  local retries="${1:-60}"

  # MySQL 用容器内 ping 结果判断是否可接收连接。
  for ((i = 1; i <= retries; i++)); do
    if docker exec openintern-mysql mysqladmin ping -h 127.0.0.1 -proot >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done

  echo "等待 MySQL 就绪超时" >&2
  return 1
}

wait_for_container_healthy() {
  local container_name="$1"
  local name="$2"
  local retries="${3:-60}"

  # 优先复用容器健康检查结果，避免服务对宿主机探测方式不一致导致误判。
  for ((i = 1; i <= retries; i++)); do
    local health_status
    health_status="$(docker inspect "${container_name}" --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}' 2>/dev/null || true)"
    if [[ "${health_status}" == "healthy" ]]; then
      return 0
    fi
    sleep 1
  done

  echo "等待 ${name} 健康检查通过超时" >&2
  return 1
}

ensure_minio_bucket() {
  # 通过独立 mc 客户端创建 bucket，避免依赖宿主机额外安装 MinIO 工具。
  docker run --rm --network host --entrypoint /bin/sh minio/mc -c \
    "mc alias set local ${MINIO_ENDPOINT} ${MINIO_ACCESS_KEY} ${MINIO_SECRET_KEY} >/dev/null && mc mb local/${MINIO_BUCKET} --ignore-existing >/dev/null"
}

ensure_minio_object_from_file() {
  local object_key="$1"
  local file_path="$2"

  if [[ ! -f "${file_path}" ]]; then
    echo "初始化 MinIO 对象失败，文件不存在: ${file_path}" >&2
    return 1
  fi

  local mounted_file_path="${file_path#"${ROOT_DIR}/"}"
  if [[ "${mounted_file_path}" == "${file_path}" ]]; then
    echo "初始化 MinIO 对象失败，文件路径不在项目目录内: ${file_path}" >&2
    return 1
  fi

  # 公共默认资源使用固定 object key，重复执行初始化脚本时只在对象不存在时上传。
  docker run --rm --network host -v "${ROOT_DIR}:/work:ro" --entrypoint /bin/sh minio/mc -c \
    "mc alias set local ${MINIO_ENDPOINT} ${MINIO_ACCESS_KEY} ${MINIO_SECRET_KEY} >/dev/null && if mc stat local/${MINIO_BUCKET}/${object_key} >/dev/null 2>&1; then exit 0; fi && mc cp /work/${mounted_file_path} local/${MINIO_BUCKET}/${object_key} >/dev/null"
}

echo "启动外部依赖..."
cd "${ROOT_DIR}"
ensure_container_running "mysql" "openintern-mysql"
ensure_container_running "redis" "openintern-redis"
ensure_container_running "minio" "openintern-minio"
ensure_container_running "openviking" "openintern-openviking"

echo "等待 MySQL、MinIO、OpenViking 就绪..."
wait_for_mysql_ready
wait_for_http_ok "http://127.0.0.1:9000/minio/health/live" "MinIO"
wait_for_container_healthy "openintern-openviking" "OpenViking"

echo "初始化 MinIO bucket..."
ensure_minio_bucket

echo "初始化默认插件图标..."
ensure_minio_object_from_file "${DEFAULT_PLUGIN_ICON_OBJECT_KEY}" "${DEFAULT_PLUGIN_ICON_PATH}"

echo "初始化默认账号..."
cd "${BACKEND_DIR}"
go run ./cmd/initdevdata \
  -config config.yaml \
  -username "${DEFAULT_USERNAME}" \
  -email "${DEFAULT_EMAIL}" \
  -password "${DEFAULT_PASSWORD}" \
  -phone "${DEFAULT_PHONE}"

echo "初始化完成。"
echo "默认账号: ${DEFAULT_USERNAME} / ${DEFAULT_EMAIL}"
