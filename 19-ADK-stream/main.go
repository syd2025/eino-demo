package main

import (
	"context"
	"fmt"
	"io"
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
	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "StreamingAssistant",
		Description: "支持流式输出的助手",
		Instruction: "你是一个友好的助手",
		Model:       chatModel,
	})

	if err != nil {
		log.Fatal("创建Agent失败：", err)
	}

	// 创建Runner
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
	})

	// --------------------------------------

	query := "请详细介绍一下Eino框架"

	fmt.Printf("用户： %s\\n\\n助手: ", query)

	iter := runner.Query(ctx, query)

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			log.Fatal("查询失败：", event.Err)
		}

		if event.Output != nil && event.Output.MessageOutput != nil {
			if event.Output.MessageOutput.IsStreaming {
				stream := event.Output.MessageOutput.MessageStream
				for {
					msg, err := stream.Recv()
					if err != nil {
						if err == io.EOF {
							break
						}

						log.Fatalf("接收流式消息失败：%v", err)
					}

					if msg != nil && msg.Content != "" {
						fmt.Print(msg.Content)
					}
				}
			}
		}
	}
}
