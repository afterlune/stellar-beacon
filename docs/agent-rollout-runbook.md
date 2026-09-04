# Agent 能力灰度与发布手册

本文用于数字空间 Agent、Provider、检索、审核、Vision、Companion bridge 和前端控制面的发布治理。开关、RBAC、灰度、紧急停机和外部平台各自承担不同职责，不能用前端隐藏按钮代替后端授权。

## 发布前不变量

发布人和复核人必须共同确认：

- 目标 commit、环境、迁移版本、Provider、模型、索引 UID 和 feature flag 已记录；
- 数据库备份、恢复方式、索引回滚、Caddy 旧产物和停止条件已准备；
- 管理员 RBAC、审计、预算、错误率、取消率、P95 和紧急开关可观察；
- 公开响应不包含 key、密码、Prompt、访客正文、内部身份或数据库/Provider 细节；
- AI 关闭时空间基础读取、登录、菜单和普通管理能力仍然可用；
- 生产容器、数据库、Meilisearch、MinIO 和 Caddy 的操作已单独登记，不会被普通应用发布隐式触发。

## Provider 基线

- Chat/Vision：`OPENAI_BASE_URL`、`OPENAI_MODEL`、`OPENAI_API_KEY`，通过 OpenAI Chat Completions 兼容协议访问 DeepSeek；
- Embedding：`ALIBAILIAN_BASE_URL`、`ALIBAILIAN_MODEL`、`ALIBAILIAN_API_KEY`，当前主模型为 `qwen3.7-text-embedding`；
- `ALIBAILIAN_MODEL2=qwen3.7-text-embedding-flash` 只能作为新索引版本的显式候选，不能与旧模型混写同一索引；
- SGLang/Qwen 本地服务是独立低内存实验，未通过内存门禁前不加入普通 Compose；
- Provider key 只从环境变量注入，文档、配置提交和日志不出现真实值。

## Feature flag 目录

所有开关默认关闭，并同时受总开关 `BENETNASCH_AI_ENABLED` 约束：

| 环境变量 | 能力 |
| --- | --- |
| `BENETNASCH_AI_PUBLIC_CHAT` | 公开 Agent 对话 |
| `BENETNASCH_AI_WRITING` | 写作预览 |
| `BENETNASCH_AI_VISION` | Vision 预览 |
| `BENETNASCH_AI_ARTICLE_INDEXING` | 文章索引任务 |
| `BENETNASCH_AI_PROVIDER_PROBE` | 管理员 Provider 烟测 |
| `BENETNASCH_AI_OBSERVABILITY` | 脱敏运行时观测 |
| `BENETNASCH_AI_BEHAVIOR` | 行为候选 Worker |
| `BENETNASCH_AI_DREAMS` / `BENETNASCH_AI_DREAM_IMAGES` | 梦境候选和图片 |
| `BENETNASCH_AI_RADIO` / `BENETNASCH_AI_VIDEOS` | 电台和视频体验 |
| `BENETNASCH_AI_CAPSULES` / `BENETNASCH_AI_TTS` | 时间胶囊和语音能力 |
| `BENETNASCH_AI_SPACE_COMPANION` | Companion 只读 bridge |
| `BENETNASCH_AI_SPACE_COMPANION_PUBLISH` | Companion 受限发布 |

Companion bridge 还必须提供不同的 `BENETNASCH_SPACE_COMPANION_READ_TOKEN` 和 `BENETNASCH_SPACE_COMPANION_PUBLISH_TOKEN`，并完成 `agent/moonfei` 主体迁移。开关不能单独绕过主体和作用域校验。

## 灰度顺序

1. **离线和代码门禁**：单元测试、边界扫描、漏洞扫描、前端构建和 Swagger 契约。
2. **隔离控制面**：管理员登录、菜单、RBAC、Provider smoke、审核队列、任务恢复、观测和回滚。
3. **隔离公开能力**：限定测试用户/网络，小规模检查检索、SSE、取消、预算和公开性过滤。
4. **隔离 Companion**：验证 read/publish token 分离、公开内容、白名单发布、幂等冲突和拒绝越权。
5. **生产发布窗口**：仅在备份、复核、监控和回滚就绪后单独审批；没有窗口时保持关闭。

## 回滚

发现错误率、敏感信息、越权、重复副作用、预算异常、P95 回退或关键页面空白时：

1. 先关闭对应能力开关或紧急开关，不修改审核历史和正常公开内容；
2. 保存脱敏指标、run ID、审计记录、错误类别和当前版本；
3. 按 Provider、应用、索引或 Caddy 的独立回滚方案恢复，不删除证据；
4. 修复并通过回归和隔离验收后，重新从管理员阶段开始灰度；
5. 生产数据、迁移和静态目录回滚必须由发布负责人执行。

## 独立发布窗口

以下动作不能由普通启动或应用发布隐式完成：

- PostgreSQL 迁移、备份、恢复和失败演练；
- Meilisearch provision、全量回填、swap、旧索引删除；
- MinIO/OSS 对象复制或清理；
- Caddy 静态目录切换和旧版本归档；
- 生产容器停止、重建、升级或卷操作。

隔离环境使用 `benetnasch-integration` 或 `benetnasch-dev` 的固定项目名、端口和卷；生产项目 `benetnasch` 不作为普通验证目标。

## 当前证据

- Go 全量测试、`go vet`、边界扫描和静态配置门禁已纳入 CI；
- `benetnasch-dev` 启动 smoke 已验证公开入口、管理入口、后端和基础设施可用；
- Companion 真实 Python 运行、完整工具/浏览器联调、生产恢复演练和生产发布切换仍需独立证据；
- 未执行的门禁必须标记为未验证，不以 fake、mock 或旧索引代替。
