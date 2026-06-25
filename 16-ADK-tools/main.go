package main

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

func main() {
	ctx := context.Background()

	chatModel, err := deepseek.NewChatModel(ctx, &deepseek.ChatModelConfig{
		APIKey:  "sk-669819aec367415e98fb61b284c09a34",
		BaseURL: "https://api.deepseek.com/v1",
		Model:   "deepseek-v4-pro",
	})
	if err != nil {
		fmt.Println("Failed to create chat model:", err)
	}

	// 创建工具
	timeTool := utils.NewTool(
		&schema.ToolInfo{
			Name:        "get_current_time",
			Desc:        "获取当前时间",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{}),
		},
		func(ctx context.Context, params map[string]any) (string, error) {
			return time.Now().Format(time.RFC3339), nil
		},
	)

	caculator := utils.NewTool(
		&schema.ToolInfo{
			Name: "caculator",
			Desc: "执行数学计算，支持加减乘除",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"expression": {
					Desc:     "数学表达式，例如： 2 + 3 * 4",
					Required: true,
					Type:     schema.String,
				},
			}),
		},
		func(ctx context.Context, params map[string]any) (string, error) {
			expression, ok := params["expression"].(string)
			if !ok {
				return "", fmt.Errorf("expression must be a string")
			}

			return fmt.Sprintf("%.2f", expression), nil
		},
	)

	// 创建Agent
	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "ToolAssistant",
		Description: "一个能够使用工具的助手Agent",
		Instruction: `你是一个智能助手，可以使用以下工具:
		- get_current_time: 获取当前时间
		- caculator： 执行数学计算
		当用户需要获取时间或进行计算时，请调用相应的工具。
		`,
		Model: chatModel,
		// 在agent上配置工具
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{timeTool, caculator},
			},
		},
		MaxIterations: 10,
	})
	if err != nil {
		fmt.Println("Failed to create agent:", err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
	})

	queries := []string{
		"现在几点了？",
		"帮我计算 123 + 34456 + 46546",
	}

	for _, query := range queries {
		fmt.Printf("用户： %s\\n\\n", query)
		iter := runner.Query(ctx, query)

		for {
			event, ok := iter.Next()
			if !ok {
				break
			}

			if event.Err != nil {
				fmt.Printf("Agent 执行错误：%v", event.Err)
			}

			if event.Output != nil && event.Output.MessageOutput != nil {
				msg := event.Output.MessageOutput.Message
				if msg != nil {
					if len(msg.ToolCalls) > 0 {
						for _, tc := range msg.ToolCalls {
							fmt.Printf("工具调用: %+v\n", tc.Function.Name)
						}
					} else if msg.Content != "" {
						fmt.Printf("助手：%s\n\n", msg.Content)
					}
				}
			}
		}

	}
}
