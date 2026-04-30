package commandruntime

// Handler は slash command の副作用処理を実行する。
type Handler func(args []string) bool

// Registry は command 名から handler への対応を表す。
type Registry map[string]Handler

// IsNonInteractiveConfigSubcommand は stdin を読まずに処理できる /config サブコマンドかを返す。
func IsNonInteractiveConfigSubcommand(args []string) bool {
	if len(args) == 0 {
		return false
	}

	switch args[0] {
	case "show":
		return len(args) == 1
	case "model":
		return len(args) >= 2
	default:
		return false
	}
}
