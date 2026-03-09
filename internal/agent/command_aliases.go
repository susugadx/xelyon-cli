package agent

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
)

// defaultCommandAliases は組み込みのコマンドエイリアス。
// config.yaml の command_aliases で上書き・追加できる。
var defaultCommandAliases = map[string]string{
	"/h": "/help",
	"/p": "/paste",
}

func resolveCommandAliasWithConfig(cmd string, cfg *config.Config) string {
	cmd = strings.ToLower(cmd)

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
