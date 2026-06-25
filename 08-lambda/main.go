package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino/components/prompt"
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

	// 定义了输入和输出的处理链
	chain := compose.NewChain[string, string]()

	chain.
		AppendLambda(compose.InvokableLambda(func(ctx context.Context, rawText string) (string, error) {
			fmt.Println("=== 步骤1：数据清晰 ===")
			cleaned := strings.TrimSpace(rawText)
			cleaned = strings.ReplaceAll(cleaned, "\n", "")
			fmt.Printf("清洗后：%s\n\n", cleaned)
			return cleaned, nil
		})).
		AppendLambda(compose.InvokableLambda(func(ctx context.Context, text string) (map[string]any, error) {
			fmt.Println("=== 步骤2：转换为AI分析输入 ===")
			return map[string]any{"text": text}, nil
		})).
		AppendGraph(func() *compose.Chain[map[string]any, *schema.Message] {
			analysisChain := compose.NewChain[map[string]any, *schema.Message]()

			template := prompt.FromMessages(
				schema.FString,
				schema.SystemMessage("你是一个文本分析专家。请分析以下文本的关键信息、主题和情感。"),
				schema.UserMessage("{text}"),
			)
			analysisChain.AppendChatTemplate(template).AppendChatModel(chatModel)

			return analysisChain
		}()).
		AppendLambda(compose.InvokableLambda(func(ctx context.Context, msg *schema.Message) (string, error) {
			fmt.Println("=== 步骤3：处理AI分析结果 ===")
			return msg.Content, nil
		}))

	runnable, err := chain.Compile(ctx)
	if err != nil {
		log.Fatalf("编译失败: %v", err)
	}

	rawInput := `
		Eino 是一个强大的AI开发框架。它提供了丰富的组件和灵活的编排能力。
		开发者可以快速构建AI应用，并且轻松集成各种AI模型。Eino 的设计注重可扩展性和易用性，非常适合各种规模的项目。
	`

	result, err := runnable.Invoke(ctx, rawInput)
	if err != nil {
		log.Fatalf("运行失败: %v", err)
	}

	fmt.Println("最终结果:", result)
}
