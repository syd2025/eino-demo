package main

import (
	"os"
	"path/filepath"
	"strings"
)

// DenyList 硬拒绝列表（Gate 1）：始终禁止的模式
func DenyList() []string {
	return []string{
		"rm -rf /",
		"sudo",
		"shutdown",
		"reboot",
		"mkfs",
		"dd if=",
		"> /dev/sda",
	}
}

// PermissionRule 权限规则（Gate 2）
type PermissionRule struct {
	Tools []string
	Check func(args map[string]interface{}) bool
	Msg   string
}

// PermissionRules 规则列表（Gate 2）
var PermissionRules = []PermissionRule{
	{
		Tools: []string{"write_file", "edit_file"},
		Check: func(args map[string]interface{}) bool {
			path, _ := args["path"].(string)
			return !isPathInWorkspace(path)
		},
		Msg: "Writing outside workspace",
	},
	{
		Tools: []string{"bash"},
		Check: func(args map[string]interface{}) bool {
			cmd, _ := args["command"].(string)
			keywords := []string{"rm ", "> /etc/", "chmod 777"}
			for _, kw := range keywords {
				if strings.Contains(cmd, kw) {
					return true
				}
			}
			return false
		},
		Msg: "Potentially destructive command",
	},
}

func isPathInWorkspace(path string) bool {
	if path == "" {
		return false
	}
	workDir, err := os.Getwd()
	if err != nil {
		return false
	}
	absWork, _ := filepath.Abs(workDir)
	full := filepath.Join(workDir, path)
	clean := filepath.Clean(full)
	absClean, _ := filepath.Abs(clean)
	rel, err := filepath.Rel(absWork, absClean)
	if err != nil || strings.Contains(rel, "..") || filepath.IsAbs(rel) {
		return false
	}
	return true
}
