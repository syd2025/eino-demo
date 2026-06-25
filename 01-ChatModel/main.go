package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/joho/godotenv"
)

func main() {
	// 说明：命令行参数解析
	var instruction string
	flag.StringVar(&instruction, "instruction", "You are a helpful assistant.", "Instruction to execute")
	flag.Parse()

	// 加载环境变量
	err := godotenv.Load()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "Error loading environment variables:", err)
		os.Exit(1)
	}

	// 命令行传递参数
	// strings.TrimSpace: 去除字符串首尾的空白字符
	// strings.Join: 将字符串切片连接成一个字符串，使用空格作为分隔符
	query := "用一句话解释 Eino 的 Component 设计解决了什么问题？"

	ctx := context.Background()
	cm, err := newChatModel(ctx)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "Error creating chat model:", err)
		os.Exit(1)
	}

	messages := []*schema.Message{
		schema.SystemMessage(instruction), // 系统提示词
		schema.UserMessage(query),         // 用户提示词
	}

	_, _ = fmt.Fprint(os.Stdout, "[assistant] ")

	// 第一种方式
	// response, err := cm.Generate(ctx, messages)
	// if err != nil {
	// 	_, _ = fmt.Fprintln(os.Stderr, err)
	// 	os.Exit(1)
	// }
	// fmt.Println(response.Content)

	// 第二种方式：流式响应
	stream, err := cm.Stream(ctx, messages)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	defer stream.Close()
	for {
		frame, err := stream.Recv() // 接收流式响应

		// 处理流式响应中的错误和结束标志
		// 1. 判断是否结束
		if errors.Is(err, io.EOF) {
			break
		}
		// 2. 判断是否有错误
		if err != nil {
			_, _ = fmt.Fprintln(os.Stdout, err)
			os.Exit(1)
		}
		// 3. 输出内容
		if frame != nil {
			_, _ = fmt.Fprint(os.Stdout, frame.Content)
		}
	}

	// 	输出换行符
	_, _ = fmt.Fprintln(os.Stdout)
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
