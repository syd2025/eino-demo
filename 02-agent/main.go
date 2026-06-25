package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/joho/godotenv"
)

func main() {
	var instruction string
	flag.StringVar(&instruction, "instruction", "You are a helpful assistant.", "")
	flag.Parse()

	// 加载环境变量
	err := godotenv.Load()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "Error loading environment variables:", err)
		os.Exit(1)
	}

	ctx := context.Background()
	cm, err := newChatModel(ctx)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "Error creating chat model:", err)
		os.Exit(1)
	}

	// 创建一个智能体
	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "ChatModelAgent",
		Description: "A minimal ChatModelAgent with in-memory multi-turn history.",
		Instruction: instruction,
		Model:       cm,
	})
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// 创建一个智能体执行器，启用流式输出
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
	})

	// 历史上下文，最长支持16轮对话
	hisory := make([]*schema.Message, 0, 16)
	// 接收用户输入并执行，直到用户输入空行或 EOF
	scanner := bufio.NewScanner(os.Stdin)
	for {
		_, _ = fmt.Fprint(os.Stdout, "you> ")
		// 处理用户输入的结束标志和错误
		if !scanner.Scan() {
			break
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			break
		}
		hisory = append(hisory, schema.UserMessage(line))

		// 执行对话
		events := runner.Run(ctx, hisory)
		content, err := printAndCollectAssistantFromEvents(events)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		hisory = append(hisory, schema.AssistantMessage(content, nil))
	}

	if err := scanner.Err(); err != nil {
		// 处理错误
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newChatModel(ctx context.Context) (model.ToolCallingChatModel, error) {
	if os.Getenv("MODEL_TYPE") == "ark" {
		return ark.NewChatModel(ctx, &ark.ChatModelConfig{
			APIKey:  os.Getenv("ARK_API_KEY"),
			Model:   os.Getenv("ARK_MODEL"),
			BaseURL: os.Getenv("ARK_BASE_URL"),
		})
	}
	return openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:  os.Getenv("OPENAI_API_KEY"),
		Model:   os.Getenv("OPENAI_MODEL"),
		BaseURL: os.Getenv("OPENAI_BASE_URL"),
		ByAzure: os.Getenv("OPENAI_BY_AZURE") == "true",
	})
}

func printAndCollectAssistantFromEvents(events *adk.AsyncIterator[*adk.AgentEvent]) (string, error) {
	var sb strings.Builder

	for {
		event, ok := events.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			return "", event.Err
		}
		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}

		mv := event.Output.MessageOutput
		if mv.Role != schema.Assistant {
			continue
		}

		if mv.IsStreaming {
			mv.MessageStream.SetAutomaticClose()
			for {
				frame, err := mv.MessageStream.Recv()
				if errors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					return "", err
				}
				if frame != nil && frame.Content != "" {
					sb.WriteString(frame.Content)
					_, _ = fmt.Fprint(os.Stdout, frame.Content)
				}
			}
			_, _ = fmt.Fprintln(os.Stdout)
			continue
		}

		if mv.Message != nil {
			sb.WriteString(mv.Message.Content)
			_, _ = fmt.Fprintln(os.Stdout, mv.Message.Content)
		} else {
			_, _ = fmt.Fprintln(os.Stdout)
		}
	}

	return sb.String(), nil
}
