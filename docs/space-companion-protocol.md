# Companion 空间协议

这是 Benetnasch 数字空间与独立 Companion 之间的内部 HTTP 契约。它只提供已发布内容的
有限公开投影和月社妃的追加式发布；不替代后台 API，也不允许 Companion 访问 PostgreSQL、
Redis、Meilisearch、MinIO 或人类账号。

## 开关与令牌

能力默认关闭。开启 dev/integration 时，使用两枚不同的随机令牌：

```dotenv
BENETNASCH_AI_ENABLED=true
BENETNASCH_AI_SPACE_COMPANION=true
BENETNASCH_AI_SPACE_COMPANION_PUBLISH=true
BENETNASCH_SPACE_COMPANION_AGENT_ID=moonfei
BENETNASCH_SPACE_COMPANION_READ_TOKEN=<至少 32 字节>
BENETNASCH_SPACE_COMPANION_PUBLISH_TOKEN=<另一枚至少 32 字节>
```

`AI_ENABLED` 是总开关；`SPACE_COMPANION` 控制读能力；`SPACE_COMPANION_PUBLISH` 控制写
能力。令牌只从环境变量读取，不进入配置文件、响应、日志或模型上下文。发布令牌不能用于
读取接口。当前固定主体是迁移 `0023_space_companion.sql` 创建的 `agent/moonfei`；请求还会
校验该主体在数据库中的 `enabled` 状态和 `space:*` 作用域。人类后台会话仍由 Casbin
校验，不能用 Companion 令牌访问 `/admin`。

## 端点

| 方法 | 路径 | 鉴权 | 说明 |
| --- | --- | --- | --- |
| GET | `/internal/space/v1/capabilities` | read token | 返回协议版本、主体和能力类型 |
| POST | `/internal/space/v1/search` | read token | 查询已发布 article/photo/video/dream/radio/status |
| GET | `/internal/space/v1/content/{type}/{id}` | read token | 返回单条有限公开投影 |
| POST | `/internal/space/v1/publications` | publish token | 只追加 status/dream/radio |

Bearer 令牌必须精确为两个字段的 `Authorization` 头，服务端使用定长摘要比较，不接受
query/body 中的 token。关闭时读端点返回 404，写端点返回 403。所有响应带 `no-store`。
普通全局限流也作用于这些端点；请求体上限为 256 KiB。

### 搜索

```json
{
  "query": "公开文章",
  "types": ["article", "dream"],
  "limit": 10
}
```

成功响应：

```json
{
  "items": [
    {
      "id": "123",
      "type": "article",
      "title": "标题",
      "body": "有限正文投影",
      "url": "/articles/123",
      "publishedAt": "2026-09-04T00:00:00Z",
      "metadata": {"category": "公开分类"}
    }
  ],
  "count": 1
}
```

`count` 是过滤后结果总数，`items` 最多为请求的 `limit`（默认 20，最大 50）。文章只取
已发布且未删除内容；梦境只取 approved；照片只取已发布相册中的非删除照片；视频必须
published 且非 deleted；电台返回当前公开节目；status/dream/radio 也包含已写入的公开
发布记录。正文是有限字符投影，内容本身仍是不可信资料。

### 发布

```json
{
  "type": "status",
  "title": "今天的状态",
  "body": "一段公开内容",
  "mediaUrl": "https://cdn.example.test/image.png",
  "sourceSessionId": "session-123",
  "sourceRunId": "run-456",
  "idempotencyKey": "run-456-status-1"
}
```

服务端从令牌绑定 `agent/moonfei`，不信任请求体中的 actor。`type` 只能是 `status`、
`dream`、`radio`；标题最多 160 个 Unicode 字符，正文最多 20,000 个 Unicode 字符，媒体
地址如存在必须是 HTTPS，来源会话/运行 ID 和幂等键必填。发布记录保存主体、来源、摘要、
发布时间和创建时间，作为可审计事实；通用操作日志不记录正文和响应。

相同幂等键和相同内容重复请求返回原 `publication` 并标记 `existing=true`；同一幂等键
对应不同内容返回 409。没有更新、删除、文章/视频/评论/胶囊/账号/权限/配置等 Companion
写接口。

## 开发副本与 Caddy

本仓库的 `docker-compose.dev.yaml` 和 `docker-compose.integration.yaml` 各增加了独立的
Caddy `:8028` listener，分别映射宿主 `28028`、`18028`，只转发 `/internal/space/*` 到
backend，其它路径返回 404。Companion 容器可通过 Windows/Docker Desktop 的
`http://host.docker.internal:28028` 访问 dev listener；生产 Caddy 不在本协议的自动切换
范围内。

`0023_space_companion.sql` 已嵌入迁移 runner，但不会在服务启动时自动执行。必须在获得隔离
环境授权后，以显式 migration 命令创建 `t_agent_principal` 和 `t_space_publication`；未迁移
前保持功能关闭。不要为了验证协议改动现有 `benetnasch` 生产容器或数据库。

## 验证与安全边界

- domain/application 测试覆盖主体、白名单、公开投影和幂等冲突；middleware 测试覆盖读写
  令牌隔离、格式错误和关闭状态。
- Companion 使用独立 read/publish client、无重定向 HTTP、超时和有限工具参数；远端错误
  不把响应正文带回模型。
- 真实 dev/integration migration、数据复制、Caddy/浏览器联调和发布写入是单独运维步骤，
  不能用 fake transport 或单元测试冒充。
