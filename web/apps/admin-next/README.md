# 管理台

管理台使用 Vue 3、Vite、Pinia 和 Arco Design。它通过 `/api/v1` 调用 Go 后端，后端 RBAC 决定最终权限。

## 开发与构建

从 `web` 目录运行：

```powershell
npm ci
npm run serve --workspace=@stellar-beacon/admin-next
npm run build:admin
```

开发服务器默认地址为 `http://127.0.0.1:8082`。API 代理目标由 `VITE_ADMIN_API_TARGET` 设置，也可使用 `VITE_API_TARGET`。

## 检查

基线 E2E 使用本地 mock：

```powershell
npm run test:e2e:baseline --workspace=@stellar-beacon/admin-next
```

真实后端联调使用仓库根目录的 `scripts/integration/`，见[部署手册](../../../docs/runbooks/deployment.md#隔离联调)。
