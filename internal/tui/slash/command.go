package slash

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/commandruntime"
)

// Command は composer から抽出した slash command 入力を表す。
type Command struct {
	Input        string
	Payload      string
	ResolvedName string
	Args         []string
	ParseStatus  commandruntime.SplitStatus
}

// TrimmedInput は入力を trim し、slash command かどうかを返す。
func TrimmedInput(value string) (string, bool) {
	input := strings.TrimSpace(value)
	return input, strings.HasPrefix(input, "/")
}

// NewCommand は command 入力と payload から Command を構築する。
// ResolvedName は command token の小文字正規化だけを行う。
// alias 解決は command catalog を source of truth とする。
func NewCommand(input, payload string) Command {
	cmdParts, parseStatus := commandruntime.SplitStrict(input)
	resolvedCommand := input
	var args []string
	if len(cmdParts) > 0 {
		resolvedCommand = strings.ToLower(cmdParts[0])
		args = append([]string(nil), cmdParts[1:]...)
	}
	return Command{
		Input:        input,
		Payload:      payload,
		ResolvedName: resolvedCommand,
		Args:         args,
		ParseStatus:  parseStatus,
	}
}

// ParseOK は command parse が成功したかを返す。
func (c Command) ParseOK() bool {
	return c.ParseStatus.IsOK()
}

// IsBare は command が指定名だけで引数を持たないかを返す。
func (c Command) IsBare(name string) bool {
	return c.ResolvedName == name && len(c.Args) == 0
}

// Matches は command の alias 解決後の名前が指定名と一致するかを返す。
func (c Command) Matches(name string) bool {
	return c.ResolvedName == name
}
