package plan

import "fmt"

// BuildInvestigationPrompt generates the prompt for the investigation phase.
// userRequest is the original user request to investigate.
func BuildInvestigationPrompt(userRequest string) string {
	return fmt.Sprintf(`USER REQUEST: %s

You are in PLAN MODE (Investigation Phase).

### READ-ONLY ONLY
Modification tools are FORBIDDEN: write_file, str_replace, delete_file

Allowed: inspect_symbol, search_code, read_file, list_dir, web_search, bash (read-only git commands only: git status, git diff, git log)

### INVESTIGATION CHECKLIST
-  Use inspect_symbol for known Go symbols (returns definition + callers + refs + tests)
-  Use search_code for broad/regex discovery across the codebase
-  Use read_file for detailed implementation context
-  For local changes (single function, local bug fix): read the target, check for immediate dependencies, then plan
-  For shared changes (interface, public API, config, rename, delete): find ALL usages, dependencies, and tests before planning
-  Check for existing patterns to follow
-  Avoid broad exploration when the target is already clear

### AFTER INVESTIGATION
When ready, output your implementation plan as text that includes a single JSON object matching the Plan schema.
- The runtime extracts it via ExtractPlanJSON/ParsePlan
- Plan should contain IMPLEMENTATION steps, not investigation steps
- Do NOT create steps like "investigate X" or "read file Y"
- Each step should be an ACTION that modifies the codebase

Start investigation now.`, userRequest)
}

// BuildPlanRequestMessage generates the system message asking for a plan
// when a modification tool is detected during investigation.
// toolName is the name of the detected modification tool.
func BuildPlanRequestMessage(toolName string) string {
	return fmt.Sprintf(`[SYSTEM] You tried to use a modification tool (%s) during the investigation phase.

Before using modification tools, you must provide an implementation plan.
Output your plan as text that includes a single JSON object matching the Plan schema.
The runtime extracts it via ExtractPlanJSON/ParsePlan.

Do NOT call create_plan/update_plan tools.`, toolName)
}
