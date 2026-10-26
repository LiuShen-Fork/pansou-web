#!/usr/bin/env bash
# 本地启动后端开发服务（默认 http://localhost:8888）。
# 会自动读取 deploy/upstream-defaults.env 的频道/插件默认值。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

mkdir -p data/cache

if [ -f deploy/upstream-defaults.env ]; then
  # shellcheck disable=SC1091
  . deploy/upstream-defaults.env
fi

export CHANNELS="${CHANNELS:-${UPSTREAM_CHANNELS:-tgsearchers7}}"
export ENABLED_PLUGINS="${ENABLED_PLUGINS:-${UPSTREAM_ENABLED_PLUGINS:-}}"
export PORT="${PORT:-8888}"
export ADMIN_DB_PATH="${ADMIN_DB_PATH:-$ROOT/data/pansou-admin.db}"
export CACHE_PATH="${CACHE_PATH:-$ROOT/data/cache}"
export ASYNC_PLUGIN_ENABLED="${ASYNC_PLUGIN_ENABLED:-true}"

# 上游 go.mod 固定 Go 1.25，sonic 也不支持 386；本机若不是 1.25/amd64 时自动切换。
export GOTOOLCHAIN="${GOTOOLCHAIN:-go1.25.0}"
export GOSUMDB="${GOSUMDB:-sum.golang.org}"
if [ -z "${GOARCH:-}" ]; then
  export GOARCH=amd64
fi

echo "=========================================="
echo " PanSou 后端开发服务"
echo " 地址:   http://localhost:${PORT}"
echo " 数据库: ${ADMIN_DB_PATH}"
echo " 频道:   ${CHANNELS:0:60}..."
echo " 插件:   ${ENABLED_PLUGINS:0:80}..."
echo "=========================================="

cd backend
exec go run .
