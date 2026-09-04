# Benetnasch Admin Next

这是管理后台的 Vue 3 + Vite + Pinia + Arco Design 渐进式迁移入口，和现有
`web/admin` 并行存在。当前已迁移登录、Token 恢复、动态菜单、布局、403/404、文章/
分类/标签/评论/用户/角色/日志/任务/相册/友链列表与管理，以及 AI Studio 预览、审核队列、
人设、版本化审核策略、记忆断言/冲突审核、网站配置、关于我、个人中心和安全的任务
“执行一次”；其余复杂编辑页面仍逐步替换为明确的占位页，旧后台仍是可部署的回滚版本。

## 开发与构建

```shell
npm ci
npm run serve
npm run build
```

默认访问 `http://127.0.0.1:8082`，API 使用相对路径 `/api`，开发代理目标默认为
`https://localhost:7777`，可通过 `VITE_ADMIN_API_TARGET` 覆盖。菜单中的后端组件路径
只会进入前端白名单映射；未迁移组件显示占位页，不执行任意路径加载。
任务页只有后端返回 `canRunOnce=true` 的固定白名单目标可执行一次；该动作只消费一个
已入队任务，不能执行数据库中的任意 `invokeTarget`。

## 基线测试

```shell
npx playwright install chromium
npm run test:e2e:baseline
```

测试默认 mock 登录和菜单 API，不启动或修改任何 PostgreSQL、Redis、Caddy、Meilisearch
或 MinIO 容器。隔离联调启动后，可设置 `E2E_BASE_URL=http://127.0.0.1:18018` 对
Caddy 提供的新后台预览入口运行同一套测试；该入口不会切换主 Caddy 的静态目录。

完整隔离后台验收可从仓库根目录执行：

```powershell
pwsh ./scripts/integration-browser-e2e.ps1 -AdminNextOnly
```

该脚本使用已注入的 `E2E_ADMIN_EMAIL`/`E2E_ADMIN_PASSWORD` 在 loopback 预览入口取得
临时 token，并在同一进程内启用登录测试和只读菜单矩阵；token 不会输出或落盘，脚本结束
后也会恢复环境变量，且登录请求不跟随 HTTP 重定向。登录会产生后端登录/操作日志，
所以这不是数据库零写入测试；脚本
不会启动、停止或修改任何容器，也不会切换生产 Caddy 静态目录。

默认模式兼容尚未执行 Agent 菜单迁移的旧后端；发布窗口应使用：

```powershell
pwsh ./scripts/integration-browser-e2e.ps1 -AdminNextOnly -RequireAgentRoutes
```

严格模式会把 AI Studio、人设、审核策略和记忆菜单列为必需入口，缺失任一项即失败。

如果只需要把本地 admin-next 接到已经运行的 Caddy，可让 Vite 保留 `/api` 前缀，
例如（PowerShell）：

```powershell
$env:E2E_REAL_INTEGRATION = '1'
$env:E2E_ADMIN_TOKEN = '<已有管理员 token>'
$env:VITE_ADMIN_API_TARGET = 'http://127.0.0.1:18008'
$env:VITE_ADMIN_API_PRESERVE_API_PREFIX = '1'
& ../../scripts/admin-next-readonly-e2e.ps1
```

该命令默认只执行 token 会话下的 GET/HEAD/OPTIONS 页面验收，不执行登录、增删改、
上传或迁移。需要验证真实登录流程时，必须另外显式设置
`E2E_ADMIN_ALLOW_LOGIN=1`，并提供 `E2E_ADMIN_EMAIL` 与 `E2E_ADMIN_PASSWORD`；
登录会产生后端登录/操作日志，因此不属于只读门禁。

只读矩阵会遍历已迁移的列表、编辑页、详情页、日志详情和 Agent 管理页，并在每个入口
及刷新后确认 URL 没有被重定向、内容根节点和模块标识可见；同时把 API 的非 GET/HEAD/OPTIONS、
HTTP 错误和网络请求失败都作为失败条件。它只证明当前 token 的 RBAC、API 响应和前端路由
能够联通，不会把后端 GET 请求可能产生的审计日志写入误认为“数据库完全无写入”。

也可以直接运行 `scripts/admin-next-readonly-e2e.ps1`；脚本会校验 API 目标只能是
现有 loopback Caddy `18008`，不传 `-AdminNextBaseUrl` 时只启动本地 Vite admin-next，
传入时仅接受隔离预览 `http://127.0.0.1:18018`。现有后端尚未执行 Agent 菜单迁移时，
Agent 管理入口按兼容模式跳过；在完整发布验收中使用：

```powershell
& ../../scripts/admin-next-readonly-e2e.ps1 -RequireAgentRoutes
```

此开关会把 AI Studio、人设、审核策略和记忆菜单列为必需入口，缺少任一入口即失败，
避免把旧后端的核心模块通过误判成完整迁移。

真实隔离 CRUD 验收需要显式授权写入，并且只接受 Caddy `18018`：

```powershell
pwsh ./scripts/admin-next-crud-e2e.ps1 -AllowWrites
```

该流程通过页面完成分类、标签、友链和说说的新增、编辑、刷新读取与删除，测试数据会在
结束时清理；没有 `-AllowWrites` 时会在读取环境和启动 Playwright 前拒绝执行。
