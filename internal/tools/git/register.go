package git

import (
	"github.com/susugadx/xelyon-cli/internal/tools"
)

// GitCommitTool is the tool for git commit
type GitCommitTool struct{}

// Name returns the tool name
func (t *GitCommitTool) Name() string {
	return "git_commit"
}

func (t *GitCommitTool) Description() string {
	return "Creates a git commit with the specified message."
}

func (t *GitCommitTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"message": map[string]interface{}{"type": "string", "description": "Commit message"},
		},
		"required":             []string{"message"},
		"additionalProperties": false,
	}
}

// Run executes the git commit tool
func (t *GitCommitTool) Run(args map[string]string) (string, *tools.FileChange, error) {
	message := args["message"]
	output := ExecuteGitCommit(message)
	return output, nil, nil
}

// GitCheckoutTool is the tool for git checkout
type GitCheckoutTool struct{}

// Name returns the tool name
func (t *GitCheckoutTool) Name() string {
	return "git_checkout"
}

func (t *GitCheckoutTool) Description() string {
	return "Switches branches or restores files."
}

func (t *GitCheckoutTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"target": map[string]interface{}{"type": "string", "description": "Branch name or file path to checkout"},
		},
		"required":             []string{"target"},
		"additionalProperties": false,
	}
}

// Run executes the git checkout tool
func (t *GitCheckoutTool) Run(args map[string]string) (string, *tools.FileChange, error) {
	target := args["target"]
	output, backupPath, err := ExecuteGitCheckout(target)
	if err != nil {
		return output, nil, err
	}

	var change *tools.FileChange
	if backupPath != "" {
		change = &tools.FileChange{
			FilePath:    target,
			BackupPath:  backupPath,
			Tool:        "git_checkout",
			Description: "Restored from HEAD: " + target,
		}
	}

	return output, change, nil
}

// RegisterTools registers git tools with the registry
func RegisterTools(registry *tools.Registry) {
	registry.Register(&GitCommitTool{})
	registry.Register(&GitCheckoutTool{})
}

func init() {
	RegisterTools(tools.DefaultRegistry)
}
