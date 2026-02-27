package plan

import (
	"fmt"
	"strings"
)

// BuildStepPrompt generates the prompt for executing a specific plan step.
// stepID is the step number, description is the step description,
// tools is the list of expected tools for this step.
func BuildStepPrompt(stepID int, description string, tools []string) string {
	toolsHint := ""
	if len(tools) > 0 {
		toolsHint = fmt.Sprintf("\n\nYou MUST use the following tools: %s\nExecute directly without asking for confirmation.", strings.Join(tools, ", "))
	}

	return fmt.Sprintf(`Execute step %d

### Task
%s
%s

### Rules
- Execute ONLY this step — do NOT execute other steps or claim they are already done
- Do NOT use update_plan to set status to completed — the system handles completion automatically
- You MUST actually call the required tools — do NOT just describe what you would do in text
- If this step requires str_replace, you MUST call str_replace — text descriptions of changes are NOT execution
- Execute autonomously - do NOT ask "Should I proceed?"
- Prefer str_replace for edits to existing files, write_file for new files or full rewrites
- If a tool fails, try alternative approach


### Report Format (MANDATORY)
WHAT: [What you did]
FILES: [Files changed with line numbers]
RESULT: [Success/Failure with reason]
NEXT: [Info for next step]

Execute now.`, stepID, description, toolsHint)
}
