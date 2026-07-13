# PanSou Web

🚀 镜像集成Pansou前后端，开箱即用。

[![Multi-Arch](https://img.shields.io/badge/arch-amd64%20%7C%20arm64-blue)](https://github.com/fish2018/pansou-web)

## 快速开始

### 一键启动

```bash
docker run -d --name pansou -p 80:80 ghcr.io/fish2018/pansou-web
```

访问：http://localhost

### Docker Compose（推荐）

```bash
# 下载配置文件
curl -o docker-compose.yml https://raw.githubusercontent.com/fish2018/pansou-web/main/docker-compose.yml

# 启动服务
docker-compose up -d

# 查看日志
docker-compose logs -f
```

## 支持架构

镜像支持以下CPU架构：
- `linux/amd64` - Intel/AMD 64位处理器
- `linux/arm64` - ARM 64位处理器

Docker会自动选择适合您系统的架构版本。

## 环境变量

### 基础配置

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `DOMAIN` | `localhost` | 访问域名 |
| `ENABLED_PLUGINS` | `labi,zhizhen,shandian,duoduo,muou,wanou` | 启用的搜索插件（逗号分隔） |

> 🔌 **重要变更**: 从当前版本开始，必须通过 `ENABLED_PLUGINS` 显式指定要启用的插件，否则不会启用任何插件。

### Telegram频道配置

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `CHANNELS` | **已内置数十个频道，开箱即用，无需配置** | Telegram频道列表（逗号分隔） |

> 💡 **自定义频道**: 如需自定义，使用 `CHANNELS` 环境变量覆盖默认配置。

### 代理配置

| 变量名 | 默认值 | 说明 | 示例 |
|--------|--------|------|------|
| `PROXY` | 无 | 代理服务器地址 | `socks5://xxx.xxx.xxx.xxx:7897` |

**支持的代理类型：**
- SOCKS5代理: `socks5://xxx.xxx.xxx.xxx:7897`

**使用场景：**
- 访问被墙的Telegram频道
- 加速国外资源访问
- 企业内网代理

```bash
# 示例
-e PROXY=socks5://xxx.xxx.xxx.xxx:7897
```

### 性能配置

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `CACHE_ENABLED` | `true` | 是否启用缓存 |
| `CACHE_TTL` | `60` | 缓存过期时间（分） |
| `MAX_CONCURRENCY` | `200` | 最大并发数 |
| `MAX_PAGES` | `30` | 最大搜索页数 |

> ⚡ **性能优化**: 镜像已内置优化配置，通常无需调整。

### 认证配置（可选）

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `AUTH_ENABLED` | `false` | 是否启用认证功能 |
| `AUTH_USERS` | - | 用户账号配置，格式：`user1:pass1,user2:pass2` |
| `AUTH_TOKEN_EXPIRY` | `24` | Token有效期（小时） |
| `AUTH_JWT_SECRET` | 自动生成 | JWT签名密钥，建议手动设置 |

> 🔐 **安全认证**: 启用后，访问应用需要登录。适合需要访问控制的场景。

---

## 前端页面配置和访问控制说明

本仓库的 Web 页面主要提供搜索入口、搜索偏好配置、API 调试文档和部分账号型插件的管理入口。部署前建议先区分“服务端全局配置”和“浏览器本地配置”：

| 配置位置 | 保存位置 | 影响范围 |
|--------|--------|--------|
| Docker / Compose 环境变量 | 容器环境变量 | 全局生效，决定后端实际启用的插件、频道、缓存、代理和认证 |
| Web 页面“搜索配置” | 当前浏览器 `localStorage` | 只影响当前浏览器发起搜索时携带的频道、插件、网盘类型参数 |
| 账号型插件管理 | 浏览器本地记录 + 后端插件接口 | 用于 QQ 频道、观影、盘链、微博等需要登录态/账号数据的插件 |

### 页面里的“搜索配置”会改服务器吗？

不会。页面里的频道、插件、网盘类型选择会写入当前浏览器的 `localStorage`，用于后续搜索请求拼接 `channels`、`plugins`、`cloud_types` 参数。其他访问者看不到你的浏览器本地配置，也不会因此修改服务器上的 `ENABLED_PLUGINS`、`CHANNELS` 或 Docker 配置。

真正的全局可用范围仍以后端为准：

- `ENABLED_PLUGINS` 决定服务端允许启用的插件集合。
- `CHANNELS` 决定服务端默认 Telegram 频道集合。
- `CACHE_ENABLED`、`CACHE_TTL`、`MAX_CONCURRENCY`、`MAX_PAGES` 决定服务端缓存和搜索性能策略。
- `PROXY` 用于服务端访问 Telegram 或外部资源时走代理。

### 启用认证后限制哪些功能？

当 `AUTH_ENABLED=true` 时，前端会显示登录弹窗，并把登录得到的 token 放在请求头 `Authorization: Bearer <token>` 中。真正的安全边界在后端接口校验，不在前端按钮是否隐藏。

一般理解：

- 网页搜索会受后端认证限制；未登录时搜索接口应返回 401。
- API 搜索同样会受后端认证限制；直接调用 `/api/search` 也需要带 token。
- `/api/auth/login` 用于获取 token，`/api/auth/verify` 用于校验 token。
- 页面“搜索配置”即使能打开，也只是浏览器本地偏好；未登录用户不能绕过后端认证调用受保护接口。

公网部署建议启用认证，并使用强密码和固定的 `AUTH_JWT_SECRET`：

```bash
-e AUTH_ENABLED=true
-e AUTH_USERS=admin:MyStrongPassword
-e AUTH_TOKEN_EXPIRY=168
-e AUTH_JWT_SECRET=$(openssl rand -base64 32)
```

如果你希望整个站点完全私有，也可以在反向代理或面板层再加一层访问控制。

### 常用管理项怎么用？

- 搜索页：输入关键词后发起聚合搜索。前端会根据本地搜索配置优先搜索 Telegram、插件或两者。
- 搜索配置：选择当前浏览器本次要使用的 Telegram 频道、插件、网盘类型和链接检测开关。
- 导出配置：把当前选择的插件和频道导出成 `ENABLED_PLUGINS` / `CHANNELS`，方便写回 Docker 环境变量。
- API 文档：调试 `/api/search`、`/api/health`、`/api/auth/*` 等接口；启用认证后需要填写或使用当前登录 token。
- 账号管理：仅当后端启用了对应插件时显示，用于管理 QQ 频道、观影、盘链、微博等插件所需的账号或登录态。

---

## 🎯 常见配置场景

### 场景1：个人使用（最小配置）

```bash
docker run -d \
  --name pansou \
  -p 80:80 \
  -v pansou-data:/app/data \
  --restart unless-stopped \
  ghcr.io/fish2018/pansou-web:latest
```

### 场景2：公网服务（带域名和SSL）

```bash
# 1. 准备SSL证书
mkdir -p /opt/pansou/ssl
# 将证书放到 /opt/pansou/ssl/fullchain.pem 和 privkey.pem

# 2. 启动容器
docker run -d \
  --name pansou \
  -p 80:80 \
  -p 443:443 \
  -e DOMAIN=pansou.example.com \
  -v pansou-data:/app/data \
  -v /opt/pansou/ssl:/app/data/ssl:ro \
  --restart unless-stopped \
  ghcr.io/fish2018/pansou-web:latest
```

### 场景3：需要代理访问Telegram

```bash
docker run -d \
  --name pansou \
  -p 80:80 \
  -e PROXY=socks5://xxx.xxx.xxx.xxx:7897 \
  -v pansou-data:/app/data \
  --restart unless-stopped \
  ghcr.io/fish2018/pansou-web:latest
```

**注意：**
- 使用 `--network host` 以访问宿主机代理
- 或者将代理服务也容器化并使用 docker 网络

### 场景4：启用访问认证

```bash
docker run -d \
  --name pansou \
  -p 80:80 \
  -e AUTH_ENABLED=true \
  -e AUTH_USERS=admin:MySecretPass123,viewer:ViewOnly456 \
  -e AUTH_TOKEN_EXPIRY=168 \
  -e AUTH_JWT_SECRET=$(openssl rand -base64 32) \
  -v pansou-data:/app/data \
  --restart unless-stopped \
  ghcr.io/fish2018/pansou-web:latest
```

---

## 📁 数据目录说明

### 卷挂载

**推荐：使用命名卷（Docker管理）**
```bash
-v pansou-data:/app/data
```

**或：使用绑定挂载（指定宿主机路径）**
```bash
-v /opt/pansou/data:/app/data
```

### 目录结构详解

```
/app/data/
│
├── cache/                          # 缓存目录（约100MB-1GB）
│   ├── disk/                       # 磁盘缓存
│   │   ├── [hash].cache           # 搜索结果缓存
│   │   └── metadata.db            # 缓存元数据
│   └── qqpd_users/                # QQPD插件数据
│       └── [hash].json            # 用户配置和频道
│
├── logs/                           # 日志目录（建议定期清理）
│   ├── backend/                    # 后端日志
│   │   └── pansou.log             # 主日志文件
│   └── nginx/                      # Nginx日志
│       ├── access.log             # 访问日志
│       └── error.log              # 错误日志
│
└── ssl/                            # SSL证书目录
    ├── fullchain.pem              # 完整证书链
    └── privkey.pem                # 私钥
```
