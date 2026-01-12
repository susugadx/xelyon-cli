package agent

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
)

// defaultCommandAliases は組み込みのコマンドエイリアス。
// config.yaml の command_aliases で上書き・追加できる。
var defaultCommandAliases = map[string]string{
	"/h": "/help",
	"/m": "/memory",
	"/p": "/paste",
}

// resolveCommandAlias は / で始まるコマンドをエイリアス解決して返す。
// 未定義なら元の cmd を返す。
//
// 仕様:
// - 解決は最大 maxDepth 回まで（循環/多段の無限ループ防止）
// - config.command_aliases は default より優先
// - 大文字小文字は無視（内部で lower に正規化）
func resolveCommandAlias(cmd string) string {
	cmd = strings.ToLower(cmd)

	cfg := config.GetGlobalConfig()
	userAliases := map[string]string{}
	if cfg != nil && cfg.CommandAliases != nil {
		userAliases = cfg.CommandAliases
	}

	const maxDepth = 10
	cur := cmd
	for i := 0; i < maxDepth; i++ {
		next, ok := lookupAlias(cur, userAliases)
		if !ok {
			return cur
		}
		next = strings.ToLower(next)
		if next == cur {
			return cur
		}
		cur = next
	}

	// 深すぎる場合は安全のため元のコマンドを返す
	return cmd
}

func lookupAlias(cmd string, userAliases map[string]string) (string, bool) {
	if v, ok := userAliases[cmd]; ok {
		return v, true
	}
	if v, ok := defaultCommandAliases[cmd]; ok {
		return v, true
	}
	return "", false
}
