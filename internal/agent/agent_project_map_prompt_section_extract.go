package agent

import promptpkg "github.com/susugadx/xelyon-cli/internal/prompt"

func extractProjectMapSection(systemPrompt string) string {
	return promptpkg.ExtractProjectMapSectionCompat(systemPrompt)
}

func stripProjectMapSection(systemPrompt string) string {
	return promptpkg.StripProjectMapSectionCompat(systemPrompt)
}
