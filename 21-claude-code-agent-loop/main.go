package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/joho/godotenv"
)

/**
 * @Description: 21-claude-code-agent-loop

 The entire secret of an AI coding agent in one pattern:

    while stop_reason == "tool_use":
        response = LLM(messages, tools)
        execute tools
        append results

    +----------+      +-------+      +---------+
    |   User   | ---> |  LLM  | ---> |  Tool   |
    |  prompt  |      |       |      | execute |
    +----------+      +---+---+      +----+----+
                          ^               |
                          |   tool_result |
                          +---------------+
                          (loop continues)

This is the core loop: feed tool results back to the model
until the model decides to stop. Production agents layer
policy, hooks, and lifecycle controls on top.
*/

// loadEnv 加载 .env 文件。文件不存在仅警告，不中断启动。
func loadEnv() {
	if err := godotenv.Load(); err != nil {
		log.Printf("warning: failed to load .env file: %v", err)
	}
}

// runBash 在受限目录下执行 shell 命令。
//
// 返回约定（注意：eino 工具框架在 error 非空时只回传 error message，
// 所以诊断信息必须放进 error 里，否则 LLM 和用户看不到）：
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
	// 工作目录获取失败时，让命令在默认目录运行而不是吞错
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

func main() {
	ctx := context.Background()

	loadEnv()

	// 必需的环境变量从启动期就校验，避免 nil 模型导致后续 panic
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

	fmt.Println("s01: Agent Loop")
	fmt.Println("输入问题，回车发送。输入 q / exit 退出。")

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

	bashTool := utils.NewTool(
		&schema.ToolInfo{
			Name: "bash",
			Desc: "Run a shell command",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"command": {
					Type:     schema.String,
					Desc:     "The shell command to run",
					Required: true,
				},
			}),
		},
		func(ctx context.Context, params map[string]interface{}) (string, error) {
			command, ok := params["command"].(string)
			if !ok {
				return "", fmt.Errorf("command must be a string")
			}
			command = strings.TrimSpace(command)
			if command == "" {
				return "", fmt.Errorf("command is empty")
			}
			return runBash(command)
		},
	)

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "SimpleAssistant",
		Description: "a simple assistant that can answer user questions.",
		Instruction: systemPrompt,
		Model:       chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{bashTool},
			},
		},
	})
	if err != nil {
		log.Fatalf("failed to create agent: %v", err)
	}

	// Runner 在内部维护 session，多次 Query 自动保留多轮上下文
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: false,
	})

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("\033[36ms01 >> \033[0m")
		if !scanner.Scan() {
			break
		}
		query := strings.TrimSpace(scanner.Text())
		if query == "" {
			continue // 空输入重新提示，而不是退出
		}
		if lower := strings.ToLower(query); lower == "q" || lower == "exit" {
			break
		}

		iter := runner.Query(ctx, query)
		for {
			event, ok := iter.Next()
			if !ok {
				break
			}
			if event.Err != nil {
				log.Printf("agent 执行错误: %v", event.Err)
				continue
			}
			if event.Output == nil || event.Output.MessageOutput == nil {
				continue
			}
			msg := event.Output.MessageOutput.Message
			if msg == nil {
				continue
			}
			if len(msg.ToolCalls) > 0 {
				for _, tc := range msg.ToolCalls {
					fmt.Printf("工具调用: %s\n", tc.Function.Name)
				}
				continue
			}
			if msg.Content != "" {
				fmt.Printf("\n助手：%s\n\n", msg.Content)
			}
		}
	}
}
