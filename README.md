# Benetnasch

Benetnasch 是一个 Vue 博客前台 + Vue 管理端 + Go API 的博客系统。当前仓库采用按职责分层、按业务域组织的结构；博客前台保留原有视觉、素材和页面交互，管理端使用保留的 `admin-next`。

## 目录

- `cmd/benetnasch`：生产 API 进程入口
- `cmd/integration-seed`：仅供隔离联调使用的数据初始化程序
- `internal/domain`：领域实体、端口和错误
- `internal/application`：应用服务与业务编排
- `internal/interfaces/http`：版本化路由、handler、DTO 和中间件
- `internal/infrastructure`：PostgreSQL、Redis、Meilisearch、MinIO、邮件和配置适配器
- `resources`：运行时资源
- `deploy`：Compose、Caddy、配置和数据库初始化文件
- `web/apps`：`blog` 前台与 `admin-next` 管理端
- `web/packages`：共享 API 契约与请求客户端
- `scripts/checks`、`scripts/integration`：检查与隔离联调脚本
- `docs`：API、运行手册和 ADR

## API

外部 API 只有版本化入口：

- `/api/v1/public/*`：博客公开内容
- `/api/v1/auth/*`：登录、注册和当前用户
- `/api/v1/admin/*`：管理端能力

响应统一为 `{ "code": "OK", "message": "...", "data": ... }`；分页数据统一为 `{ items, total, page, pageSize }`。旧 HTTP 路由不再注册。

## 开发检查

```shell
go test ./...
go vet ./...
git diff --check
```

前端：

```shell
cd web
npm ci
npm run build:blog
npm run build:admin
```

## 隔离联调

联调栈使用独立 Compose 项目 `benetnasch-integration`，不会停止、重建或修改已有 PostgreSQL、Redis、Meilisearch、MinIO、Caddy 容器及其数据。

```powershell
Copy-Item .env.integration.example .env.integration
pwsh ./scripts/integration/deploy.ps1
```

访问：

- 博客：`http://127.0.0.1:18080`
- 管理端：`http://127.0.0.1:18008`
- 隔离 API：`http://127.0.0.1:17777`

分步执行：

```powershell
pwsh ./scripts/integration/up.ps1
pwsh ./scripts/integration/repair-sequences.ps1
pwsh ./scripts/integration/seed.ps1
pwsh ./scripts/integration/smoke.ps1
```

结束时只清理隔离项目：

```powershell
pwsh ./scripts/integration/down.ps1 -RemoveVolumes
```

## 部署

生产配置位于 `deploy/config/`，生产 Compose 和 Caddy 文件分别位于 `deploy/compose/production.yaml` 与 `deploy/caddy/production.Caddyfile`。数据库初始化文件位于 `deploy/db/init/`。发布前请阅读 [部署运行手册](docs/runbooks/deployment.md)；生产容器和数据变更必须在明确的发布窗口执行。
