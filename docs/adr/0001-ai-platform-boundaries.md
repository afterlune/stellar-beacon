# ADR-0001：数字生命体与 Agent 平台边界

状态：已接受

日期：2026-08-28

## 背景

项目计划增加对话、知识检索、写作辅助和行为记录能力，但现有服务必须在 AI 全部关闭时保持原有 API、前端和容器行为。模型供应商协议、Agent 编排框架和搜索实现也不应渗透到 application 层。

## 决策

1. Go 后端的生产 Agent 主线使用 Eino。Eino、OpenAI、Anthropic 和 Meilisearch SDK 类型只能出现在 infra 适配器中，application 通过 `app/domain/port` 自有接口调用。
2. OpenAI Responses、Anthropic Messages 是首批一等协议；SGLang 使用 OpenAI Chat Completions 兼容协议。不接入 Ollama。
   当前 Provider 基线使用 DeepSeek `deepseek-v4-flash-vision-exp` 作为 `OPENAI_*` 命名空间
   下、通过 OpenAI Chat Completions 协议访问的 Chat/Vision 模型，使用阿里百炼
   `qwen3.7-text-embedding` 作为独立 Embedding Provider；
   两者都只在对应能力路由中生效，不能跨能力隐式复用。
3. Meilisearch 继续承担关键词、语义和混合检索，不迁移 PostgreSQL，不新增向量数据库容器。
4. LangChain 不进入生产 Go 服务。只有在 Eino 无法满足持久化、长时间运行和人工介入工作流时，才评估独立 LangGraph sidecar。
5. 所有模型生成内容默认进入人工审核队列，模型不能直接发布。
6. Provider API Key 只来自环境变量；数据库只保存 Provider、模型和非敏感路由配置。
7. AI 功能由配置开关控制，默认关闭；数据库迁移通过显式 CLI 执行，服务启动不自动改表。

## 后果

- application 与供应商解耦，适配器可以独立替换，也便于用固定 mock server 做协议测试。
- 早期会增加 domain port、迁移和审核状态的建模成本，但可以避免将来被 SDK 类型和不可回滚的自动迁移锁定。
- LangGraph 不作为当前依赖或容器引入，减少运行时和部署面；后续若需要 sidecar，必须有独立边界与回滚方案。
