package plan

import "fmt"

// BuildInvestigationPrompt generates the prompt for the investigation phase.
// userRequest is the original user request to investigate.
func BuildInvestigationPrompt(userRequest string) string {
	return fmt.Sprintf(`USER REQUEST: %s

You are in PLAN MODE (Investigation Phase).

IMPORTANT RULES:
1. First, investigate the codebase to understand what needs to be done
2. You CAN use read-only tools freely: read_file, search_file, search_code, list_dir, git_status, git_log, git_diff, lint, test, web_search
3. Do NOT use any modification tools yet (write_file, str_replace, delete_file, bash, etc.)
4. When you have gathered enough information, output a plan for the implementation

After investigation, output your plan in this JSON format:
{"plan": {
  "summary": "Brief summary of what will be done",
  "steps": [
    {"id": 1, "description": "Step description", "tools": ["tool1"]},
    {"id": 2, "description": "Step description", "tools": ["tool2"]}
  ]
}}

Start your investigation now. Use read_file, search_code, list_dir etc. to understand the codebase.`, userRequest)
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
