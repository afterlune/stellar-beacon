# Benetnasch 文档中心

本目录是 Benetnasch 数字空间的长期技术上下文。它描述当前代码、配置、Agent 路线、Companion 居民边界和发布约束；实际行为以代码、配置和测试为准。

## 项目概览

Benetnasch 是基于 Gin、xorm、PostgreSQL、Redis 和 Casbin 的数字空间后端，并包含三个前端入口：

- `web/blog`：Vue 3 数字空间前台（保留 blog 目录名以兼容现有构建）；
- `web/admin`：迁移期间保留的稳定 Vue 2 管理后台；
- `web/admin-next`：Vue 3 + Vite + Pinia + Arco Design 的新管理后台。

空间侧 Agent 由 Go application/domain port 和 infra 中的 Eino Provider、RAG、持久任务、审核与体验模块组成。月社妃由独立的 Companion 项目提供生命运行时，通过受限空间 API 作为 Agent 居民访问 Benetnasch。LangChain 不进入生产 Go 服务，LangGraph 仍按 ADR-0003 延后。新能力默认关闭，生产容器、数据库迁移、Meilisearch 索引操作和 Caddy 静态目录切换都必须独立审批。

## 快速入口

| 目标 | 文档 |
| --- | --- |
| 当前产品目标、边界和实施计划 | [数字空间当前计划](digital-space-plan.md) |
| Prompt 注入、工具越权、SSRF、隐私和成本威胁 | [Agent 威胁模型](agent-threat-model.md) |
| Provider、RAG、审核和功能开关验收 | [Agent 评测标准](agent-evaluation-rubric.md) |
| 白名单开放、回滚和发布窗口 | [Agent 发布手册](agent-rollout-runbook.md) |
| Chunk 索引 provision、回填、swap 和回滚 | [搜索切换手册](agent-search-cutover-runbook.md) |
| Caddy 静态目录切换、归档和保留周期 | [前端性能与发布资产门禁](frontend-performance-gates.md)、[Agent 发布手册](agent-rollout-runbook.md) |
| 迁移前滚、checksum、备份恢复和失败处理 | [数据库迁移手册](agent-database-migration-runbook.md) |
| Agent 路由与 Casbin 菜单/资源种子 | [Casbin 资源说明](agent-casbin-resources.md) |
| 依赖漏洞与升级记录 | [安全依赖说明](security-dependencies.md) |
| 架构边界与 Companion 借鉴决策 | [ADR-0001](adr/0001-ai-platform-boundaries.md)、[ADR-0002](adr/0002-companion-reliability-patterns.md) |
| Eino 主线与 LangGraph 条件评估 | [ADR-0003](adr/0003-eino-langgraph-conditional-evaluation.md) |
| 数字空间与 Companion 居民边界 | [ADR-0004](adr/0004-digital-space-resident-architecture.md) |
| Companion 空间协议、令牌和开发入口 | [空间协议说明](space-companion-protocol.md) |
| API 字段和 Swagger | [swagger.yaml](swagger.yaml)、[swagger.json](swagger.json) |
| 本地生产数据开发副本 | [benetnasch-dev 运行手册](dev-clone-runbook.md) |

## 本地验证

默认安全前置检查：

```powershell
pwsh ./scripts/safe-preflight.ps1
```

该命令执行 Go 测试、`go vet`、SQL/分层/依赖边界扫描、旧日志检查、前端构建预算和
admin-next 发布资产完整性检查；可选的浏览器基线也只使用本地 mock。它不会启动或停止
容器、执行数据库迁移、写入 Meilisearch 或切换 Caddy 静态目录。

前端构建、隔离联调和只读验收入口见 [web/README.md](../web/README.md)。需要真实 PostgreSQL、Redis、Caddy、Meilisearch 或 MinIO 数据的测试，必须在独立隔离环境或经过批准的发布窗口执行，并在结果中区分实际运行与跳过的门禁。

## 文档维护规则

1. 文档用于导航和记录边界，不能替代代码审查；文档与代码冲突时以代码为准，并在修复后更新文档。
2. 接口、事件、配置、模块职责、依赖方向或启动方式变化时，在同一变更中更新相关文档和测试。
3. 新的长期技术取舍写入 `docs/adr/`；普通修复不重复抄写 Git 历史。
4. 不提交运行时数据库、备份、日志、密钥、证书、token 或前端 `dist` 产物。
