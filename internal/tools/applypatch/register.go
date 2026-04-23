package applypatch

import (
	"github.com/susugadx/xelyon-cli/internal/tools"
)

func init() {
	tools.DefaultRegistry.Register(&ApplyPatchTool{})
}

// ApplyPatchTool は apply_patch ツールを登録する。
type ApplyPatchTool struct{}

func (t *ApplyPatchTool) Name() string { return "apply_patch" }

func (t *ApplyPatchTool) Description() string { return applyPatchDescription }

func (t *ApplyPatchTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"patch": map[string]interface{}{
				"type":        "string",
				"description": "The patch text. Must start with '*** Begin Patch' and end with '*** End Patch'.",
			},
		},
		"required":             []string{"patch"},
		"additionalProperties": false,
	}
}
