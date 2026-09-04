# 数字空间 Web 入口

本目录包含三个独立前端，它们共享 Benetnasch 的空间 API，但职责不同：

- `blog`：数字空间公开入口，目录名保留以兼容现有构建和部署路径；
- `admin`：稳定的 Vue 2 管理入口，作为迁移期间的兼容和回滚入口；
- `admin-next`：Vue 3 + Vite + Pinia + Arco Design 管理入口。

三个应用默认通过相对路径访问 `/api`。浏览器生产请求由 Caddy 转发，开发服务器可通过环境变量覆盖 API 目标。

## 本地开发

```powershell
cd web/blog
npm ci
npm run serve
```

```powershell
cd web/admin
npm ci
npm run serve
```

```powershell
cd web/admin-next
npm ci
npm run serve
```

默认情况下：

- `blog` 使用 Vite，API 目标由 `VITE_API_TARGET` 覆盖；
- `admin` 使用 Vue CLI，API 目标由 `VUE_APP_API_TARGET` 覆盖；
- `admin-next` 使用 Vite，API 目标由 `VITE_ADMIN_API_TARGET` 覆盖。

## 构建

```powershell
cd web/blog; npm ci; npm run build
cd ../admin; npm ci; npm run build
cd ../admin-next; npm ci; npm run build
```

产物位于各应用的 `dist`，不提交到 Git。`blog` 仍提供 `build:legacy`/`serve:legacy` 作为构建兼容路径；兼容产物位于 `dist-legacy`，不会覆盖当前产物。

生产静态目录不由本仓库自动切换，也不执行远程复制。静态目录切换必须使用带备份、哈希校验、归档和回滚的独立发布流程。

## 隔离查看环境

完整数字空间查看环境使用 `benetnasch-dev` Compose 项目：

| 入口 | 地址 |
| --- | --- |
| 公开空间 | `http://127.0.0.1:28080` |
| 稳定管理入口 | `http://127.0.0.1:28008` |
| admin-next | `http://127.0.0.1:28018` |
| Companion bridge | `http://127.0.0.1:28028` |

启动、数据边界和停止命令见 [开发副本运行手册](../docs/dev-clone-runbook.md)。该项目只使用独立端口和 `benetnasch-dev-*` 命名卷，不操作生产 Compose。

另有固定的 `benetnasch-integration` 隔离验收栈，端口为 `18080`、`18008`、`18018` 和 `18028`。所有会改变容器、数据库、Meilisearch 或 MinIO 的脚本默认拒绝执行，必须在已批准的隔离窗口显式授权。

## 前端验证

公开入口和 admin-next 提供本地 mock 基线：

```powershell
cd web/blog
npm ci
npm run test:e2e:baseline
```

```powershell
cd web/admin-next
npm ci
npx playwright install chromium
npm run test:e2e:baseline
```

真实隔离 API 验收从仓库根目录执行：

```powershell
pwsh ./scripts/integration-browser-e2e.ps1 -AdminNextOnly
```

完整管理入口验收使用 `-RequireAgentRoutes`；真实增删改验收使用专门的 `admin-next-crud-e2e.ps1 -AllowWrites`，只接受隔离 Caddy 地址，并在结束时清理测试数据。

静态资源门禁：

```powershell
pwsh ./scripts/check-frontend-budgets.ps1
pwsh ./scripts/check-admin-next-release-assets.ps1
```

这些检查不启动、停止或修改任何容器。真实设备帧率、生产 RUM 和生产 Caddy 发布证据必须单独采集。

## 前端安全边界

- 前端路由只允许进入已登记的组件白名单，不根据后端返回的任意路径动态加载模块。
- 前端隐藏按钮不等于权限控制，所有管理权限由后端认证、Casbin 和 feature flag 决定。
- 不在前端包、日志、测试产物或错误页面中写入 token、Provider key、Prompt 或数据库数据。
- 图片、视频和外链内容按后端公开性、URL 白名单、CSP、懒加载和大小限制处理。
