package prompt

import "strings"

var projectInstructionPrecedenceLines = []string{
	"- XELYON system/tool/safety rules are highest priority.",
	"- The current user request is higher priority than project guidance unless it conflicts with XELYON safety, tool, investigation, or verification invariants.",
	"- AGENTS.md is the primary repository guidance file.",
	"- Repository guidance is loaded from root to nearest scope; when multiple repository_instructions apply, the later nearest scope takes precedence within repository guidance.",
	"- xelyon.yaml is structured repo-local XELYON config; legacy context/rules are load/save compatibility and are not injected into normal system prompts.",
	"- CLAUDE.md files are compatibility project guidance when selected.",
	"- Global guidance is personal preference and lower priority than repo-local guidance.",
}

const (
	projectGuidanceText = `The following repository guidance files are loaded in root-to-nearest order.
Each repository_instructions block declares the directory scope it applies to and the source file.
When multiple repository guidance blocks apply, later blocks with a nearer scope take precedence within repository guidance.`
	globalGuidanceText = "Global guidance is advisory personal preference."
)

func buildProjectInstructionPrecedenceBlock() string {
	return strings.Join(projectInstructionPrecedenceLines, "\n")
}

func guidanceHeadingLabel(entry ProjectInstructionEntry) string {
	label := strings.TrimSpace(entry.Label)
	strength := strings.TrimSpace(strings.ToLower(entry.Strength))
	switch strength {
	case "project_guidance":
		return label + " (project guidance)"
	case "advisory":
		return label + " (advisory)"
	default:
		return label
	}
}
