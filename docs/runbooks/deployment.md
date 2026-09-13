# 部署运行手册

## 前置条件

准备 Docker/Compose、PostgreSQL、Redis、Meilisearch、MinIO 和 Caddy。生产数据与容器属于发布资产，除非在明确发布窗口内，不执行重建、迁移或切换。

复制 `.env.example` 为 `.env` 并填写密钥。运行时配置放在 `deploy/config/`：基础配置为 `base.yaml`，按 `STELLAR_BEACON_ENV` 选择 `dev.yaml`、`integration.yaml` 或 `prod.yaml`。

## 构建

```shell
docker build -t stellar-beacon:latest .
```

前端在 workspace 根目录构建：

```shell
cd web
npm ci
npm run build:blog
npm run build:admin
```

静态产物分别为 `web/apps/blog/dist` 和 `web/apps/admin-next/dist`，复制到 Caddy 的 `blog` 与 `admin` 静态目录。产物不纳入 Git。

## 生产文件

- Compose：`deploy/compose/production.yaml`
- Caddy：`deploy/caddy/production.Caddyfile`
- 运行时资源：`resources/`
- 配置：`deploy/config/`
- 数据库初始化：`deploy/db/init/001-stellar-beacon.sql`

生产 Compose 仍连接既有服务。修改数据库、Redis、搜索索引、对象存储或 Caddy 前，必须单独安排发布窗口并保留审计记录。

## 隔离联调

隔离联调使用 `deploy/compose/integration.yaml` 和 `scripts/integration/`，端口为博客 `18080`、管理端 `18008`、后端 `17777`、Meilisearch `17700`、MinIO `19000/19001`。它使用独立容器和数据卷，不触碰生产容器。

```powershell
Copy-Item .env.integration.example .env.integration
pwsh ./scripts/integration/deploy.ps1
```

清理时只操作隔离项目：

```powershell
pwsh ./scripts/integration/down.ps1 -RemoveVolumes
```

## API 验收

接口入口是 `/api/v1/public`、`/api/v1/auth` 和 `/api/v1/admin`。正常响应为 `code: "OK"`；分页响应必须包含 `items`、`total`、`page`、`pageSize`。使用 `scripts/integration/smoke.ps1` 验证 Caddy、前后端、PostgreSQL、Redis、Meilisearch 和 MinIO 链路。
