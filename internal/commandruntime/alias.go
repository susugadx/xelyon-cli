package commandruntime

import "strings"

// DefaultAliases は組み込み slash command alias を返す。
func DefaultAliases() map[string]string {
	return map[string]string{
		"/h": "/help",
	}
}

// ResolveAlias は user aliases と組み込み aliases を使って command 名を正規化する。
func ResolveAlias(cmd string, userAliases map[string]string) string {
	cmd = strings.ToLower(cmd)
	builtins := DefaultAliases()

	const maxDepth = 10
	cur := cmd
	for i := 0; i < maxDepth; i++ {
		next, ok := lookupAlias(cur, userAliases, builtins)
		if !ok {
			return cur
		}
		next = strings.ToLower(next)
		if next == cur {
			return cur
		}
		cur = next
	}

	return cmd
}

func lookupAlias(cmd string, userAliases, builtins map[string]string) (string, bool) {
	if v, ok := userAliases[cmd]; ok {
		return v, true
	}
	if v, ok := builtins[cmd]; ok {
		return v, true
	}
	return "", false
}
