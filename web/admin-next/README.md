# Benetnasch Admin Next

`admin-next` 是数字空间控制台的 Vue 3 + Vite + Pinia + Arco Design 入口，与稳定管理入口并行运行。它负责空间内容、用户权限、任务、审核和 Agent 控制面的可视化操作；真正的权限判断始终在后端完成。

当前已覆盖登录、Token 恢复、动态菜单、布局、403/404、文章、分类、标签、评论、用户、角色、日志、任务、相册、友链、网站配置、关于我、个人中心以及 AI Studio、审核、Agent 人设、审核策略和记忆审核入口。未迁移的复杂页面必须显示明确占位，不得根据后端路径任意加载组件。

## 开发和构建

```powershell
npm ci
npm run serve
npm run build
```

默认开发地址为 `http://127.0.0.1:8082`。API 保持 `/api` 前缀，开发代理目标默认为 `https://localhost:7777`，可用 `VITE_ADMIN_API_TARGET` 覆盖。

任务页只有后端返回 `canRunOnce=true` 且目标属于固定白名单时，才允许执行一次；前端不能执行数据库返回的任意 `invokeTarget`。

## 测试

本地 mock 基线：

```powershell
npx playwright install chromium
npm run test:e2e:baseline
```

基线测试不启动、不停止、不修改 PostgreSQL、Redis、Caddy、Meilisearch 或 MinIO。

真实隔离入口由仓库根目录脚本负责：

```powershell
pwsh ../../scripts/integration-browser-e2e.ps1 -AdminNextOnly
pwsh ../../scripts/integration-browser-e2e.ps1 -AdminNextOnly -RequireAgentRoutes
```

脚本只接受 loopback 隔离 Caddy（`127.0.0.1:18018`），临时 token 不输出、不落盘。登录会产生后端认证和操作日志，因此不能把该流程描述成“零写入”。

需要验证本地已运行的 `benetnasch-dev` 时，使用 `http://127.0.0.1:28018`；该入口不会切换生产 Caddy。

真实增删改验收必须显式授权：

```powershell
pwsh ../../scripts/admin-next-crud-e2e.ps1 -AllowWrites
```

该流程仅针对隔离数据执行分类、标签、友链和说说的新增、编辑、读取和清理。没有 `-AllowWrites` 时不得执行写入验收。

## 路由和权限要求

- 菜单组件只能映射到前端本地白名单；未知组件进入占位页。
- 刷新详情页、列表页、日志页和 Agent 控制页必须保持正确 URL，不得出现空白或任意重定向。
- 所有非 GET/HEAD/OPTIONS 操作必须有明确页面动作和后端权限；只读矩阵不能替代后端 RBAC 测试。
- Agent 菜单由显式迁移和 Casbin 资源控制；空间 Companion token 不能登录管理控制台。
- AI 输出只能进入审核流程，前端不能直接批准、发布或扩大模型权限。

## 发布边界

构建产物位于 `dist`，不提交到仓库。生产静态目录切换必须使用独立发布脚本，先进行哈希校验、备份和归档；本 README、前端测试或本地 Caddy 都不会切换生产目录。
