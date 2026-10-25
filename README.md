# Qingyu PanSou

🚀 PanSou 前后端同仓维护版：Vue 3 前端 + Go 后端 + 账户体系 + 管理后台。

- 前端：`frontend/`（Vue 3 + Vite + Tailwind）
- 后端：`backend/`（以 git subtree 引入上游 [fish2018/pansou](https://github.com/fish2018/pansou)，并叠加账户/后台能力）
- 镜像：根目录多阶段 `Dockerfile` 一次构建前后端，支持 `linux/amd64` / `linux/arm64`
- 上游数据：构建时与运行时都会自动抓取上游的 TG 频道与启用插件列表，不再需要手工维护

## 快速开始

### Docker Compose（推荐）

```bash
curl -O https://raw.githubusercontent.com/willow-god/pansou-web/main/docker-compose.yml
docker compose up -d
```

访问 `http://localhost`。

### 直接运行镜像

```bash
docker run -d --name pansou \
  -p 80:80 \
  -v pansou-data:/app/data \
  --restart unless-stopped \
  ghcr.io/willow-god/pansou-web:latest
```

> 所有持久化数据（缓存、日志、SSL 证书、账户数据库）都在 `/app/data`，升级镜像不会丢失。

## 默认管理员

首次启动会自动创建：

| 账号 | 密码 |
| --- | --- |
| `admin` | `admin123456` |

登录后会强制要求修改密码，修改前只能访问“用户中心/修改密码”等必要接口。

> 🔐 公网部署前请务必修改默认密码，并在管理后台启用 HTTPS/反向代理。
> 忘记密码时可删除数据卷中的 `pansou-admin.db` 重新初始化（会清空账户数据）。

## 账户与权限

登录页与后台支持以下组合，均可在「管理后台 → 认证与站点」中开关：

| 功能 | 说明 |
| --- | --- |
| 用户名/邮箱 + 密码登录 | 管理员始终可以密码登录，避免关闭普通登录后把自己锁死 |
| 邮箱注册 | 开启后强制 SMTP 邮箱验证码验证；未配置 SMTP 时无法保存该开关 |
| OAuth2 登录 | 已绑定第三方账号的用户可登录 |
| OAuth2 注册 | 第三方账号首次登录时自动创建本地用户 |
| 个人 API 令牌 | 用户中心生成 `ps_...` 令牌，用于 `Authorization: Bearer <令牌>` |
| 每日搜索配额 | 全局默认值 + 用户级覆盖，0 表示不限 |
| API 必须登录 | 开启后 `/api/search`、链接检测与插件 Web 接口需要会话或令牌 |
| 前端直接搜索 | 关闭后匿名用户不能使用网页搜索，必须登录 |

### OAuth2 服务商

管理后台内置 GitHub、Google、Linux DO 与通用 OIDC 预设。回调地址统一为：

```
https://你的域名/api/auth/oauth/callback
```

请在对应服务商后台登记该地址，并填写 Client ID / Client Secret。

### API 使用

```bash
# 网页匿名搜索（受后台开关与匿名限流控制）
curl "http://localhost/api/web/search?kw=仙逆"

# 登录用户 / API 令牌搜索
curl -H "Authorization: Bearer ps_xxxxxxxx" \
     "http://localhost/api/search?kw=仙逆"
```

用户可在「用户中心 → API 令牌」重置令牌；旧令牌会立即失效。

## 目录结构

```text
.
├── frontend/                 # Vue 3 前端
│   ├── src/
│   │   ├── components/       # 搜索、结果、账户、管理后台组件
│   │   ├── stores/auth.ts    # 登录态与站点开关
│   │   └── api/auth.ts       # 账户/管理接口封装
│   └── package.json
├── backend/                  # 上游 pansou 后端 + 本仓库改造
│   ├── admin/                # 新增：账户、会话、OAuth、SMTP、设置、配额、上游同步
│   ├── api/                  # 已接入账户中间件与 /api/web/search
│   └── UPSTREAM.md           # 上游同步说明与改造清单
├── deploy/
│   └── upstream-defaults.env # 构建时抓取的上游频道/插件快照
├── scripts/
│   ├── sync-upstream-defaults.mjs
│   ├── sync-backend.sh
│   └── check.sh
├── Dockerfile
├── docker-compose.yml
└── start.sh
```

## 本地开发

要求：Node.js 22+、Go 1.25（本仓库 `go.mod` 固定 Go 1.25；`scripts/check.sh` 会自动切换工具链）。

```bash
# 安装前端依赖
npm --prefix frontend ci

# 启动后端（默认 :8888）
cd backend && go run .

# 另开终端启动前端（默认 :3000，自动代理 /api 到 :8888）
npm run dev
```

一键校验（同步上游默认值 + 前端构建 + 后端测试/编译）：

```bash
bash scripts/check.sh
```

### 关键环境变量

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `PORT` | `8888` | 后端监听端口 |
| `ADMIN_DB_PATH` | `data/pansou-admin.db` | 账户数据库路径；镜像内为 `/app/data/pansou-admin.db` |
| `UPSTREAM_DEFAULTS_OUTPUT` | `/app/data/upstream-defaults.env`（容器内） | 运行期同步写出的默认值文件 |
| `UPSTREAM_DOCKERFILE_URL` | 无 | 自定义上游 Dockerfile 镜像地址；不设置时依次尝试 GitHub raw 与 jsDelivr |
| `CHANNELS` | 上游快照 | 显式覆盖 TG 频道列表，逗号分隔 |
| `ENABLED_PLUGINS` | 上游快照 | 显式覆盖启用插件，逗号分隔 |
| `PROXY` | 无 | 访问 TG/外站使用的代理 |
| `CACHE_ENABLED` / `CACHE_TTL` | `true` / `60` | 缓存开关与分钟数 |

完整的上游运行参数（并发、超时、插件配置等）仍与 [pansou](https://github.com/fish2018/pansou) 一致。

## 上游自动更新（重点）

过去 TG 频道与插件列表写死在 Dockerfile 里，上游更新后容易长期遗漏。现在有三层保障：

1. **构建时**：`scripts/sync-upstream-defaults.mjs` 读取上游 `Dockerfile` 的
   `CHANNELS` / `ENABLED_PLUGINS`，生成 `deploy/upstream-defaults.env` 并打包进镜像。
   抓取失败时使用仓库内已提交的快照，不会中断构建。
2. **运行时（管理后台）**：默认每 24 小时抓取一次上游 Dockerfile。
   - GitHub raw 不可达时自动回退到 jsDelivr；也可用 `UPSTREAM_DOCKERFILE_URL` 指向自己的镜像；
   - **TG 频道立即热更新**，无需重启；
   - 插件列表写入 `/app/data/upstream-defaults.env`，重启后生效（新插件代码仍需更新镜像）；
   - 可在「管理后台 → 上游自动同步」修改仓库/分支/间隔或点击“立即同步”，
     页面会显示上次检查、成功时间、上游 revision、失败原因与是否需要重启。
3. **代码同步**：`scripts/sync-backend.sh` 使用 git subtree 拉取上游后端；
   GitHub Actions 每周自动执行并创建 PR（`.github/workflows/sync-backend.yml`），
   每天更新频道/插件快照（`sync-upstream-data.yml`）。

> 运行期同步不会执行上游代码，只是更新配置数据；插件代码的更新仍需重新构建镜像。

## 构建镜像

```bash
docker build -t pansou-web .
```

多阶段构建：

1. `node:22-alpine` 构建前端；
2. `golang:1.25-alpine` 构建后端（`CGO_ENABLED=0`）；
3. `nginx:alpine` 运行 nginx + Go 后端，`start.sh` 负责健康检查与自动重启。

GitHub Actions 在 release 或手动触发时构建并推送到 GHCR：
`.github/workflows/build-docker.yml`。

## 数据与备份

备份 `/app/data` 即可，重点内容：

- `pansou-admin.db`：用户、会话、设置、OAuth、SMTP、搜索日志；
- `upstream-defaults.env`：运行期同步的上游默认值；
- `cache/`、`logs/`、`ssl/`。

## 从旧版 `AUTH_USERS` 迁移

旧版基于环境变量的简单认证（`AUTH_ENABLED` / `AUTH_USERS`）已由账户体系取代。
升级后请：

1. 使用默认管理员登录；
2. 在「用户管理」中为原有用户创建账号；
3. 在「认证与站点」中按需开启邮箱注册 / OAuth；
4. 删除旧的 `AUTH_USERS` 配置，避免混淆。

## 致谢

- 后端与插件生态：[fish2018/pansou](https://github.com/fish2018/pansou)
- 原前端项目：[fish2018/pansou-web](https://github.com/fish2018/pansou-web)
