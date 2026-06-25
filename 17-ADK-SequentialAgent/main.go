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

	// 创建多个子agent
	analyzerAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "AnalyzerAgent",
		Description: "分析用户需求，提取关键信息",
		Instruction: "你是一个分析用户需求的助手，请用简洁明了的方式回答用户问题。",
		Model:       chatModel,
		ToolsConfig: adk.ToolsConfig{},
		OutputKey:   "analysis",
	})
	if err != nil {
		log.Fatalf("创建Analyzer Agent 失败： %v", err)
	}

	solutionAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "solutionGenerator",
		Description: "根据分析结果生成解决方案",
		Instruction: "你是一个解决方案生成器，请根据需求分析结果生成详细的解决方案。可以使用{analysis}获取需求分析结果",
		Model:       chatModel,
	})

	if err != nil {
		log.Fatalf("创建Solution Agent 失败： %v", err)
	}

	// 创建SequentialAgent
	SequentialAgent, err := adk.NewSequentialAgent(ctx, &adk.SequentialAgentConfig{
		Name:        "AnalysisWorkflow",
		Description: "需求分析和解决方案生成工作流",
		SubAgents:   []adk.Agent{analyzerAgent, solutionAgent},
	})
	if err != nil {
		log.Fatalf("创建Sequential Agent 失败： %v", err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           SequentialAgent,
		EnableStreaming: false,
	})

	query := "我想开发一个智能客服系统，需要支持多轮对话和知识库检索"
	fmt.Printf("用户： %s\\n\\n", query)

	iter := runner.Query(ctx, query)
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			log.Fatalf("查询失败: %v", event.Err)
		}

		if event.Output != nil && event.Output.MessageOutput != nil {
			msg := event.Output.MessageOutput.Message
			if msg != nil {
				fmt.Printf("助手：%s\n\n", msg.Content)
			}
		}
	}
}
