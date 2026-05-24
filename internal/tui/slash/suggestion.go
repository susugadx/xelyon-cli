package slash

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/commandcatalog"
)

// Suggestion は TUI の slash 候補表示と入力補完に必要な情報を表す。
type Suggestion struct {
	Label      string
	InsertText string
	// SubmitText は Enter 実行時だけ InsertText と異なる command を使いたい場合に指定する。
	SubmitText  string
	Description string
	Category    commandcatalog.CommandCategory
	// CategoryLabel は候補表示用の短いカテゴリ名。空なら Category から導出する。
	CategoryLabel string
	ArgHint       string
	Detail        string
	HasArgs       bool
	// SubmitOnEnter は候補表示中の Enter で、この候補を確定して実行してよいかを表す。
	SubmitOnEnter bool
}

// CompletionText は候補を入力欄へ補完するときの文字列を返す。
func (s Suggestion) CompletionText(appendArgSpace bool) string {
	if appendArgSpace && s.HasArgs && !strings.HasSuffix(s.InsertText, " ") {
		return s.InsertText + " "
	}
	return s.InsertText
}

// SubmissionText は候補を Enter で実行するときの文字列を返す。
func (s Suggestion) SubmissionText() string {
	if s.SubmitText != "" {
		return s.SubmitText
	}
	return s.InsertText
}

// CategoryDisplayLabel は TUI 候補の左カラムに出すカテゴリ名を返す。
func (s Suggestion) CategoryDisplayLabel() string {
	if strings.TrimSpace(s.CategoryLabel) != "" {
		return strings.TrimSpace(s.CategoryLabel)
	}
	return commandcatalog.CommandCategoryDisplayLabel(s.Category)
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
		SubmitText:    commandSuggestionSubmitText(cmd),
		Description:   cmd.Description,
		Category:      cmd.Category,
		CategoryLabel: cmd.EffectiveCategoryDisplayLabel(),
		ArgHint:       cmd.Args,
		Detail:        cmd.Description,
		HasArgs:       cmd.Args != "",
		SubmitOnEnter: true,
	}
}

func commandSuggestionSubmitText(cmd commandcatalog.CommandInfo) string {
	switch cmd.Name {
	case "/plan":
		return "/plan toggle"
	default:
		return ""
	}
}
