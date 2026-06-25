package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

/**
 * 计算器工具
 // InvokableTool 是可以被 ToolsNode 执行的工具
	type InvokableTool interface {
		BaseTool
		// InvokableRun 执行工具,参数是 JSON 编码的字符串,返回字符串结果
		InvokableRun(ctx context.Context, argumentsInJSON string, opts ...Option) (string, error)
	}
*/

type CalculatorTool struct{}

func (t *CalculatorTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "Calculator",
		Desc: "执行基本的数学计算（加、减、乘、除）",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"operation": {
				Type:     "string",
				Desc:     "运算类型: add(加),subtract(减),multply(乘),divide(除)",
				Required: true,
			},
			"a": {
				Type:     "number",
				Desc:     "第一个数字",
				Required: true,
			},
			"b": {
				Type:     "number",
				Desc:     "第二个数字",
				Required: true,
			},
		}),
	}, nil
}

// 参数结构
type CalculatorParams struct {
	Operation string  `json:"operation"`
	A         float64 `json:"a"`
	B         float64 `json:"b"`
}

// 输出结构
type CalculatorResult struct {
	Result float64 `json:"result"`
	Error  string  `json:"error,omitempty"`
}

// 执行计算
func (t *CalculatorTool) InvokeRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var params CalculatorParams
	// 将出入的struct对象转换成json对象
	if err := json.Unmarshal([]byte(argumentsInJSON), &params); err != nil {
		return "", fmt.Errorf("解析参数失败：%w", err)
	}

	var result float64
	switch params.Operation {
	case "add":
		result = params.A + params.B
	case "subtract":
		result = params.A - params.B
	case "multiply":
		result = params.A * params.B
	case "divide":
		if params.B == 0 {
			return "", fmt.Errorf("除数不能为零")
		}
		result = params.A / params.B
	default:
		resultJSON, _ := json.Marshal(CalculatorResult{
			Error: "不支持的操作类型",
		})
		return string(resultJSON), nil
	}

	// 3. 返回结果
	resultJSON, err := json.Marshal(CalculatorResult{
		Result: result,
	})
	if err != nil {
		return "", err
	}
	return string(resultJSON), nil
}

func main() {
	ctx := context.Background()
	calculator := &CalculatorTool{}

	testCases := []struct {
		operation string
		a, b      float64
	}{
		{"add", 5, 3},
		{"subtract", 5, 3},
		{"multiply", 5, 3},
		{"divide", 5, 3},
		{"divide", 5, 0}, // 测试除数为零的情况
	}

	for _, tc := range testCases {
		params := CalculatorParams{
			Operation: tc.operation,
			A:         tc.a,
			B:         tc.b,
		}

		paramsJSON, _ := json.Marshal(params)

		result, err := calculator.InvokeRun(ctx, string(paramsJSON))

		if err != nil {

			fmt.Printf("执行失败: %v\n", err)

			continue

		}

		fmt.Printf("执行成功: %s\n", result)
	}
}
