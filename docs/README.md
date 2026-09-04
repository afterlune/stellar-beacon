# Benetnasch 文档中心

本目录只记录当前数字空间的产品边界、技术决策、接口契约、验证方法和发布约束。代码、配置、测试和真实运行证据优先于文档；文档与实现冲突时，以实现为准并在同一变更中修正文档。

## 当前产品模型

Benetnasch 是数字空间，不以传统内容站点作为产品中心。空间保存公开内容和受控的 Agent 能力；Eino 是空间侧的通用 Agent 引擎，不是任何居民的人格引擎。

月社妃由独立的 `D:\Git\companion` 项目提供生命运行时。她以 `agent/moonfei` 身份通过受限协议访问空间，拥有独立的入口、登录、Canon、私有记忆、语音和自主行为运行时。

## 快速入口

| 目标 | 文档 |
| --- | --- |
| 唯一产品目标、当前状态和未完成项 | [数字空间当前计划](digital-space-plan.md) |
| 总体架构、Provider 与 Eino 边界 | [ADR-0001](adr/0001-ai-platform-boundaries.md) |
| Companion 可靠性模式借鉴 | [ADR-0002](adr/0002-companion-reliability-patterns.md) |
| Eino 主线与 LangGraph 条件评估 | [ADR-0003](adr/0003-eino-langgraph-conditional-evaluation.md) |
| 数字空间与月社妃边界 | [ADR-0004](adr/0004-digital-space-resident-architecture.md) |
| Companion HTTP 契约、令牌和限制 | [空间协议](space-companion-protocol.md) |
| Agent 路由、后台资源和机器主体 | [Agent 资源说明](agent-casbin-resources.md) |
| Prompt 注入、越权、SSRF、隐私和成本风险 | [Agent 威胁模型](agent-threat-model.md) |
| Provider、审核、Vision 和写作质量检查 | [Agent 评测标准](agent-evaluation-rubric.md) |
| Feature flag、灰度、回滚和生产窗口 | [Agent 发布手册](agent-rollout-runbook.md) |
| 迁移、备份、恢复和失败处理 | [数据库运维手册](agent-database-migration-runbook.md) |
| `article_chunks_<version>` 回填和切换 | [检索索引手册](agent-search-cutover-runbook.md) |
| 隔离开发副本 `benetnasch-dev` | [开发副本手册](dev-clone-runbook.md) |
| 前端构建、性能和发布资产 | [Web 说明](../web/README.md)、[性能门禁](frontend-performance-gates.md) |
| Go/JavaScript/容器依赖安全 | [依赖安全基线](security-dependencies.md) |
| 当前 HTTP API | [Swagger YAML](swagger.yaml)、[Swagger JSON](swagger.json) |

## 系统边界

```text
用户/居民
   │
   ├─ Web /api ───────────────┐
   └─ Companion space API      │
                               ▼
                         Gin facade
                               │
                    application + domain port
                       │       │       │
                 PostgreSQL  Redis  Provider/Eino/Meili/Storage
```

- `app/application` 不能依赖 xorm、Provider SDK、Meilisearch、Redis 或对象存储类型。
- 人类后台权限由 Casbin 和后台会话控制；Companion 机器主体由独立 token、数据库主体和 `space:*` 作用域控制。
- 公开查询只返回已发布内容；草稿、账号、日志、凭据、Prompt 和后台数据不属于空间公开知识。
- Agent 输出默认是待审核候选；模型不能扩大工具白名单、改变权限或直接发布受保护内容。

## 安全运行规则

- 新能力 feature flag 默认关闭，密钥只通过环境变量注入。
- 服务启动不自动执行迁移、索引 provision、回填、swap 或对象存储写入。
- 生产 PostgreSQL、Redis、Meilisearch、MinIO、Caddy 和现有 Compose 不由普通开发命令操作。
- 真实外部服务验证必须在隔离 Compose 或明确批准的发布窗口执行，并记录目标、凭据来源、结果和回滚证据。
- 日志使用标准库 `log/slog`，不得记录密钥、完整 Prompt、访客正文、Provider 参数或 SQL 参数。

## 本地检查

```powershell
pwsh ./scripts/safe-preflight.ps1
go test -count=1 ./...
go vet ./...
```

前置检查不会启动或停止容器，不执行数据库迁移，不写入 Meilisearch/MinIO，也不切换生产 Caddy。Companion 的 Python 测试在 `D:\Git\companion` 内独立执行。

## 文档维护

1. 只有长期技术取舍进入 `docs/adr/`；普通修复不复制 Git 历史。
2. 接口、配置、事件、依赖方向、数据流或启动方式改变时，同步更新相关契约、测试和运行手册。
3. 所有状态必须标记为代码完成、隔离验证、生产验证或明确延期，不能用 mock 结果代替真实证据。
4. 不提交日志、数据库、备份、证书、token、密钥或前端构建产物。
