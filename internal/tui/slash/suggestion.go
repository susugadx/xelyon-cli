package slash

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/commandcatalog"
)

// Suggestion は TUI の slash 候補表示と入力補完に必要な情報を表す。
type Suggestion struct {
	Label       string
	InsertText  string
	Description string
	HasArgs     bool
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
		Label:       label,
		InsertText:  cmd.Name,
		Description: cmd.Description,
		HasArgs:     cmd.Args != "",
	}
}
