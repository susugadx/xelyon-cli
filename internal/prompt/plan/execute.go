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

	return fmt.Sprintf(`Execute step %d

### Task
%s
%s

### CRITICAL RULES (ALWAYS FOLLOW)

#### 1. Report Format (MANDATORY)
You MUST report your results in this exact format:

WHAT: [Detailed description of what you did]
FILES: [List of files changed with line numbers]
RESULT: [Success or Failure with specific reason]
NEXT: [Information for the next step]

#### 2. Code Quality (ALWAYS)
- ALWAYS follow Project Context (XELYON.md) rules if present
- ALWAYS run the appropriate formatter for the language
- ALWAYS follow the project's existing code style
- ALWAYS write comments for new functions
- ALWAYS verify changes work without errors

#### 3. Execution Rules
- Execute this step autonomously without asking questions
- Use tools directly - do NOT ask "Should I proceed?" or "Do you want me to..."
- For file changes: use str_replace for existing files, write_file for new files only
- If a tool fails, try an alternative approach - don't retry blindly
- Only stop for SafetyLow operations (delete_file, dangerous bash commands)

#### 4. Reporting Rules (CRITICAL)
- NEVER report just "done" or "completed"
- NEVER give vague responses
- ALWAYS be specific about what changed
- ALWAYS include file paths and line numbers

### REMEMBER (IMPORTANT - Read 3 times)
1. Report in detail, not just "done"
2. Report in detail, not just "done"
3. Report in detail, not just "done"


### Instructions
Execute this step now. When done, report using the MANDATORY format above.`, stepID, description, toolsHint)
}
