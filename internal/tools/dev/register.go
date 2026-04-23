package dev

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

// BashTool は bash コマンドを実行するツール。
type BashTool struct{}

func (t *BashTool) Name() string { return "bash" }

func (t *BashTool) Description() string {
	return tools.ToolDescriptions[t.Name()]
}

func (t *BashTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command": map[string]interface{}{"type": "string", "description": "Shell command to execute"},
		},
		"required":             []string{"command"},
		"additionalProperties": false,
	}
}

func (t *BashTool) Run(execCtx tools.ExecutionContext, args map[string]string) (string, *tools.FileChange, error) {
	command := strings.TrimSpace(args["command"])
	if command == "" {
		return "", nil, fmt.Errorf("bash command is empty. Provide a valid command string in the 'command' argument")
	}
	output := ExecuteBashWithContextAndPromptIOAndConfig(execCtx.EffectiveContext(), execCtx.PromptIO(), execCtx.EffectiveConfig(), args["command"])
	return output, nil, nil
}

// RegisterTools は dev ツール群をレジストリへ登録する。
func RegisterTools(registry *tools.Registry) {
	registry.Register(&BashTool{})
}

func init() {
	RegisterTools(tools.DefaultRegistry)
}
