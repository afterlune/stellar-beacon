# benetnasch-dev 本地开发副本

这是一套与生产 Compose 隔离的本地查看环境。项目名固定为
`benetnasch-dev`，不得把下面的命令改成生产项目 `benetnasch`，也不要复用生产端口。

## 入口

| 入口 | 地址 |
| --- | --- |
| 博客前台 | <http://127.0.0.1:28080> |
| 旧管理后台 | <http://127.0.0.1:28008> |
| admin-next | <http://127.0.0.1:28018> |
| Companion 空间桥接（独立 Caddy） | <http://127.0.0.1:28028> |
| 后端直连（自签名 HTTPS） | <https://127.0.0.1:28777> |
| Meilisearch | <http://127.0.0.1:27700> |
| MinIO 控制台 | <http://127.0.0.1:29001> |

开发 Compose 的 PostgreSQL 和 Redis 端口分别是 `25432`、`26379`。Caddy
负责三个前端入口、`/api` 反向代理和仅供 Companion 使用的 `/internal/space/*` 入口；
前端不需要改 API 地址。

## 启动

后端使用 Windows 原生 Go 交叉编译，避免在 WSL/Docker 内编译造成额外内存压力。
运行时二进制和 resource 副本放在系统临时目录，不加入仓库：

```powershell
$cloneRoot = Join-Path ([System.IO.Path]::GetTempPath()) ('benetnasch-dev-clone-' + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Force -Path $cloneRoot | Out-Null
pwsh ./scripts/build-linux-amd64.ps1 -OutputPath (Join-Path $cloneRoot 'benetnasch')
Copy-Item -LiteralPath ./resource -Destination (Join-Path $cloneRoot 'resource') -Recurse
New-Item -ItemType Directory -Force -Path (Join-Path $cloneRoot 'resource/log') | Out-Null
$env:DEV_RUNTIME_DIR = $cloneRoot
$env:DEV_RESOURCE_DIR = Join-Path $cloneRoot 'resource'
docker compose -p benetnasch-dev --env-file .env.integration -f docker-compose.dev.yaml up -d
```

如果副本已经初始化，只需重新准备当前二进制和 resource 副本后再次执行最后一条
命令；不要加 `--build`。Compose 使用独立的 `benetnasch-dev-*` 命名卷。

## 数据边界

本次副本初始化采取以下数据策略：

- PostgreSQL：从现有生产容器 `pg` 执行只读 `pg_dump`，恢复到
  `benetnasch-dev-postgresql-1`，不执行生产迁移；
- Meilisearch：从生产 `meili` 只读复制 `articles` 索引的设置和文档到
  `benetnasch-dev-meilisearch-1`；
- Redis：不复制。它只保存缓存、会话和令牌，开发实例从空库开始；
- MinIO：不复制生产对象。生产配置实际使用阿里云 OSS，数据库中已有的公开资源 URL
  继续指向原资源；开发环境的新上传只写入开发 MinIO。

生产 `pg`、`redis`、`meili`、`caddy` 容器不由这套 Compose 管理。禁止对生产数据库执行
迁移、对生产 Meilisearch 写入、对生产 MinIO 写入，或切换生产 Caddy 静态目录。

## 停止开发副本

只操作开发 Compose 项目：

```powershell
docker compose -p benetnasch-dev --env-file .env.integration -f docker-compose.dev.yaml stop
```

不要使用没有 `-p benetnasch-dev` 和 `-f docker-compose.dev.yaml` 的宽泛 `docker compose`
命令，以免误选生产项目。
