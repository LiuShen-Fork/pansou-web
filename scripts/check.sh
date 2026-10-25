#!/usr/bin/env bash
# 本地/CI 通用校验：前端构建 + 后端测试/编译。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

# 上游后端 go.mod 固定 go 1.25，且 sonic 不支持 32 位/过新的 Go。
# 本机若使用 386 架构或更高版本 Go，这里显式切到 go1.25 + amd64。
export GOTOOLCHAIN="${GOTOOLCHAIN:-go1.25.0}"
export GOSUMDB="${GOSUMDB:-sum.golang.org}"
if [ -z "${GOARCH:-}" ]; then
  export GOARCH=amd64
fi

echo "==> 同步上游默认数据（失败时使用已提交快照）"
node scripts/sync-upstream-defaults.mjs

echo "==> 前端依赖"
if [ ! -d frontend/node_modules ]; then
  npm --prefix frontend ci
fi

echo "==> 前端构建"
npm --prefix frontend run build

echo "==> 后端测试"
(cd backend && go test ./...)

echo "==> 后端编译"
(cd backend && go build -trimpath -o ../bin/pansou .)

echo "==> 全部通过"
