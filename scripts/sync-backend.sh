#!/usr/bin/env bash
# 同步上游 fish2018/pansou 后端源码到 backend/。
#
# backend/ 以 git subtree 的形式引入，本仓库对其做的认证/后台改造会作为普通提交
# 保留下来；同步本质上是一次三方合并，冲突会明确停在 backend/ 下，便于逐个人工处理。
#
# 用法（在仓库根目录执行）：
#   bash scripts/sync-backend.sh            # 同步上游 main
#   bash scripts/sync-backend.sh v1.2.3     # 同步指定 ref
#
# 同步完成后务必：
#   1. 处理冲突，确认 backend/admin 与 api/router.go、main.go 的改造仍然生效；
#   2. 运行 bash scripts/check.sh；
#   3. 提交并推送。
set -euo pipefail

REF="${1:-main}"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

echo "[sync-backend] 同步 fish2018/pansou@${REF} -> backend/"
git subtree pull \
  --prefix=backend \
  https://github.com/fish2018/pansou.git \
  "$REF" \
  --squash \
  -m "chore: 同步上游 pansou 后端（${REF}）"

echo "[sync-backend] 更新上游默认频道/插件快照"
node scripts/sync-upstream-defaults.mjs

echo "[sync-backend] 完成。请运行 bash scripts/check.sh 做一次完整校验。"
