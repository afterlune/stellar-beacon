# 数字空间数据库迁移、备份与恢复手册

本文覆盖 Agent、审核、记忆、媒体、控制面和 Companion 主体相关迁移。迁移由显式命令执行；服务启动、HTTP 请求和后台 Worker 不自动改表。

## 迁移事实

- 迁移文件位于 `app/infra/persistence/migration/migrations`，使用四位版本号和 checksum；
- runner 使用 advisory lock 串行执行，未知版本、缺失连续版本或 checksum drift 会阻断；
- `migrate status` 只读取状态，不创建 `schema_migrations`、不加锁、不执行 SQL；
- `migrate --allow-writes` 是唯一应用迁移的 CLI 写入口；
- 启动前必须确认目标环境、commit、迁移范围、应用兼容性、备份和回滚方式。

只读状态检查：

```powershell
go run . migrate status
```

执行迁移前必须显式确认写入授权：

```powershell
go run . migrate --allow-writes
```

上述命令不应直接对生产环境执行。生产窗口必须由发布负责人提供目标、审批单、备份证据和回滚计划。

## 关键迁移

- `0000_baseline_existing_schema`：声明已有空间表基线；
- `0001`—`0014`：Agent 任务、审核、行为、活动、时间胶囊、媒体和人设模型；
- `0015`—`0022`：管理员 Agent 资源、审核策略、记忆审核、Provider/Vision 和脱敏观测资源；
- `0023_space_companion`：创建 `t_agent_principal` 和 `t_space_publication`，种子 `agent/moonfei`，只允许状态/梦境/电台发布。

`0023` 不创建人类 Casbin 会话、不授予管理员权限，也不会自动启用 Companion bridge。重复执行保持 operator 对主体 `enabled` 的禁用决定，不把禁用主体重新打开。

## 迁移前检查

发布记录至少包含：

- Git commit、后端二进制哈希、目标 Compose 项目和数据库连接范围；
- 当前 `schema_migrations` 版本、名称和 checksum；
- 新表、索引、约束、锁影响、磁盘空间和估计耗时；
- 旧应用在新 schema 上是否仍能读写，以及前端是否包含所需菜单组件；
- Provider/Embedding/Vision/Companion 开关是否仍按计划关闭；
- 备份位置、恢复负责人、校验方式、观察期和停止条件。

## 隔离环境流程

隔离环境使用固定 `benetnasch-integration` 或 `benetnasch-dev` 项目，不得访问生产目标：

1. 只读确认 Compose 服务、项目标签、端口、数据卷和目标 DSN；
2. 先备份隔离数据库并记录 SHA-256、表计数和 migration 状态；
3. 运行迁移，随后再次运行 `migrate status` 检查全部 applied 且无 drift；
4. 重启/检查 backend、公开入口、管理入口和 Companion bridge；
5. 验证主体、作用域、公开性过滤、白名单、幂等、审计和回滚；
6. 把真实输出放在仓库外，并注明未执行项。

仓库提供的 `scripts/integration-*.ps1` 默认只针对隔离 integration Compose，并通过授权开关保护容器、数据库、索引和对象写入。不要把未指定项目名的宽泛 Compose 命令用于发布。

## 备份与恢复

隔离备份恢复脚本可以生成 dump、恢复到隔离临时数据库并校验表计数和 migration digest：

```powershell
pwsh ./scripts/integration-backup-restore.ps1 -AllowWrites -AllowRestore
```

它不是生产备份证据。生产备份由部署平台在独立窗口完成，仓库中的 `production-backup-evidence.ps1` 只校验外部提供的 manifest、哈希、恢复时间和保留期，不连接生产 PostgreSQL，也不执行备份/恢复。

恢复验收必须确认：

- dump 可读取且哈希与记录一致；
- 恢复库表计数、关键约束和 migration digest 与预期一致；
- 应用能够连接恢复库并通过最小公开/管理 smoke；
- 未将恢复库、dump、token、日志或对象复制物提交到仓库。

## 失败与回滚

- 单个迁移 SQL、checksum、版本记录或提交失败时，当前迁移事务回滚，后续版本停止；
- 先用只读 status 确认失败版本没有错误标记为 applied，确认失败迁移创建的对象已回滚；
- 修复同一版本后重新执行，不通过手工插入 `schema_migrations` 绕过 runner；
- 应用不兼容时回滚应用版本或关闭对应 feature flag，保留 schema 和审计证据；
- 生产恢复失败时停止后续写入并升级发布负责人，不删除备份、不猜测性清理数据。

隔离失败恢复测试位于 `app/infra/persistence/migration/runner_integration_test.go`，不能代替生产恢复演练。

## 当前状态

- Agent 与 Companion 迁移已写入并有 parser/unit 测试；
- `benetnasch-dev` 当前启动使用已有隔离卷，文档重建本身没有执行迁移或数据复制；
- 隔离迁移、明确数据复制和完整恢复证据仍需单独执行；
- 生产迁移、备份恢复和现有容器操作不属于普通开发任务。
