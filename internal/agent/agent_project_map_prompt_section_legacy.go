package agent

import "github.com/susugadx/xelyon-cli/internal/prompt"

func extractProjectMapSection(systemPrompt string) string {
	return prompt.ExtractProjectMapSectionCompat(systemPrompt)
}

func stripProjectMapSection(systemPrompt string) string {
	return prompt.StripProjectMapSectionCompat(systemPrompt)
}
