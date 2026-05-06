package slash

import "strings"

const (
	thinkingCommandName  = "/thinking"
	thinkingCommandAlias = "/think"
)

type thinkingOption struct {
	value       string
	labelSuffix string
	description string
}

var thinkingOptions = []thinkingOption{
	{value: "on", description: "Enable Extended Thinking with the current level"},
	{value: "off", description: "Disable Extended Thinking"},
	{value: "low", description: "Enable low thinking effort"},
	{value: "medium", description: "Enable medium thinking effort"},
	{value: "high", description: "Enable high thinking effort"},
	{value: "xhigh", labelSuffix: " (max)", description: "Enable maximum thinking effort"},
}

func thinkingArgumentSuggestions(req argumentSuggestionRequest) ([]Suggestion, bool) {
	if req.command != thinkingCommandName && req.command != thinkingCommandAlias {
		return nil, false
	}

	argPrefix := strings.ToLower(req.argPrefix)
	submitOnEnter := argPrefix != ""
	suggestions := make([]Suggestion, 0, len(thinkingOptions))
	for _, option := range thinkingOptions {
		if !strings.HasPrefix(option.value, argPrefix) {
			continue
		}
		suggestions = append(suggestions, newArgumentSuggestion(
			thinkingCommandName,
			option.value,
			option.labelSuffix,
			option.description,
			submitOnEnter,
		))
	}
	return suggestions, true
}
