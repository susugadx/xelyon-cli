package openai

import (
	"crypto/sha256"
	"fmt"

	openairesponses "github.com/susugadx/xelyon-cli/internal/api/providers/openai_responses"
)

// BuildPromptCacheKey はプロジェクト・モデル・プロンプトに基づく動的キャッシュキーを生成する。
func BuildPromptCacheKey(model, systemPrompt string) string {
	return openairesponses.BuildPromptCacheKey(model, systemPrompt)
}

// buildPromptCacheKeyWithCwd はテスト用に cwd を引数で受け取るバージョン。
func buildPromptCacheKeyWithCwd(cwd, model, systemPrompt string) string {
	return openairesponses.BuildPromptCacheKeyWithCwd(cwd, model, systemPrompt)
}

// shortHash は入力文字列の SHA-256 先頭8文字を返す。
func shortHash(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h[:4])
}
