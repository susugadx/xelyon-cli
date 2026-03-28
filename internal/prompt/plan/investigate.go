package plan

import "fmt"

// BuildInvestigationPrompt generates the prompt for the investigation phase.
// userRequest is the original user request to investigate.
func BuildInvestigationPrompt(userRequest string) string {
	return fmt.Sprintf(`USER REQUEST: %s

You are in PLAN MODE (Investigation Phase).

### READ-ONLY ONLY
Modification tools are FORBIDDEN: apply_patch, write_file, str_replace, delete_file

Allowed: search_code, read_file, web_search, bash (read-only git commands only: git status, git diff, git log)

### INVESTIGATION CHECKLIST
-  Use search_code for code discovery. It uses language-aware routing across symbol-aware resolution, literal search, and regex search. Prefer mode=auto, short symbol queries when possible, and regex only when needed.
-  Use read_file for detailed implementation context
- Prefer parallel investigation: batch independent read_file/search_code steps in one response
- Reading 2+ independent files -> one read_file call with all paths
- Searching multiple independent patterns -> prefer one search_code call with comma-separated patterns
  - For related code discovery, multi-pattern search_code is the default. Use one combined query for target + helpers + references/callers + tests instead of serial narrow searches.
   - If you are about to issue a second search_code call for the same task, first stop and check whether the searches should be merged into one comma-separated multi-pattern query.
  - After the initial search_code, prefer moving to read_file. A follow-up search_code should usually be a corrective multi-pattern refinement.
- For local changes (single function, local bug fix): read the target, check for immediate dependencies, then plan
- For shared changes (interface, public API, config, rename, delete): find ALL usages, dependencies, and tests before planning
- Check for existing patterns to follow
- Avoid broad exploration when the target is already clear

### EXAMPLES
- Symbol review -> search_code(pattern="chatCore", path="internal/agent/agent_chat.go")
- Need implementation + tests -> read_file(paths=["impl.go", "impl_test.go"])
- Broad pattern search -> search_code(pattern="handleError,validateInput")

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
