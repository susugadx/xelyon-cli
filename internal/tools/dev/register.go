package dev

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

// BashTool executes bash commands
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
	output := ExecuteBashWithOutput(execCtx.Output(), args["command"])
	return output, nil, nil
}

// RegisterTools registers all dev tools to the registry
func RegisterTools(registry *tools.Registry) {
	registry.Register(&BashTool{})
}

func init() {
	RegisterTools(tools.DefaultRegistry)
}
