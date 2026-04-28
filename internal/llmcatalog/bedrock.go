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

func knownBedrockMaxOutputTokens(model string) (int, bool) {
	model = trimBedrockInferenceProfilePrefix(model)
	for _, rule := range bedrockMaxOutputTokenPrefixes {
		if strings.HasPrefix(model, rule.Pattern) {
			return rule.Limit, true
		}
	}
	return 0, false
}

func trimBedrockInferenceProfilePrefix(model string) string {
	model = strings.ToLower(strings.TrimSpace(model))
	for _, prefix := range []string{"global.", "us.", "eu.", "apac."} {
		if strings.HasPrefix(model, prefix) {
			return strings.TrimPrefix(model, prefix)
		}
	}
	return model
}

var bedrockMaxOutputTokenPrefixes = []ModelLimit{
	{Pattern: "amazon.nova-2", Limit: 64000},
	{Pattern: "amazon.nova-premier-v1", Limit: 25000},
	{Pattern: "amazon.nova-pro-v1", Limit: 5000},
	{Pattern: "amazon.nova-lite-v1", Limit: 5000},
	{Pattern: "amazon.nova-micro-v1", Limit: 5000},
	{Pattern: "meta.llama4", Limit: 8000},
	{Pattern: "meta.llama", Limit: 4000},
	{Pattern: "mistral.magistral-small", Limit: 40000},
	{Pattern: "mistral.pixtral-large", Limit: 16000},
	{Pattern: "mistral.ministral-14b", Limit: 8000},
	{Pattern: "mistral.", Limit: 4000},
	{Pattern: "cohere.command-r", Limit: 4000},
	{Pattern: "ai21.jamba-1-5", Limit: 4000},
	{Pattern: "writer.palmyra-x5", Limit: 8000},
	{Pattern: "deepseek.r1", Limit: 8000},
	{Pattern: "qwen.qwen3-coder", Limit: 16000},
	{Pattern: "qwen.qwen3", Limit: 8000},
	{Pattern: "minimax.minimax-m2", Limit: 8000},
	{Pattern: "nvidia.nemotron-nano-3-30b", Limit: 8000},
	{Pattern: "zai.glm-4-7", Limit: 4000},
	{Pattern: "openai.gpt-oss", Limit: 16000},
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
	"deepseek.",
	"qwen.",
	"minimax.",
	"nvidia.",
	"zai.",
	"openai.gpt-oss",
}
