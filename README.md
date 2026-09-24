# 星际信标 · Stellar Beacon

星际信标（Stellar Beacon）是一个 Vue 博客前台 + Vue 管理端 + Go API 的博客系统。当前仓库采用按职责分层、按业务域组织的结构；博客前台使用「星图仪器 / Astral Instrument」设计语言（规范见 [`docs/design/blog-frontend.md`](docs/design/blog-frontend.md)），管理端使用保留的 `admin-next`。

产品按公共空间与个人空间组织：`/` 是公共发现面，`/u/{handle}` 是作者的公开主页，`/studio` 是仅登录本人可用的私有创作空间。登录后页头可直接在公共空间与我的空间之间切换；未设置公开地址时，公开主页入口保持禁用。

公开主页会用现有公开接口自动策展代表作、主题系列、公开书单与最近随想。单个模块请求失败时只隐藏该模块，不影响作者其他公开内容的浏览；没有热度文章时，代表作回退到最新文章。

公开作者主页同时提供面向搜索引擎和社交平台的服务端 HTML：动态标题、简介、头像、作者统计、代表作链接与 ProfilePage JSON-LD。浏览器内可调用系统分享或复制 canonical 链接，sitemap 会收录所有有公开内容的作者主页。

## 目录

- `cmd/stellar-beacon`：生产 API 进程入口
- `cmd/integration-seed`：仅供隔离联调使用的数据初始化程序
- `internal/domain`：领域实体、端口和错误
- `internal/application`：应用服务与业务编排
- `internal/interfaces/http`：版本化路由、handler、DTO 和中间件
- `internal/infrastructure`：PostgreSQL、Redis、Meilisearch、MinIO、邮件和配置适配器
- `resources`：运行时资源
- `deploy`：Compose、Caddy、配置和数据库初始化文件
- `web/apps`：`blog` 前台与 `admin-next` 管理端
- `web/packages`：共享 API 契约与请求客户端
- `scripts/checks`、`scripts/integration`：检查与隔离联调脚本
- `docs`：API、运行手册和 ADR

## API

外部 API 只有版本化入口：

- `/api/v1/public/*`：博客公开内容
- `/api/v1/auth/*`：登录、注册和当前用户
- `/api/v1/admin/*`：管理端能力

响应统一为 `{ "code": "OK", "message": "...", "data": ... }`；分页数据统一为 `{ items, total, page, pageSize }`。旧 HTTP 路由不再注册。

- `GET /u/{handle}`：作者公开主页；普通浏览器由 Caddy 返回 Vue SPA，搜索引擎与社交平台爬虫由后端输出动态 SEO HTML。

博客前台的公共发现面都是只读公开接口，且只返回 `status=1` 且审核可见的内容：

- `GET /api/v1/public/feed?sort=latest|hot|featured`：公共信息流；`hot` 按近 7 天的收藏、审核通过评论、点赞与独立读者加权排序。
- `GET /api/v1/public/authors?sort=articles|followers|active`：作者榜；`active` 按最近一次公开发布排序。
- `GET /api/v1/public/topics`：话题广场，分类、标签与系列复用同一热度口径聚合。
- `GET /api/v1/public/collections?sort=latest|hot`：公开书单发现页；只列出至少收录一篇当前公开、审核可见文章的书单。`hot` 按成员文章近 7 天热度，加书单全部时间的 `4 × 已审核评论 + 3 × 点赞` 排序。
- `GET /api/v1/public/collections/{slug}` 与 `GET /api/v1/public/authors/{handle}/collections`：按链接打开书单，或在作者主页列出其公开书单；失效、私有或审核隐藏的文章会实时从公开详情中移除。
- `/api/v1/studio/collections*`：登录用户创建与维护私有、链接可见或公开书单，支持手动排序、逐篇推荐语和最多 500 篇文章；`private` 仅本人可见，`unlisted` 不进入发现页但可通过链接访问。
- `PUT`/`DELETE /api/v1/auth/me/collection-subscriptions/{collectionId}`：订阅或退订公开、链接可见的他人书单；支持 `PUT .../mute` 单书单静音、`GET /api/v1/auth/me/collection-feed` 更新流。
- `PUT /api/v1/auth/me/collection-reactions`、`GET ...` 与 `GET .../state`：公开、链接可见书单支持独立点赞与收藏、当前账号状态查询和私人收藏列表；收藏按书单全部时间加 `5 ×` 进入热门分，私有书单拒绝互动。书单评论复用 `/api/v1/public/comments`，`type=6`、`topicId` 为书单 ID，审核通过后才计入公开评论数并发送互动通知。
- `PUT /api/v1/auth/me/comment-reactions`：所有已审核评论与回复支持幂等点赞/取消点赞；评论列表返回 `likeCount` 和当前账号 `liked`。评论点赞不参与文章或书单热度。
- `PUT `/api/v1/studio/collections/{collectionId}/comments/{commentId}/pin` 与 `DELETE /api/v1/studio/collections/{collectionId}/comments/{commentId}`：仅书单作者可治理自己书单下的评论。置顶只接受已审核、未删除的根评论，每个书单最多一条，置顶新评论会自动取消原置顶；删除为软删除，删根评论会级联隐藏其全部回复，删回复只隐藏该条，作者可通过批量恢复或申诉队列恢复对应评论。私有书单与未审核评论仍不可见、不计入公开评论数。
- `GET`/`POST /api/v1/studio/collections/{collectionId}/comments*`：书单作者的评论治理面。`GET .../comments` 返回含已删除项、置顶标记、回复数与待处理举报数的治理列表；`POST .../comments/batch` 支持 `delete`/`pin`/`unpin` 批量操作，一次批量置顶只保留最新的一条根评论，未通过的单条以 `failed[].message` 返回；`POST .../comments/restore` 恢复软删除评论，仅还原同一次根评论删除级联隐藏的回复。所有治理动作写入 `t_comment_moderation_log`（含 before/after 快照与批量 `batch_id`），通用 `t_operation_log` 亦记录请求。
- `GET /api/v1/admin/collections/{collectionId}/comments` 与 `PUT /api/v1/admin/comments/{commentId}/restore`：管理端书单评论治理面。评论列表在原有字段上增加 `isTop`、`isDelete`、`collectionId` 与待处理举报数，软删除评论仍可见并可恢复；管理员只有审核、隐藏与恢复权限，置顶仍归书单作者。
- `POST /api/v1/auth/me/comment-reports`：登录读者可举报书单内已审核评论或回复，原因限定为垃圾、骚扰、色情、违法、隐私或其他，同一读者对同一评论只能保留一条待处理举报。
- `GET`/`PUT /api/v1/studio/collections/{collectionId}/comment-reports*` 与 `/api/v1/admin/comment-reports*`：作者与管理端按评论聚合查看举报队列，并执行忽略、隐藏或恢复；所有处理结果写入治理审计并向举报者发送站内通知。
- `POST`/`GET /api/v1/auth/me/comment-appeals*`：被作者删除的评论仅评论作者本人登录时仍可见并带 `isDelete=1` 标记；作者可提交申诉，书单作者驳回后可升级给管理员终审。
- `GET`/`PUT /api/v1/studio/collections/{collectionId}/comment-appeals*` 与 `/api/v1/admin/comment-appeals*`：书单作者处理一级申诉，管理员处理升级后的终审；恢复会重新公开评论，作者、申诉人与相关管理方收到站内治理结果通知。
- `GET`/`PUT`/`DELETE /api/v1/auth/me/topic-subscriptions*`：读者按**归一化话题名**（分类或标签，跨作者）订阅、静音与取消订阅；`GET /api/v1/auth/me/topic-feed` 返回订阅话题下的新文章，通知中心新增 `topic` 分组（受账号级 `notifyTopic` 开关与单订阅静音控制）。
- `POST /api/v1/auth/me/recommendations/query`：登录态“为你推荐”；组合关注、话题订阅、点赞收藏与请求体内的本机阅读种子，排除本人和已关注作者，结果带可解释理由与游标。
- `GET`/`PUT`/`DELETE /api/v1/auth/me/recommendation-feedback*`：管理隐藏文章、减少作者或主题的推荐偏好并恢复。

## 开发检查

```shell
go test ./...
go vet ./...
git diff --check
```

前端：

```shell
cd web
npm ci
npm run build:blog
npm run build:admin
```

## 隔离联调

联调栈使用 Compose 项目 `stellar-beacon-integration-v17`，复用当前 V17 的 PostgreSQL、Redis、Meilisearch、MinIO 数据卷；应用数据复制到新数据库 `stellar_beacon` 和新桶 `stellar-beacon-integration`。旧数据库 `benetnasch`、旧桶 `benetnasch-integration` 与旧容器保留作回退；生产数据库和 OSS 桶不迁移。

```powershell
Copy-Item .env.integration.example .env.integration
pwsh ./scripts/integration/deploy.ps1
```

访问：

- 博客：`http://127.0.0.1:18080`
- 管理端：`http://127.0.0.1:18008`
- 隔离 API：`http://127.0.0.1:17777`

分步执行：

```powershell
pwsh ./scripts/integration/up.ps1
pwsh ./scripts/integration/repair-sequences.ps1
pwsh ./scripts/integration/seed.ps1
pwsh ./scripts/integration/smoke.ps1
```

一键执行真实前后端主链路联调（会重建隔离栈、执行迁移/种子/冒烟、博客真实后端 E2E、管理端真实集成套件和博客视觉门禁）：

```powershell
pwsh ./scripts/integration/verify.ps1
```

结束时只清理隔离项目：

```powershell
pwsh ./scripts/integration/down.ps1 -RemoveVolumes
```

## 部署

新 Linux 主机可用独立 Compose 栈一次构建博客、管理端与 API。先准备 x86_64 Linux、Docker Compose v2、指向主机的根域名和 `admin.` 子域名，并开放 TCP 80/443（启用 HTTP/3 时也开放 UDP 443）：

```shell
cp .env.production.example .env.production
# 编辑 .env.production：换掉所有示例密钥，并填写域名、SMTP 和对象存储配置
docker compose --env-file .env.production -f deploy/compose/production-standalone.yaml --profile minio up -d --build
docker compose --env-file .env.production -f deploy/compose/production-standalone.yaml --profile minio exec -it backend /app/stellar-beacon bootstrap-admin --email admin@example.com
```

首次启动会执行版本化数据库迁移，生成空白博客配置，不会导入历史 SQL、文章、账号或日志。首次管理员密码在终端中隐藏输入。改用阿里云 OSS 时，填写 OSS endpoint、bucket、区域和访问密钥，并从命令中去掉 `--profile minio`；MinIO 数据卷不会启动。`.env.production` 含生产凭据，不要提交到 Git。更完整的安装、更新和备份说明见[部署运行手册](docs/runbooks/deployment.md)。

原有 Windows 生产栈保持独立，仍使用 `deploy/compose/production.yaml` 与 `deploy/caddy/production.Caddyfile`；不要在现有数据卷上运行新栈的初始化步骤。
