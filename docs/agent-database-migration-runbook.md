# Agent 数据库迁移、备份与失败恢复手册

本手册针对 Agent 相关 `schema_migrations` 和显式迁移。它不由服务启动流程执行；
`go run . migrate --allow-writes` 是唯一明确的迁移写入口；`migrate status` 只读。当前仓库不会在本轮自动执行迁移，也不会
停止、重建或修改现有 PostgreSQL/Redis/Caddy/Meilisearch/MinIO 容器。

## 1. 迁移前检查

在独立发布单中记录：

- 发布二进制版本、Git commit、目标环境和迁移编号；
- 当前 `schema_migrations` 版本、名称和 checksum；
- 预计新增表、索引、约束、锁影响和所需磁盘空间；
- 若包含 `0017_agent_memory_history_conflicts`，确认断言历史、冲突和冲突成员表的
  外键/部分唯一索引可用，并确认应用发布顺序不会在迁移前调用 history/conflict 读写；
- 若包含 `0018_agent_memory_admin_rbac`，确认 `0015_agent_admin_rbac` 已存在 AI
  资源父节点，并确认新 API 仍由管理员认证/Casbin 保护；该迁移不创建菜单，持久化
  开关仍须单独显式启用；
- 若包含 `0019_agent_memory_admin_menu`，确认 admin-next 已发布包含
  `/ai/Memory.vue` 的构建产物，并确认菜单入口只绑定现有启用的 `admin` 角色；
- 若包含 `0021_agent_vision_rbac`，确认 `0015_agent_admin_rbac` 已存在 AI 资源父节点，
  `/admin/ai/vision/preview` 已由管理员认证/Casbin 保护，且 `ai.vision` 仍保持关闭直到
  Vision Provider smoke 和审核链验收完成；该迁移只增加资源授权，不执行模型调用。
- 若包含 `0022_ai_observability_rbac`，确认 `0015_agent_admin_rbac` 已存在 AI 资源父节点，
  `/admin/ai/observability` 已由管理员认证/Casbin 保护；该迁移只增加脱敏观测接口的资源
  授权，不执行 Provider 烟测、搜索请求或外部监控写入，`ai.observability` 仍须单独显式启用。
- 若包含 `0023_space_companion`，确认目标是隔离 Compose，已准备独立的 read/publish
  令牌且两者不同，`agent/moonfei` 只作为独立 agent 主体存在；确认 Companion bridge
  的开关仍关闭或与迁移窗口一致。该迁移创建公开发布审计表，不授予人类后台角色权限，
  也不改变现有 Casbin 会话。
- 应用兼容窗口：旧版本是否能在新 schema 上继续读写；
- 数据库备份位置、恢复负责人、回滚方式和验证结果。
- 若历史导入曾显式写入自增 ID，额外检查所有 identity/serial sequence 是否不落后于
  对应表的最大 ID；尤其要检查 `t_operation_log`，否则日志 INSERT 可能触发主键约束
  （例如 `_copy_14`）。这属于独立的数据修复动作，不由服务启动或普通迁移自动执行。

已应用迁移的文件名或内容禁止直接修改。Runner 会校验 name/checksum，发现漂移时
拒绝继续，修复必须新增更高版本迁移。

## 2. 备份与恢复演练

在发布窗口使用经过批准的 PostgreSQL 备份工具或平台完成：

1. 生成包含 schema、数据、权限和扩展信息的可恢复备份；
2. 记录备份时间、数据库版本、文件校验和与保留期限；
3. 在隔离数据库恢复一次，并验证 `schema_migrations`、核心文章/用户/评论数据及
   Agent 新表可读；
4. 未完成恢复验证不得执行生产迁移。

隔离演练入口为：

```powershell
# 仅生成备份产物（不创建恢复库）
pwsh ./scripts/integration-backup-restore.ps1 -AllowWrites

# 生成备份并恢复到临时库进行校验
pwsh ./scripts/integration-backup-restore.ps1 -AllowWrites -AllowRestore
```

该脚本固定使用 `benetnasch-integration` 项目和仓库内的隔离 Compose 文件，主动忽略
本机 Compose override；它把 custom dump 和 globals 导出到仓库外的临时目录，并恢复到
随机命名的临时数据库，再比较迁移数量、迁移 checksum 摘要和核心/Agent 表行数。仅备份
模式只需 `-AllowWrites`，不会创建恢复库；只有额外传入 `-AllowRestore` 才执行恢复校验。
恢复数据库会在验证后删除，备份产物和脱敏 `metadata.json` 留在指定目录供发布单留存；
脚本不会覆盖已有文件，也不会把数据库导出物写入仓库。入口不会自动启动、停止或重建
容器。生产备份仍由部署平台在独立发布窗口执行。

生产发布单的备份证据可以在产物已经由平台附加后执行只读校验：

```powershell
# 只显示证据要求，不读取或改变任何外部状态
pwsh ./scripts/production-backup-evidence.ps1 -Action Plan

# 校验平台提供的生产备份产物、恢复演练标记、SHA-256 和保留期限
pwsh ./scripts/production-backup-evidence.ps1 `
  -Action Verify `
  -BackupDirectory 'D:\release-evidence\<ticket>\postgres-backup' `
  -ReleaseTicket '<ticket>' `
  -MinimumRetentionHours 168
```

`Verify` 要求目录位于仓库之外，且包含 `benetnasch.dump`、
`postgres-globals.sql` 和 `metadata.json`。metadata 使用
`benetnasch.production-backup.v1` 契约，必须标记 `environment=production`、
`restoreEnvironment=isolated`、`restoreVerified=true`，并记录与文件匹配的 SHA-256
及未到期的保留截止时间。该入口只读校验本地证据，不连接 PostgreSQL、不调用 Docker、
不执行备份/恢复、不创建或删除文件；校验通过也不代替平台实际完成备份和恢复演练。

## 3. 前滚

只在窗口内执行：

```text
go run . migrate --allow-writes
```

Runner 会先在同一 PostgreSQL session 上获取 advisory lock，串行化同一数据库的并发
迁移命令（必须显式传入 `--allow-writes`）；每个迁移步骤均在独立事务内执行：SQL 成功且版本记录成功后才提交。任何
SQL、版本记录或提交失败都会回滚当前迁移并停止后续步骤，已经提交的更早版本保持
可追踪。数据库中存在当前二进制未包含的已应用版本时会直接阻断，避免旧二进制误判
schema 状态。命令返回成功后重新读取 `schema_migrations`，保存版本、名称和 checksum。

隔离联调可以使用：

```powershell
pwsh ./scripts/integration-migrate.ps1 -AllowWrites -VerifyIdempotency
```

该脚本只针对 `benetnasch-integration` 隔离项目，并且默认拒绝写入；只有在批准的隔离
窗口中显式增加 `-AllowWrites` 才会执行迁移。第二次执行用于验证已应用版本会被
checksum 校验并跳过，不代表生产迁移已经执行。

迁移窗口前只读取当前状态时使用：

```powershell
pwsh ./scripts/integration-migration-status.ps1
```

该脚本只在已运行的隔离 backend 容器内执行 `migrate status`，不会启动、重建或修改
Compose 服务，也不会创建 `schema_migrations`、获取 advisory lock 或执行迁移 SQL。
若容器中的二进制尚未包含 `migrate status`，脚本应直接失败并等待下一次批准的镜像
部署，不得改用 `integration-migrate.ps1` 代替只读检查。

`integration-up.ps1` 在启动（或使用 `--build` 更新）隔离栈后也会先执行同一能力探测；
因此 `integration-deploy.ps1` 不再生成不会被 Dockerfile 使用的主机侧 backend 二进制，
而是以镜像 builder 的当前源码编译结果作为唯一 backend 产物。能力探测失败时不会进入
后续迁移步骤。

Windows 主机内存不足以承担 Docker/WSL Go builder 时，已获批的隔离窗口可以先运行
`scripts/integration-native-backend.ps1 -Build -AllowContainerChanges`。该入口使用宿主机
原生 Go 交叉编译结果，只备份并替换 `benetnasch-integration` backend，严格校验 Compose
project/service label 和二进制哈希；它不执行迁移、seed、Meilisearch/MinIO 写入，生产发布
仍须使用独立发布手册。容器若之后被 Compose 重建，原生二进制不会自动保留，需重新部署。

隔离 Compose 的 PostgreSQL 和 Meilisearch 数据卷带有版本纪元后缀（当前分别为 `v16` 和
`v153`）。如果旧卷来自 PostgreSQL 14 或 Meilisearch 1.1，禁止直接挂载到新引擎，也不
删除旧卷；更新隔离栈时会创建新的版本化卷，随后按本节显式执行 migration/seed。旧卷只
作为隔离回滚参考，不属于生产数据，也不应被提交到仓库。

## 4. 失败处理

如果迁移失败：

1. 保留完整命令输出、数据库日志和失败版本，不手工插入 `schema_migrations`；
2. 检查当前迁移事务是否已回滚，并确认失败版本没有错误地标记为已应用；
3. 先恢复应用兼容性或修复迁移脚本，再以新二进制重试；已应用版本不能编辑；
4. 若数据/schema 已不满足兼容窗口，停止应用写入，按备份恢复演练执行恢复；
5. 恢复后再次运行只读校验和隔离 smoke，复核人批准后才能继续前滚。

迁移失败不自动触发数据删除、不自动回退已经提交的旧迁移，也不自动重启或重建任何
容器。只有明确审批的恢复操作才能改变数据库状态。

隔离环境可以用下面的命令验证失败事务和修复后重试：

```powershell
pwsh ./scripts/integration-migration-failure-recovery.ps1 -AllowWrites
```

该入口固定检查 `benetnasch-integration` 的 PostgreSQL 容器，然后在隔离数据库的随机
schema 中运行 `integration` 构建标签测试：第一条迁移成功，第二条迁移故意失败；测试
检查失败版本没有写入 `schema_migrations`、失败迁移创建的表已回滚，随后用修复后的同一
版本迁移重试并确认状态为 applied。测试结束会删除随机 schema，不会启动、重建或修改
Compose 服务，也不会触碰生产数据库。

如果进程被强制终止导致历史 probe schema 残留，可在隔离 repair 窗口执行：

```powershell
pwsh ./scripts/integration-clean-migration-probes.ps1 -AllowWrites
```

该脚本固定使用 `benetnasch-integration`，只接受 `migration_probe_<数字>` 名称，且会先
校验 schema 只包含本演练生成的六个表/索引对象；任一条件不满足就拒绝删除。清理后应再
执行 `integration-migration-status.ps1`，确认正式迁移表未变化。

如果出现操作日志主键重复，先停止继续写入并在批准的隔离/发布窗口检查 identity
sequence；仓库会把 PostgreSQL `23505` 视为永久错误而不盲目重试。隔离环境的修复和
校验脚本为 `scripts/integration-repair-sequences.ps1`，其 SQL 只在对应的隔离 Compose
项目中执行，且默认拒绝写入，必须显式传入 `-AllowWrites`；本仓库任务不会对现有
PostgreSQL 容器运行该脚本。脚本同时处理
`is_called=false` 且序列值刚好等于表最大 ID 的边界，避免下一次 `nextval` 再次发出已
存在的主键。

## 5. 验收清单

- `schema_migrations` 连续、无重复版本，checksum 与发布二进制一致；
- 文章、用户、评论、RBAC 和操作日志主链路读写正常；
- Agent job/review/task/memory/projection 等表的约束和索引存在；其中 Memory v3 还应
  验证断言修订快照可追加、同一主体/谓词最多一个 open conflict，resolve/reject 后
  成员状态和历史记录一致；0018 的六个 memory admin 资源已绑定现有启用 admin
  角色；0019 的 `/ai-memory` 菜单组件已注册且未落到占位页；
- 0021 的视觉预览资源已绑定现有启用的 `admin` 角色，视觉 feature flag 和 Provider
  发布门禁状态与发布单一致；
- 0022 的运行时观测资源已绑定现有启用的 `admin` 角色，观测 feature flag、外部监控
  采集凭据和发布单状态一致；接口只返回脱敏快照；
- 旧版本兼容性和新版本 API smoke 通过；
- 备份可恢复，且恢复点和恢复耗时符合发布单；
- 应用日志没有把数据库凭据、完整用户正文或 Provider 密钥写出；
- 迁移完成后，索引回填、Meilisearch swap、Caddy 静态目录切换仍需独立发布步骤。

## 6. 当前代码证据

- 迁移 Runner：`app/infra/persistence/migration/runner.go`；
- Companion 主体与发布表迁移：`app/infra/persistence/migration/migrations/0023_space_companion.sql`；
- Companion 主体运行时校验：`app/infra/persistence/repository/space_principal_repository.go`、
  `app/infra/middlewares/space_companion.go`；
- advisory lock、未知版本阻断、checksum、排序和非法文件测试：
  `app/infra/persistence/migration/runner_test.go`；
- 显式命令：`cmd/migrate.go`；
- 隔离迁移与二次幂等检查：`scripts/integration-migrate.ps1`、
  `scripts/integration-deploy.ps1`。
- 隔离迁移失败回滚和修复后重试：
  `scripts/integration-migration-failure-recovery.ps1`、
  `app/infra/persistence/migration/runner_integration_test.go`；CI integration job
  在启用 `TESTCONTAINERS_ENABLED=1` 时使用固定 digest 的临时 PostgreSQL 重复该测试。
- 隔离备份、临时数据库恢复和行数/checksum 验证：
  `scripts/integration-backup-restore.ps1`；它只提供演练入口，未执行不代表生产备份
  可恢复门禁已完成。
- 生产发布单备份证据的只读校验：`scripts/production-backup-evidence.ps1`；它不执行
  生产备份、恢复或数据库连接，`OPS-02` 仍须等待独立发布窗口的真实结果。
