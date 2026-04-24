package mcp

import (
	"fmt"
	"os"
	"strings"
)

// 安全なMCPコマンドのホワイトリスト
var allowedMCPCommands = map[string]bool{
	"npx":     true,
	"node":    true,
	"python":  true,
	"python3": true,
	"uvx":     true,
	"docker":  true,
}

// validateMCPCommand はMCPコマンドの安全性を検証
func validateMCPCommand(command string) error {
	if command == "" {
		return fmt.Errorf("empty command")
	}

	// ホワイトリストチェック
	if !allowedMCPCommands[command] {
		return fmt.Errorf("command '%s' is not in the allowed list. Allowed: npx, node, python, python3, uvx, docker", command)
	}

	// パストラバーサルチェック
	if strings.Contains(command, "..") || strings.Contains(command, "/") {
		return fmt.Errorf("command contains path traversal characters")
	}

	return nil
}

// sanitizeEnv は環境変数を構築する
// システム環境変数から安全なものをコピーし、customEnvの値をすべて追加する
// customEnvの値は ${VAR} 形式で環境変数を参照できる
func sanitizeEnv(customEnv map[string]string) []string {
	// 安全な環境変数のホワイトリスト
	safeEnvKeys := map[string]bool{
		"PATH":         true,
		"HOME":         true,
		"USER":         true,
		"LANG":         true,
		"LC_ALL":       true,
		"NODE_OPTIONS": true,
		"PYTHONPATH":   true,
	}

	env := []string{}

	// システム環境変数から安全なもののみコピー
	for _, e := range os.Environ() {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 && safeEnvKeys[parts[0]] {
			env = append(env, e)
		}
	}

	// カスタム環境変数をすべて追加（${VAR} を展開）
	for k, v := range customEnv {
		expandedValue := os.ExpandEnv(v)
		env = append(env, fmt.Sprintf("%s=%s", k, expandedValue))
	}

	return env
}
