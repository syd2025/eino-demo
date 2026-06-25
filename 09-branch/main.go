package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/compose"
)

// branchCondition 根据输入的编程语言决定执行哪个分支

// 根据输入条件，选择走不同的分支
func main() {
	ctx := context.Background()

	branchCondition := func(ctx context.Context, input map[string]any) (string, error) {
		language := input["language"].(string)
		language = strings.ToLower(language)

		fmt.Printf("检测到语言：%s\\n", language)

		if language == "go" || language == "golang" {
			return "go_branch", nil
		} else if language == "python" {
			return "python_branch", nil
		}
		return "other_branch", nil
	}

	goBranch := compose.InvokableLambda(func(ctx context.Context, input map[string]any) (map[string]any, error) {
		fmt.Println("进入 Go 分支")
		input["advice"] = "推荐使用Eino框架进行Go开发"
		input["features"] = []string{"高性能", "易用性", "丰富的功能"}
		return input, nil
	})

	pythonBranch := compose.InvokableLambda(func(ctx context.Context, input map[string]any) (map[string]any, error) {
		fmt.Println("执行Python分支")
		input["advice"] = "推荐使用Eino框架进行Python开发"
		input["features"] = []string{"高性能", "易用性", "丰富的功能"}
		return input, nil
	})

	otherBranch := compose.InvokableLambda(func(ctx context.Context, input map[string]any) (map[string]any, error) {
		fmt.Println("执行其他语言分支")
		input["advice"] = "建议参考该语言的AI开发库"
		input["features"] = []string{"待探索"}
		return input, nil
	})

	chain := compose.NewChain[map[string]any, map[string]any]()

	chain.AppendLambda(compose.InvokableLambda(func(ctx context.Context, input map[string]any) (map[string]any, error) {
		fmt.Println("====开始处理====")
		return input, nil
	})).
		AppendBranch(
			compose.NewChainBranch(branchCondition).
				AddLambda("go_branch", goBranch).
				AddLambda("python_branch", pythonBranch).
				AddLambda("other_branch", otherBranch),
		).
		AppendLambda(compose.InvokableLambda(func(ctx context.Context, input map[string]any) (map[string]any, error) {
			fmt.Println("====处理完成====")
			return input, nil
		}))

	runnable, err := chain.Compile(ctx) // 编译
	if err != nil {
		fmt.Printf("编译链式调用失败: %v\n", err)
		return
	}

	testCases := []map[string]any{
		{"language": "Go", "task": "开发Go应用"},
		{"language": "Python", "task": "开发Python应用"},
		{"language": "Java", "task": "开发Java应用"},
		{"language": "C++", "task": "开发C++应用"},
	}

	for _, testCase := range testCases {
		result, err := runnable.Invoke(ctx, testCase)
		if err != nil {
			fmt.Printf("处理测试用例失败: %v\n", err)
			continue
		}
		fmt.Printf("建议: %s\n", result["advice"])
		fmt.Printf("功能亮点: %v\n", result["features"])
		fmt.Println("====================================")
	}
}
