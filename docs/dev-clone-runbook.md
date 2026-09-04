# benetnasch-dev 隔离开发副本

`benetnasch-dev` 是用于查看数字空间、管理入口和 Companion bridge 的独立 Compose 项目。它必须与生产项目 `benetnasch` 使用不同的项目名、端口和命名卷。

## 入口

| 能力 | 地址 |
| --- | --- |
| 数字空间公开入口 | `http://127.0.0.1:28080` |
| 稳定管理入口 | `http://127.0.0.1:28008` |
| admin-next | `http://127.0.0.1:28018` |
| Companion bridge | `http://127.0.0.1:28028` |
| 后端直连 | `https://127.0.0.1:28777` |
| PostgreSQL | `127.0.0.1:25432` |
| Redis | `127.0.0.1:26379` |
| Meilisearch | `http://127.0.0.1:27700` |
| MinIO API/控制台 | `http://127.0.0.1:29000` / `http://127.0.0.1:29001` |

## 启动

`.env.integration` 必须是仓库外或被 `.gitignore` 忽略的本地文件，不能提交。后端使用 Windows 原生 Go 工具链交叉编译，避免在 WSL/Docker 内编译：

```powershell
$runtimeRoot = Join-Path ([System.IO.Path]::GetTempPath()) ('benetnasch-dev-runtime-' + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Force -Path $runtimeRoot | Out-Null
pwsh ./scripts/build-linux-amd64.ps1 -OutputPath (Join-Path $runtimeRoot 'benetnasch') -Parallelism 1
Copy-Item -LiteralPath ./resource -Destination (Join-Path $runtimeRoot 'resource') -Recurse
New-Item -ItemType Directory -Force -Path (Join-Path $runtimeRoot 'resource/log') | Out-Null
$env:DEV_RUNTIME_DIR = $runtimeRoot
$env:DEV_RESOURCE_DIR = Join-Path $runtimeRoot 'resource'
docker compose -p benetnasch-dev --env-file .env.integration -f docker-compose.dev.yaml up -d
```

Compose 使用独立的 `benetnasch-dev-postgres-data`、`benetnasch-dev-redis-data`、`benetnasch-dev-meili-data` 和 `benetnasch-dev-minio-data` 卷，不加 `--build`，也不依赖生产容器。

启动只负责服务和静态入口，不自动执行迁移、seed、索引 provision、回填、swap 或数据复制。

## 数据边界

- PostgreSQL 数据复制只能写入 dev PostgreSQL；源数据读取、备份、恢复和行数校验必须在独立授权窗口完成。
- Redis 不复制生产会话、缓存或 token，dev 实例从隔离数据开始。
- Meilisearch 的索引复制或回填只能写入 dev Meilisearch，不能把 dev 目标误指向生产 UID。
- MinIO 是 dev 新上传的对象存储；已有公开 URL 是否指向外部 OSS 由隔离数据决定。
- 禁止将数据库备份、索引导出、对象、token、日志或运行时二进制放入仓库。

迁移使用显式命令，且只允许在已授权的隔离项目中执行：

```powershell
pwsh ./scripts/integration-migration-status.ps1
pwsh ./scripts/integration-migrate.ps1 -AllowWrites
```

上述 integration 脚本默认针对 `benetnasch-integration`；dev 副本的迁移和数据复制应使用明确指向 `benetnasch-dev` 的经过复核命令，不得只依赖当前 shell 的默认 Compose 项目。

## Companion bridge

bridge 由 `BENETNASCH_AI_SPACE_COMPANION` 和 `BENETNASCH_AI_SPACE_COMPANION_PUBLISH` 控制，默认关闭。启用时必须提供不同的 read/publish token，并先执行 `0023_space_companion.sql` 到 dev 数据库。

Companion 以 `http://host.docker.internal:28028` 访问宿主机 bridge；生产 Caddy、生产 backend 和生产数据不会被这套入口代理。

## 验收

只读启动检查：

```powershell
docker compose -p benetnasch-dev --env-file .env.integration -f docker-compose.dev.yaml ps
```

完整验收需要单独验证：公开入口、管理入口、菜单和刷新、空间能力、公开检索、公开内容读取、发布白名单、token 隔离、幂等冲突以及 Companion 工具调用。未执行的项目必须保留为“未验证”。

## 停止

只操作开发 Compose 项目：

```powershell
docker compose -p benetnasch-dev --env-file .env.integration -f docker-compose.dev.yaml stop
```

不要使用没有 `-p benetnasch-dev` 和 `-f docker-compose.dev.yaml` 的宽泛命令；不要对生产项目执行 `down`、`rm`、`volume rm` 或重建。
