package main

import (
	"context"
	"fmt"
	"log"

	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino/adk"
)

// 基于ChatModel的Agent，支持工具调用
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

	// 创建Agent
	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "SimpleAssistant",
		Description: "一个简单的助手Agent, 能够回答用户问题",
		Instruction: "你是一个友好的助手，请用简洁明了的方式回答用户问题。",
		Model:       chatModel,
		ToolsConfig: adk.ToolsConfig{},
	})
	if err != nil {
		fmt.Println("Failed to create agent:", err)
	}

	// 创建runner
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: false,
	})

	query := "什么是Java？"
	fmt.Printf("用户： %s\\n\\n", query)

	iter := runner.Query(ctx, query)
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			log.Fatalf("Agent 执行错误：%v", event.Err)
		}

		if event.Output != nil && event.Output.MessageOutput != nil {
			msg := event.Output.MessageOutput.Message
			if msg != nil {
				fmt.Printf("助手：%s\n\n", msg.Content)
			}
		}
	}
}
