# 清羽盘搜

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
>
> **忘记管理员密码**：使用 `PANSOU_RESET_ADMIN=1` 启动一次后端/容器，会把第一个管理员重置为默认密码并强制改密。
> 也可以删除数据卷中的 `pansou-admin.db` 重新初始化（会清空账户数据）。

## 账户与权限

登录页与后台支持以下组合，均可在「管理后台 → 认证与站点」中开关：

| 功能 | 说明 |
| --- | --- |
| 用户名/邮箱 + 密码登录 | 管理员始终可以密码登录，避免关闭普通登录后把自己锁死 |
| 邮箱注册 | 开启后强制 SMTP 邮箱验证码验证；未配置 SMTP 时无法保存该开关 |
| OAuth2 登录 | 已绑定第三方账号的用户可登录 |
| OAuth2 注册 | 第三方账号首次登录时自动创建本地用户 |
| 个人 API 令牌 | 用户中心最多创建 10 个 `ps_...` 令牌；明文只显示一次，所有令牌共享账号额度 |
| 每日搜索配额 | 全局默认值 + 用户级覆盖；可额外设置“仅 API”的每日额度 |
| 匿名搜索防护 | 每分钟限流 + 单 IP 每日额度 + 全站每日额度，均可独立配置 |
| 网页搜索凭证 | 匿名 `/api/web/search` 需同源页面签发的短期凭证，不能直接当公开 API 调用 |
| API 必须登录 | 开启后 `/api/search`、链接检测与插件 Web 接口需要会话或令牌 |
| 前端直接搜索 | 关闭后匿名用户不能使用网页搜索，必须登录 |
| 管理员唯一 | 数据库层强制最多一个管理员，接口不能新增、提升或删除管理员 |

### OAuth2 服务商

管理后台内置 GitHub、Google、Linux DO 与通用 OIDC 预设。回调地址统一为：

```
https://你的域名/api/auth/oauth/callback
```

请在对应服务商后台登记该地址，并填写 Client ID / Client Secret。

### API 使用

```bash
# 第三方应用 / 脚本：使用个人 API 令牌（推荐）
curl -H "Authorization: Bearer ps_xxxxxxxx" \
     "http://localhost/api/search?kw=仙逆"
```

`/api/search` 的请求参数与响应结构保持与上游 PanSou 一致，可直接接入视频源、机器人等应用：

- GET 查询参数：`kw`、`channels`、`conc`、`refresh`、`res`、`src`、`plugins`、`cloud_types`、`ext`、`filter`
- POST JSON 字段：`kw`、`channels`、`conc`、`refresh`、`res`、`src`、`plugins`、`cloud_types`、`ext`、`filter`
- 成功响应：`{"code":0,"message":"success","data":{"total":…,"results":[…],"merged_by_type":{…}}}`
- 令牌可放在 `Authorization: Bearer ps_…` 或 `X-API-Token: ps_…`

与上游的差异只有访问控制：默认 `API 必须登录`，未携带有效令牌会返回 `401`；超出每日额度返回 `429`（`code` 为 `API_DAILY_LIMIT_REACHED` 等）。`/api/check/links` 同样支持这套令牌。

网页端匿名搜索走 `/api/web/search`。它不会作为公开 API 开放：页面会先向
`/api/web/token` 换取与 IP、User-Agent 绑定的短期凭证，搜索时通过
`X-Web-Token` 携带；后端同时校验同源来源、每分钟频率、单 IP 每日额度与全站每日额度。
第三方应用请使用上面的 API 令牌方式，而不是绕过页面直接调用网页接口。

用户可在「用户中心 → API 令牌」创建多个命名令牌，明文只在创建时展示一次；
删除或重建后旧令牌立即失效。所有令牌共享同一账号的每日额度，管理员还可为
单个用户设置独立的“每日 API 额度”。

### 站点外观与公告

「管理后台 → 认证与站点 → 站点信息」支持自定义站点名称、Slogan、描述与图标。
图标可以填写外链、站内路径，或直接上传图片（≤ 512KB，保存为 Data URL），
会同步更新页头、登录框与浏览器标签页图标。

站点公告支持 Markdown（标题、列表、粗体、链接、引用、代码块等），
访问首页时以弹窗展示；同一公告 5 分钟内不会重复打扰，修改内容后会重新提示。
前端右上角提供浅色 / 深色 / 跟随系统三种主题，选择会保存在本地。

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

# 终端 1：启动后端（默认 :8888，自动加载上游频道/插件默认值）
npm run dev:backend
# 等价于 bash scripts/dev-backend.sh

# 终端 2：启动前端（默认 :3000，自动代理 /api 到 :8888）
npm run dev
```

打开 <http://localhost:3000>，首次登录 `admin / 123456`，会要求立即修改密码。
本地数据位于 `data/pansou-admin.db`，删除该文件即可重置本地账户。

一键校验（同步上游默认值 + 前端构建 + 后端测试/编译）：

```bash
bash scripts/check.sh
```

### 关键环境变量（可选）

代理、缓存、并发、网页搜索签名与上游同步地址都已经可以在「管理后台 → 运行参数 / 上游自动同步」中维护，**正常部署不再需要写这些环境变量**。环境变量只在首次启动或后台没有保存记录时作为默认值。

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `PORT` | `8888` | 后端监听端口（容器内） |
| `ADMIN_DB_PATH` | `data/pansou-admin.db` | 账户数据库路径；镜像内为 `/app/data/pansou-admin.db` |
| `CHANNELS` | 上游快照 | 显式覆盖 TG 频道列表；设置后会阻止重启时加载自动同步结果，一般不要设置 |
| `ENABLED_PLUGINS` | 上游快照 | 显式覆盖启用插件，同上 |
| `PROXY` / `NO_PROXY` | 无 | 后台「运行参数」未保存代理时使用的默认值 |
| `WEB_TOKEN_SECRET` | 随机生成 | 后台未保存签名密钥时使用；多副本部署应统一 |
| `CACHE_ENABLED` / `CACHE_TTL` | `true` / `60` | 后台未保存缓存设置时使用 |
| `CONCURRENCY` / `OUTBOUND_MAX_CONCURRENCY` | 自动 / `128` | 后台未保存并发设置时使用 |

### 运行参数（管理后台）

「管理后台 → 运行参数」中的值写入 `pansou-admin.db`，优先级高于环境变量：

| 参数 | 生效方式 |
| --- | --- |
| 出站代理（http / https / socks5） | 重启容器 |
| 缓存开关 | 重启容器 |
| 缓存 TTL | 立即生效（只影响新写入） |
| 默认搜索并发 | 立即生效；填 0 表示自动计算 |
| 出口并发上限 | 重启容器 |
| 网页搜索签名密钥 | 立即生效，现有网页凭证失效 |
| 上游 Dockerfile 地址 / 默认值输出路径 | 下次同步时生效 |

上游 Dockerfile 地址与默认值输出路径在「上游自动同步」页面维护。输出路径必须是容器内可写的绝对路径；`start.sh` 只会在重启时自动加载默认位置 `/app/data/upstream-defaults.env`，非默认路径需要自行确保启动时可读取。

完整的上游运行参数（超时、插件配置等）仍与 [pansou](https://github.com/fish2018/pansou) 一致。

## 上游自动更新（重点）

过去 TG 频道与插件列表写死在 Dockerfile 里，上游更新后容易长期遗漏。现在有三层保障：

1. **构建时**：`scripts/sync-upstream-defaults.mjs` 读取上游 `Dockerfile` 的
   `CHANNELS` / `ENABLED_PLUGINS`，生成 `deploy/upstream-defaults.env` 并打包进镜像。
   抓取失败时使用仓库内已提交的快照，不会中断构建。
2. **运行时（管理后台）**：默认每 24 小时抓取一次上游 Dockerfile。
   - GitHub raw 不可达时自动回退到 jsDelivr；也可用 `UPSTREAM_DOCKERFILE_URL` 指向自己的镜像；
   - **TG 频道与插件列表都会热更新**，无需重启；
   - 启用列表同时写入 `/app/data/upstream-defaults.env`，容器重启后保持一致；
   - 可在「管理后台 → 上游自动同步」修改仓库/分支/间隔或点击“立即同步”，
     页面会显示上次检查、成功时间、上游 revision、失败原因与是否需要重启；
   - 极少数环境无法注入插件热重载时，页面会提示重启容器。
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
