# 星际信标前端工作区 · Stellar Beacon Web Workspace

`apps/blog` 是博客前台，采用「星图仪器 / Astral Instrument」设计语言（规范见 [`docs/design/blog-frontend.md`](../docs/design/blog-frontend.md)）；`apps/admin-next` 是管理端；公共传输契约和请求客户端位于 `packages/`。

## 安装与构建

在 `web` 目录执行：

```shell
npm ci
npm run build:blog
npm run build:admin
```

也可以运行 `npm run build` 一次构建所有 workspace。

构建结果为 `web/apps/blog/dist` 和 `web/apps/admin-next/dist`。构建产物不提交 Git。

## 本地开发

```shell
npm run serve --workspace=@stellar-beacon/blog
npm run serve --workspace=@stellar-beacon/admin-next
```

前台默认使用 Vite 的 `8080` 端口；API 代理目标由 `VITE_API_TARGET` 指定，例如联调栈填 `http://127.0.0.1:18080`（Caddy 入口）。代理**不做路径重写**：后端注册的就是 `/api/v1/*`，多剥一层前缀会 404。管理端默认使用 `8082` 端口，目标由 `VITE_ADMIN_API_TARGET` 或 `VITE_API_TARGET` 指定。两个应用均只请求版本化接口 `/api/v1/...`。

前台的设计令牌在 `apps/blog/src/styles/tokens/`，组件基元类在 `apps/blog/src/styles/primitives/`。改视觉前先读设计规范，不要再写裸色值或阶梯外的间距。

## 联调

完整联调使用仓库根目录的 `scripts/integration/` 脚本与隔离 Compose 项目，不触碰已有数据库、缓存、搜索、对象存储或 Caddy 容器。
