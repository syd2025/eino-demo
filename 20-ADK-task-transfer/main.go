package main

import (
	"context"
	"fmt"
	"log"

	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino/adk"
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

	// 创建Agent
	generalAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "GeneralAssistant",
		Description: "通用助手，可以处理各种问问题，也可以将任务转移给专业Agent",
		Instruction: `你是一个通用助手。你可以：
		1. 直接回答简单问题
		2. 将复杂问题转交给 TechExpert
		3. 将数学问题转移给 MathExpert
		`,
		Model: chatModel,
	})

	if err != nil {
		log.Fatal("创建代理失败:", err)
	}

	//  技术专家

	techExpert, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "TechExpert",
		Description: "技术专家，专门处理编程和技术问题",
		Instruction: "你是一个技术专家，请详细解答编程和技术相关问题",
		Model:       chatModel,
	})
	if err != nil {
		log.Fatal("创建技术专家代理失败:", err)
	}

	// 数学专家
	mathExpert, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "MathExpert",
		Description: "数学专家，专门处理数学问题",
		Instruction: "你是一个数学专家，请详细解答数学相关的问题",
		Model:       chatModel,
	})

	if err != nil {
		log.Fatal("创建数学专家代理失败:", err)
	}

	generalAgentWithSubs, err := adk.SetSubAgents(ctx, generalAgent, []adk.Agent{techExpert, mathExpert})

	if err != nil {
		log.Fatal("设置子代理失败:", err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           generalAgentWithSubs,
		EnableStreaming: false,
	})

	queries := []string{
		"你好，今天天气怎么样?",
		"Go语言中如何实现并发？",
		"如何计算圆的面积？",
	}

	for _, query := range queries {
		fmt.Printf("\n=== 用户： %s  ===\n", query)
		iter := runner.Query(ctx, query)
		for {

			event, ok := iter.Next()
			if !ok {
				break
			}

			if event.Err != nil {
				log.Fatal("查询失败:", event.Err)
			}

			if event.Output != nil && event.Output.MessageOutput != nil {

				msg := event.Output.MessageOutput.Message

				if msg != nil {
					if len(msg.ToolCalls) > 0 {
						for _, tc := range msg.ToolCalls {
							fmt.Printf("工具调用: %+v\n", tc)
							if tc.Function.Name == "transfer_to_agent" {
								fmt.Printf("转移代理调用: %+v\n", tc.Function)
							}
						}
					} else if msg.Content != "" {
						fmt.Printf("[%s]: %s\n", event.AgentName, msg.Content)
					}
				}
			}
		}
	}
}
