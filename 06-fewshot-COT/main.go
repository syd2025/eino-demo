package main

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
)

type PromptTemplate struct{}

func (p *PromptTemplate) Translator(sourceLang, targetLang string) prompt.ChatTemplate {
	// 模版格式化逻辑
	return prompt.FromMessages(
		schema.FString,
		// 系统提示词，指导模型如何进行翻译
		schema.SystemMessage(fmt.Sprintf(
			"你是一个专业的翻译助手，请将%s翻译成%s, \\n"+
				"要求：\\n"+
				"1. 保持原文的语气和风格\\n"+
				"2. 确保翻译的准确性\\n"+
				"3. 不要添加任何额外的信息\\n"+
				"4. 如果原文中有专业术语，请保持不变",
			sourceLang, targetLang,
		)),
		// 用户提示词，提供需要翻译的文本
		schema.UserMessage("{text}"),
	)
}

func (p *PromptTemplate) CodeReviewer(language string) prompt.ChatTemplate {
	// 模版格式化逻辑
	return prompt.FromMessages(
		schema.FString,
		// 系统提示词
		schema.SystemMessage(fmt.Sprintf(
			"你是一个专业的代码审查助手，请审查以下%s代码，\\n"+
				"要求：\\n"+
				"1. 检查代码风格和规范\\n"+
				"2. 确保代码的正确性和效率\\n"+
				"3. 不要添加任何额外的信息\\n"+
				"4. 如果有改进建议，请具体说明",
			language,
		)),
		// 用户提示词
		schema.UserMessage("请审查以下代码：\\n\\n ```{language}\\n{code}\\n```"),
	)
}

func main() {
	// 提示词模版管理
	ctx := context.Background()
	templates := &PromptTemplate{}

	chatModel, err := deepseek.NewChatModel(ctx, &deepseek.ChatModelConfig{
		APIKey:  "sk-669819aec367415e98fb61b284c09a34",
		BaseURL: "https://api.deepseek.com/v1",
		Model:   "deepseek-v4-pro",
	})

	if err != nil {
		fmt.Println("创建聊天模型失败:", err)
	}

	fmt.Println("======翻译示例======")
	translatorTemplate := templates.Translator("英语", "中文")
	messages, _ := translatorTemplate.Format(ctx, map[string]any{
		"text": "Hello, how are you?",
	})

	response, err := chatModel.Generate(ctx, messages)
	if err != nil {
		fmt.Println("生成响应失败:", err)
	}
	fmt.Println("翻译结果:", response.Content)

}
