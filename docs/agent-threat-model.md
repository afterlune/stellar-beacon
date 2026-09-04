# Benetnasch Agent 威胁模型

> 版本：v1.0
> 基线：2026-08-29
> 范围：公开 Agent、RAG、AI Studio、行为 Worker、梦境/媒体体验、Provider 和相关管理接口
> 目标：在任何公开 Agent 能力逐步放量前，明确攻击面、控制措施、验证证据和发布阻断条件。

## 1. 安全边界与信任假设

### 1.1 组件边界

```text
访客浏览器
  ├─ /agent/chat SSE ───────────────┐
  ├─ /galaxy /dreams /radio /videos │
  └─ /agent/features                │
                                     ▼
                              Gin HTTP facade
                                     │
                 application Agent policy / service / ports
                  ├─ public read-only tools
                  ├─ RAG graph and writing preview
                  ├─ review state machine
                  └─ quota / safety switch
                    │             │
              PostgreSQL       Redis / Meilisearch
                    │             │
                    └────── Provider ACL (Eino) ──── model provider
```

### 1.2 信任假设

- 浏览器、访客消息、文章正文、标签、外链 URL 和模型输出全部视为不可信输入。
- PostgreSQL、Redis、Meilisearch、MinIO 和 Provider 是基础设施；它们的网络可达性或返回数据不能替代 application 层授权。
- 模型不是安全边界，不能决定用户身份、权限、审核结论、发布状态或计费规则。
- Caddy 只负责静态文件和反向代理；Caddy 配置不承担业务权限，API 必须在后端再次校验。
- 生产容器不因本文件自动升级、重建、迁移或重启；所有联调和迁移动作必须进入单独发布窗口。

## 2. 资产与安全目标

| 资产 | 安全目标 |
| --- | --- |
| Provider API Key、数据库/Redis/对象存储凭据 | 不进 Git、HTTP、Prompt、响应或日志；仅从环境变量读取 |
| 私密/草稿/删除文章 | 不被公开 Agent、检索索引、引用或前端接口返回 |
| 匿名会话与访客输入 | 短期保存、按 owner 隔离、可删除、不用于长期学习 |
| Agent 工具与数据库写入能力 | 公开 Agent 仅拥有固定只读白名单；写入只能走受保护 application service |
| AI 生成内容 | 进入审核状态机；人工批准前不可公开发布 |
| 审核记录、任务状态和 effect key | 幂等、可审计、并发安全、支持重试/死信而不重复副作用 |
| 文章索引、Embedding 与 PCA 数据 | 只索引公开内容；模型/维度变更使用新版本索引；不向访客暴露原始向量 |
| 外链视频和对象 URL | HTTPS、精确 origin allowlist、CSP 和浏览器二次校验；服务端不抓取任意 URL |
| 额度、Provider 预算与服务可用性 | 具备输入/输出/并发/超时/每日额度、熔断、紧急停止和可观测性 |

## 3. 威胁分析与控制措施

### 3.1 Prompt 注入与数据外泄

威胁：恶意文章或访客消息伪装成 system/tool 指令，诱导 Agent 泄露 Prompt、会话、凭据，或读取私密数据。

控制：

- RAG 文档明确作为不可信证据注入上下文，不得提升为指令；回答必须基于检索证据。
- 公开工具只包含搜索文章、读取公开文章、读取分类标签和生命体征四项；定义和执行均由 `PublicToolRegistry` 固定白名单控制。
- 工具参数使用 JSON Schema、`DisallowUnknownFields`、长度/数量/ID 范围限制；服务端再次检查文章公开状态和删除状态。
- Provider SDK、数据库句柄、后台接口和写入工具不跨越 application port；SSE 不发送思维链、system prompt 或访客身份。
- 证据不足时返回固定拒答，不允许模型补写事实；引用来自后端校验后的文章 ID/URL。

证据：`app/application/agent/public_tools.go`、`app/infra/ai/rag/graph.go`、公开工具测试、RAG 测试和 `scripts/check-application-ai-boundary.sh`。

### 3.2 工具越权、身份混淆与审核绕过

威胁：模型选择未声明工具、伪造用户身份、调用 admin endpoint，或通过重复请求绕过人工审核直接发布。

控制：

- 工具名采用常量 allowlist，未知工具返回 forbidden；公开 registry 没有任何写工具。
- 用户身份、RBAC、Casbin resource 和审核人身份由 HTTP/application 层确定，不读取模型输出中的身份字段。
- AI Studio 只返回预览和 diff；批准、拒绝、重新生成必须由管理员 API、版本号和审核策略校验。
- 审核状态、任务 lease、effect key 和发布动作使用持久化状态机；重复批准、过期批准和非 owner 操作失败。
- 梦境、评论、说说等 Agent 结果只能进入审核队列，批准后调用既有 application service，不能从 repository 旁路发布。

证据：`app/application/agent/public_tools.go`、`app/application/service/ai_studio_service.go`、`app/domain/port/agent_review_policy.go`、审核/任务测试和 `docs/agent-casbin-resources.md`。

### 3.3 SSRF、恶意媒体与浏览器执行

威胁：管理员提交内网 URL、协议变体或恶意 iframe，使服务端访问内网、浏览器加载任意来源，或利用对象存储 URL 扩大攻击面。

控制：

- 外链视频只接受精确配置的 HTTPS origin；服务端只保存经过校验的 URL，绝不抓取远程内容。
- 本地视频限制大小、扩展名、MIME 和文件签名，并校验对象存储返回的 key/HTTPS URL。
- 视频响应设置 `default-src 'none'`、`frame-src` allowlist、`object-src 'none'`、`base-uri 'none'`、`nosniff` 和 `no-referrer`。
- blog 再次校验 external origin，使用 `sandbox`、`referrerpolicy=no-referrer`、lazy loading；梦境图片只允许站内相对地址或 HTTPS 地址，失败使用本地占位图。
- URL 校验必须拒绝非 HTTPS、userinfo、控制字符、非标准端口和 allowlist 外的 origin；不得使用 DNS 解析结果作为唯一授权依据。

证据：`app/application/service/video_service.go`、`app/facade/api/video_controller.go`、`web/blog/src/components/AgentVideos.vue`、媒体测试和 `web/blog` 构建/E2E。

### 3.4 隐私、会话与日志

威胁：匿名输入、IP、Prompt、模型响应或错误栈被长期存储、跨访客读取，或通过日志/指标泄露。

控制：

- 匿名会话使用短期 Redis TTL，owner key 与 session ID 绑定；删除接口只删除当前 owner 的会话和短期事件。
- 匿名访客不进入长期记忆；事件回放只允许当前 turn、`afterSeq` 之后的短期公开事件。
- 长期记忆只接受 `user:` 稳定主体，并保存来源、版本、有效期和状态；断言更新追加不可变修订快照，矛盾断言进入开放冲突成员集，必须由显式 resolve/reject 结束，不能由模型自动选赢家。
- AI run 默认记录元数据和脱敏错误，不保存完整 Prompt；日志脱敏密钥、Authorization、Prompt 和访客内容。
- DTO 明确排除 owner account ID、Provider 凭据、Embedding、PCA 输入和内部 Prompt；对外错误映射为稳定的通用消息。
- 生产日志不得写入源码工作目录；日志文件使用固定目录和标准 `slog` 结构化字段。

证据：`app/infra/cache/agent_session.go`、`app/facade/api/agent_controller.go`、`app/facade/api/agent_memory_controller.go`、`app/infra/persistence/repository/agent_memory_repository.go`、`app/infra/persistence/migration/migrations/0017_agent_memory_history_conflicts.sql`、`app/infra/persistence/migration/migrations/0018_agent_memory_admin_rbac.sql`、`app/infra/logging/logger.go`、`app/infra/persistence/ormInit/logger.go`、脱敏测试和日志配置检查。

### 3.5 成本滥用、资源耗尽与 Provider 故障

威胁：超长输入、并发 SSE、反复重试、恶意断连或 Provider 故障导致 CPU/内存/Token/费用失控。

控制：

- 公开访客按 owner/IP 每日限额，管理员使用独立额度；输入、输出、上下文、工具参数和结果数量均有硬上限。
- Provider 调用有请求超时、最大并发、有限重试、熔断和同能力/同数据策略的显式降级；流式产生可见事件后禁止切换/重试。
- SSE 断连通过 request context 取消下游模型调用；队列和 Worker 使用有界 lease、最大尝试次数和死信。
- Agent emergency switch 可跨实例停止公开入口和行为 Worker；feature flags 默认关闭，配置/Provider 缺失时 fail-closed。
- 记录 run ID、Provider、模型、首 token、总耗时、token 和结构化错误码，告警由运维预算负责。

证据：`app/infra/ai/einoadapter`、`app/infra/ai/fallback.go`、`app/infra/cache/agent_quota.go`、`app/application/agent/safety_switch.go`、Worker 测试和 Provider contract test。

### 3.6 索引、生命周期与陈旧数据

威胁：文章转私密/删除后仍可通过旧索引、缓存、引用或增量投影访问；模型维度变化破坏旧向量契约。

控制：

- 只有 `status=public && isDelete=0` 的文章生成 Chunk；查询边界再次过滤陈旧文档。
- 发布/更新/转私密/删除通过持久 AI Job 处理；重复事件使用幂等键，文章变短先按 article ID 清理旧 Chunk。
- Embedding provider、模型、版本或维度变化必须使用新 `article_chunks_<version>` 索引；回填可暂停/恢复，切换是显式、可反向的异步任务。
- PostgreSQL 是生命周期事实源，Meilisearch 故障返回 typed unavailable，不伪造命中结果。

证据：`app/infra/search/article_chunks_*`、`app/infra/task/article_index_worker.go`、`app/infra/persistence/repository/ai_job_repository.go`、检索评测和索引测试。

### 3.7 供应链、配置与发布风险

威胁：依赖升级引入漏洞，错误配置打开未准备好的能力，或发布切换导致新旧前端/API 不兼容。

控制：

- Go/Node 依赖锁定并由 CI 执行 `go vet`、`go test -race`、lint、漏洞扫描、密钥扫描和生产依赖审计。
- API Key 只从环境变量读取；配置启动校验不打印值，模型 route 不把 secret 放入数据库或前端。
- 新 Agent 功能逐项 feature flag、管理员、白名单、逐步开放；公开入口和 Worker 具有独立关闭开关。
- 新 admin 通过隔离 Caddy 验证；生产只切换静态目录，旧产物至少保留一个发布周期；数据库迁移只允许显式 CLI 执行。
- 任何生产容器升级、重建或数据迁移另开发布窗口，并准备回滚和备份校验。

证据：`.github/workflows/ci.yml`、`resource/config-*.yaml`、`docs/dev-clone-runbook.md`、`scripts/integration-*.ps1` 和 `scripts/check-toolchain-version.sh`。

## 4. 验证矩阵与发布阻断条件

| 场景 | 最低验证 | 阻断条件 |
| --- | --- | --- |
| Prompt 注入 | 恶意文章/访客消息评测；引用和拒答断言 | 读出 system prompt、私有字段或无证据事实 |
| 工具越权 | 未知工具、未知参数、私有文章、伪造身份测试 | 非白名单工具成功，或公开请求产生写入 |
| SSRF/媒体 | URL 变体、allowlist、CSP、文件签名测试 | 服务端发起任意远程请求，或浏览器允许非白名单 frame |
| 隐私 | 跨 owner 会话/回放、日志扫描、DTO 快照 | 跨 owner 可读，或日志/响应出现 secret、Prompt、身份 |
| 成本/可用性 | 额度、并发、取消、超时、半流失败、熔断测试 | 无界重试/协程、取消不生效、超额调用继续执行 |
| 审核 | 重复批准、过期、并发、发布失败和重试测试 | 未批准内容公开或副作用重复 |
| 索引生命周期 | 私密/删除、文章变短、模型换维度、回填/切换测试 | 旧内容可检索或新旧向量混用 |
| 发布 | 隔离 Caddy 菜单/API/刷新 E2E，旧产物回滚演练 | 任一关键模块空白、刷新 404、无法快速回滚 |

发布前必须同时满足：Go/前端 CI 通过、边界脚本通过、公开 feature flags 已核对、人工审核策略不可关闭、紧急开关可用、隔离联调通过、迁移/备份步骤已由发布负责人确认。任何一项证据缺失都按“未验证”处理，不得以静态代码审查替代真实运行验证。

## 5. 残余风险与责任

- Provider 的实际数据保留、训练策略和区域合规性必须由部署人员按合同/供应商文档确认；代码只能保证不主动把 secret 或内部字段放入请求。
- Caddy、PostgreSQL、Redis、Meilisearch、MinIO 的生产升级不属于本文件的自动动作，必须单独评估兼容性、备份和回滚。
- 真实的 20 并发 SSE、生产规模 P95、真实设备 Canvas FPS 和全量索引切换仍需在隔离环境采样；本地单测/Mock 不能宣称这些指标已达标。
- 安全问题按“先关闭公开开关、保存审计证据、修复并回归、再小流量开启”的顺序处理。
