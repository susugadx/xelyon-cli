package prompt

import "strings"

// providerPrefixes はプロバイダー別のシステムプロンプトプレフィックス
// クリティカルルールを冒頭に注入してモデルの遵守率を上げる
var providerPrefixes = map[string]string{
	"gemini": `## ⚠️ ABSOLUTE RULES (NEVER SKIP)
1. **ALWAYS read_file BEFORE str_replace** - NO EXCEPTIONS
2. **NEVER guess file contents** - read first, edit second
3. If you haven't read it this session, you CANNOT edit it
4. **After running bash verification (test/build/lint), WAIT for the result** - do NOT declare completion until you see the output
5. **Tool calls must be actual JSON, NOT inside markdown code blocks** - ` + "```json...```" + ` is for display only

`,
}

// GetProviderPrefix はプロバイダー名に応じたプレフィックスを返す
// 未登録プロバイダーは空文字を返す
func GetProviderPrefix(provider string) string {
	return providerPrefixes[strings.ToLower(provider)]
}

// BuildProviderSystemPrompt はプロバイダー別プレフィックスをシステムプロンプトの冒頭に注入する
func BuildProviderSystemPrompt(base, providerName string) string {
	prefix := GetProviderPrefix(providerName)
	if prefix == "" {
		return base
	}
	return prefix + base
}
