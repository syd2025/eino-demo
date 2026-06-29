package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ═══════════════════════════════════════════════════════════
// FROM s01 (unchanged): runBash
// ═══════════════════════════════════════════════════════════

// runBash 在受限目录下执行 shell 命令。
//
// 返回约定：
//   - 成功（exit 0）          -> (output, nil)
//   - 失败（非 0 退出码）     -> (output 含错误信息, nil)
//   - 超时（>120s）           -> (output 含错误信息, nil)
//   - 危险命令                -> (output 含错误信息, nil)
func runBash(command string) string {
	dangerous := []string{
		"rm -rf /",
		"sudo",
		"shutdown",
		"reboot",
		"> /dev/",
	}

	for _, d := range dangerous {
		if strings.Contains(command, d) {
			return fmt.Sprintf("Error: Dangerous command blocked (%q)", d)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", command)
	if dir, err := os.Getwd(); err == nil {
		cmd.Dir = dir
	} else {
		log.Printf("warning: failed to get working directory: %v", err)
	}

	out, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(out))

	if ctxErr := ctx.Err(); errors.Is(ctxErr, context.DeadlineExceeded) {
		return fmt.Sprintf("Error: Timeout (120s)\n%s", output)
	}

	const maxLen = 50000
	if len(output) > maxLen {
		output = output[:maxLen]
	}

	if err != nil {
		return fmt.Sprintf("Error: %v\n%s", err, output)
	}

	if output == "" {
		return "(no output)"
	}
	return output
}

// ═══════════════════════════════════════════════════════════
// NEW in s02: safePath + 4 个新工具
// ═══════════════════════════════════════════════════════════

// safePath 校验并返回安全的绝对路径，防止路径穿越工作区。
func safePath(p string) (string, error) {
	workDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	absWork, err := filepath.Abs(workDir)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute workdir: %w", err)
	}

	full := filepath.Join(workDir, p)
	clean := filepath.Clean(full)
	absClean, err := filepath.Abs(clean)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	rel, err := filepath.Rel(absWork, absClean)
	if err != nil || strings.Contains(rel, "..") || filepath.IsAbs(rel) {
		return "", fmt.Errorf("path escapes workspace: %s", p)
	}

	return absClean, nil
}

// runRead 读取文件内容，支持可选的 limit 行数限制。
func runRead(path string, limit int) string {
	safePath, err := safePath(path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	data, err := os.ReadFile(safePath)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if limit > 0 && limit < len(lines) {
		lines = append(lines[:limit], fmt.Sprintf("... (%d more lines)", len(lines)-limit))
	}

	return strings.Join(lines, "\n")
}

// runWrite 写入内容到文件，自动创建父目录。
func runWrite(path string, content string) string {
	safePath, err := safePath(path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	parentDir := filepath.Dir(safePath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	if err := os.WriteFile(safePath, []byte(content), 0644); err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	return fmt.Sprintf("Wrote %d bytes to %s", len(content), path)
}

// runEdit 替换文件中首次出现的 oldText 为 newText。
func runEdit(path string, oldText string, newText string) string {
	safePath, err := safePath(path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	data, err := os.ReadFile(safePath)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	text := string(data)
	if !strings.Contains(text, oldText) {
		return fmt.Sprintf("Error: text not found in %s", path)
	}

	replaced := strings.Replace(text, oldText, newText, 1)
	if err := os.WriteFile(safePath, []byte(replaced), 0644); err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	return fmt.Sprintf("Edited %s", path)
}

// runGlob 在当前工作目录下按 glob pattern 匹配文件。
func runGlob(pattern string) string {
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	absPattern := filepath.Join(workDir, pattern)
	matches, err := filepath.Glob(absPattern)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	absWork, _ := filepath.Abs(workDir)

	var results []string
	for _, match := range matches {
		rel, err := filepath.Rel(absWork, match)
		if err != nil {
			continue
		}
		if strings.Contains(rel, "..") || filepath.IsAbs(rel) {
			continue
		}
		results = append(results, rel)
	}

	if len(results) == 0 {
		return "(no matches)"
	}
	return strings.Join(results, "\n")
}

// ═══════════════════════════════════════════════════════════
// NEW in s02: 工具名 → 执行函数的映射（查表分发）
// Python 对应: TOOL_HANDLERS = {"bash": run_bash, "read_file": run_read, ...}
// ═══════════════════════════════════════════════════════════

// toolParams 解析后的工具参数。
type toolParams struct {
	// bash
	Command string `json:"command,omitempty"`
	// read_file
	Path  string `json:"path,omitempty"`
	Limit int    `json:"limit,omitempty"`
	// write_file
	Content string `json:"content,omitempty"`
	// edit_file
	OldText string `json:"old_text,omitempty"`
	NewText string `json:"new_text,omitempty"`
	// glob
	Pattern string `json:"pattern,omitempty"`
}

// toolHandler 是工具处理函数类型：接收解析后的参数，返回结果字符串。
type toolHandler func(ctx context.Context, params *toolParams) string

// toolHandlers 工具名 → 处理函数的映射。
var toolHandlers = map[string]toolHandler{
	"bash": func(ctx context.Context, p *toolParams) string {
		return runBash(p.Command)
	},
	"read_file": func(ctx context.Context, p *toolParams) string {
		return runRead(p.Path, p.Limit)
	},
	"write_file": func(ctx context.Context, p *toolParams) string {
		return runWrite(p.Path, p.Content)
	},
	"edit_file": func(ctx context.Context, p *toolParams) string {
		return runEdit(p.Path, p.OldText, p.NewText)
	},
	"glob": func(ctx context.Context, p *toolParams) string {
		return runGlob(p.Pattern)
	},
}
