# ADR-0002：借鉴 Companion 的可靠性模式

- 状态：已接受
- 日期：2026-09-05

## 背景

独立 Companion 项目在持久任务、事件重放、审批防重放和记忆版本化方面有成熟的本地优先模式。这些模式可以完善 Benetnasch 的 Eino 空间 Agent，但两个项目的身份、数据和运行时边界不同，不能直接复制 Companion 的实现。

## 决策

1. 每次 Provider 调用由空间生成稳定 `RunID`，并单独保存 Provider 返回的 ID。流式输出开始后不自动重试，避免重复内容。
2. SSE 和后台 Agent 事件使用 `eventId + sessionId + turnId + seq` 信封；Redis 只保存短期游标，长期任务、审核事件和副作用记录进入 PostgreSQL。
3. 长任务使用持久状态机、数据库租约、有限重规划、等待状态和副作用幂等键。Worker 重启后从已保存状态恢复，不覆盖旧计划版本。
4. 公开空间 Agent 不获得写工具。后台写作和行为生成只产生待审核候选；审核动作绑定会话、目标、内容摘要和有效期，并使用条件更新防止重复消费。
5. 需要长期记忆时，使用带来源、置信度、有效期、版本和冲突状态的断言模型。匿名访客数据默认不进入长期学习，公开内容检索继续使用 Meilisearch。

## Companion 边界

- Companion 的人格、Canon、私有记忆、语音和自主行为继续由 Companion 项目负责。
- Benetnasch 只负责空间公开内容、机器主体、权限、审计和受限发布事实。
- 不把 Companion 的 Python/FastAPI、SQLite、NATS 或人格编排复制进 Go 服务。
- 两个项目之间只通过版本化空间协议交互，不共享数据库连接、Redis、Meilisearch、MinIO 或人类会话。

## 不引入

- 不增加 pgvector、FAISS、WebRTC/WebTransport、本地语音模型或系统 Shell 工具作为 Agent 依赖。
- 不因理论上的长流程需求提前引入 LangChain/LangGraph；只有 ADR-0003 的量化条件满足时才单独评估。

## 结果

可靠性能力按可关闭、可测试、可回滚的顺序推进。持久化和事件字段会增加迁移与契约成本，但能够避免流式重复、副作用重放、重启丢任务和跨项目身份混淆。
