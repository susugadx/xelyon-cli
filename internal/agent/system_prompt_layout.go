package agent

import "github.com/susugadx/xelyon-cli/internal/api"

func parseSystemPromptLayout(systemPrompt string) api.SystemPromptLayout {
	return api.SplitSystemPromptLayout(systemPrompt)
}
