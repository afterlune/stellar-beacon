# 前端工作区

`apps/blog` 是博客前台，`apps/admin-next` 是管理台；`packages/` 放共享 API 契约和请求客户端。

## 安装与构建

在 `web` 目录执行：

```shell
npm ci
npm run build:blog
npm run build:admin
```

构建结果位于 `apps/blog/dist` 和 `apps/admin-next/dist`，不提交到 Git。

## 本地开发

```shell
npm run serve --workspace=@stellar-beacon/blog
npm run serve --workspace=@stellar-beacon/admin-next
```

博客前台默认使用 8080 端口，管理台默认使用 8082 端口。前台 API 代理地址由 `VITE_API_TARGET` 设置；管理台优先读取 `VITE_ADMIN_API_TARGET`，未设置时使用 `VITE_API_TARGET`。代理保留 `/api/v1` 路径。

前台样式令牌和组件基元位于 `apps/blog/src/styles/`，设计约定见 [`docs/design/blog-frontend.md`](../docs/design/blog-frontend.md)。

## 隔离联调

从仓库根目录运行：

```powershell
pwsh ./scripts/integration/verify.ps1
```

脚本会重建隔离联调栈、迁移并准备测试数据，然后运行前后台集成流程和前台视觉检查。该栈使用独立 Compose 项目；详细说明见[部署手册](../docs/runbooks/deployment.md#隔离联调)。
