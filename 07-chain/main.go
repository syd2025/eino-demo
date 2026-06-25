package main

import (
	"context"
	"fmt"
	"log"

	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// chain链式编排
// Eino中用于组合多个组件的编排方式，它将多个组件按照顺序链接，前一个组件的输出作为后一个组件的额输入，形成一个处理的流水线
func main() {

	ctx := context.Background()

	chatTemplate := prompt.FromMessages(
		schema.FString,
		schema.SystemMessage("你是一个{role}"),
		schema.UserMessage("{question}"),
	)

	// 2. 创建ChatModel
	chatModel, err := deepseek.NewChatModel(ctx, &deepseek.ChatModelConfig{
		APIKey:  "sk-669819aec367415e98fb61b284c09a34",
		BaseURL: "https://api.deepseek.com/v1",
		Model:   "deepseek-v4-pro",
	})
	if err != nil {
		fmt.Println("创建模型失败")
	}

	chain := compose.NewChain[map[string]any, *schema.Message]()
	chain.
		AppendChatTemplate(chatTemplate). // 第一步调用： 格式化模板
		AppendChatModel(chatModel)        // 第二步： 调用模型

	// 编译链
	runnable, err := chain.Compile(ctx)
	if err != nil {
		log.Fatalf("编译失败: %v", err)
	}

	// 输入参数
	input := map[string]any{
		"role":     "Go语言专家",
		"question": "Go语言的goroutine是什么？",
	}

	output, err := runnable.Invoke(ctx, input)
	if err != nil {
		log.Fatalf("调用失败: %v", err)
	}

	fmt.Println(output.Content)
}
