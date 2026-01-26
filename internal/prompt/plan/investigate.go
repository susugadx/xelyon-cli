package plan

import "fmt"

// BuildInvestigationPrompt generates the prompt for the investigation phase.
// userRequest is the original user request to investigate.
func BuildInvestigationPrompt(userRequest string) string {
	return fmt.Sprintf(`USER REQUEST: %s

You are in PLAN MODE (Investigation Phase).

## INVESTIGATION RULES:
1. Think first: identify ALL files/areas you need to examine
2. Batch tool calls: read multiple files together, don't read one-by-one
3. Read-only tools allowed: read_file, search_file, search_code, list_dir, lsp_references, lsp_definition, lint, web_search
4. bash allowed for read-only commands only: git status, git log, git diff, ls, cat, etc.
5. Do NOT use modification tools: write_file, str_replace, delete_file, git_commit, or destructive bash commands

## INVESTIGATION CHECKLIST:
- [ ] Understand the current implementation (read relevant files)
- [ ] Find related code (search for usages, dependencies)
- [ ] Check for existing patterns to follow
- [ ] Identify potential impacts of changes

## OUTPUT:
When ready, output your plan in JSON:
{"plan": {
  "summary": "Brief summary of what will be done",
  "steps": [
    {"id": 1, "description": "Specific action", "tools": ["tool1"]},
    {"id": 2, "description": "Specific action", "tools": ["tool2"]}
  ]
}}

Each step should be concrete and actionable, not vague.

Start investigation now.`, userRequest)
}

// BuildPlanRequestMessage generates the system message asking for a plan
// when a modification tool is detected during investigation.
// toolName is the name of the detected modification tool.
func BuildPlanRequestMessage(toolName string) string {
	return fmt.Sprintf(`[SYSTEM] You tried to use a modification tool (%s) during the investigation phase.

Before using modification tools, you must output an implementation plan.
Output your plan now in this JSON format:

{"plan": {
  "summary": "Brief summary of what will be done",
  "steps": [
    {"id": 1, "description": "Step description", "tools": ["tool1"]},
    {"id": 2, "description": "Step description", "tools": ["tool2"]}
  ]
}}`, toolName)
}
