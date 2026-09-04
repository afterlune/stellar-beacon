# Agent 灰度、开关与发布窗口手册

本文是 Agent 能力的发布治理手册。配置开关、RBAC、紧急停机和外部发布平台各自承担
不同职责：开关决定能力是否装配，RBAC 决定谁能进入后台操作，灰度控制谁能观察公开
入口，紧急开关用于快速止损。不能用前端隐藏按钮替代后端授权，也不能把浏览器提交的
身份、权限或灰度标记当作可信输入。

## 1. 发布前不变量

每次启用前必须由发布人和复核人确认：

- `ai.enabled` 和目标能力开关均已明确记录，未知或缺失配置保持关闭；
- 数据库迁移、Provider、Redis、Meilisearch、MinIO 和 Caddy 的依赖已分别检查；
- 管理员 RBAC、审计日志、预算告警、错误率和紧急开关可用；
- 公开响应不包含 API key、密码、Prompt、内部错误、访客身份或审核内部字段；
- 已准备降级/回滚步骤、观察时间、负责人和停止条件；
- 生产容器升级、重建、数据迁移和静态目录切换均已单独登记发布窗口。

### 1.1 当前 Provider 基线

- Chat/Vision：`OPENAI_BASE_URL=https://api.deepseek.com`、
  `OPENAI_MODEL=deepseek-v4-flash-vision-exp`，协议为 `openai_chat_completions`；API key
  只从 `OPENAI_API_KEY` 注入。
- Embedding：使用阿里百炼 OpenAI 兼容接口，`ALIBAILIAN_BASE_URL`、
  `ALIBAILIAN_MODEL=qwen3.7-text-embedding`；索引契约固定为 1024 维，不能混写旧模型向量。
  `ALIBAILIAN_MODEL2=qwen3.7-text-embedding-flash` 仅作为显式换模候选，换模必须创建新的
  索引版本，不能作为运行时自动 fallback。
- 这些配置不等于能力已发布：`ai.enabled` 和具体 feature flag 仍须显式开启；真实 Provider
  smoke、`article_chunks_v2` provision/backfill 和 P95 采样需要隔离环境或独立发布窗口。
- 图片输入仅在 user 消息允许，URL/base64 必须经过 adapter 的格式、大小和主机边界校验；
  不把原始图片、Prompt 或 Provider 参数写入日志。
- 管理员视觉预览由独立的 `ai.vision` 开关控制，入口为
  `POST /admin/ai/vision/preview`；请求只允许一个受限的 HTTPS/HTTP 公网图片 URL 或
  有 MIME 类型的 base64 图片，结果先写入 pending review，不自动修改文章。

#### 本地 Qwen Embedding 低内存 smoke

本地可用 ModelScope 下载的 `Qwen/Qwen3-Embedding-0.6B` 做隔离烟测。配置位于
`docker-compose.integration.qwen.yaml`，不纳入普通 integration Compose；模型目录必须位于
仓库外并以只读 bind mount 注入：

```powershell
# 默认只执行路径、Compose 和内存预检，不启动任何容器。
pwsh ./scripts/integration-qwen-up.ps1 `
  -ModelPath 'D:/Models/Qwen3-Embedding-0.6B'

# 只有隔离发布窗口明确授权时才允许启动 qwen-embedding。
pwsh ./scripts/integration-qwen-up.ps1 `
  -ModelPath 'D:/Models/Qwen3-Embedding-0.6B' `
  -Start -AllowContainerChanges
```

启动器会在 Compose 配置校验前后各做一次内存预检，模型目录必须位于仓库外且不是
symlink/reparse point；启动时只允许显式选择 `qwen-embedding`，如需让隔离 backend
改用本地路由，必须额外传入 `-IncludeBackend`。它不会启动 PostgreSQL、Redis、Caddy、
Meilisearch 或 MinIO，也不会把本地 Qwen 路由用于文章回填。

该 overlay 固定为低内存 smoke profile：`mem-fraction-static=0.35`、4096 token
上限、单并发并关闭 CUDA Graph，适合 8 GiB 显存的本机验证，不用于吞吐压测。容器另外设置
5 GiB 主机内存硬上限、禁止 swap 扩张、2 个 CPU、256 个进程并关闭自动重启；资源不足时应
让可选容器 OOM 退出，不能把 Docker Desktop/WSL 拖到整机卡死。服务只暴露隔离 loopback
端口 `30000`，模型挂载只读；一次请求应验证 `/v1/embeddings` 返回 1024 维有限向量。
只选择 `qwen-embedding` 时不会改变 backend；在明确选择 `backend` 后，overlay 才会打开
`BENETNASCH_AI_LOCAL_EMBEDDING`，将 Provider probe 路由到同一 Compose 网络中的 SGLang，
并强制文章索引保持关闭。验证命令为：

```powershell
pwsh ./scripts/integration-qwen-up.ps1 `
  -ModelPath 'D:/Models/Qwen3-Embedding-0.6B' `
  -Start -IncludeBackend -AllowContainerChanges

pwsh ./scripts/integration-provider-smoke.ps1 `
  -UseCase embedding -UseLocalSGLang -AllowExternalProviderCalls -AllowWrites
```

Provider smoke 还会检查实际运行的 Compose service 是否确实使用 5 GiB 内存/交换上限；旧的
无上限 Qwen 容器即使仍能响应，也会在登录和模型请求前被拒绝，必须按上面的 `--force-recreate`
重新创建。

该本地路由不改变普通 integration 或生产默认的 AliBailian 配置，也不触发文章回填、索引
写入或切换；正式使用 Qwen 向量仍须单独完成质量/P95 和版本化索引发布门禁。
烟测结束后应停止可选的 `qwen-embedding`，并用不带该 overlay 的普通 integration Compose
恢复 backend，释放显存/内存并回到 AI 默认关闭状态。

如果 Windows 主机可用物理内存不足 12 GiB、Docker Desktop/WSL 总容量不足 8 GiB、扣除当前
运行容器后可用容量不足 6 GiB，或已经出现明显换页，直接跳过本节，不启动 SGLang；本地回填、
批量评估和 `article-index` 命令均不属于低内存 smoke。运行中出现卡顿时只停止这个隔离服务：

```powershell
docker compose -p benetnasch-integration `
  --env-file .env.integration `
  --profile qwen-low-memory `
  -f docker-compose.integration.yaml `
  -f docker-compose.integration.qwen.yaml `
  stop qwen-embedding
```

随后按上面的“烟测结束”步骤恢复 backend。不要通过增加 `mem-fraction-static`、token 上限或
并发来“修复”卡顿，也不要把本地 Qwen overlay 加到生产 Compose。

Provider 契约依据（访问日期 2026-09-01）：DeepSeek 官方 [Vision 文档](https://api-docs.deepseek.com/guides/vision/)
确认 `deepseek-v4-flash-vision-exp` 支持 JPEG/PNG/GIF/WebP，并通过 OpenAI 兼容
Chat Completions 的 user `image_url` 传图；阿里百炼 OpenAI 兼容 `/embeddings` 接口已用
`.env.integration` 中的 `qwen3.7-text-embedding` 和 `qwen3.7-text-embedding-flash` 各完成
真实 HTTP smoke，均返回 1024 维向量。仓库因此固定 Vision 为 DeepSeek 路由、Embedding
为阿里百炼 `qwen3.7-text-embedding`/1024 维；真实可用性仍必须通过隔离 Provider smoke
验证，不能只凭文档视为已上线。

DeepSeek 的 Chat/Vision key 不得填入 `ALIBAILIAN_API_KEY` 或被路由兼容逻辑隐式复用。
本项目对 `https://api.deepseek.com/v1/embeddings` 的只读能力探测返回 HTTP `404`；该
结果只作为 fail-closed 配置证据，不会把 Chat 模型输出伪装成向量，也不会因此启动文章
回填。只有独立 Embedding Provider 返回与 `article_chunks_v2` 契约一致的向量后，才可
进入 backfill、质量/P95 和索引切换门禁。

隔离环境的真实 Provider 烟测使用 `scripts/integration-provider-smoke.ps1`，支持
`chat`、`vision` 和 `embedding`。`.env.integration`
可选注入 `OPENAI_API_KEY`、`OPENAI_BASE_URL`、`OPENAI_MODEL` 和 AliBailian Embedding
配置；示例文件只保留空 key。脚本只接受 loopback `18018`，且必须同时传入
`-AllowExternalProviderCalls -AllowWrites`，后者用于明确承认操作审计日志会写入隔离数据库。
它不会自动开启 `ai.enabled`/`ai.provider_probe`，也不会修改配置、索引或容器；只有在批准的
隔离窗口中先在未纳入 Git 的 `.env.integration` 中显式设置
`BENETNASCH_AI_ENABLED=true` 和 `BENETNASCH_AI_PROVIDER_PROBE=true`，重建/启动隔离
backend 并确认模型路由后，才可执行：

```powershell
pwsh ./scripts/integration-provider-smoke.ps1 `
  -UseCase vision -AllowExternalProviderCalls -AllowWrites
```

Chat/Vision 的成功标准是生成和流式能力均可达；Embedding 的成功标准是向量能力可达，
且不误报生成/流式能力。三者的响应均不得包含 key、Prompt、向量、请求/响应正文或授权
字段；失败时只查隔离 backend 的脱敏日志。Provider key 缺失、flag 关闭或目标不是隔离
loopback 时脚本会在外部调用前退出。当前默认 Embedding 路由为 AliBailian
`qwen3.7-text-embedding`，可执行：

```powershell
pwsh ./scripts/integration-provider-smoke.ps1 `
  -UseCase embedding -AllowExternalProviderCalls -AllowWrites
```

Provider smoke 通过后，真实 Vision 图片评测仍需另行执行；它要求经过授权且位于仓库外的
图片目录，脚本会按固定 `fixtureId`、仓库外 manifest 的显式映射，或受支持的 `D:\样本`
目录命名逐项调用管理员预览接口，并把模型预览写入仓库外的新结果文件供负责人验收：

```powershell
pwsh ./scripts/integration-vision-eval.ps1 `
  -FixtureDirectory C:\secure\benetnasch-vision-fixtures `
  -OutputPath C:\secure\benetnasch-vision-runs\run-001.jsonl `
  -AllowExternalProviderCalls -AllowWrites
```

该入口不会自动批准 pending review、发布内容或覆盖既有结果；没有真实 key、authorized
fixtures 或明确隔离窗口时保持 fail-closed。

2026-09-01 已在隔离 Compose 中用仓库外只读目录 `D:\图片` 完成三个真实样本调用：
`vision-scene-001`、`vision-ocr-001` 和 `vision-privacy-001`。三个请求都返回非空预览，
只产生 pending review；结果 JSONL 在系统临时目录，未写入仓库。该结果只能证明调用链路
可执行；随后负责人决定当前版本不把“两名独立评测人”设为 Vision 发布门禁。

已有图片不能重命名时，可使用仓库外的 manifest；value 只能是 `FixtureDirectory` 直接下的
文件名，必须由人工确认类别，脚本不会按普通文件名推断 OCR、图表、安全或隐私样本；
`D:\样本` 的六类固定中文文件名是唯一内置例外：

```powershell
pwsh ./scripts/integration-vision-eval.ps1 `
  -FixtureDirectory D:\图片 `
  -FixtureManifestPath C:\secure\benetnasch-vision-fixtures\manifest.json `
  -PreflightOnly
```

`BENETNASCH_AI_*` 环境覆盖只在值为合法布尔值时生效，非法值 fail-closed；Compose 默认
传入 `false`，不会改变现有配置文件和生产默认关闭行为。文章索引回填同理使用
`BENETNASCH_AI_ENABLED=true` 与 `BENETNASCH_AI_ARTICLE_INDEXING=true`，但必须另行完成
候选索引 provision 和发布单审批。

## 2. 固定灰度顺序

### 阶段 A：管理员

先只启用后台可见能力：AI Studio 预览、审核列表/动作、任务状态、索引 backfill
控制和观测指标。启用观测时使用 `BENETNASCH_AI_OBSERVABILITY=true`，并在迁移
`0022_ai_observability_rbac` 已完成后，由外部监控以管理员身份采集
`GET /api/admin/ai/observability`；管理员操作必须经 Casbin/RBAC，生成内容只能进入 pending 审核记录，
不能直接写文章、评论或说说。此阶段使用隔离环境或后台 API，公开 feature flag 保持关闭。

### 阶段 B：白名单

确认阶段 A 的审计、预算、失败重试和回滚都正常后，再对经过登记的测试用户/测试网络
开放一个或多个公开能力。白名单应在可信的发布网关、认证层或 Caddy 的独立隔离入口
实现，不接受浏览器自报的 `role`、`userId`、`allowlisted`、`rollout` 等字段；匿名 Agent
仍必须使用服务端 owner key、Redis 配额和应用层并发闸门。

白名单阶段的每次请求记录版本、目标 flag、入口、结果类别和 run ID，不记录原始 Prompt、
凭据材料或完整私密正文。没有可信白名单能力时，不得把全量匿名流量误称为白名单灰度，
应停留在管理员或隔离环境阶段。

### 阶段 C：逐步开放

按小比例、短观察窗逐步提高流量；每一步都要满足上一窗口的门禁。任一门禁失败，立即
执行对应能力的关闭动作，保留其他主链路和无关 Agent 能力。公开前台只依据
`GET /agent/features` 渲染，接口失败或紧急停机时必须 fail closed。

## 3. Feature flag 目录

| 开关 | 能力 | 主要依赖 | 首次开放方式 | 立即回滚 |
| --- | --- | --- | --- | --- |
| `ai.observability` | AI/搜索脱敏运行时观测 | 0022 RBAC、外部监控采集 | 管理员只读采集 | 关闭观测开关，停止采集 |
| `ai.article_indexing` | Chunk 索引 worker | 新索引、Embedding、backfill 表 | 先离线回填候选 UID | 关闭 worker，保留旧索引 |
| `ai.content_understanding` | 摘要/分类候选 | Job 表、写作路由 | 管理员手动执行一次 | 关闭 worker，不删 pending |
| `ai.writing` | 写作预览 | 写作 Provider、审核表 | 管理员预览 | 关闭预览入口，保留审核记录 |
| `ai.vision` | 管理员视觉理解预览 | Vision Provider、审核表、0021 RBAC | 管理员预览 | 关闭视觉入口，保留 pending 审核记录 |
| `ai.public_chat` | 公开 SSE Agent | Chat Provider、Redis | 白名单后小流量 | 关闭公开聊天，保留短期会话 |
| `ai.behavior` | 自主候选生成 | 虚拟账号、审核/行为表 | 管理员审核 dry run | emergency stop + 关闭 worker |
| `ai.dreams` | 梦境候选/档案 | 梦境迁移、审核策略 | 先只显示已批准数据 | 关闭公开档案/生成 |
| `ai.dream_images` | 梦境图片任务 | Image Provider、对象存储 | 已批准梦境小批量 | 关闭图片 worker，保留占位图 |
| `ai.capsules` | 时间胶囊 | 胶囊迁移、认证用户 | 管理员和测试用户 | 关闭入口，不改变已封存状态 |
| `ai.vitals` | 聚合生命体征 | 活动表、聚合查询 | 只读小流量 | 关闭组件/接口 |
| `ai.galaxy` | 内容星河 | projection 表、PCA worker | 只读小流量 | 关闭组件/查询 |
| `ai.radio` | 文本电台 | 公开文章、Chat Provider | 手动刷新小流量 | 关闭电台 |
| `ai.videos` | 视频列表/上传 | 视频迁移、MinIO/HTTPS 白名单 | 管理员上传后公开 | 关闭列表/上传 |
| `ai.tts` | 用户点击 TTS | 浏览器 Web Speech API | 用户主动点击 | 关闭 `ttsEnabled` |

`ai.enabled=false` 是全局总闸；`ai.emergency_stop=true` 是跨实例止损开关。关闭任何
能力都不能回滚已批准的审核记录、已发布的正常内容或用户已封存的胶囊，除非另有独立
数据修复流程和审批。

公开聊天的 `ai.limits.max_output_tokens` 会同时作为两次模型调用（工具回合和最终回答）
的输出预算；应用层默认 1200，公开聊天硬上限为 4096。它与 `max_answer_runes` 是两道
独立门禁，配置过大时仍会被应用层上限截断。权限判定只负责 RBAC，不能绕过
`AccessLimiter`；管理员请求也经过 IP/全局限流，避免“有权限”等同于无限速。

Caddy 模板不再自行处理 CORS 或 OPTIONS，也不允许 `*` 配合凭据。CORS 决策统一由后端
的精确 Origin 白名单完成；`handle_path /api` 必须位于 SPA fallback 之前。生产主机上的
外部 `Caddyfile` 不会被应用启动流程自动替换，需在独立发布单中同步并用 Caddy 配置校验器
复核。

## 4. 统一回滚步骤

1. 记录触发门禁的时间、flag、版本、请求量和错误类别，不把内部错误详情写入公开响应；
2. 优先关闭目标能力 flag；若无法确认影响范围，先设置共享 emergency stop；
3. 停止对应 worker 的新领取，等待当前任务在租约/取消边界收敛；
4. 对索引能力暂停 backfill 并保留旧 UID；对审核能力保留 pending/publish_failed 记录；
5. 检查主链路：登录、文章读取、后台菜单和普通文章搜索不能因 Agent 回滚而失败；
6. 验证 `GET /agent/features` 已反映关闭状态，浏览器 fail closed，审计和预算告警仍工作；
7. 复盘根因并由复核人批准后，才可回到管理员阶段，不得直接跳回全量开放。

## 5. 各类门禁

- 安全：未知工具/参数、私密文章、越权身份、Prompt 注入、SSRF 或秘密泄漏为零容忍；
- 可靠性：Provider 错误、Redis/Meili 故障、SSE 断连和 worker 取消必须有界并可恢复；
- 质量：RAG 引用/拒答、写作人工评分、审核通过率/重复率和检索 P95 达到本项目门槛；
- 成本：token 使用、单日预算、并发槽位和异常重试均在告警范围内；
- 产品：任何公开生成内容仍需人工审核，且不覆盖原文章或用户内容。

## 6. 独立生产发布窗口

以下动作必须分别建单、单独审批、单独备份和单独回滚，不能在普通应用发布中隐式完成：

- PostgreSQL schema migration、数据修复、序列修复或删除；
- Meilisearch 版本升级、全量索引创建/回填、swap 或旧索引删除；
- PostgreSQL、Redis、Meilisearch、MinIO、Caddy 容器的升级、重建、配置/数据卷变更；
- Caddy 生产静态目录切换和旧产物清理。

发布单至少包含：变更目标、影响范围、备份位置和恢复演练、开始/停止时间、操作者、
复核人、监控门禁、回滚命令/计划、验证结果和遗留风险。未获得独立窗口时，本仓库的
启动流程只做配置校验和依赖装配，不执行上述动作。

### 6.1 Caddy 静态目录切换记录

生产 Caddy 当前以 `/srv/vue/admin` 提供旧后台；主机侧目录由部署环境映射，不能把下面
示例路径直接当作本机事实。发布人先用仓库产物校验器和第二人复核新后台入口哈希，再在
批准的主机上执行只读计划（`StaticRoot` 应是包含 `admin` 子目录的主机目录，
`ArchiveRoot` 必须位于该目录之外）：

`Plan`/`Apply` 会拒绝静态根和已有归档树中的 reparse point/junction；`Apply` 还要求
`ArchiveRoot` 已由运维预创建为普通目录，脚本不会自动创建未验证的路径；`VerifyRetention` 只
接受直接位于 `ArchiveRoot/<release-id>/manifest.json` 的活动 manifest，避免发布路径或
保留校验被链接和嵌套伪记录劫持。

```powershell
pwsh ./scripts/production-caddy-admin-switch.ps1 `
  -Action Plan `
  -StaticRoot 'D:\containerData\caddy\static\vue' `
  -ArchiveRoot 'D:\containerData\caddy\releases' `
  -SourceDist 'web/admin-next/dist' `
  -ReleaseId 'admin-next-<release-id>'
```

确认发布单、目标目录和 SHA-256 后，才允许在同一窗口执行 `Apply`：

```powershell
pwsh ./scripts/production-caddy-admin-switch.ps1 `
  -Action Apply `
  -StaticRoot 'D:\containerData\caddy\static\vue' `
  -ArchiveRoot 'D:\containerData\caddy\releases' `
  -SourceDist 'web/admin-next/dist' `
  -ReleaseId 'admin-next-<release-id>' `
  -ReleaseTicket '<ticket>' `
  -ExpectedIndexSha256 '<64-char-sha256>' `
  -RetentionHours 168 `
  -AllowProductionSwitch -ConfirmProductionTarget
```

切换后立即通过 Caddy 对登录、菜单、列表、编辑入口和硬刷新执行发布单指定的验证，
并保留脚本输出、manifest 和旧目录。至少一个发布周期后执行只读保留校验：

```powershell
pwsh ./scripts/production-caddy-admin-switch.ps1 `
  -Action VerifyRetention `
  -StaticRoot 'D:\containerData\caddy\static\vue' `
  -ArchiveRoot 'D:\containerData\caddy\releases'
```

`VerifyRetention` 返回非零或旧产物哈希不一致时，不得清理归档；回滚只使用 manifest
记录的旧目录和独立审批，不通过删除或覆盖未知目录完成。

## 7. 当前仓库证据

- 公开 flag：`app/facade/api/agent_features_controller.go`；
- 全局/能力开关：`app/infra/config/config.go`；
- 紧急开关：`app/application/agent/safety_switch.go`；
- 后台任务白名单：`app/infra/task/job_runner.go`；
- 管理员视觉预览：`app/application/service/ai_vision_service.go`、
  `app/facade/api/ai_studio_controller.go` 和 `0021_agent_vision_rbac.sql`；
- 索引切换：`docs/agent-search-cutover-runbook.md`；
- 隔离联调入口：`scripts/integration-*.ps1`；
- 本地安全前置检查：`scripts/safe-preflight.ps1`。它只执行本地 Go/前端检查、边界
  扫描、遗留日志检查和可选的 mock 浏览器基线，不启动或修改任何容器、数据库、索引
  或生产 Caddy。真实隔离浏览器、迁移、索引回填/切换和备份恢复仍须单独发布窗口。

本手册本身不改变任何现有容器、数据库、索引或生产 Caddy 配置。
