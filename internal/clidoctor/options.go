package clidoctor

import (
	"io"
	"time"
)

// DefaultTimeout は doctor live smoke request の既定 timeout。
const DefaultTimeout = 120 * time.Second

// CommonOptions は provider doctor で共有する CLI option。
type CommonOptions struct {
	CatalogModel         string
	Smoke                bool
	ToolSmoke            bool
	Capabilities         bool
	RequiredCapabilities []string
	Timeout              time.Duration
	JSON                 bool
	PrintRequest         bool
}

// AzureOptions は Azure OpenAI doctor の CLI option。
type AzureOptions struct {
	CommonOptions
	Deployment     string
	RetentionSmoke bool
	PrintConfig    bool
}

// BedrockOptions は Bedrock doctor の CLI option。
type BedrockOptions struct {
	CommonOptions
	Model         string
	ImageSmoke    bool
	ThinkingSmoke bool
}

// ClaudeOptions は Claude doctor の CLI option。
type ClaudeOptions struct {
	CommonOptions
	Model          string
	ImageSmoke     bool
	ThinkingSmoke  bool
	WebSearchSmoke bool
}

// DeepSeekOptions は DeepSeek doctor の CLI option。
type DeepSeekOptions struct {
	CommonOptions
	Model string
}

// GeminiOptions は Gemini doctor の CLI option。
type GeminiOptions struct {
	CommonOptions
	Model          string
	ImageSmoke     bool
	WebSearchSmoke bool
}

// GroqOptions は Groq doctor の CLI option。
type GroqOptions struct {
	CommonOptions
	Model string
}

// KimiOptions は Kimi doctor の CLI option。
type KimiOptions struct {
	CommonOptions
	Model          string
	ModelChanged   bool
	ImageSmoke     bool
	WebSearchSmoke bool
}

// MCPOptions は MCP doctor の CLI option。
type MCPOptions struct {
	JSON         bool
	Connect      bool
	Server       string
	IncludeTools bool
}

// OllamaOptions は Ollama doctor の CLI option。
type OllamaOptions struct {
	CommonOptions
	Model string
}

// OpenAIOptions は OpenAI doctor の CLI option。
type OpenAIOptions struct {
	CommonOptions
	Model          string
	RetentionSmoke bool
}

// OpenAISubscriptionOptions は OpenAI Subscription doctor の CLI option。
type OpenAISubscriptionOptions struct {
	CommonOptions
	Model          string
	RetentionSmoke bool
	CacheSmoke     bool
	CompactSmoke   bool
	ThinkingSmoke  bool
	SmokeOutput    io.Writer
}

// OpenRouterOptions は OpenRouter doctor の CLI option。
type OpenRouterOptions struct {
	CommonOptions
	Model string
}
