package slash

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/commandcatalog"
)

// AliasResolver は command 名を alias 解決する関数を表す。
type AliasResolver func(string) string

// Command は composer から抽出した slash command 入力を表す。
type Command struct {
	Input        string
	Payload      string
	ResolvedName string
	ArgCount     int
}

// TrimmedInput は入力を trim し、slash command かどうかを返す。
func TrimmedInput(value string) (string, bool) {
	input := strings.TrimSpace(value)
	return input, strings.HasPrefix(input, "/")
}

// NewCommand は command 入力と payload から Command を構築する。
func NewCommand(input, payload string, resolve AliasResolver) Command {
	cmdParts := strings.Fields(input)
	resolvedCommand := input
	if len(cmdParts) > 0 {
		resolvedCommand = cmdParts[0]
		if resolve != nil {
			resolvedCommand = resolve(cmdParts[0])
		}
	}
	return Command{
		Input:        input,
		Payload:      payload,
		ResolvedName: resolvedCommand,
		ArgCount:     len(cmdParts),
	}
}

// IsBare は command が指定名だけで引数を持たないかを返す。
func (c Command) IsBare(name string) bool {
	return c.ResolvedName == name && c.ArgCount == 1
}

// Matches は command の alias 解決後の名前が指定名と一致するかを返す。
func (c Command) Matches(name string) bool {
	return c.ResolvedName == name
}

// Suggestions は prefix に一致する slash command 候補を返す。
func Suggestions(prefix string) []commandcatalog.CommandInfo {
	return commandcatalog.MatchPrefix(prefix)
}
