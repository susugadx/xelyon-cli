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
	HideCategory  bool
	ArgHint       string
	Detail        string
	HasArgs       bool
	// ExpandOnEnter は Enter で実行せず、引数付き補完へ進む候補を表す。
	ExpandOnEnter bool
	// CompleteOnEnter は Enter で実行せず、入力欄へ補完して閉じる候補を表す。
	CompleteOnEnter bool
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
	if s.HideCategory {
		return ""
	}
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
		Detail:        commandSuggestionDetail(cmd),
		HasArgs:       cmd.Args != "",
		ExpandOnEnter: commandSuggestionExpandsOnEnter(cmd),
		SubmitOnEnter: commandSuggestionSubmitsOnEnter(cmd),
	}
}

func commandSuggestionDetail(cmd commandcatalog.CommandInfo) string {
	detail := strings.TrimSpace(cmd.Description)
	if len(cmd.SubCommands) == 0 {
		return detail
	}

	parts := make([]string, 0, len(cmd.SubCommands)+1)
	if detail != "" {
		parts = append(parts, detail)
	}
	for _, sub := range cmd.SubCommands {
		name := strings.TrimSpace(sub.Name)
		if value, ok := subcommandArgumentValue(cmd.Name, sub.Name); ok && value != "" {
			name = value
		}
		description := strings.TrimSpace(sub.Description)
		if name == "" || description == "" {
			continue
		}
		parts = append(parts, name+": "+description)
	}
	return strings.Join(parts, " · ")
}

func commandSuggestionExpandsOnEnter(cmd commandcatalog.CommandInfo) bool {
	switch cmd.Name {
	case "/thinking", "/skills":
		return true
	default:
		return false
	}
}

func commandSuggestionSubmitsOnEnter(cmd commandcatalog.CommandInfo) bool {
	switch cmd.Name {
	case "/skills", "/thinking":
		return false
	default:
		return true
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
