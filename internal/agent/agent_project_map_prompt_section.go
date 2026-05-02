package agent

import (
	"strings"

	promptpkg "github.com/susugadx/xelyon-cli/internal/prompt"
)

func renderProjectMapFocusOverlay(paths []string) string {
	paths = dedupeProjectMapPriorityPaths(paths)
	if len(paths) == 0 {
		return ""
	}
	if len(paths) > projectMapFocusMaxPaths {
		paths = paths[:projectMapFocusMaxPaths]
	}

	var b strings.Builder
	b.WriteString("Focus files for current task:\n")
	for _, path := range paths {
		b.WriteString("- ")
		b.WriteString(path)
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}

func composeProjectMapPromptSection(baseSection, focusSection string) string {
	baseSection = strings.TrimRight(baseSection, "\n")
	focusSection = strings.TrimRight(focusSection, "\n")

	switch {
	case baseSection == "":
		if focusSection == "" {
			return ""
		}
		return "## Project Map\n\n" + focusSection
	case focusSection == "":
		return baseSection
	default:
		return baseSection + "\n\n" + focusSection
	}
}

func countProjectMapFocusLines(section string) int {
	if strings.TrimSpace(section) == "" {
		return 0
	}
	count := 0
	for _, line := range strings.Split(section, "\n") {
		if strings.HasPrefix(line, "- ") {
			count++
		}
	}
	return count
}

func dedupeProjectMapPriorityPaths(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	deduped := make([]string, 0, len(paths))
	for _, path := range paths {
		if path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		deduped = append(deduped, path)
	}
	return deduped
}

func appendProjectMapSection(systemPrompt, section string) string {
	if strings.TrimSpace(section) == "" {
		return systemPrompt
	}
	layout := parseSystemPromptLayout(systemPrompt)
	layout.SetDynamic(promptpkg.InjectProjectMapSection(layout.Dynamic, section))
	return layout.Compose()
}
