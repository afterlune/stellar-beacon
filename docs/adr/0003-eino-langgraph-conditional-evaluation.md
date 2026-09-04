# ADR-0003：Eino 主线与 LangGraph 条件评估

- 状态：已接受，LangGraph 延后
- 日期：2026-09-05

## 决策

Benetnasch 当前只使用 Eino 作为 Go 服务内数字空间 Agent 的编排主线。不引入 LangGraph、LangChain、Python sidecar 或新的生产容器。

请求级的检索、Prompt、模型调用、流式输出、重试、熔断和取消由 Eino、Provider ACL 和 application port 共同完成；跨请求/跨进程的任务、审核、租约、事件、游标和副作用由 PostgreSQL、Redis 和持久 Worker 状态机完成。

只有真实故障演练证明现有 Eino + 持久状态机无法满足跨进程暂停恢复、人工中断恢复或多 Agent 长流程的量化要求时，才允许启动独立 LangGraph sidecar POC。POC 不能直接进入生产链路。

## 当前边界证据

| 能力 | 当前实现 | 结论 |
| --- | --- | --- |
| 固定请求级 RAG | Eino Graph 编排标准化、检索、上下文、Prompt、生成和结果 | 已满足请求级流程 |
| Provider 可靠性 | 有界重试、并发闸门、Context 取消、流式边界和 circuit breaker | 不需要第二套编排框架 |
| 跨进程任务 | 持久 Job、租约、checkpoint、重试、死信和 supervisor | 状态在 Eino 外部保存 |
| 审核和副作用 | 审核记录、动作幂等、目标/摘要绑定和条件更新 | 人工介入不依赖进程内 Graph |
| 当前缺口 | 固定 RAG Graph 没有自身的跨进程 checkpoint 与 human interrupt/resume | 只有真实长流程需求出现后再评估 |

普通 Provider 超时、SSE 断开、审核重复提交或 Worker 失败不能单独触发 LangGraph 评估；这些场景已有 Eino ACL、Context、租约、重试和幂等控制。

## 触发条件

必须先提交包含 workflow 版本、状态、重启点、恢复目标、副作用和现有实现失败证据的可复现故障报告，并满足以下至少一项：

- 同类跨进程暂停恢复故障在隔离演练中连续复现，且现有状态机无法在既定时间内恢复到幂等安全状态；
- 审核流程确实需要 Graph 内多次人工 interrupt、回退、重规划和并行子 Agent，现有 Job/Review 状态机无法保持版本和副作用边界；
- 必须依赖仅存在于 Python/LangGraph 生态、且无法由受限 Go internal API 替代的能力；
- 量化结果显示 sidecar 在恢复成功率或维护成本上有明确、可复核的收益。

## POC 约束

触发后仍需单独立项：

1. sidecar 不直连 PostgreSQL、Redis、Meilisearch 或 MinIO，只调用带服务身份认证的受限 Go internal API；
2. checkpoint、interrupt、人工恢复、幂等副作用和服务重启分别设计故障用例；
3. 对比 Eino 主线和 sidecar 的恢复成功率、P95、部署复杂度、观测覆盖、权限面和回滚成本；
4. POC 默认关闭、可删除，LangGraph 类型和协议不能泄漏到 Go application；
5. 只有比较结果明确优于现有组合，才新增生产 ADR。

## 结果

当前部署面保持单一 Go/Eino 主线。需要长流程时先扩展已有持久状态、租约、事件和审核模型；LangGraph 相关工作保持延期，不作为当前数字空间启动条件。
