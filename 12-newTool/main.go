package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
)

// 使用NewTool快速创建

type TimeParams struct {
	Format string `json:"format"`
}

type TimeResult struct {
	CurrentTime string `json:"current_time"`
}

func GetCurrentTime(ctx context.Context, params *TimeParams) (*TimeResult, error) {
	now := time.Now()
	var result string
	switch params.Format {
	case "date":
		result = now.Format("2006-01-02")
	case "time":
		result = now.Format("15:04:04")
	default:
		result = now.Format("2006-01-02 15:04:04")
	}
	return &TimeResult{CurrentTime: result}, nil
}

func main() {
	ctx := context.Background()

	timeTool := utils.NewTool(&schema.ToolInfo{
		Name: "get_current_time",
		Desc: "获取当前时间",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"format": {
				Type:     schema.String,
				Desc:     "时间格式：date(日期)、time(时间)、datetime(日期时间)",
				Required: false,
			},
		}),
	}, GetCurrentTime)

	// 测试工具
	testFormats := []string{"date", "time", "datetime", ""}

	for _, format := range testFormats {
		params := TimeParams{Format: format}
		b, _ := json.Marshal(params)
		// 工具执行
		ouputJSON, err := timeTool.InvokableRun(ctx, string(b))
		if err != nil {
			log.Printf("执行失败: %v", err)
			continue
		}
		fmt.Printf("格式化-%s, 结果-%s\n", format, ouputJSON)
	}
}
