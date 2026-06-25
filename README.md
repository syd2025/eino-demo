# eino-demo

基于 [CloudWeGo eino](https://github.com/cloudwego/eino) 框架的 Go 语言 AI 应用开发示例集合，包含 22 个渐进式示例，覆盖从基础模型调用到复杂多智能体协作的全链路。

## 项目依赖

| 依赖 | 版本 |
|---|---|
| `github.com/cloudwego/eino` | v0.9.0-alpha.20 |
| `github.com/cloudwego/eino-ext/components/model/ark` | v0.1.67 |
| `github.com/cloudwego/eino-ext/components/model/deepseek` | v0.1.5 |
| `github.com/cloudwego/eino-ext/components/model/openai` | v0.1.13 |
| `github.com/joho/godotenv` | v1.5.1 |

## 快速开始

```bash
# 设置环境变量（以 DeepSeek 为例）
echo "OPENAI_API_KEY=your_deepseek_api_key" > .env
echo "OPENAI_BASE_URL=https://api.deepseek.com/v1" >> .env
echo "OPENAI_MODEL=deepseek-chat" >> .env

# 运行任意示例
cd 01-ChatModel
go run .
```

## 示例目录

### 基础入门

| 序号 | 目录 | 说明 | 核心 API |
|---|---|---|---|
| 01 | [01-ChatModel](./01-ChatModel/) | 聊天模型基础调用（流式 & 非流式），支持 Ark/OpenAI 后端切换 | `ChatModel.Generate`, `ChatModel.Stream` |
| 02 | [02-agent](./02-agent/) | 使用 ADK 创建 ChatModelAgent，支持多轮对话记忆 | `adk.NewChatModelAgent`, `adk.NewRunner` |
| 03 | [03-prompt](./03-prompt/) | PromptTemplate 消息模板基础，使用 `FString` 占位符 | `prompt.FromMessages`, `schema.FString` |

### 提示词工程

| 序号 | 目录 | 说明 | 核心 API |
|---|---|---|---|
| 04 | [04-prompt-FString](./04-prompt-FString/) | PromptTemplate 结合 DeepSeek 模型调用 | `prompt.FromMessages` + `chatModel.Generate` |
| 05 | [05-prompt-Object](./05-prompt-Object/) | Go 结构体字段值作为模板变量填充 | 结构体 → `map[string]any` 映射 |
| 06 | [06-fewshot-COT](./06-fewshot-COT/) | Few-shot / Chain-of-Thought 可复用 Prompt 模板封装 | 方法封装的 `ChatTemplate` 构建器 |
| 13 | [13-dynamic-prompt-template](./13-dynamic-prompt-template/) | 运行时根据对话风格动态生成 System Prompt | 动态 Prompt 构建 |

### 编排能力

| 序号 | 目录 | 说明 | 核心 API |
|---|---|---|---|
| 07 | [07-chain](./07-chain/) | Chain 链式编排：ChatTemplate + ChatModel 串联 | `compose.NewChain`, `chain.Compile`, `runnable.Invoke` |
| 08 | [08-lambda](./08-lambda/) | Lambda 自定义节点 + 子链嵌入，灵活编排 | `chain.AppendLambda`, `chain.AppendGraph` |
| 09 | [09-branch](./09-branch/) | Branch 条件分支路由（按编程语言分发） | `compose.NewChainBranch`, `chain.AppendBranch` |
| 10 | [10-chain-demo](./10-chain-demo/) | 多步骤 Chain 实战：文章生成流水线 | Lambda + ChatModel 组合编排 |

### 工具系统

| 序号 | 目录 | 说明 | 核心 API |
|---|---|---|---|
| 11 | [11-tool](./11-tool/) | 手动实现 `tool.InvokableTool` 接口（Calculator） | `tool.InvokableTool`, `schema.ToolInfo` |
| 12 | [12-newTool](./12-newTool/) | 使用 `utils.NewTool` 快速创建工具 | `utils.NewTool` |
| 14 | [14-react-agent](./14-react-agent/) | ReAct 模式 Agent，模型自主决定工具调用 | `react.NewAgent`, `agent.Generate` |

### ADK 高级

| 序号 | 目录 | 说明 | 核心 API |
|---|---|---|---|
| 15 | [15-ADK-basic](./15-ADK-basic/) | ADK 基础 Agent，单次查询获取结果 | `adk.NewChatModelAgent`, `runner.Query` |
| 16 | [16-ADK-tools](./16-ADK-tools/) | ADK Agent 集成工具（时间查询 + 计算器） | `adk.ToolsConfig`, `compose.ToolsNodeConfig` |
| 17 | [17-ADK-SequentialAgent](./17-ADK-SequentialAgent/) | 顺序执行 Agent：需求分析 → 方案生成 | `adk.NewSequentialAgent`, `OutputKey` |
| 18 | [18-ADK-Loop](./18-ADK-Loop/) | 循环迭代 Agent：方案生成 ↔ 批判反馈 | `adk.NewLoopAgent`, `MaxIterations` |
| 19 | [19-ADK-stream](./19-ADK-stream/) | ADK 流式输出，逐帧接收模型响应 | `EnableStreaming`, `MessageStream.Recv` |
| 20 | [20-ADK-task-transfer](./20-ADK-task-transfer/) | 任务转移：通用 Agent 路由到专业子 Agent | `adk.SetSubAgents`, `transfer_to_agent` |

### 底层原理

| 序号 | 目录 | 说明 | 核心 API |
|---|---|---|---|
| 21 | [21-claude-code-agent-loop](./21-claude-code-agent-loop/) | 手动 Agent Loop（单工具 bash），模拟 Claude Code 内部循环 | `Generate` + `FinishReason` 判断 + 手动消息管理 |
| 22 | [22-claude-code-agent-tools](./22-claude-code-agent-tools/) | 手动 Agent Loop（多工具），bash/read_file/write_file/edit_file/glob | 工具分发 map + `safePath` 安全校验 |

## 学习路线

```
基础入门 (01-03)
    │  模型调用、Agent 概念、Prompt 模板
    ▼
提示词工程 (04-06, 13)
    │  结构体绑定、Few-shot/COT、动态模板
    ▼
编排能力 (07-10)
    │  Chain 链式编排、Lambda 节点、Branch 分支
    ▼
工具系统 (11-12, 14)
    │  自定义 Tool、React Agent
    ▼
ADK 高级 (15-20)
    │  顺序/循环 Agent、流式输出、任务转移
    ▼
底层原理 (21-22)
    手动 Agent Loop、多工具分发、安全控制
```

## 环境配置

在项目根目录创建 `.env` 文件：

```bash
OPENAI_API_KEY=your_api_key
OPENAI_BASE_URL=https://api.deepseek.com/v1
OPENAI_MODEL=deepseek-chat
```

## 常见问题

### 网络问题

```bash
# Windows PowerShell
$env:HTTP_PROXY="http://127.0.0.1:7890"
$env:HTTPS_PROXY="http://127.0.0.1:7890"

# Linux / macOS
export HTTP_PROXY="http://127.0.0.1:7890"
export HTTPS_PROXY="http://127.0.0.1:7890"
```

### 模块下载慢

```bash
go env -w GOPROXY=https://goproxy.cn,direct
```
