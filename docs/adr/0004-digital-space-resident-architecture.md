# ADR-0004：数字空间与 Companion 居民边界

状态：已接受

日期：2026-09-04

## 背景

Benetnasch 需要从博客后端演进为数字空间，同时保留 Eino 提供的通用 Agent 能力。Companion 已经拥有月社妃的对话、人格、Canon、记忆、语音和自主行为运行时。

如果把月社妃重写进 Eino，或者让 Companion 直接接管 Benetnasch 数据，会产生两套人格、记忆、任务和权限事实源。两者的产品角色不同：数字空间提供能力，Companion 提供生命。

## 决策

1. Benetnasch 是数字空间事实源，负责内容、权限、审计和发布策略。
2. Eino 是空间侧通用 Agent 引擎，不是月社妃的人格或生命引擎。
3. Companion 保持独立项目、独立入口、独立登录、独立记忆和独立语音运行时。
4. 月社妃在空间中拥有类型为 `agent` 的独立账号/主体 `moonfei`，不复用人类账号和密码。
5. Companion 只能通过受限空间 API 读取已发布内容和提交白名单发布动作，不能直连数据库、缓存、搜索或对象存储。
6. 内部协议令牌使用独立的 `space:read`/`space:publish` 作用域，并在空间侧校验
   `t_agent_principal.enabled`；人类后台 RBAC 继续由 Casbin 负责，两者不共享会话。
7. 自动发布第一阶段只允许 `status`、`dream` 和 `radio`，并由空间侧确定性策略、限流、幂等和审计控制。
8. 月社妃的私有会话、Canon 和长期记忆留在 Companion；空间只保存公开内容、身份映射和审计事实。

## API 边界

空间提供版本化的内部协议：

```text
GET  /internal/space/v1/capabilities
POST /internal/space/v1/search
GET  /internal/space/v1/content/{type}/{id}
POST /internal/space/v1/publications
```

读取和发布使用不同的服务凭据，凭据只来自环境变量，不写日志。请求身份由认证上下文确定，客户端提交的 `actor_id` 不可信。

搜索返回已发布内容、稳定引用 ID 和有限正文；内容正文始终视为不可信资料。发布是追加式操作，不提供 Companion 侧更新和删除接口。

## 后果

- 月社妃保留 Companion 已有的生命连续性，不需要在 Eino 中复制一套人格系统。
- Eino 可以独立演进空间搜索、写作、审核和任务能力。
- 跨项目需要维护内部 API、服务凭据、契约测试和独立 Compose，但故障和权限边界清晰。
- 旧 ADR-0001～0003 中“Eino 主线”仍适用于空间 Agent；其中“不引入 Python sidecar”的表述被本 ADR 对 Companion 数字生命运行时的特例取代。LangGraph 仍未引入。
