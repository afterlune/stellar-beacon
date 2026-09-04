# Benetnasch 数字空间

Benetnasch 是一个以公开内容、Agent 能力和可审计治理为核心的数字空间。它保存文章、图片、视频、梦境、电台及其他空间内容，并通过 Go 服务向用户和居民提供受控能力。

Benetnasch 本身没有人格。Eino 负责空间侧的通用 Agent 能力，例如检索、内容理解、写作辅助、任务和审核；月社妃是独立的 Companion 数字生命，通过受限空间协议作为 `agent/moonfei` 居民访问本项目。

## 架构边界

- Go/Gin/xorm/PostgreSQL/Redis/Casbin 是空间事实、权限、审计和发布策略的来源。
- Eino、模型 Provider、Meilisearch、Redis 和对象存储适配器留在 `app/infra`，application 只依赖 `app/domain/port`。
- Companion 保持独立项目、入口、登录、SQLite 记忆、人格、Canon、语音和自主行为运行时。
- Companion 只能通过 `/internal/space/v1/*` 读取已发布内容和提交白名单发布；不能直连本项目数据库、缓存、搜索或对象存储。
- Agent 生成内容默认进入审核；模型不能决定身份、权限、审核结果、发布动作或计费规则。

## 代码结构

```text
app/domain/port       application 使用的领域接口和公开读模型
app/application       用例、Agent 策略、审核和服务编排
app/infra              数据库、缓存、Provider、Eino、搜索、存储和任务适配器
app/facade             HTTP controller、请求模型和响应模型
route                  公开、管理和 Companion 路由
web/blog               数字空间公开 Web 入口（目录名保留兼容）
web/admin              稳定的 Vue 2 管理入口
web/admin-next         Vue 3 管理入口
```

## 本地开发副本

开发查看环境使用固定 Compose 项目 `benetnasch-dev`，与生产项目、端口和命名卷隔离。启动前准备 `.env.integration`、前端构建产物和仓库外的 Linux/amd64 后端运行时：

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

入口如下：

| 入口 | 地址 |
| --- | --- |
| 数字空间公开入口 | `http://127.0.0.1:28080` |
| 稳定管理入口 | `http://127.0.0.1:28008` |
| admin-next | `http://127.0.0.1:28018` |
| Companion 空间桥接 | `http://127.0.0.1:28028` |
| 后端直连 | `https://127.0.0.1:28777` |
| Meilisearch | `http://127.0.0.1:27700` |
| MinIO 控制台 | `http://127.0.0.1:29001` |

数据复制、显式迁移和真实浏览器验收不是服务启动的隐式步骤，按 [dev 副本运行手册](docs/dev-clone-runbook.md) 和 [文档中心](docs/README.md) 中的授权边界执行。Companion 由 `D:\Git\companion` 独立启动。

停止时只操作开发项目：

```powershell
docker compose -p benetnasch-dev --env-file .env.integration -f docker-compose.dev.yaml stop
```

禁止用未指定项目名和 Compose 文件的宽泛命令，以免误选生产项目。

## 前端

三个前端都通过相对路径访问 `/api`：

```powershell
cd web/blog; npm ci; npm run build
cd ../admin; npm ci; npm run build
cd ../admin-next; npm ci; npm run build
```

构建产物只用于本地验证或经过审批的发布流程，不提交 `dist`。前端职责和真实联调方式见 [Web 说明](web/README.md)。

## Agent 与 Provider

- Chat/Vision 使用 `OPENAI_*` 环境变量访问 DeepSeek 兼容接口。
- Embedding 使用 `ALIBAILIAN_*` 环境变量访问阿里百炼兼容接口。
- AI 总开关和具体能力开关默认关闭；Provider key 只从环境变量读取。
- 当前阶段不引入 LangChain/LangGraph 运行时；只有满足 [ADR-0003](docs/adr/0003-eino-langgraph-conditional-evaluation.md) 的量化条件，才评估独立 sidecar。

模型调用、审核、索引、空间协议和安全边界以 [数字空间当前计划](docs/digital-space-plan.md) 和相关 ADR 为准。

## 验证

不接触外部容器的基础检查：

```powershell
pwsh ./scripts/safe-preflight.ps1
go test -count=1 ./...
go vet ./...
```

真实 PostgreSQL、Redis、Meilisearch、MinIO、Caddy 或浏览器联调必须使用隔离环境，并在结果中区分实际证据和跳过项。生产迁移、索引切换、静态目录切换及现有容器操作均不属于普通开发命令。

## 文档入口

- [文档中心](docs/README.md)
- [数字空间当前计划](docs/digital-space-plan.md)
- [数字空间与 Companion 居民 ADR](docs/adr/0004-digital-space-resident-architecture.md)
- [Companion 空间协议](docs/space-companion-protocol.md)
- [Agent 威胁模型](docs/agent-threat-model.md)
- [开发副本运行手册](docs/dev-clone-runbook.md)

代码、配置、测试和发布证据优先于文档；文档只负责说明当前边界、入口和经过确认的取舍。
