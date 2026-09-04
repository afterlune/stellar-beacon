# Benetnasch 数字空间当前计划

> 当前唯一产品与实施基线：2026-09-04。代码、配置、测试和发布证据优先于本文；本文只记录当前目标、边界和未完成工作。

## 产品目标

Benetnasch 不再以“博客”作为产品中心，而是一个数字空间：保存文章、图片、视频、梦境、电台和其他可公开内容，并向居民和用户提供安全的 Agent 能力。

数字空间本身没有人格和生命。Eino 只负责空间侧的通用 Agent 能力，例如搜索、内容理解、写作辅助、任务和审核。

Companion 项目本身就是月社妃数字生命：负责她的对话、人格、Canon、私有记忆、语音、节律和自主行为。她以数字空间中的独立居民身份存在，不能被实现成 Benetnasch 内部的另一套人格服务。

## 固定架构边界

### 数字空间平面

- Go/Gin/xorm/PostgreSQL/Redis/Casbin 负责空间事实、权限、审计和发布。
- Eino、Provider 和 Meilisearch 留在 `infra`，通过 domain port 被 application 使用。
- 空间 Agent 是非人格化、可审计的能力，不拥有月社妃的记忆或身份。
- 公开检索只返回已发布内容；草稿、账号、日志、凭据和系统配置不属于公开知识。

### 数字生命平面

- `D:\Git\companion` 保持独立项目、独立入口、独立登录和独立 SQLite 记忆。
- 月社妃拥有独立的 Agent 账号/主体 `moonfei`，不复用人类用户账号和密码。
- Companion 通过空间 API 像用户一样读取公开内容和提交允许的发布动作。
- Companion 不直连 Benetnasch 的 PostgreSQL、Redis、Meilisearch 或 MinIO，也不调用 Benetnasch 的 Eino Agent 作为自身人格引擎。

## 当前实施顺序

1. 在 `dev` 分支固定本计划和 ADR，清理旧部署说明、数据库导出物和旧剩余清单。
2. 增加 `moonfei` Agent 主体、固定机器作用域和审计身份；人类后台 RBAC 继续由 Casbin
   负责，不能把内部服务令牌伪装成后台用户。
3. 增加只读空间知识 API，供 Companion 查询已发布内容。
4. 增加受限追加式发布 API，仅允许状态、梦境和电台。
5. 在开发副本启动独立 Companion，完成登录、搜索、引用、发布和拒绝路径联调。
6. 完成开发环境证据后，再单独申请生产迁移、发布、索引和 Caddy 操作。

## 执行状态（2026-09-04）

| 项目 | 状态 | 证据或剩余动作 |
| --- | --- | --- |
| `dev` 分支、当前计划/ADR、旧文档与导出物清理 | 已完成 | 当前仓库分支为 `dev`；旧部署说明、SQL 导出物和旧剩余清单已删除 |
| `agent/moonfei` 主体与审计事实 | 代码完成 | `0023_space_companion.sql` 创建主体和发布表；重复迁移不会重新启用被禁用主体 |
| 主体授权 | 代码完成 | 独立 read/publish token、数据库主体 `enabled` 和 `space:*` 作用域；不复用人类 Casbin 会话 |
| 只读公开知识 API | 代码完成 | 版本化 `/internal/space/v1` API、公开投影、正文/类型/请求上限和单元测试 |
| 受限追加发布 API | 代码完成 | 仅 `status`/`dream`/`radio`，HTTPS 媒体、来源、幂等、冲突和审计表均已实现 |
| Companion 独立客户端与工具 | 代码完成 | 独立配置、HTTP 客户端、读写令牌隔离、工具白名单和 Python 测试已加入 |
| 隔离 Caddy listener | 配置完成 | dev/integration 使用宿主 `28028`/`18028`，生产 Caddy 未修改 |
| 隔离数据库迁移、数据复制和真实浏览器联调 | 待执行 | 需要按隔离 Compose 运维授权执行；当前未以 fake 或静态检查冒充真实证据 |
| 生产迁移、索引/Caddy 切换和现有容器操作 | 明确不执行 | 不属于本阶段，也未获得生产发布窗口授权 |

## 发布权限

月社妃可以自主追加发布：

- `status`：状态、心情、活动记录；
- `dream`：梦境；
- `radio`：电台节目。

发布边界由 Benetnasch 的确定性策略和 feature flag 控制，模型不能扩大白名单。文章、视频、评论、时间胶囊、人设 Prompt、账号、权限和配置不能自动发布、修改或删除。

所有跨服务写入都必须带来源会话、来源运行 ID 和幂等键；重复请求只能返回原结果，不能产生重复公开内容。

## 明确不做

- 不在 Companion 内嵌第二套 Eino Agent。
- 不把 Companion Python/FastAPI 运行时复制到 Go 服务。
- 不用 Eino 重写月社妃的人格、记忆、语音和生命状态。
- 不引入 pgvector、FAISS、LangGraph 或 LangChain 作为本阶段必要依赖。
- 不在生产环境自动迁移、切换索引、切换 Caddy 静态目录或修改现有容器。

## 验收标准

- [x] `moonfei` 身份可被明确审计，不能冒充人类用户。
- [x] Companion 能通过受限协议查询已发布文章、图片、视频、梦境和电台，并携带稳定引用。
- [x] Companion 无法通过该协议读取草稿、日志、凭据和后台管理数据。
- [x] 状态、梦境、电台发布具备限流、校验、幂等和审计。
- [x] 文章、视频、删除、修改和权限操作被协议拒绝。
- [ ] `benetnasch-dev` 与独立 Companion Compose 的真实浏览器联调通过。
- [x] Go 全量测试、边界扫描、`go vet` 和 `git diff --check` 通过；Companion 测试需在具备 Python 运行时的环境执行。

## 运维状态

- 当前开发分支：`dev`。
- 远程目标：`origin/dev`。
- 当前生产 `benetnasch` 与 `main` 不属于本计划的执行对象。
- 外部 CodeBuddy plan 和 `.codebuddy/` 仅作第三方评估资料，不作为项目当前事实来源。
