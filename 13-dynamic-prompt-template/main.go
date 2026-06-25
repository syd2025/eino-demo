package main

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
)

type ConversationStyle string

const (
	StyleProfessional ConversationStyle = "professional"
	StyleFriendly     ConversationStyle = "friendly"
	StyleCasual       ConversationStyle = "casual"
	StyleFormal       ConversationStyle = "formal"
)

func createDynamicPrompt(style ConversationStyle, domain string) prompt.ChatTemplate {
	var systemPrompt string
	switch style {
	case StyleProfessional:
		systemPrompt = fmt.Sprintf("你是一位专业的%s专家,请用专业、准确的语言回答问题。", domain)
	case StyleFriendly:
		systemPrompt = fmt.Sprintf("你是一位懂%s朋友, 请用轻松、通俗的方式聊天。", domain)
	case StyleCasual:
		systemPrompt = fmt.Sprintf("你是一位热情友好的%s助手，请用温暖、鼓励的语气交流", domain)
	case StyleFormal:
		systemPrompt = fmt.Sprintf("你是一位正式的%s顾问，请使用严谨、正式的表达方式", domain)
	default:
		systemPrompt = fmt.Sprintf("你是一个%s助手", domain)
	}
	return prompt.FromMessages(
		schema.FString,
		schema.SystemMessage(systemPrompt), // 系统提示词
		schema.UserMessage("{query}"),      // 用户提示词
	)
}

func main() {
	// 提示词模版管理
	ctx := context.Background()

	chatModel, err := deepseek.NewChatModel(ctx, &deepseek.ChatModelConfig{
		APIKey:  "sk-669819aec367415e98fb61b284c09a34",
		BaseURL: "https://api.deepseek.com/v1",
		Model:   "deepseek-v4-pro",
	})

	if err != nil {
		fmt.Println("创建聊天模型失败:", err)
	}

	query := "什么是微服务架构？"

	styles := []ConversationStyle{
		StyleProfessional,
		StyleCasual,
		StyleFriendly,
	}

	for _, style := range styles {
		fmt.Printf("\\n=========%s 风格=========\\n", style)

		template := createDynamicPrompt(style, "软件架构")
		messages, _ := template.Format(ctx, map[string]any{
			"query": query,
		})

		response, err := chatModel.Generate(ctx, messages)
		if err != nil {
			fmt.Println("生成响应失败:", err)
		}
		fmt.Println(response.Content)
	}
}
