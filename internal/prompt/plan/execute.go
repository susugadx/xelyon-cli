package plan

import (
	"fmt"
	"strings"
)

// BuildStepPrompt generates the prompt for executing a specific plan step.
// stepID is the step number, description is the step description,
// tools is the list of expected tools for this step.
func BuildStepPrompt(stepID int, description string, tools []string) string {
	// 期待するツールがある場合、明示的に指示
	toolsHint := ""
	if len(tools) > 0 {
		toolsHint = fmt.Sprintf("\n\nYou MUST use the following tools to complete this step: %s\nDo NOT ask for confirmation - execute the tools directly.", strings.Join(tools, ", "))
	}

	return fmt.Sprintf(`Execute step %d of the implementation plan:
%s

IMPORTANT INSTRUCTIONS:
1. Execute this step autonomously without asking questions
2. Use tools directly - do NOT ask "Should I proceed?" or "Do you want me to..."
3. If you need to create/modify files, use write_file or str_replace directly
4. Only stop for SafetyLow operations (delete_file, dangerous bash commands)%s`, stepID, description, toolsHint)
}
