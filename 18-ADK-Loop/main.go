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
		Name:        "MainAgent",
		Description: "负责生成初步解决方案",
		Instruction: "你是一个问题解决专家。请根据用户问题生成详细的解决方案。如果解决方案需要改进，请说明需要改进的地方",
		Model:       chatModel,
		OutputKey:   "solution",
	})
	if err != nil {
		log.Fatalf("创建Main Agent 失败： %v", err)
	}

	// Agent 2： 批判反馈 Agent
	critiqueAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "critiqueAgent",
		Description: "对解决方案进行批判和反馈",
		Instruction: "你是一个批判和反馈专家。请对解决方案进行批判和反馈。如果解决方案需要改进，请说明需要改进的地方",
		Model:       chatModel,
		OutputKey:   "critique",
	})
	if err != nil {
		log.Fatalf("创建critique Agent 失败： %v", err)
	}

	// agent 3:
	loopAgent, err := adk.NewLoopAgent(ctx, &adk.LoopAgentConfig{
		Name:          "loopAgent",
		Description:   "迭代反思性智能体，通过多轮迭代优化解决方案",
		SubAgents:     []adk.Agent{analyzerAgent, critiqueAgent},
		MaxIterations: 5,
	})

	if err != nil {
		log.Fatalf("创建loopAgent失败： %v", err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           loopAgent,
		EnableStreaming: false,
	})

	///////////////////////////////////////////////////////////

	query := "如何设计一个高性能的分布式缓存系统？"
	fmt.Printf("用户： %s\\n\\n", query)

	iter := runner.Query(ctx, query)
	iteration := 0
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
				if event.AgentName == "MainAgent" {
					iteration++
					fmt.Printf("分析结果：%s\n\n", msg)
				} else if event.AgentName == "critiqueAgent" {
					fmt.Printf("批判反馈：%s\n\n", msg)
				}

				fmt.Printf("[%s]: %s\\n", event.AgentName, msg.Content)
			}
		}
	}
}
