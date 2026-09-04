# Benetnasch 协作与工程约束

本文件是仓库级的长期协作规则。用户、系统和开发者指令优先于本文件；代码、配置和测试是最终事实来源，文档只负责导航和记录取舍。

## 开始任务

- 先阅读 [README](README.md)、[文档中心](docs/README.md) 和与任务相关的路线图、ADR、接口或运行手册。
- 用 `rg` / `rg --files` 定位实现、路由、配置、测试和脚本，再沿调用链确认影响范围。
- 修改前检查 `git status`、`git diff --stat`，保留用户已有的无关改动；不要使用 `git reset --hard` 或 `git checkout --` 覆盖工作树。

## 运行环境安全

- 不停止、重建、迁移或修改用户现有的 PostgreSQL、Redis、Caddy、Meilisearch、MinIO 容器。
- 未经单独授权，不执行数据库迁移，不向 Meilisearch provision、回填、swap、删除或写入数据；发布窗口操作只写入手册和可审计脚本。
- `.codebuddy/` 是第三方评估目录，保持不改动、不过度扫描。
- 不把密钥、token、运行时数据、日志、证书、数据库导出物或前端构建产物加入 Git。
- 所有外部输入都要考虑鉴权、大小限制、Origin、速率限制、日志脱敏、超时和取消传播。

## 分层与实现

- `app/application` 只依赖 `app/domain/port`；Eino、Provider、Meilisearch、Redis、对象存储和 xorm 类型留在 `infra`/bootstrap 边界。
- 请求 Context 必须沿 handler、service、repository 和外部适配器传播；后台任务使用 supervisor 的生命周期。
- Agent 生成内容默认先进入审核；模型不能决定身份、权限、审核结果、发布动作或计费规则。
- 新能力必须由独立 feature flag 控制，默认关闭；迁移、索引和 Caddy 静态目录切换必须是显式运维步骤。
- 日志使用标准库 `log/slog`，禁止输出密钥、完整 Prompt、访客正文、Provider 参数或数据库查询参数。

## 验证与文档

- 相关改动至少运行对应单元测试、`git diff --check` 和相关边界扫描；跨模块改动再运行 `go test -count=1 ./...` 与 `go vet ./...`。
- 不具备外部服务、凭据或发布窗口时，明确记录跳过项；离线 fake、旧索引和 mock 浏览器不能冒充真实生产门禁。
- 接口、事件、配置、依赖方向、数据流或启动方式变化时，同步更新 `docs/`；长期技术取舍写入 `docs/adr/`。
- 除非用户明确要求，不自动 commit、push、改远程或创建 PR。

常用的无副作用检查入口是 `scripts/safe-preflight.ps1`。它不会启动/停止容器、执行迁移、写入 Meilisearch 或切换 Caddy 静态目录。
