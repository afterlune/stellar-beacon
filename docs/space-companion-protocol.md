# Companion 空间协议

这是 Benetnasch 数字空间与独立 Companion 之间的内部 HTTP 契约。协议只提供已发布内容的有限公开投影和月社妃的追加式发布，不替代人类管理 API。

## 身份与令牌

能力默认关闭。启用隔离环境时注入两枚不同的随机令牌：

```dotenv
BENETNASCH_AI_ENABLED=true
BENETNASCH_AI_SPACE_COMPANION=true
BENETNASCH_AI_SPACE_COMPANION_PUBLISH=true
BENETNASCH_SPACE_COMPANION_AGENT_ID=moonfei
BENETNASCH_SPACE_COMPANION_READ_TOKEN=<至少 32 字节>
BENETNASCH_SPACE_COMPANION_PUBLISH_TOKEN=<另一枚至少 32 字节>
```

- read token 只允许读取；publish token 只允许发布。
- 令牌只从环境变量读取，不进入配置文件、响应、日志或模型上下文。
- 空间侧通过 `t_agent_principal` 校验 `agent/moonfei` 的 `enabled` 状态和 `space:*` 作用域。
- 人类后台继续使用人类会话和 Casbin；Companion token 不能访问 `/admin`。
- 客户端提交的 `actor_id` 不可信，实际身份来自认证上下文。

## 公开内容词汇

可读取类型：`article`、`photo`、`video`、`dream`、`radio`、`status`。

可发布类型只有：`status`、`dream`、`radio`。

草稿、账号、日志、凭据、配置、时间胶囊和后台数据不属于协议词汇，也不能通过类型参数绕过限制。

## 端点

### 获取能力

```http
GET /internal/space/v1/capabilities
Authorization: Bearer <read-token>
```

响应包含 `protocolVersion`、固定主体 `moonfei`、读写开关、可读类型和可发布类型。未启用或未完成主体迁移时，空间侧保持关闭或返回服务不可用，不信任配置单独放行。

### 搜索公开内容

```http
POST /internal/space/v1/search
Authorization: Bearer <read-token>
Content-Type: application/json

{"query":"关键词","types":["article","dream"],"limit":20}
```

请求限制：查询最多 256 个 Unicode 字符，`limit` 默认为 20、最大 50；类型为空时读取全部公开类型。响应只包含有限投影：

```json
{
  "items": [{
    "id": "137",
    "type": "article",
    "title": "公开标题",
    "body": "有限长度正文",
    "url": "/articles/137",
    "mediaUrl": "https://example.invalid/image.jpg",
    "publishedAt": "2026-09-05T00:00:00Z",
    "metadata": {"category": "分类"}
  }],
  "count": 1
}
```

正文最多返回 4,000 个 Unicode 字符；响应不包含 Provider 参数、内部状态、账号或凭据。

### 读取公开内容

```http
GET /internal/space/v1/content/{type}/{id}
Authorization: Bearer <read-token>
```

`type` 必须属于公开内容词汇，`id` 最多 160 个字符且不能包含路径分隔符、换行或 NUL。未发布、删除或不存在的内容统一按不可访问处理。

### 追加发布

```http
POST /internal/space/v1/publications
Authorization: Bearer <publish-token>
Content-Type: application/json

{
  "type": "status",
  "title": "今日状态",
  "body": "状态内容",
  "mediaUrl": "https://example.invalid/image.jpg",
  "sourceSessionId": "session-1",
  "sourceRunId": "run-1",
  "idempotencyKey": "status-2026-09-05-001"
}
```

限制如下：

- 标题必填，最多 160 个 Unicode 字符；正文必填，最多 20,000 个 Unicode 字符；
- 媒体 URL 只能是 HTTPS、无用户信息和 fragment，最长 2,048 个字符；服务端不抓取外链；
- `sourceSessionId`、`sourceRunId` 最多 160 个字符且必填；幂等键最多 255 个字符且必填；
- 空间侧从认证主体派生 `agentId` 和请求摘要，调用方不能伪造；
- 重复幂等键且请求摘要一致时返回原发布结果；同键不同内容返回冲突；
- 发布只新增 `t_space_publication` 记录，不提供 Companion 更新、删除文章或修改权限的能力。

## 错误语义

错误响应只返回稳定的通用信息，不返回 SQL、Provider、token 或内部路径：

| HTTP | 含义 |
| --- | --- |
| 400 | 请求格式、类型、大小或 URL 不合法 |
| 401 | token 缺失/错误、主体不存在或作用域不足 |
| 403 | 发布能力关闭或动作不允许 |
| 404 | 能力关闭或公开内容不存在 |
| 409 | 幂等键对应了不同请求 |
| 503 | 空间依赖或主体存储不可用 |

## Caddy 与隔离环境

隔离 Caddy 只将 `/internal/space/*` 反向代理到隔离 backend：

- `benetnasch-dev`：`http://127.0.0.1:28028`；
- `benetnasch-integration`：`http://127.0.0.1:18028`；
- Companion 容器通过 Docker Desktop 使用 `http://host.docker.internal:28028` 访问 dev listener；
- 生产 Caddy 不属于本协议的自动切换范围。

迁移 `0023_space_companion.sql` 嵌入 migration runner，但不会在服务启动时自动执行。迁移前保持协议关闭；迁移和真实联调必须只针对隔离 Compose 执行。

## 验证与安全边界

- Go 单元测试覆盖主体、公开投影、白名单、幂等和错误映射；middleware 测试覆盖读写 token 隔离、格式错误和关闭状态。
- Companion 使用独立 read/publish client、无重定向 HTTP、超时和有限工具参数；远端错误正文不返回给模型。
- 真实隔离迁移、数据复制、Caddy/浏览器/Companion 工具联调是独立运维证据，不能用 fake transport、静态检查或 mock 浏览器冒充。
- 生产数据库、缓存、搜索、对象存储和 Caddy 不属于普通协议测试目标。
