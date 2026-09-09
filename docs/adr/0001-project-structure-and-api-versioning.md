# ADR-0001：按业务域重组项目并启用版本化 API

- 状态：已采用
- 日期：2026-09-09

## 决策

后端入口放在 `cmd/benetnasch`，核心代码放在 `internal`，并把 domain、application、interfaces/http、infrastructure 分开；前端统一放在 `web/apps`，可复用传输代码放在 `web/packages`；部署文件统一放在 `deploy`。

HTTP 外部入口固定为 `/api/v1/public`、`/api/v1/auth` 和 `/api/v1/admin`，旧 HTTP 路由不保留兼容别名。响应统一使用字符串 `code`、`message`、`data`，分页统一使用 `items`、`total`、`page`、`pageSize`。

## 取舍

现有业务服务先保留其稳定的领域端口和测试，HTTP 层通过统一 DTO/响应边界完成迁移；前台视觉和资源不重做。博客请求客户端在展示层短期提供旧字段适配，以便内容页面平滑切换，新的代码只新增版本化路径。

数据库初始化和序列修复脚本随部署目录迁移，但不在开发或验证阶段自动操作用户已有容器。
