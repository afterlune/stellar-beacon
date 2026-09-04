# ADR-0002：借鉴 Companion 的 Agent 可靠性模式

状态：已接受

日期：2026-08-28

## 背景

对 Companion（Python/FastAPI 本地优先 Agent 项目）的代码、架构文档和测试进行对照审查后，发现其中的持久任务、事件重放、审批防重放和记忆版本化模式，可以补充 Benetnasch 的 Eino 长期路线。两者的业务边界和技术栈不同，不能直接复制运行时实现。

## 决策

1. Provider 调用的 `RunID` 由 Benetnasch 生成并贯穿一次调用；Provider 返回的 ID 单独保存为 `ProviderRunID`。流式输出开始后不再自动重试，避免重复内容。
2. SSE 和后台 Agent 事件使用 `eventId + sessionId + turnId + seq` 信封。Redis 只承载匿名会话的短期游标；后台任务和审核事件才进入 PostgreSQL outbox。
3. 长任务使用持久状态机、数据库租约、有限重规划、问题等待和副作用幂等键。Worker 重启后恢复未完成步骤，旧计划版本保留不覆盖。
4. 公开 Agent 不获得写工具。后台写作和行为生成只进入审核队列；审核动作绑定会话、目标、内容摘要和有效期，并通过条件更新防止重复消费。
5. 后续需要长期记忆时，使用带来源、置信度、有效期、版本和冲突状态的断言模型。匿名访客数据默认不进入长期学习，文章检索继续使用 Meilisearch。

## 不引入的内容

- 不把 Companion 的 Python/FastAPI 编排、SQLite 存储或 NATS 集群平面复制进 Go 服务。
- 不增加 pgvector、FAISS、WebRTC/WebTransport、本地语音模型或系统 Shell 工具作为当前 Agent 依赖。
- 不因为存在理论上的长流程需求而提前引入 LangChain/LangGraph；只有 Eino 的恢复能力经过真实故障评测仍不足时，才按既有 ADR 评估独立 sidecar。

## 结果

- Eino 和供应商 SDK 仍被限制在 infra，application 只依赖项目自有 port。
- 可靠性能力按 M1 → M4 → M5 → M9 逐步落地，每步可关闭、可测试和可回滚。
- 持久化和事件字段会增加迁移、审计和前端契约成本，但能避免流式重复、副作用重放和重启丢任务。
