package main

import (
	"context"
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

	// 定义了一个模版，包含系统消息、用户消息和助手消息，并使用占位符来动态填充内容
	template := prompt.FromMessages(
		schema.FString, // 定义一个字符串模板，包含占位符
		schema.SystemMessage("你是{role},你的专长是{expertise}"),
		schema.UserMessage("我的问题是：{question}"),
		schema.AssistantMessage("我理解了，让我思考一下...", nil),
		schema.UserMessage("请详细说明"),
	)

	template2 := prompt.FromMessages(
		schema.FString,
		schema.SystemMessage("你是一个智能推荐助手"),
		schema.UserMessage(`用户信息：
			姓名：{name}
			年龄： {age}
			兴趣： {interests}
			VIP等级： {vip_level}

			请根据以上信息推荐合适的内容。
		`),
	)

	user := UserProfile{
		Name:      "张三",
		Age:       28,
		Interests: []string{"编程", "阅读", "旅行"},
		VIPLevel:  3,
	}

	// 定义了一个变量映射，将占位符替换为实际的值
	variables := map[string]any{
		"role":      "一位经验丰富的软件架构师",
		"expertise": "微服务的架构设计",
		"question":  "如何设计一个高可用的分布式系统？",
	}

	variables2 := map[string]any{
		"name":      user.Name,
		"age":       user.Age,
		"interests": user.Interests,
		"vip_level": user.VIPLevel,
	}

	messages, err := template.Format(ctx, variables)
	if err != nil {
		log.Fatalf("格式化失败: %v", err)
	}

	messages2, err := template2.Format(ctx, variables2)
	if err != nil {
		log.Fatalf("格式化失败: %v", err)
	}

	log.Printf("生成的消息: %v", messages)
	log.Printf("生成的消息2: %v", messages2)

	for i, msg := range messages {
		log.Printf("消息 %d: %s", i, msg)
	}

	for i, msg := range messages2 {
		log.Printf("消息 %d: %s", i, msg)
	}
}
