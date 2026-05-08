package slash

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/commandcatalog"
)

var catalogSubcommandSuggestionCommands = map[string]struct{}{
	"/skills": {},
}

func catalogSubcommandArgumentSuggestions(req argumentSuggestionRequest) ([]Suggestion, bool) {
	cmd, ok := commandcatalog.Find(req.command)
	if !ok || len(cmd.SubCommands) == 0 || !catalogSubcommandSuggestionsEnabled(cmd.Name) {
		return nil, false
	}

	argPrefix := strings.ToLower(req.argPrefix)
	suggestions := make([]Suggestion, 0, len(cmd.SubCommands))
	for _, sub := range cmd.SubCommands {
		value, ok := subcommandArgumentValue(cmd.Name, sub.Name)
		if !ok || !strings.HasPrefix(strings.ToLower(value), argPrefix) {
			continue
		}
		insertText, hasArgs := subcommandInsertText(cmd.Name, value)
		suggestions = append(suggestions, Suggestion{
			Label:         sub.Name,
			InsertText:    insertText,
			Description:   sub.Description,
			Category:      cmd.Category,
			ArgHint:       value,
			Detail:        sub.Description,
			HasArgs:       hasArgs,
			SubmitOnEnter: true,
		})
	}
	return suggestions, true
}

func catalogSubcommandSuggestionsEnabled(commandName string) bool {
	_, ok := catalogSubcommandSuggestionCommands[commandName]
	return ok
}

func subcommandArgumentValue(commandName, subcommandName string) (string, bool) {
	prefix := commandName + " "
	if !strings.HasPrefix(subcommandName, prefix) {
		return "", false
	}
	return strings.TrimSpace(strings.TrimPrefix(subcommandName, prefix)), true
}

func subcommandInsertText(commandName, value string) (string, bool) {
	tokens := strings.Fields(value)
	insertTokens := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if isArgumentPlaceholder(token) {
			break
		}
		insertTokens = append(insertTokens, token)
	}

	insertText := commandName
	if len(insertTokens) > 0 {
		insertText += " " + strings.Join(insertTokens, " ")
	}
	return insertText, len(insertTokens) < len(tokens)
}

func isArgumentPlaceholder(token string) bool {
	return strings.HasPrefix(token, "<") || strings.HasPrefix(token, "[")
}
