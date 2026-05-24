package tui

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/commandcatalog"
	"github.com/susugadx/xelyon-cli/internal/providerpicker"
	"github.com/susugadx/xelyon-cli/internal/tui/slash"
)

func providerArgumentSuggestions(candidates []providerpicker.ProviderCandidate, argPrefix string) []slash.Suggestion {
	argPrefix = strings.ToLower(strings.TrimSpace(argPrefix))
	submitOnEnter := argPrefix != ""
	suggestions := make([]slash.Suggestion, 0, len(candidates))
	for _, candidate := range candidates {
		if !strings.HasPrefix(strings.ToLower(candidate.Key), argPrefix) {
			continue
		}
		insertText := "/provider " + candidate.Key
		suggestions = append(suggestions, slash.Suggestion{
			Label:         candidate.Key,
			InsertText:    insertText,
			Description:   providerCandidateDescription(candidate),
			Category:      commandcatalog.CommandCategoryModel,
			CategoryLabel: "provider",
			HideCategory:  true,
			Detail:        "Switch to " + candidate.Label,
			SubmitOnEnter: submitOnEnter,
		})
	}
	return suggestions
}

func modelArgumentSuggestions(candidates []providerpicker.ModelCandidate, argPrefix string) []slash.Suggestion {
	argPrefix = strings.ToLower(strings.TrimSpace(argPrefix))
	submitOnEnter := argPrefix != ""
	suggestions := make([]slash.Suggestion, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.Custom || !strings.HasPrefix(strings.ToLower(candidate.Name), argPrefix) {
			continue
		}
		insertText := "/model " + candidate.Name
		suggestions = append(suggestions, slash.Suggestion{
			Label:         candidate.Name,
			InsertText:    insertText,
			Description:   modelCandidateDescription(candidate),
			Category:      commandcatalog.CommandCategoryModel,
			CategoryLabel: "model",
			HideCategory:  true,
			Detail:        "Switch current provider model to " + candidate.Name,
			SubmitOnEnter: submitOnEnter,
		})
	}
	return suggestions
}

func providerCandidateDescription(candidate providerpicker.ProviderCandidate) string {
	var parts []string
	if candidate.Label != "" && candidate.Label != candidate.Key {
		parts = append(parts, candidate.Label)
	}
	if candidate.Current {
		parts = append(parts, "current")
	}
	if candidate.CredentialStatus != "" {
		parts = append(parts, string(candidate.CredentialStatus))
	}
	return strings.Join(parts, " · ")
}

func modelCandidateDescription(candidate providerpicker.ModelCandidate) string {
	var parts []string
	if candidate.Current {
		parts = append(parts, "current")
	}
	if candidate.Default {
		parts = append(parts, "default")
	}
	return strings.Join(parts, " · ")
}
