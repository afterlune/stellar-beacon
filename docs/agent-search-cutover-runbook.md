# 数字空间检索索引回填与切换手册

本文用于 `article_chunks_<version>` 的显式 provision、全量回填、质量验证、swap 和回滚。它不是应用启动动作；服务和 Worker 不会自动创建索引、回填或切换。

## 当前隔离状态

截至 2026-09-02 的隔离证据：阿里百炼 `qwen3.7-text-embedding` 已完成接口验证，`article_chunks_v1` 完成 10 篇公开文章、177 个 Chunk 的回填和只读质量/P95 检查；候选 UID `article_chunks_v2` 保留为回滚入口。该证据只适用于隔离环境，不代表生产 Meilisearch 已操作。

模型、Provider、版本或向量维度改变时，必须新建索引 UID，不能把不同契约混写在同一索引中。`qwen3.7-text-embedding-flash` 只能作为另一完整版本的显式候选。

## 硬性约束

- 只操作发布单中明确声明的候选 UID；
- 先备份数据库和目标 Meilisearch，再进入任何写入或切换阶段；
- 公开性过滤必须在源查询和索引写入两侧执行，删除/私密化内容不能继续出现；
- 候选索引必须非空且文档数达到代表性门槛，空索引禁止 swap；
- `rename=false` 交换 UID 的文档、设置和任务历史但不交换 UID 名称，必须记录两端状态；
- 旧索引在观察期结束、备份可恢复且经过复核前不得删除；
- 任何生产索引操作都必须另开窗口，不由启动、HTTP handler 或 Worker 构造函数触发。

## 只读预检

记录以下内容后才可申请写入：

1. 当前索引 UID、候选 UID、文档数、设置和应用配置；
2. Embedding Provider、模型、模型版本、维度、批大小和索引契约；
3. 公开文章数、预期 Chunk 数、估计请求量、预算、超时和取消策略；
4. 数据库备份、Meilisearch 备份、回滚 UID、停止条件和观察时长。

只读计划脚本不执行 provision、回填或 swap：

```powershell
pwsh ./scripts/integration-article-index-plan.ps1
```

## 隔离回填与质量门禁

在获批的隔离窗口中，按以下顺序执行仓库脚本：

1. 创建并锁定候选索引设置；
2. 只读扫描公开文章并执行有界批处理；
3. 等待 Meilisearch 任务完成，不以“任务已入队”作为成功；
4. 验证候选文档数、字段、公开性、重复、空内容和模型元数据；
5. 使用固定查询计算 Recall@8、MRR、无结果准确率和 P50/P95/最大耗时；
6. 保存任务 UID、索引 UID、文档数、质量报告和错误摘要到仓库外。

示例入口（均需隔离授权和具体参数）：

```powershell
pwsh ./scripts/integration-article-index-backfill.ps1 -AllowWrites
pwsh ./scripts/integration-search-readonly.ps1 -MinimumDocumentCount 1
```

质量门禁失败时不发送 swap 请求，先修复数据、Provider、限流或查询契约。

## 原子切换与回滚

切换前必须再次读取两端实时文档数并确认候选非空、当前索引可回滚：

```powershell
pwsh ./scripts/integration-article-index-swap.ps1 -AllowWrites
```

切换成功后验证公开搜索、私密/删除过滤、Agent 引用、后台搜索、任务状态和 P95。观察期内保留旧 UID、任务记录和质量证据。出现召回下降、错误率、延迟或权限异常时，使用反向 swap 回到旧 UID，并重新执行最小查询验证。

## 禁止事项

- 不在启动流程、普通请求或后台 Worker 中执行索引写入；
- 不复用 UID 承载新模型/维度；
- 不跳过备份、公开性过滤、质量门禁或回滚演练；
- 不删除数据卷或修改生产容器来“修复”索引；
- 不把隔离证据、旧索引或 mock 结果描述成生产完成。
