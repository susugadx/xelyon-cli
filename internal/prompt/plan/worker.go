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

	return fmt.Sprintf(`## Execute Step %d

### Task
%s

### Previous Context
%s

### Suggested Tools
%s

### Rules
- Execute autonomously without asking questions
- str_replace for existing files, write_file for new files only
- If a tool fails, try alternative approach
- Run formatter after changes

### Report Format (MANDATORY)
WHAT: [What you did]
FILES: [Files changed with line numbers]
RESULT: [Success/Failure with reason]
NEXT: [Info for other workers]

Execute now.`, step.ID, step.Description, context, toolsList)
}

func BuildInvestigationExecutionPrompt(query string) string {
	return fmt.Sprintf(`## INVESTIGATION QUERY
%s

### READ-ONLY ONLY
Modification tools are FORBIDDEN: write_file, str_replace, delete_file

Allowed: read_file, search_code, search_file, list_dir, lsp_find, bash (read-only), web_search

### Report Format (MANDATORY)
QUERY: [The original query]
FINDINGS: [Detailed findings with file paths and line numbers]
FILES_EXAMINED: [List of files you read]
CONCLUSION: [Summary and recommendations]

Focus on answering the query comprehensively.`, query)
}
