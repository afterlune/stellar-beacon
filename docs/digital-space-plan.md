# Benetnasch 数字空间当前计划

> 唯一产品与实施基线：2026-09-05。代码、配置、测试和真实运行证据优先于本文；本文只记录当前目标、固定边界和剩余工作。

## 产品目标

Benetnasch 是数字空间：保存文章、图片、视频、梦境、电台和其他可公开内容，并向用户和空间居民提供安全、可审计的 Agent 能力。

数字空间本身没有人格。Eino 负责搜索、内容理解、写作辅助、任务、审核等通用空间能力；它不拥有任何居民的 Canon、私有记忆或身份。

月社妃是独立 Companion 项目的数字生命，负责对话、人格、Canon、私有记忆、语音、节律和自主行为。她以 `agent/moonfei` 居民身份访问数字空间，而不是被重写成 Benetnasch 内部的一套人格服务。

## 固定边界

### 数字空间

- Go/Gin/xorm/PostgreSQL/Redis/Casbin 保存空间事实、权限、审计和发布策略。
- Eino、模型 Provider、Meilisearch、Redis 和对象存储适配器位于 `infra`，通过 `domain/port` 供 application 使用。
- 公开检索只返回已发布内容；草稿、账号、日志、凭据、Prompt 和系统配置永不进入公开知识。
- Agent 生成内容默认进入审核队列；模型不能决定身份、权限、审核结果、发布动作或计费规则。

### Companion 居民

- `D:\Git\companion` 保持独立项目、入口、登录、SQLite 记忆和运行时。
- 月社妃不复用人类用户账号和密码，不直连 Benetnasch PostgreSQL、Redis、Meilisearch 或 MinIO。
- Companion 只通过版本化空间协议读取已发布内容，并提交受限的追加发布动作。
- 第一阶段自主发布白名单只有 `status`、`dream`、`radio`；文章、视频、评论、时间胶囊、权限和配置禁止自动修改或删除。

## 当前实施顺序

1. 在 `dev` 分支维护本计划、ADR、协议和运行手册，统一数字空间的产品和技术叙事。
2. 维护独立的 `agent/moonfei` 主体、机器作用域、来源会话和审计事实。
3. 通过只读空间 API 提供已发布内容、有限正文和稳定引用。
4. 通过受限发布 API 接收状态、梦境和电台的追加内容，执行校验、限流、幂等和审计。
5. 在隔离 `benetnasch-dev` 与 Companion Compose 中完成真实迁移、数据准备、浏览器和工具联调。
6. 只有隔离证据完整后，才另行申请生产迁移、索引切换、静态目录切换或其他发布窗口操作。

## 执行状态（2026-09-05）

| 工作项 | 状态 | 事实与剩余动作 |
| --- | --- | --- |
| `dev` 分支和当前文档基线 | 已完成 | 本次文档重建已落在 `dev`；`main` 和生产链路不属于本计划 |
| `agent/moonfei` 主体与审计事实 | 代码完成 | 迁移 `0023_space_companion.sql` 创建主体和发布表；重复迁移不会重新启用被禁用主体 |
| 独立机器授权 | 代码完成 | read/publish token、主体 `enabled` 和 `space:*` 作用域分离，不借用人类 Casbin 会话 |
| 只读公开知识 API | 代码完成 | `/internal/space/v1` 提供能力、搜索和内容读取，带公开性过滤和请求上限 |
| 受限追加发布 API | 代码完成 | 仅允许 `status`/`dream`/`radio`，支持媒体校验、来源、幂等、冲突和审计 |
| Companion 独立客户端与工具 | 代码完成 | 独立配置、读写 token 隔离、工具白名单和测试已加入 Companion |
| `benetnasch-dev` 启动 smoke | 已验证 | 隔离前后台、后端、PostgreSQL、Redis、Meilisearch、MinIO 和 Caddy 已启动，公开 API 返回隔离数据 |
| `0023` 隔离迁移与明确数据复制 | 待执行 | 当前启动使用已有隔离卷；尚未把生产数据复制流程和新迁移作为本次文档变更的一部分执行 |
| Companion 真实浏览器/工具联调 | 待执行 | 需要 Companion Python 运行时、bridge 开关、独立 token 和完整隔离 E2E 证据 |
| 生产迁移、索引、Caddy 和现有容器操作 | 明确不执行 | 需要单独发布窗口和备份/回滚证据 |

## 空间协议

```text
GET  /internal/space/v1/capabilities
POST /internal/space/v1/search
GET  /internal/space/v1/content/{type}/{id}
POST /internal/space/v1/publications
```

读取和发布使用不同服务 token。客户端传入的 `actor_id` 不可信，空间侧根据认证主体确定身份；发布采用追加式语义，不提供 Companion 更新和删除接口。

## 验收标准

- [x] 月社妃是可审计的独立 `agent` 主体，不能冒充人类用户。
- [x] 协议只返回已发布内容和稳定引用，不能读取草稿、日志、凭据或后台数据。
- [x] 状态、梦境、电台发布具备白名单、限流、校验、幂等和审计。
- [x] 文章、视频、评论、删除、修改和权限操作被协议拒绝。
- [x] Go 测试、边界扫描、`go vet` 和文档安全检查通过。
- [ ] 隔离数据库迁移、数据复制和完整 Companion Compose/浏览器联调通过。
- [ ] Companion Python 测试在具备 Python 运行时的环境通过。

## 明确不做

- 不在 Benetnasch 内嵌月社妃的第二套人格、记忆、语音或生命状态。
- 不把 Companion Python/FastAPI/SQLite 运行时复制到 Go 服务。
- 不以 Eino、LangChain 或 LangGraph 替代 Companion 的人格运行时。
- 不在本阶段引入 pgvector、FAISS 或新的向量数据库容器。
- 不在生产环境自动迁移、切换索引、切换 Caddy 静态目录或修改现有容器。

## 运维事实

- 当前执行分支：`dev`；远程目标：`origin/dev`。
- `benetnasch-dev` 使用独立端口和命名卷；生产 Compose 项目名为 `benetnasch`，不得混用。
- AI、Vision、Embedding、空间桥接和自主发布开关默认关闭，开启必须记录环境、token 来源和回滚动作。
- `.codebuddy/` 仅是第三方评估资料，不是项目事实来源，也不参与文档维护。
