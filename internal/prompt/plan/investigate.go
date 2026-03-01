package plan

import "fmt"

// BuildInvestigationPrompt generates the prompt for the investigation phase.
// userRequest is the original user request to investigate.
func BuildInvestigationPrompt(userRequest string) string {
	return fmt.Sprintf(`USER REQUEST: %s

You are in PLAN MODE (Investigation Phase).

### READ-ONLY ONLY
Modification tools are FORBIDDEN: write_file, str_replace, delete_file

Allowed: read_file, list_dir, search_code, bash (find/read-only), web_search

### INVESTIGATION CHECKLIST
-  Understand the current implementation (read relevant files)
-  Find related code (search for usages, dependencies)
-  Check for existing patterns to follow
-  Identify potential impacts of changes

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
