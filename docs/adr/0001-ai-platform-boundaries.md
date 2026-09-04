# ADR-0001：数字空间 Agent 平台边界

- 状态：已接受
- 日期：2026-09-05

## 背景

Benetnasch 需要提供检索、内容理解、写作辅助、审核和持久任务能力，但空间事实、用户权限和发布策略必须继续由 Go 服务确定。模型 Provider、Agent 编排框架和搜索 SDK 不能渗透到 application，AI 关闭时原有空间 API 和管理入口仍必须可用。

## 决策

1. Eino 是 Go 服务内空间 Agent 的主线。Eino、Provider SDK、Meilisearch 和存储客户端只允许位于 `app/infra` 适配器；`app/application` 只依赖 `app/domain/port`。
2. Chat/Vision 通过 `OPENAI_*` 环境变量访问 DeepSeek 兼容接口；Embedding 通过 `ALIBAILIAN_*` 环境变量访问阿里百炼兼容接口。模型配置按能力隔离，不能隐式跨能力复用。
3. Meilisearch 继续承担关键词、语义和混合检索；本阶段不迁移 PostgreSQL，也不引入 pgvector、FAISS 或新的向量数据库容器。
4. LangChain 不进入生产 Go 服务。只有真实故障演练证明 Eino 加持久状态机无法满足跨进程暂停恢复、人工中断恢复或多 Agent 长流程时，才按 ADR-0003 评估独立 LangGraph sidecar。
5. 模型生成内容默认进入人工审核队列。模型不能决定身份、权限、审核结果、发布动作或计费规则。
6. Provider key 只来自环境变量；数据库只保存 Provider、模型和非敏感路由配置。日志不得包含 key、完整 Prompt、访客正文、Provider 参数或 SQL 参数。
7. AI 能力由独立 feature flag 控制，默认关闭；数据库迁移、索引操作和生产发布必须使用显式运维入口，服务启动不自动改表。

## 影响

- application 与 Provider 解耦，可使用 fake 或固定 mock server 进行契约测试。
- 需要通过 domain port、审核状态、任务租约和审计记录表达更多边界，但避免供应商 SDK 锁定核心用例。
- 空间 Agent 与 Companion 居民是两套不同的职责，月社妃的身份、人格和记忆由 ADR-0004 管理。
- 发布时必须分别验证代码、迁移、Provider、索引、Caddy 和容器，不把启动成功当成完整发布证据。

## 不做的事情

- 不在 application 引入 Eino、Provider、xorm、Redis 或 Meilisearch 类型。
- 不因“未来可能需要”提前增加 LangGraph、LangChain 或 Python sidecar。
- 不允许 Agent 绕过审核直接修改受保护空间事实。
