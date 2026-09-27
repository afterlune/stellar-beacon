# 部署手册

## 全新 Linux 主机

独立生产栈面向 Linux x86_64 单机。主机只需安装 Docker Engine 和 Compose v2；Go、Node、PostgreSQL、Redis、Meilisearch 和 Caddy 都由 Compose 管理。

### 准备域名和配置

将 `SITE_DOMAIN` 和 `ADMIN_DOMAIN` 的 A/AAAA 记录指向主机。开放 TCP 80、443；需要 HTTP/3 时也开放 UDP 443。Caddy 会通过 ACME 申请证书。

复制环境文件：

```shell
cp .env.production.example .env.production
```

在 `.env.production` 中填写域名、`ACME_EMAIL`、SMTP 和对象存储配置。替换所有 `replace-with-...` 示例值；数据库、Redis、Meilisearch 和 MinIO 使用各自的随机密钥，可用 `openssl rand -hex 32` 生成。SMTP 密码应使用邮件服务商的应用密码。该文件含部署凭据，不要提交或打入镜像。

### 启动

默认使用 Compose 内的 MinIO：

```shell
docker compose --env-file .env.production -f deploy/compose/production-standalone.yaml --profile minio up -d --build
```

改用阿里云 OSS 时，将 `OBJECT_STORAGE_PROVIDER` 设为 `aliyun`，填写 endpoint、region、bucket、公开地址和访问密钥，并从 Compose 命令中省略 `--profile minio`。公开地址应由 OSS 或 CDN 提供。

首次启动会先执行版本化数据库迁移，再启动 API。迁移完成后，在交互终端创建首个管理员：

```shell
docker compose --env-file .env.production -f deploy/compose/production-standalone.yaml --profile minio exec -it backend /app/stellar-beacon bootstrap-admin --email admin@example.com
```

密码会隐藏输入并要求再次确认。若已有管理员或该邮箱已注册，命令会拒绝创建。完成后访问 `https://<SITE_DOMAIN>` 和 `https://<ADMIN_DOMAIN>`。

新站点从空白数据库启动，不会导入历史账号、文章或日志。PostgreSQL、Redis、Meilisearch、JWT 密钥、日志、Caddy 证书和 MinIO 文件使用独立卷；只有 Caddy 对外开放 HTTP/HTTPS 端口。

### 更新、健康检查与备份

更新应用时重新构建并启动：

```shell
docker compose --env-file .env.production -f deploy/compose/production-standalone.yaml --profile minio up -d --build
```

不要用 `down -v` 做日常更新，它会删除数据卷。`down` 停止服务但保留数据。发布前备份数据库和对象存储，并将副本保存在另一台主机。PostgreSQL 备份示例：

```shell
docker compose --env-file .env.production -f deploy/compose/production-standalone.yaml --profile minio exec -T postgresql pg_dump -U stellar_beacon stellar_beacon > stellar-beacon.sql
```

`/healthz` 检查进程存活，`/readyz` 检查服务是否已就绪；运维监控和负载均衡使用 `/readyz`。短暂的 Redis 或邮件故障不会因此触发容器重启。

后台任务使用五段 Cron 表达式，默认按 `Asia/Shanghai` 执行。进程退出时会先停止 HTTP 服务，再停止任务和后台 worker。

### 现有 Windows 生产栈

现有环境继续使用 `deploy/compose/production.yaml`、`deploy/caddy/production.Caddyfile` 和已有外部服务与数据目录。它与 Linux 独立栈分开维护；不要对现有数据库执行新栈的首次部署迁移。

该栈的前端产物由 `npm run build:blog` 和 `npm run build:admin` 生成，分别位于 `web/apps/blog/dist` 和 `web/apps/admin-next/dist`。

## 隔离联调

隔离联调使用 `deploy/compose/integration.yaml` 和 `scripts/integration/`。博客、管理端和 API 地址分别是 `http://127.0.0.1:18080`、`http://127.0.0.1:18008` 和 `http://127.0.0.1:17777`。该栈使用专用 Compose 项目，不连接生产容器。

```powershell
Copy-Item .env.integration.example .env.integration
pwsh ./scripts/integration/deploy.ps1
```

结束时只清理联调项目：

```powershell
pwsh ./scripts/integration/down.ps1 -RemoveVolumes
```

## API 冒烟检查

版本化 API 前缀为 `/api/v1/public`、`/api/v1/auth` 和 `/api/v1/admin`。使用 `scripts/integration/smoke.ps1` 检查 Caddy、前后端、PostgreSQL、Redis、Meilisearch 和 MinIO 请求链路，并验证公开文章及管理端内容表现接口。
