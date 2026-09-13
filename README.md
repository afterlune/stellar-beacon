# 星际信标 · Stellar Beacon

星际信标（Stellar Beacon）是一个 Vue 博客前台 + Vue 管理端 + Go API 的博客系统。当前仓库采用按职责分层、按业务域组织的结构；博客前台保留原有视觉、素材和页面交互，管理端使用保留的 `admin-next`。

## 目录

- `cmd/stellar-beacon`：生产 API 进程入口
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

联调栈使用 Compose 项目 `stellar-beacon-integration-v17`，复用当前 V17 的 PostgreSQL、Redis、Meilisearch、MinIO 数据卷；应用数据复制到新数据库 `stellar_beacon` 和新桶 `stellar-beacon-integration`。旧数据库 `benetnasch`、旧桶 `benetnasch-integration` 与旧容器保留作回退；生产数据库和 OSS 桶不迁移。

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

新 Linux 主机可用独立 Compose 栈一次构建博客、管理端与 API。先准备 x86_64 Linux、Docker Compose v2、指向主机的根域名和 `admin.` 子域名，并开放 TCP 80/443（启用 HTTP/3 时也开放 UDP 443）：

```shell
cp .env.production.example .env.production
# 编辑 .env.production：换掉所有示例密钥，并填写域名、SMTP 和对象存储配置
docker compose --env-file .env.production -f deploy/compose/production-standalone.yaml --profile minio up -d --build
docker compose --env-file .env.production -f deploy/compose/production-standalone.yaml --profile minio exec -it backend /app/stellar-beacon bootstrap-admin --email admin@example.com
```

首次启动会执行版本化数据库迁移，生成空白博客配置，不会导入历史 SQL、文章、账号或日志。首次管理员密码在终端中隐藏输入。改用阿里云 OSS 时，填写 OSS endpoint、bucket、区域和访问密钥，并从命令中去掉 `--profile minio`；MinIO 数据卷不会启动。`.env.production` 含生产凭据，不要提交到 Git。更完整的安装、更新和备份说明见[部署运行手册](docs/runbooks/deployment.md)。

原有 Windows 生产栈保持独立，仍使用 `deploy/compose/production.yaml` 与 `deploy/caddy/production.Caddyfile`；不要在现有数据卷上运行新栈的初始化步骤。
