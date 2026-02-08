package agent

import "strings"

// providerPrefixes はプロバイダー別のシステムプロンプトプレフィックス
// クリティカルルールを冒頭に注入してモデルの遵守率を上げる
var providerPrefixes = map[string]string{
	"gemini": `## ⚠️ ABSOLUTE RULES (NEVER SKIP)
1. **ALWAYS read_file BEFORE str_replace** - NO EXCEPTIONS
2. **NEVER guess file contents** - read first, edit second
3. If you haven't read it this session, you CANNOT edit it

`,
}

// getProviderPrefix はプロバイダー名に応じたプレフィックスを返す
// 未登録プロバイダーは空文字を返す
func getProviderPrefix(provider string) string {
	return providerPrefixes[strings.ToLower(provider)]
}

// BuildProviderSystemPrompt はプロバイダー別プレフィックスをシステムプロンプトの冒頭に注入する
func BuildProviderSystemPrompt(base, providerName string) string {
	prefix := getProviderPrefix(providerName)
	if prefix == "" {
		return base
	}
	return prefix + base
}
