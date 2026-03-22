package agent

import (
	"os"

	"github.com/susugadx/xelyon-cli/internal/prompt"
)

func appendDefaultEditToolExclusions(excluded []string) []string {
	result := filterStrings(excluded, "apply_patch", "str_replace", "write_file", "delete_file")
	if os.Getenv("XELYON_EDIT_TOOL") == "str_replace" {
		return appendUniqueStrings(result, "apply_patch")
	}
	return appendUniqueStrings(result, "str_replace", "write_file", "delete_file")
}

func normalModeExcludedTools() []string {
	return appendDefaultEditToolExclusions(prompt.PlanningToolNames)
}

func planModeExcludedTools() []string {
	return appendDefaultEditToolExclusions(nil)
}

func appendUniqueStrings(values []string, extras ...string) []string {
	seen := make(map[string]struct{}, len(values)+len(extras))
	for _, value := range values {
		seen[value] = struct{}{}
	}
	for _, extra := range extras {
		if _, ok := seen[extra]; ok {
			continue
		}
		values = append(values, extra)
		seen[extra] = struct{}{}
	}
	return values
}

func filterStrings(values []string, excluded ...string) []string {
	blocked := make(map[string]struct{}, len(excluded))
	for _, value := range excluded {
		blocked[value] = struct{}{}
	}

	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := blocked[value]; ok {
			continue
		}
		result = append(result, value)
	}
	return result
}
