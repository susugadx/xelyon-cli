package llmcatalog

import "strings"

// BedrockModelFamily は Bedrock 上のモデル family を表す。
type BedrockModelFamily string

const (
	// BedrockModelFamilyClaude は Anthropic Claude on Bedrock の family。
	BedrockModelFamilyClaude BedrockModelFamily = "claude"
	// BedrockModelFamilyConverse は Converse API 経路で扱う Bedrock family。
	BedrockModelFamilyConverse BedrockModelFamily = "converse"
)

// BedrockModelFamilyFor は raw model と catalog_model から Bedrock の実行 family を返す。
func BedrockModelFamilyFor(model, catalogModel string) BedrockModelFamily {
	if IsBedrockClaudeModel(model) || IsBedrockClaudeModel(catalogModel) {
		return BedrockModelFamilyClaude
	}
	return BedrockModelFamilyConverse
}

// IsBedrockClaudeModel は model が Claude on Bedrock と見なせるか返す。
func IsBedrockClaudeModel(model string) bool {
	m := strings.ToLower(strings.TrimSpace(model))
	if m == "" {
		return false
	}
	return strings.HasPrefix(m, "claude") ||
		strings.Contains(m, ".claude") ||
		strings.Contains(m, "/claude")
}

// IsBedrockModelID は model が Bedrock の model ID / inference profile ID と見なせるか返す。
func IsBedrockModelID(model string) bool {
	m := strings.ToLower(strings.TrimSpace(model))
	if m == "" {
		return false
	}
	for _, prefix := range bedrockModelIDPrefixes {
		if strings.HasPrefix(m, prefix) {
			return true
		}
	}
	return false
}

// IsBedrockConverseModel は model が Converse API 経路の候補か返す。
func IsBedrockConverseModel(model, catalogModel string) bool {
	return BedrockModelFamilyFor(model, catalogModel) == BedrockModelFamilyConverse
}

var bedrockModelIDPrefixes = []string{
	"anthropic.claude",
	"global.anthropic.",
	"us.anthropic.",
	"eu.anthropic.",
	"apac.anthropic.",
	"amazon.nova",
	"global.amazon.nova",
	"us.amazon.nova",
	"eu.amazon.nova",
	"apac.amazon.nova",
	"meta.llama",
	"us.meta.llama",
	"mistral.",
	"cohere.",
	"ai21.",
	"writer.",
	"stability.",
}
