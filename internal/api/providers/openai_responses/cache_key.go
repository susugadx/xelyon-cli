package openairesponses

import openaicompat "github.com/susugadx/xelyon-cli/internal/api/providers/openai_compat"

// BuildPromptCacheKey はプロジェクト・モデル・プロンプトに基づく動的キャッシュキーを生成する。
func BuildPromptCacheKey(model, systemPrompt string) string {
	return openaicompat.BuildPromptCacheKey(model, systemPrompt)
}

// BuildPromptCacheKeyWithCwd はテスト用に cwd を引数で受け取るバージョン。
func BuildPromptCacheKeyWithCwd(cwd, model, systemPrompt string) string {
	return openaicompat.BuildPromptCacheKeyWithCwd(cwd, model, systemPrompt)
}
