# ADR-0004：数字空间与 Companion 居民边界

- 状态：已接受
- 日期：2026-09-05

## 背景

Benetnasch 是数字空间，需要保留 Eino 提供的通用 Agent 能力；Companion 已拥有月社妃的对话、人格、Canon、记忆、语音和自主行为运行时。

如果把月社妃重写进 Eino，或让 Companion 直接接管空间数据，就会产生两套人格、记忆、任务和权限事实源。数字空间提供能力和事实，Companion 提供生命连续性，两者必须分开。

## 决策

1. Benetnasch 是空间内容、权限、审计和发布策略的事实源。
2. Eino 是空间侧通用 Agent 引擎，不是月社妃的人格或生命引擎。
3. Companion 保持独立项目、入口、登录、记忆和语音运行时。
4. 月社妃在空间中拥有类型为 `agent` 的独立主体 `agent/moonfei`，不复用人类账号和密码。
5. Companion 只能通过受限空间 API 读取已发布内容和提交白名单发布动作，不能直连 PostgreSQL、Redis、Meilisearch 或 MinIO。
6. 内部协议令牌分为 `space:read` 与 `space:publish`，由空间侧校验数据库主体的 `enabled` 状态和作用域；人类后台仍由 Casbin 和人类会话控制，两者不共享会话。
7. 自动发布第一阶段只允许 `status`、`dream`、`radio`，并由空间侧确定性策略执行大小、媒体、来源、限流、幂等和审计检查。
8. 月社妃的私有会话、Canon 和长期记忆留在 Companion；空间只保存公开发布事实、身份映射和审计记录。

## API 边界

```text
GET  /internal/space/v1/capabilities
POST /internal/space/v1/search
GET  /internal/space/v1/content/{type}/{id}
POST /internal/space/v1/publications
```

读取和发布使用不同环境变量令牌。客户端传入的 `actor_id` 不可信，身份以认证上下文为准。读取返回有限公开投影；发布是追加式操作，不提供 Companion 侧更新和删除。

## 结果

- 月社妃保留 Companion 的独立生命连续性，不需要在 Eino 中复制人格系统。
- Eino 可以独立演进空间搜索、写作、审核和任务能力。
- 跨项目只维护版本化 HTTP 契约、服务凭据、契约测试和隔离 Compose。
- `t_agent_principal`、Casbin 人类角色和 Companion 私有记忆互不混用。
