package main

import (
	"context"
	"fmt"
	"log"

	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
)

func main() {
	ctx := context.Background()

	// 1. 创建ChatTemplate
	template := prompt.FromMessages(
		schema.FString,
		schema.SystemMessage("你是一个{role}"),
		schema.UserMessage("{question}"),
	)

	// 2. 准备变量
	variables := map[string]any{
		"role":     "专业的Go语言工程师",
		"question": "Go语言的协程是什么？",
	}

	// 3. 格式化消息
	messages, err := template.Format(ctx, variables)
	if err != nil {
		log.Fatalf("格式化失败: %v", err)
	}

	// 使用生成的消息调用模型
	chatModel, err := deepseek.NewChatModel(ctx, &deepseek.ChatModelConfig{
		APIKey:  "sk-669819aec367415e98fb61b284c09a34",
		BaseURL: "https://api.deepseek.com/v1",
		Model:   "deepseek-v4-pro",
	})

	if err != nil {
		log.Fatalf("创建模型失败: %v", err)
	}

	response, err := chatModel.Generate(ctx, messages)
	if err != nil {
		log.Fatalf("生成失败: %v", err)
	}

	fmt.Printf("\\nAI 回答： \\n%s\\n", response.Content)
}
