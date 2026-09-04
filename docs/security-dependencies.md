# 依赖与供应链安全基线

依赖安全是数字空间发布门禁的一部分。版本升级必须结合 API 兼容性、镜像 digest、内存占用、回滚和真实环境验证，不能只看包管理器的可升级提示。

## Go 依赖

- CI 执行 `go test -race ./...`、`go vet ./...` 和 `govulncheck ./...`；
- 密码使用 bcrypt；不引入 OpenPGP 或无人维护的加密实现；
- Provider、Eino、Meilisearch、Redis、对象存储和 xorm 类型保持在 infra/bootstrap 边界；
- JWT 使用单一当前版本，校验算法、issuer、密钥长度和公私钥匹配；
- 升级依赖后必须复跑服务边界、SQL、日志脱敏和配置安全扫描。

## JavaScript 依赖

三套前端分别维护 lockfile：

- 公开空间入口使用 Vue 3/Vite；
- 稳定管理入口保留 Vue 2/Vue CLI 兼容链；
- admin-next 使用 Vue 3/Vite/Pinia/Arco Design。

生产依赖图中的 high/critical 漏洞由 CI 阻断。升级时分别执行：

```powershell
cd web/blog; npm ci --no-audit --no-fund; npm audit --omit=dev --audit-level=high
cd ../admin; npm ci --no-audit --no-fund; npm audit --omit=dev --audit-level=high
cd ../admin-next; npm ci --no-audit --no-fund; npm audit --omit=dev --audit-level=high
```

不要为了压制告警直接删除 lockfile、降级运行时或把开发依赖打进生产静态资产。

## 容器镜像

Compose 使用固定版本和 digest，不使用 `latest`。PostgreSQL、Redis、Caddy、Meilisearch、MinIO 和 backend 的变更必须单独评估：

- 数据格式和 migration 兼容性；
- 内存/CPU 峰值，尤其是 WSL/Docker Desktop；
- 健康检查、TLS、端口和卷隔离；
- 备份、回滚和生产观察期。

Qwen/SGLang 低内存实验使用独立 profile，不得混入普通隔离栈或生产 Compose。模型下载和缓存必须位于仓库外。

## 密钥和运行数据

- Provider key、数据库密码、Redis 密码、SMTP 凭据和 JWT 密钥只从环境变量或部署密钥系统注入；
- `.env*`、本地配置、日志、证书、数据库 dump、对象和前端 `dist` 不进入 Git；
- 日志不得记录 token、Prompt、访客正文、Provider 参数、SQL 参数或完整请求体；
- CI secret 扫描失败时先轮换凭据，再处理提交历史和代码。

## 本地门禁

```powershell
pwsh ./scripts/safe-preflight.ps1
bash scripts/check-no-openpgp.sh
go test -race ./...
go vet ./...
```

CI 还执行镜像、配置、动态 SQL、分层、日志、Compose、发布脚本和漏洞扫描。容器扫描由 CI 使用临时环境完成，不要求也不修改本机生产容器。
