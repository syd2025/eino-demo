package main

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

type ArticleRequest struct {
	Topic    string
	Keywords []string
	Length   int
}

func main() {
	ctx := context.Background()

	chatModel, err := deepseek.NewChatModel(ctx, &deepseek.ChatModelConfig{
		APIKey:  "sk-669819aec367415e98fb61b284c09a34",
		BaseURL: "https://api.deepseek.com/v1",
		Model:   "deepseek-v4-pro",
	})
	if err != nil {
		panic(err)
	}

	// 构建文章生成流水线
	chain := compose.NewChain[ArticleRequest, string]()

	chain.
		AppendLambda(
			compose.InvokableLambda(func(ctx context.Context, req ArticleRequest) (string, error) {
				fmt.Print("======第一步：生成文章大纲====")

				template := prompt.FromMessages(
					schema.FString,
					schema.SystemMessage("你是一个专业的内容策划师，请根据主题和关键词生成文章大纲"),
					schema.UserMessage("主题：{topic}\\n关键词：{keywords}\\n\\n请生成一个3-5点的文章大纲"),
				)

				messages, _ := template.Format(ctx, map[string]any{
					"topic":    req.Topic,
					"keywords": fmt.Sprintf("%v", req.Keywords),
				})

				response, err := chatModel.Generate(ctx, messages)
				if err != nil {
					return "", err
				}
				fmt.Printf("大纲：\\n%s\\n\\n", response.Content)

				return response.Content, nil
			})).
		AppendLambda(
			compose.InvokableLambda(func(ctx context.Context, outline string) (string, error) {
				fmt.Print("======第二步：生成文章内容====")

				template := prompt.FromMessages(
					schema.FString,
					schema.SystemMessage("你是一个专业的内容创作者，请根据以下文章大纲和关键词生成一篇文章,逻辑清晰。"),
					schema.UserMessage("文章大纲：{outline}\\n关键词：{keywords}\\n\\n请根据大纲和关键词生成一篇文章，要求内容丰富，逻辑清晰，字数在{length}字左右。"),
				)
				messages, _ := template.Format(ctx, map[string]any{
					"outline":  outline,
					"keywords": "科技,创新,未来",
					"length":   1000,
				})

				response, err := chatModel.Generate(ctx, messages)
				if err != nil {
					return "", err
				}
				fmt.Printf("文章内容：\\n%s\\n\\n", response.Content)
				return response.Content, nil
			})).
		AppendLambda(compose.InvokableLambda(func(ctx context.Context, content string) (string, error) {
			fmt.Println("======第三步：润色文章====")

			template := prompt.FromMessages(
				schema.FString,
				schema.SystemMessage("你是一个专业的编辑。请优化文章的语言表达，便其更加流畅、生动。"),
				schema.UserMessage("文章:\\n{draft}\\n\\n请进行润色优化"),
			)

			messages, _ := template.Format(ctx, map[string]any{
				"draft": content,
			})

			response, err := chatModel.Generate(ctx, messages)
			if err != nil {
				return "", err
			}
			return response.Content, nil
		})).
		AppendLambda(compose.InvokableLambda(func(ctx context.Context, article string) (string, error) {
			fmt.Println("======第四步：生成文章标题====")

			formatted := fmt.Sprint("# 生成的文章\\n\\n%s\\n\\n---\\n由Eino AI助手生成", article)
			return formatted, nil
		}))

	runnable, err := chain.Compile(ctx)
	if err != nil {
		panic(err)
	}

	request := ArticleRequest{
		Topic: "AI 在软件开发中的应用",
	}

	result, err := runnable.Invoke(ctx, request)
	if err != nil {
		panic(err)
	}

	fmt.Println("最终结果：", result)
}
