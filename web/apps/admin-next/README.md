# 星际信标管理台 · Stellar Beacon Admin Next

`admin-next` 是星际信标（Stellar Beacon）保留的 Vue 3 + Vite + Pinia + Arco Design 管理台。旧版 Vue 2 管理入口已淘汰；博客前台视觉和素材保持不变。

## 开发和构建

推荐从 `web` workspace 根目录执行：

```powershell
cd ../..
npm ci
npm run build:admin
```

直接开发：

```powershell
npm run serve --workspace=@stellar-beacon/admin-next
```

默认地址为 `http://127.0.0.1:8082`，API 统一使用 `/api/v1`，代理目标可由 `VITE_ADMIN_API_TARGET` 或 `VITE_API_TARGET` 覆盖。

## 测试

```powershell
npm run test:e2e:baseline --workspace=@stellar-beacon/admin-next
```

基线测试使用本地 mock，不启动、不停止、不修改任何已有数据库、缓存、搜索、对象存储或 Caddy 容器。真实联调由仓库根目录的 `scripts/integration/` 负责。

管理端只把后端菜单组件映射到本地白名单；未知组件进入占位页。后端 RBAC 仍是最终权限边界。
