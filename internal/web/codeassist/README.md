# CodeAssist

CodeAssist 是 Codebook 的 AI 对话与候选代码能力，只在 Scheduler/Web 模式运行。Executor 和 Agent 不依赖 AI 配置。

## 配置

```yaml
ai:
  provider: "rawchat"
  endpoint: "https://rawchat.cn/codex"
  model: "gpt-5.6-sol"
  timeout: "180s"
  max_output_tokens: 8192
  max_concurrency: 4
  reasoning_effort: "low"
```

| Provider | 接入方式 | API Key |
| --- | --- | --- |
| `openai` | OpenAI Responses | `OPENAI_API_KEY` |
| `rawchat` | Responses + RawChat 事件适配 | `RAWCHAT_API_KEY` |
| `qwen` | Eino ToolCallingChatModel | `QWEN_API_KEY` |

Qwen 的 `endpoint` 使用 OpenAI-compatible Chat Completions 地址，并可配置 `enable_thinking`。未配置模型时 Scheduler 仍可启动，AI 接口返回不可用错误。

## 接口与能力

| 路径 | 能力编码 | 说明 |
| --- | --- | --- |
| `POST /conversation/create` | `task:code_assist:add_conversation` | 创建项目会话 |
| `POST /conversation/list` | `task:code_assist:view` | 查询当前用户的项目会话 |
| `POST /conversation/detail` | `task:code_assist:get_conversation` | `NoSync`，依赖 `task:code_assist:view` |
| `POST /message/stream` | `task:code_assist:chat` | 发送消息并接收 SSE |
| `POST /suggestion/apply` | `task:code_assist:apply_suggestion` | 依赖 `task:codebook:add_version` |

接口前缀为 `/api/code-assist`。会话和候选代码只允许当前用户访问。

## 场景

| Recipe ID | 用途 |
| --- | --- |
| `codebook.general` | 通用解释、审阅和按需修改 |
| `codebook.review` | 只读代码审阅 |
| `codebook.edit` | 生成完整候选文件 |
| `codebook.legacy-migration` | 迁移旧运行协议 |

`recipe_id` 为空时使用 `codebook.general`。文件上下文和场景允许生成代码时，模型可调用 `propose_code`；候选代码会执行 Python 或 Shell 语法检查和旧协议提示检查。

应用候选会校验基础版本，并创建新的 Codebook 版本，不会自动切换当前版本或发布制品。

## SSE

- `message.started`
- `message.delta`
- `message.progress`
- `message.completed`
- `message.failed`
- `heartbeat`：空闲时每 15 秒发送，前端不展示。

## 数据表

`ai_conversation`、`ai_message`、`ai_suggestion` 由 `AutoMigrate` 创建。旧版 AI 表包含已删除字段时，需要在开发环境中重建这三张表。
