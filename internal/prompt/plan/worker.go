package plan

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
)

// BuildStepExecutionPrompt は Worker がステップを実行するためのプロンプト
func BuildStepExecutionPrompt(step *plan.PlanStep, context string) string {
	toolsList := "any appropriate tools"
	if len(step.Tools) > 0 {
		toolsList = strings.Join(step.Tools, ", ")
	}

	return fmt.Sprintf(`Execute step %d:
%s

Previous context from completed steps:
%s

Suggested tools: %s

Execute this step autonomously without asking questions.
Focus on completing this specific step only.
Report success or failure clearly.`, step.ID, step.Description, context, toolsList)
}

// BuildInvestigationExecutionPrompt は Worker が調査を実行するためのプロンプト
func BuildInvestigationExecutionPrompt(query string) string {
	return fmt.Sprintf(`INVESTIGATION QUERY:
%s

Execute this investigation using read-only tools:
- lsp_references: Find all references to a symbol
- lsp_definition: Go to definition
- lsp_hover: Get hover information
- lsp_diagnostics: Get diagnostics for a file
- read_file: Read file contents
- search_code: Search for code patterns
- search_file: Find files by name
- list_dir: List directory contents
- git_status: Check git status
- git_diff: View git diff

Gather relevant information and report your findings clearly.
Do NOT make any modifications to the codebase.
Focus on answering the query comprehensively.`, query)
}
