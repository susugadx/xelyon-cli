package tools

import "strings"

// readOnlyCommands は bash 実行後にキャッシュクリアが不要なコマンドのプレフィックス。
// ファイルを変更しないコマンドのみ（go mod tidy 等は除外）。
// NOTE: auto-approve 判定とは独立。discovery 系もキャッシュ判定では read-only 扱い。
var readOnlyCommands = []string{
	"ls", "cat", "pwd", "echo", "which",
	"head", "tail", "wc", "grep", "rg", "find",
	"sed -n", "diff", "file", "du", "stat",
	"md5sum", "sha256sum",
	"git status", "git log", "git diff", "git branch",
	"git ls-files", "git show", "git remote",
	"go version", "go test", "go vet", "go env",
	"node -v", "npm -v", "npm list",
	"python --version", "pip list",
	"env", "printenv", "date", "uname", "whoami", "id",
	"tree", "less", "more", "sort", "uniq", "cut", "tr",
}

// isBashReadOnly はコマンドが read-only（キャッシュクリア不要）かどうかを判定する。
// パイプ、リダイレクト、連結演算子を含む場合は安全側に倒して false を返す。
func isBashReadOnly(command string) bool {
	trimmed := strings.TrimSpace(command)
	if trimmed == "" {
		return true
	}

	// パイプ・リダイレクト・連結を含む場合は unsafe
	for _, ch := range []string{"|", ">", ">>", "&&", ";"} {
		if strings.Contains(trimmed, ch) {
			return false
		}
	}

	// 先頭コマンドが read-only リストにマッチするか
	for _, prefix := range readOnlyCommands {
		if trimmed == prefix {
			return true
		}
		if strings.HasPrefix(trimmed, prefix) && len(trimmed) > len(prefix) {
			next := trimmed[len(prefix)]
			if next == ' ' || next == '\t' {
				return true
			}
		}
	}

	return false
}

// IsReadOnlyBashCommand は bash コマンドが read-only かどうかを返す。
func IsReadOnlyBashCommand(command string) bool {
	return isBashReadOnly(command)
}
