package slash

import (
	"strings"
	"unicode"

	"github.com/susugadx/xelyon-cli/internal/commandcatalog"
)

// Suggestion は TUI の slash 候補表示と入力補完に必要な情報を表す。
type Suggestion struct {
	Label       string
	InsertText  string
	Description string
	HasArgs     bool
	// SubmitOnEnter は候補表示中の Enter で、この候補を確定して実行してよいかを表す。
	SubmitOnEnter bool
}

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

// CompletionText は候補を入力欄へ補完するときの文字列を返す。
func (s Suggestion) CompletionText(appendArgSpace bool) string {
	if appendArgSpace && s.HasArgs && !strings.HasSuffix(s.InsertText, " ") {
		return s.InsertText + " "
	}
	return s.InsertText
}

// Suggestions は prefix に一致する slash command 候補を TUI 用 item として返す。
func Suggestions(prefix string) []Suggestion {
	if suggestions, ok := argumentSuggestions(prefix); ok {
		return suggestions
	}
	commands := commandcatalog.MatchDiscoverablePrefixForSurface(prefix, commandcatalog.CommandSurfaceTUI)
	suggestions := make([]Suggestion, 0, len(commands))
	for _, cmd := range commands {
		suggestions = append(suggestions, newCommandSuggestion(cmd))
	}
	return suggestions
}

func newCommandSuggestion(cmd commandcatalog.CommandInfo) Suggestion {
	label := cmd.Name
	if cmd.Args != "" {
		label += " " + cmd.Args
	}
	return Suggestion{
		Label:         label,
		InsertText:    cmd.Name,
		Description:   cmd.Description,
		HasArgs:       cmd.Args != "",
		SubmitOnEnter: true,
	}
}

func argumentSuggestions(prefix string) ([]Suggestion, bool) {
	command, argPrefix, argOK, ok := splitCommandArgumentPrefix(prefix)
	if !ok {
		return nil, false
	}
	if command != thinkingCommandName && command != thinkingCommandAlias {
		return nil, true
	}
	if !argOK {
		return nil, true
	}
	return thinkingArgumentSuggestions(argPrefix, argPrefix != ""), true
}

func splitCommandArgumentPrefix(input string) (string, string, bool, bool) {
	input = strings.TrimLeftFunc(input, unicode.IsSpace)
	index := strings.IndexFunc(input, unicode.IsSpace)
	if index < 0 {
		return "", "", false, false
	}

	command := input[:index]
	argPrefix := strings.TrimLeftFunc(input[index:], unicode.IsSpace)
	if strings.IndexFunc(argPrefix, unicode.IsSpace) >= 0 {
		return command, "", false, true
	}
	return command, argPrefix, true, true
}

func thinkingArgumentSuggestions(argPrefix string, submitOnEnter bool) []Suggestion {
	argPrefix = strings.ToLower(argPrefix)
	suggestions := make([]Suggestion, 0, len(thinkingOptions))
	for _, option := range thinkingOptions {
		if !strings.HasPrefix(option.value, argPrefix) {
			continue
		}
		insertText := thinkingCommandName + " " + option.value
		suggestions = append(suggestions, Suggestion{
			Label:         insertText + option.labelSuffix,
			InsertText:    insertText,
			Description:   option.description,
			HasArgs:       false,
			SubmitOnEnter: submitOnEnter,
		})
	}
	return suggestions
}
