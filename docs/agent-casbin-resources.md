# Agent 能力 Casbin 资源预留

本文档登记 Agent 控制面资源和菜单种子。资源只有在对应 API 已实现后才由显式迁移增加；
迁移不会在服务启动时自动执行，也不会触碰正在运行的容器。`0015_agent_admin_rbac.sql`
会幂等创建 AI 菜单和基础资源，`0016_agent_review_policy.sql` 再增加审核策略资源，
`0018_agent_memory_admin_rbac.sql` 增加记忆审核资源，`0019_agent_memory_admin_menu.sql`
增加 admin-next 菜单入口，`0020_agent_provider_probe_rbac.sql` 增加 Provider 烟测资源，
`0021_agent_vision_rbac.sql` 增加视觉预览资源，`0022_ai_observability_rbac.sql` 增加脱敏
AI/搜索运行时观测资源；它们都只给现有、启用的 `admin` 角色补授权。观测接口只读运行时
聚合，不返回 Prompt、请求正文、Provider 凭据或搜索词。

| 资源 | 方法 | 默认策略 | 说明 |
| --- | --- | --- | --- |
| `/agent/chat` | `POST` | public flag | SSE 对话；默认关闭 |
| `/agent/sessions/:id` | `DELETE` | public flag | 清理访客会话 |
| `/agent/vitals` | `GET` | public flag | 生命体征 |
| `/galaxy` | `GET` | public flag | 星河视图 |
| `/dreams` | `GET` | public flag | 梦境内容 |
| `/radio` | `GET` | public flag | 电台内容 |
| `/videos` | `GET` | public flag | 视频内容 |
| `/capsules` | `POST` | authenticated | 创建时间胶囊 |
| `/capsules/:id` | `GET` | authenticated | 查询时间胶囊 |
| `/capsules/:id/seal` | `POST` | authenticated | 封存时间胶囊 |
| `/admin/ai/providers/test` | `POST` | admin | 只测试 chat/vision/embedding 连接，不保存密钥 |
| `/admin/ai/observability` | `GET` | admin | 脱敏 Provider 与索引指标；不执行 Provider 烟测 |
| `/admin/ai/writing/stream` | `POST` | admin | 续写、润色、摘要等 |
| `/admin/ai/writing/preview` | `POST` | admin | 生成不落库的写作预览，并创建待审记录 |
| `/admin/ai/vision/preview` | `POST` | admin | 根据图片生成待审视觉理解预览，不发布文章 |
| `/admin/ai/profile` | `GET/PATCH` | admin | 人设配置；行为策略仍由独立配置/后续策略仓储控制 |
| `/admin/ai/review-policy` | `GET/PATCH` | admin | 版本化审核边界；人工审核不可关闭 |
| `/admin/ai/memory/assertions` | `GET` | admin | 查询持久化记忆断言 |
| `/admin/ai/memory/assertions/:id/history` | `GET` | admin | 查询单条断言的不可变历史 |
| `/admin/ai/memory/assertions/:id` | `DELETE` | admin | 撤回断言但保留历史 |
| `/admin/ai/memory/conflicts` | `GET` | admin | 查询记忆冲突及成员 |
| `/admin/ai/memory/conflicts/:id/resolve` | `POST` | admin | 显式选择冲突赢家 |
| `/admin/ai/memory/conflicts/:id/reject` | `POST` | admin | 驳回冲突中的全部断言 |
| `/admin/ai/reviews` | `GET` | admin | 待审核生成物 |
| `/admin/ai/reviews/:id/approve` | `POST` | admin | 人工批准 |
| `/admin/ai/reviews/:id/partial` | `POST` | admin | 部分接受并记录人工修改内容 |
| `/admin/ai/reviews/:id/reject` | `POST` | admin | 人工拒绝 |
| `/admin/ai/reviews/:id/regenerate` | `POST` | admin | 记录重新生成操作 |
| `/admin/ai/runs` | `GET` | admin | 脱敏调用记录 |
| `/admin/videos/upload` | `POST` | admin | 上传并校验本地视频 |
| `/admin/videos/external` | `POST` | admin | 添加 HTTPS 白名单外链视频 |
| `/admin/videos/:id` | `DELETE` | admin | 软删除视频元数据 |

菜单种子使用稳定路径 `/ai-submenu`、`/ai-studio`、`/ai-profile`、`/ai-review-policy`，
页面组件路径分别为 `/ai/Studio.vue`、`/ai/Profile.vue`、`/ai/ReviewPolicy.vue`、`/ai/Memory.vue`。
记忆审核菜单入口为 `/ai-memory`；持久化开关默认关闭，且公共 Agent 对话不会读写这组数据。
启用前须由部署人员在确认窗口按迁移版本顺序执行 0015、0016、0018、0019、0020、0021；
本仓库不会自动向生产库写入这些记录。

运行时授权会先兼容读取既有 `casbin_rule` 策略，再校验 `t_resource` 与
`t_role_resource` 的显式资源绑定。这样旧部署的 Casbin wildcard 策略仍然有效，
而新迁移或后台角色页面配置的资源也会真正约束 API；资源路径使用分段匹配，避免
`/reviews/*/approve` 意外放行其它审核动作。用户角色查询同时排除已禁用角色。

数字空间 Companion 协议不属于上述人类后台资源：`/internal/space/v1/*` 使用独立的
服务令牌、`t_agent_principal` 主体和 `space:*` 作用域，由专用 middleware 校验；它不会
借用人类 Casbin 会话，也不会把 `moonfei` 映射成某个后台用户。发布事实仍通过
`t_space_publication` 保存来源会话、运行 ID 和幂等键。
