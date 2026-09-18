# 部署运行手册

## 新 Linux 主机：独立生产栈

该部署面向 Linux x86_64 单机，由 Compose 管理 API、PostgreSQL、Redis、Meilisearch、Caddy 和两个 Vue 应用；MinIO 是可选的本地对象存储。只需预装 Docker Engine 与 Compose v2，不需要单独安装 Go、Node、PostgreSQL 或 Redis。

先将域名的 A/AAAA 记录指向主机，确保 `SITE_DOMAIN` 和 `ADMIN_DOMAIN` 都可从公网访问；放通 TCP 80/443，启用 HTTP/3 时放通 UDP 443。Caddy 使用 ACME 自动签发证书。

复制并编辑部署变量：

```shell
cp .env.production.example .env.production
```

必须替换所有 `replace-with-...` 示例密钥，设置 SMTP 主机、发信地址与应用密码，并按实际域名修正 `SITE_DOMAIN`、`ADMIN_DOMAIN`、`ACME_EMAIL` 和 `OBJECT_STORAGE_PUBLIC_URL`。建议使用 `openssl rand -hex 32` 生成数据库、Redis、Meilisearch、MinIO 密钥；SMTP 密码使用服务商提供的应用密码。`.env.production` 已被 Git 忽略，不要提交或放入镜像。

使用 MinIO 时配置 `OBJECT_STORAGE_PROVIDER=minio`、endpoint `http://minio:9000`、公开 URL `https://<站点域名>/storage/<bucket>`，并以 `--profile minio` 启动：

```shell
docker compose --env-file .env.production -f deploy/compose/production-standalone.yaml --profile minio up -d --build
```

改用阿里云 OSS 时，填写 `OBJECT_STORAGE_PROVIDER=aliyun`、endpoint、region、bucket、公开 URL 和访问密钥，并省略 `--profile minio`。外部存储的公开 URL 应由对象存储/CDN 直接提供。两种模式都使用同一 API 配置入口 `deploy/config/prod-standalone.yaml`。

Compose 会构建 Go API 和包含博客、`admin-next` 静态资源的 Caddy 镜像。`migrate` 一次性服务先执行版本化迁移；只有成功后 API 才启动。PostgreSQL、Redis、Meilisearch、JWT 密钥、应用日志、Caddy 证书和 MinIO 数据使用独立命名卷，服务端口不发布到宿主机，仅 Caddy 对外开放 80/443。

迁移结束后，在交互式终端创建第一个管理员。程序会隐藏并二次确认密码；该命令在已有管理员或同邮箱账号时拒绝运行，不提供 HTTP 自动注册管理员入口：

```shell
docker compose --env-file .env.production -f deploy/compose/production-standalone.yaml --profile minio exec -it backend /app/stellar-beacon bootstrap-admin --email admin@example.com
```

阿里云 OSS 模式同样省略 `--profile minio`。完成后访问 `https://<SITE_DOMAIN>` 与 `https://<ADMIN_DOMAIN>`，用管理员邮箱和新设密码登录。

更新应用：

```shell
docker compose --env-file .env.production -f deploy/compose/production-standalone.yaml --profile minio up -d --build
```

后台任务使用标准五段 Cron，并以容器/主机的 `TZ` 为执行时区；当前 Compose 默认为 `Asia/Shanghai`。停机期间错过的执行会被跳过。API 进程使用 Redis 互斥避免多实例重复运行同一任务，并在收到 `SIGINT`/`SIGTERM` 后先停止 HTTP，再停止任务与后台 worker。

`/healthz` 是存活检查，服务进程存在即返回 200；`/readyz` 是就绪检查，启动完成后返回 200，停机过程中返回 503。现有 Compose 健康检查继续使用 `/healthz`，运维监控和负载均衡应使用 `/readyz`，避免 Redis 或邮件等外部依赖的短时故障触发容器重启。

不要用 `down -v` 做常规更新，它会删除数据库、对象文件和密钥卷。发布前备份 PostgreSQL 与对象存储；至少异机保留数据库转储和 MinIO/OSS 对象副本。数据库备份示例：

```shell
docker compose --env-file .env.production -f deploy/compose/production-standalone.yaml --profile minio exec -T postgresql pg_dump -U stellar_beacon stellar_beacon > stellar-beacon.sql
```

首次部署创建的是空白站点。历史数据库导出 `deploy/db/init/001-stellar-beacon.sql` 只供原隔离联调栈使用，新生产栈不会挂载或导入它。

## 现有 Windows 生产栈

原有生产配置继续使用 `deploy/compose/production.yaml`、`deploy/caddy/production.Caddyfile` 和已存在的外部服务/数据目录。它们与独立 Linux 栈分开维护；不要把新栈的 migration 命令指向该数据库，也不要改动其卷来进行首次安装。

旧栈仍需手工构建前端时，可在 workspace 根目录运行 `npm ci`、`npm run build:blog` 和 `npm run build:admin`。产物分别为 `web/apps/blog/dist` 与 `web/apps/admin-next/dist`，不纳入 Git。

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

接口入口是 `/api/v1/public`、`/api/v1/auth` 和 `/api/v1/admin`。正常响应为 `code: "OK"`；分页响应必须包含 `items`、`total`、`page`、`pageSize`。使用 `scripts/integration/smoke.ps1` 验证 Caddy、前后端、PostgreSQL、Redis、Meilisearch 和 MinIO 链路；smoke 还会用隔离环境中的公开文章完成阅读会话与系列/相关阅读事件上报，并校验管理端内容表现概览、文章列表、目标归因排行和详情接口。
