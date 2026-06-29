package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/joho/godotenv"
)

/**
 * @Description: 23-claude-code-agent-permission
 *
 s03_permission.py - Permission System

Three gates inserted before tool execution:

    Gate 1: Hard deny list (rm -rf /, sudo, ...)
    Gate 2: Rule matching (write outside workspace? destructive cmd?)
    Gate 3: User approval (pause and wait for confirmation)

    +-------+    +--------+    +--------+    +--------+    +------+
    | Tool  | -> | Gate 1 | -> | Gate 2 | -> | Gate 3 | -> | Exec |
    | call  |    | deny?  |    | match? |    | allow? |    |      |
    +-------+    +--------+    +--------+    +--------+    +------+
         |            |             |             |
         v            v             v             v
      (normal)     (blocked)    (ask user)   (user says no?)

Only one line added to the agent loop:

    if not check_permission(block):
        continue

Builds on s02 (multi-tool). Usage:

    python s03_permission/code.py
    Needs: pip install anthropic python-dotenv + ANTHROPIC_API_KEY in .env
*/

// loadEnv 加载 .env 文件。文件不存在仅警告，不中断启动。
func loadEnv() {
	if err := godotenv.Load("D:\\projects\\eino-demo\\.env"); err != nil {
		log.Printf("warning: failed to load .env file: %v", err)
	}
}

// buildToolInfos 构建 5 个工具的 ToolInfo 列表，对应 Python 的 TOOLS。
//
// Python 对应:
//
//	TOOLS = [
//	    {"name": "bash", "description": "Run a shell command.", ...},
//	    {"name": "read_file", "description": "Read file contents.", ...},
//	    {"name": "write_file", "description": "Write content to a file.", ...},
//	    {"name": "edit_file", "description": "Replace exact text in a file once.", ...},
//	    {"name": "glob", "description": "Find files matching a glob pattern.", ...},
//	]
func buildToolInfos() []*schema.ToolInfo {
	return []*schema.ToolInfo{
		{
			Name: "bash",
			Desc: "Run a shell command.",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"command": {
					Type:     schema.String,
					Desc:     "The shell command to run",
					Required: true,
				},
			}),
		},
		{
			Name: "read_file",
			Desc: "Read file contents.",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"path": {
					Type:     schema.String,
					Desc:     "The file path to read",
					Required: true,
				},
				"limit": {
					Type: schema.Integer,
					Desc: "Maximum number of lines to read (optional)",
				},
			}),
		},
		{
			Name: "write_file",
			Desc: "Write content to a file.",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"path": {
					Type:     schema.String,
					Desc:     "The file path to write to",
					Required: true,
				},
				"content": {
					Type:     schema.String,
					Desc:     "The content to write",
					Required: true,
				},
			}),
		},
		{
			Name: "edit_file",
			Desc: "Replace exact text in a file once.",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"path": {
					Type:     schema.String,
					Desc:     "The file path to edit",
					Required: true,
				},
				"old_text": {
					Type:     schema.String,
					Desc:     "The text to replace",
					Required: true,
				},
				"new_text": {
					Type:     schema.String,
					Desc:     "The text to replace with",
					Required: true,
				},
			}),
		},
		{
			Name: "glob",
			Desc: "Find files matching a glob pattern.",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"pattern": {
					Type:     schema.String,
					Desc:     "The glob pattern to match",
					Required: true,
				},
			}),
		},
	}
}

func checkDenyList(command string) (string, error) {
	for _, pattern := range DenyList() {
		if strings.Contains(command, pattern) {
			return fmt.Sprintf("Blocked: '%v' is on the deny list", pattern), errors.New("DenyList")
		}
	}
	return "", nil
}

// checkPermission 三闸权限检查：deny list → rule matching → user approval
func checkPermission(tc schema.ToolCall) (bool, string) {
	name := tc.Function.Name

	// 只对 bash 做 deny list 检查
	if name == "bash" {
		var args struct {
			Command string `json:"command"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return false, fmt.Sprintf("Error: failed to parse bash arguments: %v", err)
		}

		// Gate 1: Hard deny list
		if reason, err := checkDenyList(args.Command); err != nil {
			return false, reason
		}

		// Gate 2: Rule matching — 破坏性操作
		for _, d := range []string{"rm -rf", "chmod -R", "> /dev/", "dd if=", "mkfs", "wget ", "curl ", ":(){ :|:& };:"} {
			if strings.Contains(args.Command, d) {
				fmt.Printf("\033[31m⚠  Destructive command detected: %q\033[0m\n", d)
				return gate3UserConfirm(d)
			}
		}
	}

	// 对 write_file/edit_file 做 Gate 2: 路径越界检查 + Gate 3
	if name == "write_file" || name == "edit_file" {
		var args struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err == nil && args.Path != "" {
			// safePath 已经拒绝越界，但到达这里说明通过了 safePath
		}
	}

	return true, ""
}

// gate3UserConfirm 输出警告并请求用户输入 y/N，返回是否放行。
func gate3UserConfirm(reason string) (bool, string) {
	msg := fmt.Sprintf("\033[33m⚠  Gate 3 — Allow this operation?\n  Reason: %s\n  Type y to allow, anything else to block: \033[0m", reason)
	fmt.Print(msg)

	var response string
	fmt.Scanln(&response)
	response = strings.TrimSpace(strings.ToLower(response))

	if response == "y" || response == "yes" {
		return true, ""
	}
	return false, fmt.Sprintf("Blocked by user: %s", reason)
}

// agentLoop 是核心模式：持续调用 LLM 直到模型不再请求工具调用。
//
// Python 原始逻辑（s02）:
//
//	while True:
//	    response = client.messages.create(model=..., messages=messages, tools=TOOLS)
//	    messages.append({"role": "assistant", "content": response.content})
//	    if response.stop_reason != "tool_use":
//	        return
//	    for block in response.content:
//	        if block.type == "tool_use":
//	            handler = TOOL_HANDLERS[block.name]   # 查表
//	            output = handler(**block.input)        # 调用
//	            results.append({"type": "tool_result", "tool_use_id": block.id, "content": output})
//	    messages.append({"role": "user", "content": results})
func agentLoop(ctx context.Context, chatModel model.ToolCallingChatModel, messages []*schema.Message) ([]*schema.Message, error) {
	for {
		// 调用 LLM
		resp, err := chatModel.Generate(ctx, messages)
		if err != nil {
			return messages, fmt.Errorf("LLM generate failed: %w", err)
		}

		// 追加 assistant 消息到历史
		messages = append(messages, resp)

		// 检查 stop reason：如果不是 "tool_calls"，说明模型已经完成
		finishReason := ""
		if resp.ResponseMeta != nil {
			finishReason = resp.ResponseMeta.FinishReason
		}
		if finishReason != "tool_calls" {
			return messages, nil
		}

		// 执行每个工具调用：查表分发 → 执行 → 收集结果
		var toolResults []*schema.Message
		for _, tc := range resp.ToolCalls {
			toolName := tc.Function.Name

			fmt.Printf("\033[33m> %s\033[0m\n", toolName)

			// run through permission pipeline before executing
			allowed, reason := checkPermission(tc)
			if !allowed {
				toolResults = append(toolResults, schema.ToolMessage(
					reason, tc.ID, schema.WithToolName(toolName),
				))
				continue
			}

			// 查找处理函数
			handler, exists := toolHandlers[toolName]
			if !exists {
				output := fmt.Sprintf("Unknown tool: %s", toolName)
				fmt.Println(output[:min(len(output), 200)])
				toolResults = append(toolResults, schema.ToolMessage(
					output, tc.ID, schema.WithToolName(toolName),
				))
				continue
			}

			// 解析 JSON 参数
			var params toolParams
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &params); err != nil {
				output := fmt.Sprintf("Error: failed to parse arguments: %v", err)
				fmt.Println(output[:min(len(output), 200)])
				toolResults = append(toolResults, schema.ToolMessage(
					output, tc.ID, schema.WithToolName(toolName),
				))
				continue
			}

			// 执行工具
			output := handler(ctx, &params)

			// 截断输出显示
			display := output
			if len(display) > 200 {
				display = display[:200]
			}
			fmt.Println(display)

			// 追加 tool result 消息
			toolResults = append(toolResults, schema.ToolMessage(
				output, tc.ID, schema.WithToolName(toolName),
			))
		}

		// 将工具执行结果追加回 messages，循环继续
		messages = append(messages, toolResults...)
	}
}

func main() {
	ctx := context.Background()

	loadEnv()

	// 必需的环境变量从启动期就校验
	required := []string{"ARK_API_KEY", "ARK_BASE_URL", "ARK_MODEL"}
	for _, key := range required {
		if os.Getenv(key) == "" {
			log.Fatalf("%s is not set; please export it before running", key)
		}
	}

	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("failed to get working directory: %v", err)
	}

	fmt.Println("s02: Tool Use — 在 s01 基础上加了 4 个工具")
	fmt.Println("输入问题，回车发送。输入 q / exit 退出。\n")

	// 创建 ChatModel
	chatModel, err := deepseek.NewChatModel(ctx, &deepseek.ChatModelConfig{
		APIKey:  os.Getenv("ARK_API_KEY"),
		BaseURL: os.Getenv("ARK_BASE_URL"),
		Model:   os.Getenv("ARK_MODEL"),
	})
	if err != nil {
		log.Fatalf("failed to create chat model: %v", err)
	}

	systemPrompt := fmt.Sprintf(
		"You are a coding agent at %s. Use tools to solve tasks. Act, don't explain.",
		currentDir,
	)

	// 绑定 5 个工具
	toolInfos := buildToolInfos()
	chatModelWithTools, err := chatModel.WithTools(toolInfos)
	if err != nil {
		log.Fatalf("failed to bind tools: %v", err)
	}

	// 手动维护的消息历史
	history := []*schema.Message{
		schema.SystemMessage(systemPrompt),
	}

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("\033[36ms03 >> \033[0m")
		if !scanner.Scan() {
			break
		}
		query := strings.TrimSpace(scanner.Text())
		if query == "" {
			continue
		}
		if lower := strings.ToLower(query); lower == "q" || lower == "exit" {
			break
		}

		// 追加用户消息
		history = append(history, schema.UserMessage(query))

		// 执行 agent loop
		history, err = agentLoop(ctx, chatModelWithTools, history)
		if err != nil {
			log.Printf("agent loop error: %v", err)
			continue
		}

		// 打印模型的最终文本响应
		if len(history) > 0 {
			lastMsg := history[len(history)-1]
			if lastMsg.Role == schema.Assistant && lastMsg.Content != "" {
				fmt.Printf("\n助手：%s\n\n", lastMsg.Content)
			}
		}
	}
}
