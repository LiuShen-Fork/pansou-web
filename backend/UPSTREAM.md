# 上游后端来源与同步说明

本目录是 [fish2018/pansou](https://github.com/fish2018/pansou) 的后端源码，以
**git subtree** 形式引入本仓库，方便前后端同仓构建与部署。

## 当前基线

- 上游仓库：`https://github.com/fish2018/pansou`
- 引入方式：`git subtree add --prefix=backend --squash`
- 当前基线：`221665fb2c9279f22af4684ea2d8c0c331eac1cc`（2026-10-12，新增插件 488run）

## 本仓库对上游代码的改动

为了在不破坏上游结构的前提下加入账户体系与后台，改动集中在以下位置：

| 文件 | 改动 |
| --- | --- |
| `main.go` | 启动时初始化 `pansou/admin` 数据库并注册后台 |
| `api/router.go` | 挂载 `/api/auth/*`、`/api/admin/*`、`/api/web/search`，并加入访问控制中间件 |
| `go.mod` / `go.sum` | 新增 SQLite、bcrypt、OAuth2 依赖 |
| `admin/` | **新增目录**，账户、会话、OAuth、SMTP、设置、配额、管理接口全部在此 |

其余上游文件尽量保持原样，降低同步冲突面。

## 同步上游

```bash
bash scripts/sync-backend.sh          # 同步 main
bash scripts/sync-backend.sh v1.2.3   # 同步指定 tag/commit
```

同步后请检查：

1. `backend/admin/` 未被上游覆盖；
2. `api/router.go` 中 `admin.RegisterRoutes` 与 `admin.AccessGate` 仍然存在；
3. `main.go` 中 `admin.Init()` 仍然存在；
4. 运行 `bash scripts/check.sh`。

GitHub Actions 中的 `sync-backend.yml` 会定期执行同样的同步并创建 PR，
冲突时不会自动合并，需人工处理。
