package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/joho/godotenv"
)

/**
 * @Description: 21-claude-code-agent-loop
 *
 * The entire secret of an AI coding agent in one pattern:
 *
 *    while stop_reason == "tool_use":
 *        response = LLM(messages, tools)
 *        execute tools
 *        append results
 *
 *    +----------+      +-------+      +---------+
 *    |   User   | ---> |  LLM  | ---> |  Tool   |
 *    |  prompt  |      |       |      | execute |
 *    +----------+      +---+---+      +----+----+
 *                          ^               |
 *                          |   tool_result |
 *                          +---------------+
 *                          (loop continues)
 *
 * This is the core loop: feed tool results back to the model
 * until the model decides to stop. Production agents layer
 * policy, hooks, and lifecycle controls on top.
 *
 * Go + eino 版本：手动 agent loop，直接调用 LLM.Generate，
 * 检查 ResponseMeta.FinishReason，手动执行工具并追加 tool result。
 */

// loadEnv 加载 .env 文件。文件不存在仅警告，不中断启动。
func loadEnv() {
	if err := godotenv.Load("../.env"); err != nil {
		log.Printf("warning: failed to load .env file: %v", err)
	}
}

// runBash 在受限目录下执行 shell 命令。
//
// 返回约定：
//   - 成功（exit 0）          -> (output, nil)
//   - 失败（非 0 退出码）     -> ("", error)   error 中含 command + exit 信息 + output
//   - 超时（>120s）           -> ("", error)   error 中含 command + 部分 output
//   - 危险命令                -> ("", error)
func runBash(command string) (string, error) {
	dangerous := []string{
		"rm -rf /",
		"sudo",
		"shutdown",
		"reboot",
		"> /dev/",
		"mkfs",
		"dd if=",
		":(){:|:&};:", // fork bomb
		"chmod -R 777 /",
		"curl | sh",
		"curl | bash",
		"wget | sh",
		"wget | bash",
	}

	for _, d := range dangerous {
		if strings.Contains(command, d) {
			return "", fmt.Errorf("dangerous command blocked: %q", d)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", command)
	if dir, err := os.Getwd(); err == nil {
		cmd.Dir = dir
	} else {
		log.Printf("warning: failed to get working directory: %v", err)
	}

	out, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(out))

	if ctxErr := ctx.Err(); errors.Is(ctxErr, context.DeadlineExceeded) {
		return "", fmt.Errorf("command %q timed out after 120s; partial output: %s", command, output)
	}

	const maxLen = 50000
	if len(output) > maxLen {
		output = output[:maxLen] + "\n[output truncated to 50000 chars]"
	}

	if err != nil {
		return "", fmt.Errorf("command %q failed: %v\noutput:\n%s", command, err, output)
	}

	return output, nil
}

// agentLoop 是核心模式：持续调用 LLM 直到模型不再请求工具调用。
//
// Python 原始逻辑：
//
//	while True:
//	    response = client.messages.create(model=..., messages=messages, tools=tools)
//	    messages.append({"role": "assistant", "content": response.content})
//	    if response.stop_reason != "tool_use":
//	        return
//	    for block in response.content:
//	        if block.type == "tool_use":
//	            output = run_bash(block.input["command"])
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

		// 检查 stop reason：如果不是 "tool_calls"，说明模型已经完成，不再需要调用工具
		finishReason := ""
		if resp.ResponseMeta != nil {
			finishReason = resp.ResponseMeta.FinishReason
		}
		if finishReason != "tool_calls" {
			// 模型不再请求工具调用，循环结束
			return messages, nil
		}

		// 执行每个工具调用，收集结果
		var toolResults []*schema.Message
		for _, tc := range resp.ToolCalls {
			// Arguments 是 JSON 字符串，如 {"command": "echo hello"}
			// 需要反序列化解析出 command 字段
			var args struct {
				Command string `json:"command"`
			}
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
				toolResults = append(toolResults, schema.ToolMessage(
					fmt.Sprintf("Error: failed to parse arguments: %v", err),
					tc.ID,
					schema.WithToolName("bash"),
				))
				continue
			}

			command := strings.TrimSpace(args.Command)
			if command == "" {
				toolResults = append(toolResults, schema.ToolMessage(
					"Error: missing command argument",
					tc.ID,
					schema.WithToolName("bash"),
				))
				continue
			}

			fmt.Printf("\033[33m$ %s\033[0m\n", command)

			output, err := runBash(command)
			if err != nil {
				output = fmt.Sprintf("Error: %v", err)
			}

			// 截断输出显示
			display := output
			if len(display) > 200 {
				display = display[:200]
			}
			fmt.Println(display)

			// 追加 tool result 消息
			toolResults = append(toolResults, schema.ToolMessage(
				output,
				tc.ID,
				schema.WithToolName("bash"),
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

	fmt.Println("s01: Agent Loop (eino manual loop)")
	fmt.Println("输入问题，回车发送。输入 q / exit 退出。\n")

	// 创建 ChatModel（底层 BaseChatModel）
	chatModel, err := deepseek.NewChatModel(ctx, &deepseek.ChatModelConfig{
		APIKey:  os.Getenv("ARK_API_KEY"),
		BaseURL: os.Getenv("ARK_BASE_URL"),
		Model:   os.Getenv("ARK_MODEL"),
	})
	if err != nil {
		log.Fatalf("failed to create chat model: %v", err)
	}

	systemPrompt := fmt.Sprintf(
		"You are a coding agent at %s. Use bash to solve tasks. Act, don't explain.",
		currentDir,
	)

	// 绑定 bash 工具
	bashToolInfo := &schema.ToolInfo{
		Name: "bash",
		Desc: "Run a shell command.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"command": {
				Type:     schema.String,
				Desc:     "The shell command to run",
				Required: true,
			},
		}),
	}

	chatModelWithTools, err := chatModel.WithTools([]*schema.ToolInfo{bashToolInfo})
	if err != nil {
		log.Fatalf("failed to bind tools: %v", err)
	}

	// 手动维护的消息历史
	history := []*schema.Message{
		schema.SystemMessage(systemPrompt),
	}

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("\033[36ms01 >> \033[0m")
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
