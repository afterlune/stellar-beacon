# Agent 控制面资源与授权

本文说明数字空间 Agent 能力的后台资源、Casbin 角色和 Companion 机器主体边界。资源迁移不会在服务启动时自动执行；只有代码和 API 已就绪后，才在明确授权的数据库窗口按版本顺序执行。

## 两套身份平面

### 人类控制面

人类管理员通过后台会话、角色和 Casbin 资源访问控制面。前端菜单只是导航，不能代替后端授权。AI Studio、Provider 烟测、审核、记忆和观测资源均受管理员认证、资源路径匹配和审计保护。

### Companion 机器主体

`/internal/space/v1/*` 不属于人类后台资源，不借用 Casbin 会话，也不把 `moonfei` 映射成某个用户。它使用独立 read/publish token、`t_agent_principal` 和 `space:*` 作用域：

- `space:read`：能力、公开搜索和公开内容读取；
- `space:publish`：受限的状态、梦境和电台追加发布；
- 主体必须是 `agent/moonfei`、处于 enabled 状态且作用域匹配；
- token、主体缺失或迁移未完成时必须 fail closed。

## 公开 Agent 资源

| 路径 | 方法 | 身份 | 说明 |
| --- | --- | --- | --- |
| `/agent/features` | GET | feature flag | 查询公开能力状态 |
| `/agent/chat` | POST | public flag | 受限 SSE 对话，默认关闭 |
| `/agent/vitals` | GET | public flag | 空间 Agent 生命体征 |
| `/galaxy` | GET | public flag | 空间视图数据 |
| `/dreams`、`/radio`、`/videos` | GET | public flag | 已发布公开内容 |
| `/agent/sessions/:id` | DELETE | 会话身份 | 删除访客会话 |
| `/agent/sessions/:id/events` | GET | 会话身份 | 回放事件 |

公开 Agent 没有文章、评论、用户、权限、配置或发布写工具。所有模型输出都必须经过应用层策略和审核边界。

## 管理资源

| 能力 | 主要路径 | 默认状态 |
| --- | --- | --- |
| Provider 烟测 | `/admin/ai/providers/test` | admin，独立开关 |
| 写作预览 | `/admin/ai/writing/preview` | admin，生成待审记录 |
| Vision 预览 | `/admin/ai/vision/preview` | admin，生成待审记录 |
| 审核队列 | `/admin/ai/reviews/*` | admin，条件更新和幂等 |
| Agent 人设 | `/admin/ai/profile` | admin，只能修改受控配置 |
| 审核策略 | `/admin/ai/review-policy` | admin，版本化 |
| 记忆审核 | `/admin/ai/memory/*` | admin，持久化开关独立控制 |
| 运行时观测 | `/admin/ai/observability` | admin，只读脱敏指标 |
| 紧急开关 | `/admin/agent/emergency` | admin，优先关闭公开能力 |

观测和错误响应不能返回 Prompt、请求正文、Provider 凭据、搜索词、访客身份或数据库参数。

## 迁移顺序

Agent 控制面资源由对应迁移按顺序创建。新增资源前必须确认父节点、API、前端组件和 feature flag 已存在；迁移只增加显式授权，不自动启用公开能力。

Companion 主体和发布事实由 `0023_space_companion.sql` 创建，包含 `t_agent_principal` 与 `t_space_publication`。该迁移不授予人类后台角色权限，也不会在启动时自动运行。

## 匹配规则

- 先兼容既有 Casbin wildcard 策略，再校验 `t_resource` 与 `t_role_resource` 的显式绑定；
- 资源路径按分段匹配，避免 `/reviews/*/approve` 放行其他审核动作；
- 已禁用角色和主体不得获得访问权；
- 前端隐藏菜单、客户端提交的角色和请求体中的身份字段都不可信；
- 每次授权失败只返回稳定通用错误，并记录脱敏审计事实。

## 验证

验证必须覆盖：未登录、普通用户、管理员、禁用角色、未知路径、路径边界、token 读写隔离、主体禁用、feature flag 关闭和错误响应脱敏。真实数据库迁移和后台 E2E 只在隔离 Compose 或独立发布窗口执行。
