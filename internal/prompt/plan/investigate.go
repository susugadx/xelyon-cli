package plan

import "fmt"

// BuildInvestigationPrompt generates the prompt for the investigation phase.
// userRequest is the original user request to investigate.
func BuildInvestigationPrompt(userRequest string) string {
	return fmt.Sprintf(`USER REQUEST: %s

You are in PLAN MODE (Investigation Phase).

## INVESTIGATION RULES:
## INVESTIGATION RULES:
1. **Check RepoMap first**: RepoMap at the end of system prompt contains all file paths and function names with line numbers
  - Find the file/function in RepoMap before searching
  - Example: "plan_executor.go:18: func runImplementationPhase" → read_file directly

2. **PREFER LSP tools**: Use lsp_definition and lsp_references FIRST for code navigation
  - lsp_definition: Find where functions/types are defined (1 call, exact location)
  - lsp_references: Find all usages of a symbol (accurate, ignores comments)
  - lsp_hover: Get type information
  - Only use search_code when LSP doesn't work or for keyword search (TODO, error messages)

3. Read-only tools allowed: read_file, search_file, search_code, list_dir, lsp_references, lsp_definition, lint, web_search
4. bash allowed for read-only commands only: git status, git log, git diff, ls, cat, etc.
5. Do NOT use modification tools: write_file, str_replace, delete_file, git_commit

## INVESTIGATION CHECKLIST:
- [ ] Understand the current implementation (read relevant files)
- [ ] Find related code (search for usages, dependencies)
- [ ] Check for existing patterns to follow
- [ ] Identify potential impacts of changes

## AFTER INVESTIGATION:
When you have gathered enough information:
1. If you need clarification from the user, use the ask_user_question tool
2. When ready, use the create_plan tool to create your implementation plan

IMPORTANT:
- The plan should contain IMPLEMENTATION steps, not investigation steps
- Do NOT create a plan that says "investigate X" or "read file Y"
- Each step should be an ACTION that modifies the codebase (write_file, str_replace, etc.)
- If you still need to investigate, continue using read-only tools first

Do NOT output JSON directly. Always use the create_plan tool.

Start investigation now.`, userRequest)
}

// BuildPlanRequestMessage generates the system message asking for a plan
// when a modification tool is detected during investigation.
// toolName is the name of the detected modification tool.
func BuildPlanRequestMessage(toolName string) string {
	return fmt.Sprintf(`[SYSTEM] You tried to use a modification tool (%s) during the investigation phase.

Before using modification tools, you must create an implementation plan.
Use the create_plan tool now to create your plan.

Do NOT output JSON directly.`, toolName)
}
