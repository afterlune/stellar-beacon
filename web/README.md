# Web 前端

本目录包含三个独立的 Vue 应用：

- `blog`：Vue 3 博客前台
- `admin`：Vue 2 管理后台（迁移期间的稳定回滚版本）
- `admin-next`：Vue 3 + Vite + Pinia + Arco Design 管理后台迁移入口

三个应用均使用相对路径访问 `/api`。blog 的默认开发/构建入口已经迁移到 Vite，`vite.config.ts` 默认将 `/api` 代理到后端的自签名 HTTPS 地址 `https://localhost:7777`，也可以通过 `VITE_API_TARGET` 覆盖。`serve:legacy` 和 `build:legacy` 保留为 Vue CLI 回滚基线。旧 admin 仍使用 Vue CLI，并通过 `VUE_APP_API_TARGET` 覆盖代理地址；`admin-next` 使用 `VITE_ADMIN_API_TARGET` 覆盖代理地址。

## 本地开发

在项目根目录分别执行：

```shell
cd web/blog
npm ci
npm run serve
```

```shell
cd web/admin
npm ci
npm run serve
```

```shell
cd web/admin-next
npm ci
npm run serve
```

## 构建

```shell
cd web/blog
npm ci
npm run build
```

blog 的当前 Vite 基线可以用 Playwright 验证。默认会启动一个本地开发
服务器并 mock `/api` 数据，不会连接或修改现有容器：

```shell
cd web/blog
npm ci
npm run test:e2e:baseline
```

如需验证迁移前的 Vue CLI 产物，可运行：

```shell
npm run serve:legacy
npm run build:legacy
```

旧构建输出到 `web/blog/dist-legacy`，不会覆盖当前 Vite 产物
`web/blog/dist`；性能预算和 Caddy 隔离联调只使用后者。

要对已经部署在 Caddy 后面的站点运行同一组基线，设置
`E2E_BASE_URL=https://your-blog.example`；此时 Playwright 不会启动本地服务，
而是直接访问该地址。真实 API 联调仍由仓库根目录的隔离集成脚本负责。

```shell
cd web/admin
npm ci
npm run build
```

```shell
cd web/admin-next
npm ci
npm run build
```

构建结果分别位于 `web/blog/dist`、`web/admin/dist` 和 `web/admin-next/dist`。生产部署时，
迁移期间只将旧 `admin` 或隔离预览的 `admin-next` 复制到明确的 Caddy 静态目录；本仓库
不包含构建产物，也不执行远程 SSH 部署。隔离 Compose 会额外提供 `http://127.0.0.1:18018`
作为 `admin-next` 预览入口，旧后台仍位于 `18008`，两者共享相对路径 `/api`。

完整联调请使用仓库根目录的 `scripts/integration-up.ps1 -AllowContainerChanges`、
`integration-migrate.ps1 -AllowWrites`、`integration-seed.ps1 -AllowWrites` 和
`integration-smoke.ps1 -AllowWrites`。这些脚本只操作隔离的
`benetnasch-integration` Compose 项目；所有会改变容器、数据库、MinIO 或 Meilisearch
的入口都默认拒绝，必须在批准的隔离窗口显式传入对应参数。

隔离栈已存在且已准备 `.env.integration` 时，可运行
`scripts/integration-browser-e2e.ps1`，执行不带 API mock 的 blog 与 `admin-next`
浏览器验收；脚本只接受 `127.0.0.1:18080` 和 `127.0.0.1:18018`。三套前端构建后可用
`scripts/check-frontend-budgets.ps1` 检查静态体积门槛。

如果需要查看复制了生产 PostgreSQL 和 `articles` 搜索索引的本地开发副本，使用固定
Compose 项目 `benetnasch-dev`，入口是 `28080`（blog）、`28008`（旧 admin）和
`28018`（admin-next）。完整的数据边界、初始化和安全命令见
[benetnasch-dev 运行手册](../docs/dev-clone-runbook.md)。该副本不复用生产容器、端口或卷。

发布验收 admin-next 时使用：

```powershell
pwsh ./scripts/integration-browser-e2e.ps1 -AdminNextOnly -RequireAgentRoutes
```

脚本会在当前进程用已注入的管理员凭据获取临时 token，完整执行登录、菜单/RBAC、模块
列表和刷新矩阵；严格模式会把 Agent 菜单迁移列为必需项。登录请求不跟随重定向，token
不输出或落盘，脚本不会切换生产 Caddy 静态目录。

如果不允许改动现有容器，且只需要验收本地新后台，可设置已有管理员 token 后运行
`scripts/admin-next-readonly-e2e.ps1`。脚本默认把本地 `admin-next` 接到现有 Caddy
`127.0.0.1:18008`，只执行只读页面和 API 检查；不会登录、写入、上传、迁移或操作容器。
