# 数字空间 Web 性能与发布资产门禁

本文验证公开空间入口、管理入口和 admin-next 的构建资产与浏览器基线。不执行容器操作、数据库迁移、索引写入、对象存储写入或生产 Caddy 切换。

## 固定门槛

- 使用 `scripts/check-frontend-budgets.ps1` 检查三套前端的入口、总 JS/CSS、`dist` 和 `index.html` 大小；
- 公开空间 Chromium 基线检查 DOMContentLoaded、load 和 first-contentful-paint 均小于 5 秒；
- `/galaxy` 普通模式最多绘制 2,000 个点，limited 模式最多 400 个点；
- `deviceMemory <= 2`、`hardwareConcurrency <= 2` 或 `prefers-reduced-motion: reduce` 直接使用 limited 模式；
- 真实设备帧率、移动端电量和生产 RUM 不由本地基线代替。

## 本地检查

```powershell
pwsh ./scripts/safe-preflight.ps1 -BuildFrontend -RunBrowserBaseline
pwsh ./scripts/check-frontend-budgets.ps1
pwsh ./scripts/check-admin-next-release-assets.ps1
```

检查会构建或读取本地资产，默认使用 mock API，不连接生产服务。构建产物不提交到 Git。

## 隔离联调

真实 API、菜单、RBAC、刷新和 Agent 控制面验证使用隔离 Caddy：

```powershell
pwsh ./scripts/integration-browser-e2e.ps1 -AdminNextOnly -RequireAgentRoutes
```

公开空间和 admin-next 的入口必须检查：首屏、API 反向代理、路由刷新、未授权页面、错误页面、静态资源加载和控制台关键错误。管理写操作另行使用带授权的 CRUD 脚本。

## 发布资产切换

生产静态目录切换必须满足：

1. 产物由明确 commit 构建并记录入口 SHA-256；
2. 新入口在隔离 Caddy 完成菜单、刷新和权限验收；
3. 旧目录先移动到已校验的独立归档目录并保留回滚周期；
4. 切换失败保留新旧证据并恢复旧入口；
5. 保留期结束前不得清理旧产物。

仓库提供 `scripts/production-caddy-admin-switch.ps1` 作为受保护的计划/执行入口。普通文档、前端构建和本地 Compose 不得把生产目录当作目标，也不得自动重载生产 Caddy。
