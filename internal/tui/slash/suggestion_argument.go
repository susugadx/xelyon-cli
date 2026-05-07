package slash

import (
	"strings"
	"unicode"

	"github.com/susugadx/xelyon-cli/internal/commandcatalog"
)

type argumentSuggestionRequest struct {
	command       string
	argPrefix     string
	canSuggestArg bool
}

type argumentSuggestionProvider func(argumentSuggestionRequest) ([]Suggestion, bool)

var argumentSuggestionProviders = []argumentSuggestionProvider{
	catalogSubcommandArgumentSuggestions,
	thinkingArgumentSuggestions,
}

func argumentSuggestions(prefix string) ([]Suggestion, bool) {
	req, ok := newArgumentSuggestionRequest(prefix)
	if !ok {
		return nil, false
	}
	if !req.canSuggestArg {
		return nil, true
	}
	return suggestionsFromArgumentProviders(req), true
}

func newArgumentSuggestionRequest(input string) (argumentSuggestionRequest, bool) {
	input = strings.TrimLeftFunc(input, unicode.IsSpace)
	index := strings.IndexFunc(input, unicode.IsSpace)
	if index < 0 {
		return argumentSuggestionRequest{}, false
	}

	command := input[:index]
	argPrefix := strings.TrimLeftFunc(input[index:], unicode.IsSpace)
	if strings.IndexFunc(argPrefix, unicode.IsSpace) >= 0 {
		return argumentSuggestionRequest{command: command}, true
	}
	return argumentSuggestionRequest{
		command:       command,
		argPrefix:     argPrefix,
		canSuggestArg: true,
	}, true
}

func suggestionsFromArgumentProviders(req argumentSuggestionRequest) []Suggestion {
	for _, provider := range argumentSuggestionProviders {
		suggestions, ok := provider(req)
		if ok {
			return suggestions
		}
	}
	return nil
}

func newArgumentSuggestion(command, value, labelSuffix, description string, submitOnEnter bool) Suggestion {
	insertText := command + " " + value
	return Suggestion{
		Label:         insertText + labelSuffix,
		InsertText:    insertText,
		Description:   description,
		Category:      commandcatalog.CommandCategoryModel,
		ArgHint:       value,
		Detail:        description,
		HasArgs:       false,
		SubmitOnEnter: submitOnEnter,
	}
}
