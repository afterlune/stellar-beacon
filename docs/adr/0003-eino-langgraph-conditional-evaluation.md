# ADR-0003：Eino 主线与 LangGraph 条件评估

状态：已接受（LangGraph 延后）

日期：2026-08-29

## 决策摘要

Benetnasch 继续使用 Eino 作为 Go 服务内的 Agent 主线；本阶段不引入 LangGraph、
LangChain、Python sidecar 或新的容器。Eino 的编排能力、Provider 适配和故障控制
继续留在 `infra`，持久任务、审核状态、租约、事件重放和恢复游标由现有 Go/Redis/
PostgreSQL 边界负责。

只有当真实故障演练证明当前边界无法满足跨进程暂停恢复、人工中断恢复或多 Agent
长流程的量化要求时，才启动独立 LangGraph sidecar POC。POC 必须先通过安全、延迟、
恢复和运维成本对比，不能因为“未来可能需要”直接进入生产链路。

## 背景

当前 Agent 路线包含两类不同问题：

1. 请求级的检索、Prompt、模型调用、流式输出、重试、熔断和取消；
2. 需要跨请求或跨进程保存的任务、审核、事件、租约、游标和副作用。

第一类适合由 Eino Graph 和 Provider ACL 处理，第二类不能只依赖进程内 Graph 状态。
把所有问题都交给第二套编排框架会增加运行时、身份、网络、部署和观测面，也会让
现有 Go 后端承担两套恢复语义。

## 当前实现与边界证据

| 能力 | 当前实现 | 结论 |
| --- | --- | --- |
| 固定 RAG 流程 | `app/infra/ai/rag/graph.go` 使用 Eino `compose` 编译 `normalize → retrieve → assemble_context → build_prompt → generate → result` | 已满足确定性请求级编排 |
| Provider 可靠性 | `app/infra/ai/einoadapter/adapter.go` 提供有界重试、并发闸门、Context 取消和流式事件边界；`circuitbreaker.go` 隔离单路 Provider | 不需要第二套框架解决请求级故障 |
| 跨进程任务 | `app/infra/task` 的持久 Job、租约、checkpoint、重试/死信和 `Supervisor` | 持久状态已在 Eino 外部实现 |
| 审核与副作用 | `t_ai_review`/`t_ai_review_action` 及 repository 的条件更新、幂等键和内容摘要绑定 | 人工介入不依赖进程内状态 |
| 当前缺口 | RAG Graph 本身没有跨进程 checkpoint store 或 human interrupt/resume 运行时 | 只有出现真实长流程需求时才评估 sidecar |

因此，“Eino 当前限制”在本项目中的准确表述不是 Eino 已经不能用，而是当前 Graph
是请求级、无持久 checkpoint 的固定流程；持久恢复由应用层状态机承担。若未来要把
一个长流程暂停在 Graph 内并跨实例恢复，必须先证明现有状态机和 Eino 组合无法满足
需求，再评估 LangGraph。

## 故障案例与证据强度

### 已观察到的运行时故障

2026-08-25 的部署日志曾出现操作日志重复写入导致 PostgreSQL 唯一约束
`_copy_14` 冲突。该事件不是 Eino 故障，但说明“每请求无界异步写入 + 永久性数据库
错误重试”会放大故障并污染主链路。当前控制措施是有界 `LogQueue`、固定 worker、
对永久性重复键错误不重试，并由 `app/infra/persistence/repository/log_queue_test.go`
覆盖。

原始运行日志不作为仓库凭据或测试 fixture 提交；本 ADR 只保留脱敏后的故障类别和
处理结论。

### 已确定性复现的故障场景

| 场景 | 证据 | 处理结论 |
| --- | --- | --- |
| Provider 在已经发出部分 delta 后失败 | `app/infra/ai/einoadapter/adapter_test.go` 的 `TestChatAdapterStreamNeverRetriesAfterPartialOutput` | 不自动重试，避免重复内容；错误交给调用方收敛 |
| 请求 Context 已取消 | `TestChatAdapterStreamRejectsCanceledContextBeforeProviderCall`、`TestChatAdapterStreamStopsWhenConsumerDisconnects` | 取消向下游传播，不把取消误判为 Provider 故障 |
| 连续可重试 Provider 错误 | `TestChatAdapterCircuitBreakerFailsFastAfterRetryableFailures`、`circuitbreaker_test.go` | 有界重试后熔断，只允许单次 half-open 探测 |
| Worker 重复领取、租约失效或处理失败 | `app/infra/task/article_index_worker_test.go`、`persistent_article_index_backfill_test.go` | 由数据库租约、条件更新、重试和死信处理，不依赖 LangGraph |
| 审核动作重复提交或绑定错误 | `app/infra/persistence/repository/ai_review_repository_test.go` | 使用动作幂等键、会话/目标/摘要绑定和条件事务更新拒绝重放 |

这些测试证明当前控制措施的确定性行为，不等同于生产规模故障演练；跨进程崩溃、
真实网络断连和多实例恢复仍属于 M9/M8 触发后的发布门禁。

## LangGraph 触发条件

必须先提交可复现的故障报告，至少包含 workflow 版本、状态、重启点、恢复目标、
副作用和当前 Eino/应用层的失败证据。满足以下任一条件，才允许启动 M8-02：

- 同一类跨进程暂停恢复故障在隔离演练中连续复现，且现有持久状态机无法在既定恢复
  时间内恢复到幂等安全状态；
- 审核流程确实需要 Graph 内的多次人工 interrupt、回退、重规划和并行子 Agent，
  现有 Job/Review 状态机无法保持可审计的版本和副作用边界；
- 必须依赖仅存在于 Python/LangGraph 生态、且无法由受限 Go internal API 替代的能力；
- 量化评测显示独立 sidecar 在恢复成功率或维护成本上有明确、可复核的收益。

单次 Provider 超时、普通重试、SSE 断开、审核重复请求或 Worker 失败不能单独触发
LangGraph 评估；这些场景已有 Eino ACL、Context、租约和幂等控制。

## 触发后的评估约束

若触发条件成立，POC 仍必须满足：

1. sidecar 不直连 PostgreSQL、Redis、Meilisearch 或 MinIO，只调用带服务身份认证的
   受限 Go internal API；
2. checkpoint、interrupt、人工恢复、幂等副作用和服务重启各有独立故障用例；
3. 记录 Eino 主线与 sidecar 的恢复成功率、P95 延迟、部署复杂度、观测覆盖、权限
   面和故障回滚成本；
4. POC 默认关闭且可删除，不把 LangChain/LangGraph 类型或协议泄漏到 Go application；
5. 只有比较结果明确优于现有 Eino + 持久状态机组合，才另行提交生产 ADR。

## 后果

- 当前部署面保持单一 Go/Eino 主线，不增加 Python 运行时、容器或跨服务网络；
- 需要长流程时，优先扩展已有持久状态、租约、事件和审核模型，而不是把状态藏进
  编排框架；
- 真实故障演练的成本会延后，但触发条件、证据格式和比较指标已经明确；
- M8-02～M8-08 继续保持未启动，直到触发条件和独立评估结果满足要求。
