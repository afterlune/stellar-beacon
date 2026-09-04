# 前端性能门禁

这份门禁对应长期路线 M7-16。它只验证前端构建产物和本地 mock 浏览器行为，不启动、
停止或修改 PostgreSQL、Redis、Caddy、Meilisearch、MinIO 容器。

## 固定阈值

- bundle：使用 `scripts/check-frontend-budgets.ps1`，分别约束 blog、旧 admin 和
  `admin-next` 的最大/总 JS、CSS、dist 及 `index.html` 大小。
- 首屏：blog Chromium 基线要求 DOMContentLoaded、load 和可用的
  `first-contentful-paint` 均小于 5 秒。该阈值是 CI 环境稳定性门槛，不等同于生产
  RUM 目标；生产优化应另行采集 P75/P95。
- Canvas：`/galaxy` 在普通设备上必须完成真实 Canvas 绘制后的 FPS 采样；低于 45 FPS
  时必须切换到 `limited` 模式。普通模式最多绘制 2000 个点，limited 模式最多绘制
  400 个点，并关闭高开销的 `lighter` 合成。
- 低性能设备：`deviceMemory <= 2`、`hardwareConcurrency <= 2` 或
  `prefers-reduced-motion: reduce` 必须直接进入 limited 模式，点数上限为 400。

## 运行方式

在仓库根目录执行：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\safe-preflight.ps1 `
  -BuildFrontend -RunBrowserBaseline
```

这会构建三套前端、检查 bundle 门禁，并运行 blog 与 admin-next 的 Chromium 基线。
其中 blog 的 `agent-routes.spec.ts` 覆盖 FPS、低性能设备和 reduced-motion 三条路径。

## 发布资产只读校验

在进入发布窗口前，可执行：

```powershell
pwsh ./scripts/check-admin-next-release-assets.ps1
```

该脚本只读取 `web/admin-next/dist` 和 `web/admin/dist`：确认新后台入口及其本地
`src`/`href` 资源完整，应用入口 chunk 没有残留开发地址，并确认旧 admin 的入口仍可作为
回滚候选。输出包含产物数量、字节数和入口 SHA-256；`readyForSwitch` 固定为 `false`，
表示它不是生产 Caddy 切换授权，也不代表旧产物已经完成一个发布周期保留。脚本不会构建、
写入或切换任何容器、数据库、对象存储、Meilisearch 或 Caddy 目录。

`safe-preflight.ps1` 在前端预算检查后自动调用该只读校验；若还没有本地构建产物，需先按
上面的构建命令生成产物，或在 CI 中执行 `-BuildFrontend`。

CI 的 `release-assets` job 会从 `admin` 和 `admin-next` 构建 job 下载短期 artifact，再运行
同一校验器；artifact 仅保留 1 天用于该次流水线验证，不作为仓库发布物或生产备份。

## 运行时约束

星河只有在已经加载坐标、页面可见且不是 limited/reduced-motion 模式时才采样 FPS；
页面隐藏时停止采样并跳过增量轮询，恢复可见后再按需恢复。这样性能测量不会给低性能
设备增加额外绘制，也不会在后台标签页持续制造网络和 CPU 开销。

## 证据边界

本门禁不代表真实 Caddy 发布切换、移动真机帧率或生产 RUM；M7-12 的真实 RBAC
菜单/编辑/刷新 E2E 由 `scripts/integration-browser-e2e.ps1 -AdminNextOnly
-RequireAgentRoutes` 在隔离 Caddy 上单独验收。真实部署证据仍需在隔离发布窗口采集；
生产静态目录切换之前必须保留旧产物以便回滚，且 `OPS-05`/`OPS-06` 仍需要独立的发布窗口
和保留周期证据。

仓库提供可审计的切换入口 `scripts/production-caddy-admin-switch.ps1`。它默认执行
`Plan` 只读校验；`Apply` 必须同时提供隔离/生产发布单号、已复核的
`ExpectedIndexSha256`、`-AllowProductionSwitch` 和 `-ConfirmProductionTarget`，并把
当前 `admin` 目录移动到预先创建且通过路径校验的独立 `ArchiveRoot` 后再放入新产物。脚本不删除旧产物、不重载
Caddy，也不操作容器；切换失败会保留失败产物并尝试恢复旧目录。保留期结束后使用
`-Action VerifyRetention` 检查归档存在且旧入口 SHA-256 未变；该校验通过前禁止清理旧
产物。当前仓库只做脚本/夹具验证，没有把任何生产静态目录作为目标。
