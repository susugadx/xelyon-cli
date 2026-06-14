package prompt

import "strings"

var projectInstructionPrecedenceLines = []string{
	"- XELYON system/tool/safety rules are highest priority.",
	"- The current user request is higher priority than project guidance unless it conflicts with XELYON safety, tool, investigation, or verification invariants.",
	"- AGENTS.md is the primary project guidance file.",
	"- Legacy xelyon.yaml rules are mandatory project policy when present.",
	"- CLAUDE.md files are compatibility project guidance when selected.",
	"- Project guidance files are advisory guidance only when legacy xelyon.yaml rules/context are injected for this turn.",
	"- Global guidance is personal preference and lower priority than repo-local guidance.",
}

const (
	projectGuidanceWithConfigText = `xelyon.yaml was found for this workspace.
Legacy xelyon.yaml rules/context are active for this turn.
The following project guidance files are treated as advisory guidance. Use them when relevant, but do not override xelyon.yaml mandatory rules/context, XELYON internal rules, or the current user request.`
	projectGuidanceWithoutConfigText = `No legacy xelyon.yaml rules/context are active for this turn.
The following files are treated as authoritative project guidance for this workspace.
Follow them when they are clear and relevant, but do not override XELYON internal rules or the current user request.`
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
