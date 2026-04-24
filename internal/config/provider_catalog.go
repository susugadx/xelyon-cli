package config

import (
	"os"
	"strings"
)

// ProviderCatalogEntry は provider ごとの運用メタ情報を表す。
type ProviderCatalogEntry struct {
	APIKeyEnv            string
	SetupInstructions    []string
	DefaultSubAgentModel string
	SupportsImages       bool
}

var providerCatalog = map[string]ProviderCatalogEntry{
	"deepseek": {
		APIKeyEnv:            "DEEPSEEK_API_KEY",
		SetupInstructions:    []string{"export DEEPSEEK_API_KEY=your-api-key"},
		DefaultSubAgentModel: "deepseek-chat",
		SupportsImages:       false,
	},
	"openai": {
		APIKeyEnv:            "OPENAI_API_KEY",
		SetupInstructions:    []string{"export OPENAI_API_KEY=your-api-key"},
		DefaultSubAgentModel: "gpt-5.4-mini",
		SupportsImages:       true,
	},
	"claude": {
		APIKeyEnv:            "ANTHROPIC_API_KEY",
		SetupInstructions:    []string{"export ANTHROPIC_API_KEY=your-api-key"},
		DefaultSubAgentModel: "claude-haiku-4-5-20251001",
		SupportsImages:       true,
	},
	"gemini": {
		APIKeyEnv:            "GEMINI_API_KEY",
		SetupInstructions:    []string{"export GEMINI_API_KEY=your-api-key"},
		DefaultSubAgentModel: "gemini-3.1-flash-lite-preview",
		SupportsImages:       true,
	},
	"groq": {
		APIKeyEnv:            "GROQ_API_KEY",
		SetupInstructions:    []string{"export GROQ_API_KEY=your-api-key"},
		DefaultSubAgentModel: "llama-3.3-70b-versatile",
		SupportsImages:       false,
	},
	"openrouter": {
		APIKeyEnv:            "OPENROUTER_API_KEY",
		SetupInstructions:    []string{"export OPENROUTER_API_KEY=your-api-key"},
		DefaultSubAgentModel: "openai/gpt-5.4-mini",
		SupportsImages:       true,
	},
	"ollama": {
		SetupInstructions: []string{"ローカルの Ollama サーバーを起動してください"},
		SupportsImages:    false,
	},
	"bedrock": {
		SetupInstructions: []string{"AWS認証チェーン（IAMロール、環境変数、~/.aws/credentials等）を設定"},
		SupportsImages:    true,
	},
}

// ProviderCatalogEntryFor は provider 名に対応するメタ情報を返す。
func ProviderCatalogEntryFor(name string) (ProviderCatalogEntry, bool) {
	key := CanonicalProviderName(name)
	entry, ok := providerCatalog[key]
	return entry, ok
}

// ProviderAPIKeyEnv は provider に対応する API キー環境変数名を返す。
func ProviderAPIKeyEnv(name string) string {
	entry, ok := ProviderCatalogEntryFor(name)
	if !ok {
		return ""
	}
	return entry.APIKeyEnv
}

// ProviderHasAvailableCredential は provider の認証情報が利用可能かを返す。
func ProviderHasAvailableCredential(name string) bool {
	entry, ok := ProviderCatalogEntryFor(name)
	if !ok {
		return false
	}
	if entry.APIKeyEnv == "" {
		return true
	}
	return strings.TrimSpace(os.Getenv(entry.APIKeyEnv)) != ""
}

// ProviderSetupInstructions は provider の設定手順を返す。
func ProviderSetupInstructions(name string) []string {
	entry, ok := ProviderCatalogEntryFor(name)
	if !ok || len(entry.SetupInstructions) == 0 {
		return nil
	}
	clone := make([]string, len(entry.SetupInstructions))
	copy(clone, entry.SetupInstructions)
	return clone
}

// ProviderDefaultSubAgentModel は provider に紐づく既定 sub-agent model を返す。
func ProviderDefaultSubAgentModel(name string) string {
	entry, ok := ProviderCatalogEntryFor(name)
	if !ok {
		return ""
	}
	return strings.TrimSpace(entry.DefaultSubAgentModel)
}

// ProviderSupportsImages は provider が画像入力をサポートするか返す。
func ProviderSupportsImages(name string) bool {
	entry, ok := ProviderCatalogEntryFor(name)
	return ok && entry.SupportsImages
}
