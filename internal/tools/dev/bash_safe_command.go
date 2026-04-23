package dev

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
)

// verificationSafeCommands はビルド・テスト・検証・環境確認用の安全コマンド。
// コード探索系（grep, find, cat, head, tail, sed -n 等）は含めない。
// 探索は gather_context を第一選択にし、必要時のみ低レベル investigation tool を使う。
var verificationSafeCommands = map[string]bool{
	"pwd": true, "echo": true, "which": true, "env": true, "printenv": true,
	"diff": true, "file": true, "du": true, "stat": true,
	"md5sum": true, "sha256sum": true,
	"git status": true, "git log": true, "git diff": true, "git branch": true,
	"git ls-files": true, "git show": true, "git remote": true,
	"go version": true, "go build": true, "go test": true, "go vet": true,
	"go fmt": true, "go mod tidy": true, "go mod download": true,
	"go generate": true,
	"make":        true,
	"npm test":    true, "npm run": true,
	"cargo build": true, "cargo test": true, "cargo check": true, "cargo fmt": true,
	"pytest": true, "python -m pytest": true, "python -m unittest": true,
	"pip list": true,
	"node -v":  true, "npm -v": true,
	"python --version": true, "python3 --version": true,
	"rustc --version": true, "cargo --version": true,
	"java -version": true, "javac -version": true,
	"golangci-lint": true, "eslint": true, "prettier": true,
	"docker ps": true, "docker images": true,
}

// IsSafeCommand はコマンドが自動実行可能な安全コマンドかを判定する。
// verification 系のみ自動許可。discovery 系は safe_commands に明示追加されていても
// verification として扱う（探索用途の bypass にならない）。
// チェーンコマンド（&&, ||, ;）は全パーツが安全な場合のみ true を返す。
func IsSafeCommand(command string, cfg config.BashConfig) bool {
	parts := splitChainCommand(command)
	for _, part := range parts {
		if !isSingleCommandSafe(part, cfg) {
			return false
		}
	}
	return true
}

// splitChainCommand はコマンドを &&, ||, ; で分割する。
// パイプ | は分割しない（パイプチェーンは別途 dangerousPipePatterns でチェック）。
func splitChainCommand(command string) []string {
	var parts []string
	var current strings.Builder
	inSingle := false
	inDouble := false
	i := 0
	for i < len(command) {
		ch := command[i]
		if ch == '\'' && !inDouble {
			inSingle = !inSingle
			current.WriteByte(ch)
			i++
			continue
		}
		if ch == '"' && !inSingle {
			inDouble = !inDouble
			current.WriteByte(ch)
			i++
			continue
		}
		if inSingle || inDouble {
			current.WriteByte(ch)
			i++
			continue
		}
		if i+1 < len(command) && command[i:i+2] == "&&" {
			if part := strings.TrimSpace(current.String()); part != "" {
				parts = append(parts, part)
			}
			current.Reset()
			i += 2
			continue
		}
		if i+1 < len(command) && command[i:i+2] == "||" {
			if part := strings.TrimSpace(current.String()); part != "" {
				parts = append(parts, part)
			}
			current.Reset()
			i += 2
			continue
		}
		if ch == ';' {
			if part := strings.TrimSpace(current.String()); part != "" {
				parts = append(parts, part)
			}
			current.Reset()
			i++
			continue
		}
		current.WriteByte(ch)
		i++
	}
	if part := strings.TrimSpace(current.String()); part != "" {
		parts = append(parts, part)
	}
	return parts
}

// isSingleCommandSafe は単一コマンド（チェーンなし）が安全かチェックする。
// verification 系のみ true を返す。discovery 系は false。
func isSingleCommandSafe(command string, cfg config.BashConfig) bool {
	trimmed := strings.TrimSpace(command)

	for safe := range verificationSafeCommands {
		if matchCommandPrefix(trimmed, safe) {
			return true
		}
	}
	for _, safe := range cfg.SafeCommands {
		if matchCommandPrefix(trimmed, safe) {
			return true
		}
	}
	return false
}

// matchCommandPrefix はコマンドが安全なプレフィックスで始まるか判定する。
// プレフィックスの直後が空白・タブ・EOF のいずれかであることを確認する。
func matchCommandPrefix(command, prefix string) bool {
	if command == prefix {
		return true
	}
	if strings.HasPrefix(command, prefix) {
		next := command[len(prefix)]
		return next == ' ' || next == '\t'
	}
	return false
}

func isEvalInvocation(command string) bool {
	if !strings.HasPrefix(command, "eval") {
		return false
	}
	if len(command) == len("eval") {
		return true
	}
	next := command[len("eval")]
	isAlphaNumOrUnderscore := (next >= 'a' && next <= 'z') ||
		(next >= 'A' && next <= 'Z') ||
		(next >= '0' && next <= '9') ||
		next == '_'
	return !isAlphaNumOrUnderscore
}
