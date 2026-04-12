package plan

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/investigation"
	promptfragments "github.com/susugadx/xelyon-cli/internal/prompt/fragments"
)

// BuildInvestigationPrompt generates the prompt for the investigation phase.
// userRequest is the original user request to investigate.
func BuildInvestigationPrompt(userRequest string, surface investigation.Surface) string {
	surface = investigation.NormalizeSurface(surface)
	allowedTools := promptfragments.InvestigationAllowedToolsLine(surface)
	checklistLines := []string{
		promptfragments.BuildInvestigationToolingBlock(promptfragments.InvestigationToolingOptions{
			Surface:                surface,
			SearchOverrideLabel:    "a low-level expert override",
			ReadOverrideExtra:      "Use it only when you already know the exact file or range and need exact manual control.",
			IncludeBatchRead:       true,
			BatchReadOverrideLabel: "a low-level override",
		}),
		`- If the Project Map already gives an exact file or range, pass that exact target to gather_context(query="...").`,
		`- Prefer parallel investigation: batch independent gather_context/web_search steps in one response`,
		promptfragments.SharedChangeGatherContextLine(""),
		`- For local changes (single function, local bug fix): read the target, check for immediate dependencies, then plan`,
		`- For shared changes (interface, public API, config, rename, delete): find ALL usages, dependencies, and tests before planning`,
		`- Check for existing patterns to follow`,
		`- Avoid broad exploration when the target is already clear`,
	}
	if surface.AllowsLowLevelOverrides() {
		checklistLines = append(checklistLines[:2], append([]string{promptfragments.LowLevelOverridesWhenExposedLine()}, checklistLines[2:]...)...)
	}
	checklist := strings.Join(checklistLines, "\n")

	return fmt.Sprintf(`USER REQUEST: %s

You are in PLAN MODE (Investigation Phase).

### READ-ONLY ONLY
Modification tools are FORBIDDEN: apply_patch, write_file, str_replace, delete_file

%s

### INVESTIGATION CHECKLIST
%s

### EXAMPLES
- Symbol review -> gather_context(query="chatCore", path="internal/agent/agent_chat.go")
- Need implementation + tests -> gather_context(query="impl.go,impl_test.go", path="pkg", file_filter="go")
- Direct file read -> gather_context(query="internal/agent/agent.go:161-328")
- Directory summary -> gather_context(query="internal/tools/search")

### AFTER INVESTIGATION
When ready, output your implementation plan as text that includes a single JSON object matching the Plan schema.
- The runtime extracts it via ExtractPlanJSON/ParsePlan
- Plan should contain IMPLEMENTATION steps, not investigation steps
- Do NOT create steps like "investigate X" or "read file Y"
- Each step should be an ACTION that modifies the codebase

Start investigation now.`, userRequest, allowedTools, checklist)
}

// BuildPlanRequestMessage generates the system message asking for a plan
// when a modification tool is detected during investigation.
// toolName is the name of the detected modification tool.
func BuildPlanRequestMessage(toolName string) string {
	return fmt.Sprintf(`[SYSTEM] You tried to use a modification tool (%s) during the investigation phase.

Before using modification tools, you must provide an implementation plan.
Output your plan as text that includes a single JSON object matching the Plan schema.
The runtime extracts it via ExtractPlanJSON/ParsePlan.

Do not call tools in this response. Return only the implementation plan.`, toolName)
}
