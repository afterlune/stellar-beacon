# Web 前端

本目录包含两个独立的 Vue 应用：

- `blog`：Vue 3 博客前台
- `admin`：Vue 2 管理后台

两个应用均使用相对路径访问 `/api`。本地开发时，`vue.config.js` 默认将 `/api` 代理到后端的自签名 HTTPS 地址 `https://localhost:7777`，并跳过本地证书校验，也可以通过 `VUE_APP_API_TARGET` 覆盖。

## 本地开发

在项目根目录分别执行：

```shell
cd web/blog
npm ci
npm run serve
```

```shell
cd web/admin
npm ci
npm run serve
```

## 构建

```shell
cd web/blog
npm ci
npm run build
```

```shell
cd web/admin
npm ci
npm run build
```

构建结果分别位于 `web/blog/dist` 和 `web/admin/dist`。生产部署时，将它们复制到现有 Caddy 静态目录约定的 `blog` 和 `admin` 目录；本仓库不包含构建产物，也不执行远程 SSH 部署。

完整联调请使用仓库根目录的 `scripts/integration-up.ps1`、`integration-seed.ps1` 和 `integration-smoke.ps1`。这些脚本只操作隔离的 `benetnasch-integration` Compose 项目，不会改动现有 Caddy 或容器。
