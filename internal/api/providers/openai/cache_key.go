package openai

import (
	"crypto/sha256"
	"fmt"
	"os"
)

// BuildPromptCacheKey はプロジェクト・モデル・プロンプトに基づく動的キャッシュキーを生成する。
// フォーマット: "xelyon:{cwd_hash}:{model}:{prompt_hash}"
//
// cwd_hash: cwd の SHA-256 先頭8文字
// prompt_hash: systemPrompt の先頭100文字の SHA-256 先頭8文字
func BuildPromptCacheKey(model, systemPrompt string) string {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "unknown"
	}
	return buildPromptCacheKeyWithCwd(cwd, model, systemPrompt)
}

// buildPromptCacheKeyWithCwd はテスト用に cwd を引数で受け取るバージョン。
func buildPromptCacheKeyWithCwd(cwd, model, systemPrompt string) string {
	cwdHash := shortHash(cwd)

	promptPrefix := systemPrompt
	if len(promptPrefix) > 100 {
		promptPrefix = promptPrefix[:100]
	}
	promptHash := shortHash(promptPrefix)

	return fmt.Sprintf("xelyon:%s:%s:%s", cwdHash, model, promptHash)
}

// shortHash は入力文字列の SHA-256 先頭8文字を返す。
func shortHash(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h[:4])
}
