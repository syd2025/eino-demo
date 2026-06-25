package main

import (
	"context"
	"fmt"
	"log"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
)

type UserProfile struct {
	Name      string
	Age       int
	Interests []string
	VIPLevel  int
}

func main() {

	ctx := context.Background()

	template := prompt.FromMessages(
		schema.FString,
		schema.SystemMessage("你是一个智能体验助手"),
		schema.UserMessage(`用户信息：
			姓名：{name},
			年龄：{age},
			兴趣：{interests},
			VIP等级：{vipLevel}

			请根据以上信息推荐合适的内容
		`),
	)

	user := UserProfile{
		Name:      "张三",
		Age:       28,
		Interests: []string{"阅读", "旅行", "编程"},
		VIPLevel:  3,
	}

	variables := map[string]any{
		"name":      user.Name,
		"age":       user.Age,
		"interests": user.Interests,
		"vipLevel":  user.VIPLevel,
	}

	messages, err := template.Format(ctx, variables)
	if err != nil {
		log.Fatalf("格式化错误: %v", err)
	}

	for _, msg := range messages {
		fmt.Printf("[%s]\\n%s\\n\\n", msg.Role, msg.Content)
	}
}
