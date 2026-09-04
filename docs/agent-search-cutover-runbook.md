# Agent 检索索引全量重建与切换手册

本文用于 `article_chunks_<version>` 的显式全量重建和切换。它是发布窗口
手册，不是应用启动动作；Benetnasch 启动、普通请求和后台 worker 都不会自动
创建索引、修改索引设置、执行全量回填或提交 swap。

当前状态（2026-09-02）：阿里百炼 `qwen3.7-text-embedding` 已通过真实
Embedding 接口契约验证；已在获授权的隔离 Compose 中完成 `article_chunks_v1`
provision、10 篇公开文章的全量回填（177 个 Chunk）以及真实只读质量/P95 门禁。
随后以 `rename=false` 完成从 `article_chunks_v2` 到 `article_chunks_v1` 的原子
swap，task UID 为 `68` 且已 `succeeded`。当前隔离配置指向 `article_chunks_v1`，
`article_chunks_v2` 保留为可回滚索引；两个 UID 均为 177 个文档。切换后的 13 个固定
查询共采样 130 次，Recall@8/MRR/无结果准确率均为 `1.0`，P50/P95/最大耗时为
`7.48/10.20/15.11ms`。`qwen3.7-text-embedding-flash` 仍仅可作为另一个完整版本
的显式换模候选，不得与当前模型混写同一索引。以上证据仅适用于隔离环境，未执行生产
Meilisearch、数据库或 Caddy 操作。

## 0. 适用范围与硬性约束

- 只操作本次发布声明的目标 UID，例如 `article_chunks_v2`。
- 旧索引保留到观察期结束，禁止把删除旧索引作为切换的一部分。
- 目标索引的 Provider、模型、模型版本、向量维度和批大小必须与发布配置一致。
  模型、模型版本、Provider 或维度变化时，必须使用新的 `article_chunks_<version>` UID。
- 先备份数据库和 Meilisearch，再进入写入/切换阶段；备份失败直接中止。
- 只在已批准的隔离或生产发布窗口执行。现有 PostgreSQL、Redis、Caddy、Meilisearch、
  MinIO 容器不由本手册停止、重建或迁移；容器编排和数据库迁移另行审批。
- `rename=false` swap 会交换两个 UID 的文档、设置和任务历史，但不会交换 UID 本身；
  当前索引为空时禁止 swap。首次启用新版本应在发布配置中直接指向候选 UID，只有已知
  有效且有数据的当前 UID 才能进入原子 swap 流程。

## 1. 只读预检

发布人记录以下信息并由第二人复核：

| 项目 | 示例 | 要求 |
| --- | --- | --- |
| 当前索引 | `article_chunks_v1` | 已知可用，保留回滚入口 |
| 候选索引 | `article_chunks_v2` | 与当前 UID 不同 |
| Primary key | `id` | 当前和候选一致 |
| Model contract | provider/model/version/dimension | 与 embedding 配置和 backfill 状态一致 |
| Run ID | `article-index-backfill-v2` | 稳定、可审计、不可复用其他契约 |

代码入口：

- `search.NewArticleChunksIndexSpec` 构造并校验目标契约；
- `search.ValidateArticleChunksIndexMigration` 拒绝同 UID 的契约漂移；
- `search.NewArticleChunksIndexSwapPlan` 拒绝相同 UID，并校验两端契约。

预检只读确认目标 UID 不承载需要保留的数据、当前索引可查询、数据库备份可恢复，
并确认没有另一个相同 `run_id` 的 backfill 正在运行。任何检查失败都不得进入下一步。

## 2. 创建并配置候选索引

由受控的一次性运维入口显式调用。当前仓库提供对应的 CLI；它要求全局
`ai.enabled`、`ai.article_indexing` 和显式 `--allow-writes` 同时成立，并在任何写请求
前校验 Meilisearch 配置：

```text
go run . article-index provision --allow-writes --task-timeout 10m
```

命令会等待创建/设置任务全部 `succeeded` 后输出不含密钥的契约和 task UID JSON；不传
`--allow-writes`、开关关闭、配置缺失或任务超时都会在失败边界结束。该命令只负责
provision/configure，不执行数据库迁移、文章回填或 swap。

底层实现仍可由受控的一次性运维入口直接调用：

```go
tasks, err := search.ProvisionArticleChunksIndex(ctx, meili, targetSpec)
```

该入口只允许创建缺失的目标 UID，并设置固定的 searchable/filterable 字段；已存在
索引时会校验 Primary key，不会静默改主键。每个返回的 Meilisearch task 都必须等待到
`succeeded`，遇到 `failed`、超时或返回空 task 立即停止。可使用
`search.WaitForArticleChunksTasks(ctx, meili.TaskReader(), targetSpec.UID, tasks, interval)`
统一执行等待和索引归属校验。提交 swap 前，代码还会重新
读取当前和候选索引的 searchable/filterable 设置；任一索引与文章 Chunk 契约不一致都会
阻断切换。

其中 searchable attributes 的顺序属于契约（用于保持检索字段优先级），而
filterable attributes 只校验成员集合；Meilisearch 返回设置时可能对后者重新排序，
顺序变化本身不代表契约漂移。

当前仓库已在 `app/infra/search/article_chunks_index.go` 实现上述边界和 httptest；
它没有被 `InitializeRuntime` 或普通 HTTP 启动路径调用。

### 1.1 文章数据只读计划

在拥有真实 Embedding key 或进入任何索引写入窗口前，可先运行只读数据计划：

```text
pwsh ./scripts/integration-article-index-plan.ps1 -PageSize 1 -TimeoutSeconds 300
```

该命令只读取公开、未删除文章的完整正文，使用与 projector 相同的本地 Markdown
切块和 schema 校验逻辑，输出文章数、预计 Chunk 数、游标和最多 20 个异常文章 ID。
默认每页 1 篇，硬上限为 25；可用 `--max-articles N` 做受控样本。它不需要 AI/Provider key，
不会初始化模型客户端，不创建或修改 Meilisearch 索引，也不写入 backfill 状态。脚本会
把本机源码 CLI 显式指向 `127.0.0.1:15432` 的隔离 PostgreSQL；不应直接用容器内的
`postgresql:5432` 配置从宿主机运行。
只有 `complete=true` 且 `invalidArticles=0` 时，`readyForBackfill=true` 才表示
数据库内容满足进入真实 backfill 的数据前置条件；它不等价于 Provider、质量或 P95
门禁通过。

### 1.2 回填内存预算

Projector 默认对单篇公开文章和单次 Embedding 请求施加固定上限：正文 2 MiB、
Chunk 4,096 个、Embedding batch 32 条、请求正文 256 KiB。超过任一上限会在调用
Provider 或写入 Meilisearch 前返回校验错误，持久化 backfill 保留游标并进入失败/重试
流程，不会为了“尽快跑完”继续扩大内存占用。Eino Embedding 网关对所有直接调用者
也默认执行 256 KiB 输入上限，并拒绝超过 4 MiB 的配置，避免绕过 Projector 时失去
最后一道请求边界。低内存 Qwen/SGLang overlay 的 batch 仍固定为 1，应用侧本地
Embedding 并发也强制为 1，且只允许 Provider smoke；禁止拿它执行全量回填。

这些是应用进程的最后一道保护，不是主机内存或 GPU 显存充足的证明。执行真实 backfill 前仍须
先检查主机/Docker/GPU 余量、使用代表性小样本和独立的停止/恢复窗口；Qwen 启动预检默认要求
至少 16 GiB 宿主机可用内存、5,120 MiB GPU 空闲显存和 12 GiB Docker 余量。发现换页或内存
压力时立即暂停，不得通过提高 batch/page size 绕过预算。

## 3. 全量回填

目标索引配置完成且相关 task 成功后，执行持久化 backfill：

```text
go run . article-index backfill start --run-id article-index-backfill-v2 --page-size 1 --max-articles-per-run 1 --run-timeout 10m --allow-writes
go run . article-index backfill status --run-id article-index-backfill-v2
go run . article-index backfill run --run-id article-index-backfill-v2 --page-size 1 --max-articles-per-run 1 --run-timeout 10m --allow-writes
```

`start`、`run`、`pause` 和 `resume` 会改变数据库回填状态或向目标索引写入，均必须
显式传入 `--allow-writes`；`status` 只读取状态，不接受也不需要写授权。feature flag
仍是第二道门，未同时开启全局 AI 与 `ai.article_indexing` 时命令不会装配回填运行时。

默认运行参数是低内存试跑：每次最多处理 1 篇文章，最长运行 10 分钟，并在文章完成
checkpoint 后进入 `paused`。确认 Provider、索引写入、CPU/内存和延迟都稳定后，再重复
`resume`/`run`；每次继续仍建议保留这个上限。只有在已批准的窗口、已有资源观测和明确
回滚方案下，才可显式传入 `--max-articles-per-run 0` 跑到完成；不要把它与 Qwen
低内存 smoke overlay 一起使用。

隔离环境可使用仓库提供的受限编排器执行 `start` 或 `run`。它固定使用
`benetnasch-integration` Compose、loopback Meilisearch 和运行中 backend，先检查候选索引、
`.env.integration` 中的开关/key，以及容器内已经注入的 Embedding key；不会把 key 作为
`docker exec` 参数传递：

```powershell
pwsh ./scripts/integration-article-index-backfill.ps1 `
  -Action start `
  -IndexUid article_chunks_v2 `
  -RunId article-index-backfill-v2 `
  -AllowExternalProviderCalls -AllowWrites
```

`-IndexUid` 必须与 `resource/config-integration.yaml` 中当前配置的目标一致；Go 命令从运行中
配置派生目标，脚本不会把该参数转发成任意 UID。若要回填另一版本，必须先准备独立的
显式配置与发布窗口，不能只修改脚本参数。

该脚本默认使用 `PageSize=1`、`MaxArticlesPerRun=1` 和 10 分钟超时，只执行一个可恢复的
小步试跑；可通过 `-MaxArticlesPerRun` 与 `-RunTimeoutSeconds` 显式调整。`0` 代表无文章数
上限，仅允许在单独批准的全量窗口传入，且脚本仍会拒绝超过 25 的 `PageSize`。

脚本不会自动修改开关、重建 backend、provision 索引、执行迁移或提交 swap。`run` 用于
从已有持久游标继续；失败后先检查脱敏状态，再由操作者决定是否修复 Provider 后重试。

实际命令需使用部署产物和已批准的配置。运行期间：

1. `status` 必须最终为 `completed`，`lastError` 为空；
2. `processedArticles` 应与当时公开、未删除文章的 keyset 快照相符；
3. 目标索引的文档数、唯一文章数和抽样文章 Chunk 数要与回填状态及数据库抽样相符；
4. Provider、Meilisearch 或网络错误时保留游标，先 `status` 记录，再修复后 `resume/run`；
5. 文章变短、转私密或删除时必须确认旧 Chunk 已按文章 ID 清理；
6. 发布人保存 CLI JSON 输出和 Meilisearch task UID，作为本次发布证据。

暂停使用：

```text
go run . article-index backfill pause --run-id article-index-backfill-v2 --allow-writes
```

暂停只在当前文章 checkpoint 后生效。禁止通过重置游标或重新创建同名 run 绕过租约。

## 4. 质量与性能门禁

回填完成后，先在目标 UID 上执行固定查询抽样，再计算离线评测和延迟报告：

- 搜索质量使用 `search.EvaluateKnowledgeSearchDataset`；
- 内部检索延迟使用 `search.EvaluateKnowledgeSearchDatasetLatency`，关注 P50/P95/最大值；
- 模型首 token 不混入检索 P95，使用 `ai.RunMetricsObserver` 的 first-token 指标；
- 评测错误、目标索引不可用、公开性过滤异常或引用来源缺失，均为阻断条件；
- `P95 <= 300ms` 是内部检索目标，不得用 fake/空索引测试结果替代真实代表性数据采样。

隔离环境的只读采样入口为：

```powershell
pwsh ./scripts/integration-search-readonly.ps1 `
  -IndexUid article_chunks_v1 `
  -DatasetFileName integration-dataset.json `
  -SamplesPerQuery 10 `
  -MinimumDocumentCount 177
```

脚本会先从仓库外的 `.env.integration` 加载并隔离 `MEILI_MASTER_KEY`；不会使用父级 shell
中同名变量，也不会打印 key。缺少该文件或 key 时在任何 Meilisearch 请求前失败。

该脚本只对已存在的版本化候选索引执行 GET 元数据检查和 POST search 查询，拒绝非本机
`17700` 目标、非 `article_chunks_<version>` UID、缺少 `MEILI_MASTER_KEY` 或 Primary key
不为 `id` 的索引；它还会读取索引文档统计，要求发布人按本次发布的代表性规模显式设置
一个正数 `MinimumDocumentCount`。统计值在发起任何搜索请求前就会与该值比较；未达到
该值时立即阻断并明确记录没有发送 benchmark search 请求。脚本同时校验
固定数据集的期望命中、无结果样本和每个命中的
`status=1 && isDelete=0`，并在 JSON 报告中计算按查询宏平均的 Recall@8、MRR 和无结果
准确率（重复采样只用于延迟，质量指标取每个查询的首个样本）。索引不存在、质量检查
失败或 P95 超过目标会以非零状态退出，不会执行 provision、写入或删除任何索引。

至少抽查：存在文章、无结果、中文长查询、分类/标签/时间过滤、私密文章、已删除文章、
文章变短后的尾部 Chunk，以及同一文章多 Chunk 去重。结果必须只包含公开文章。

## 5. 提交原子切换

确认目标索引已 provision、回填完成、质量/延迟门禁通过且当前线上查询仍健康后，构造：

仓库也提供了受控的 `article-index swap` CLI。它要求操作者显式提供当前索引的完整
Embedding 契约，候选契约从当前配置读取，并在提交后等待 swap task 成功；不传
`--allow-writes` 或契约与 UID 不一致都会在 Meilisearch 请求前失败：

```powershell
go run . article-index swap `
  --current-index article_chunks_v1 `
  --current-index-version v1 `
  --current-provider <previous-provider> `
  --current-model <previous-model> `
  --current-model-version <previous-model-version> `
  --current-dimension 1024 `
  --current-batch-size 32 `
  --candidate-index article_chunks_v2 `
  --allow-writes
```

该命令只负责已经通过前置门禁后的原子 swap，不执行 provision、回填、数据库迁移或
质量/P95 评测；输出仅包含两端 UID 和已完成 task UID。回滚时交换 current/candidate
参数并重新复核发布单，不删除旧索引。

隔离环境建议使用 `scripts/integration-article-index-swap.ps1` 作为最终写入口。它要求
`-AllowWrites` 和位于仓库外的质量报告，并校验报告的候选 UID、`pass`、质量门禁、代表性
文档数、`P95 <= 300ms` 与时间新鲜度；只读确认运行中的 backend 暴露 swap CLI 后才提交
Meilisearch swap。它不接收或传递 Provider key，不自动删除旧索引：

```powershell
pwsh ./scripts/integration-article-index-swap.ps1 `
  -CurrentIndex article_chunks_v1 `
  -CurrentIndexVersion v1 `
  -CurrentProvider '<previous-provider>' `
  -CurrentModel '<previous-model>' `
  -CurrentModelVersion '<previous-model-version>' `
  -CandidateIndex article_chunks_v2 `
  -QualityEvidencePath 'C:\\Temp\\article-chunks-v2-quality.json' `
  -CurrentMinimumDocumentCount 1 `
  -AllowWrites
```

```go
plan, err := search.NewArticleChunksIndexSwapPlan(currentSpec, targetSpec)
task, err := search.SwapArticleChunksIndex(ctx, meili, plan)
_, err = search.WaitForArticleChunksTasks(ctx, meili.TaskReader(), targetSpec.UID, []*meilisearch.TaskInfo{task}, interval)
```

切换入口会先检查两端索引存在且 Primary key 匹配，再提交一个 `rename=false` 的异步
swap task。必须等待该 task `succeeded`，并重新执行第 4 节的最小抽样；提交 task 不等于
切换完成。切换前后记录当前/候选 UID、task UID、时间、操作者和验证结果。

2026-09-02 隔离执行记录：先确认 `article_chunks_v2` 与 `article_chunks_v1` 均为
`177` 个文档、主键均为 `id` 且未处于 indexing；v1 的质量报告通过全部门禁。随后在
当前源码配置指向 v1、并显式为隔离 CLI 设置 `BENETNASCH_AI_ARTICLE_INDEXING=true`
的前提下，提交以下等价于本节的受控命令：

```powershell
docker compose -p benetnasch-integration --env-file .env.integration `
  -f docker-compose.integration.yaml exec -T backend sh -c `
  'BENETNASCH_AI_ARTICLE_INDEXING=true /app/benetnasch article-index swap --current-index article_chunks_v2 --current-index-version v2 --current-provider alibailian --current-model qwen3.7-text-embedding --current-model-version qwen3.7-text-embedding --current-dimension 1024 --current-batch-size 32 --candidate-index article_chunks_v1 --allow-writes'
```

task `68` 等待至 `succeeded` 后，重新对配置指向的 v1 执行质量/P95 门禁并确认
`177` 个文档；v2 未删除，仍可通过反向 swap 回滚。此前“v1 为空、swap 被阻断”的
段落属于历史记录，不再描述当前隔离状态。

## 6. 回滚与观察期

如果切换后出现质量、延迟、公开性过滤或错误率回归：

1. 立即停止继续回填和新版本写入；
2. 确认原切换 task 已完成且两个 UID 仍存在；
3. 使用 `plan.Reverse()` 生成反向计划，复核后提交反向 swap；
4. 等待反向 task 成功，再执行最小查询、权限和性能验证；
5. 保留两个索引、task 记录、backfill 状态和日志，直到观察期结束。

旧索引只有在单独的后续发布窗口、备份可恢复、无回滚需求且经过第二人复核后，才可
制定删除计划。回滚失败时保持当前数据不再进行猜测性删除，并升级给发布负责人。

## 7. 禁止事项

- 不在 `bootstrap.InitializeRuntime`、HTTP handler 或 worker 构造函数里调用 provisioning、
  backfill 或 swap。
- 不复用旧 UID 承载新的模型/版本/维度契约。
- 不以“任务已入队”作为“索引已完成”的证据。
- 不跳过公开性过滤、备份、质量门禁或回滚演练。
- 不通过修改现有容器、删除数据卷或执行未审批迁移来修复切换问题。
