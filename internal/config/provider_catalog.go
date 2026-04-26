package config

import (
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
)

// ProviderCatalogEntry は provider ごとの運用メタ情報を表す。
type ProviderCatalogEntry struct {
	APIKeyEnv            string
	SetupInstructions    []string
	DefaultSubAgentModel string
	SupportsImages       bool
}

// ProviderCatalogEntryFor は provider 名に対応するメタ情報を返す。
func ProviderCatalogEntryFor(name string) (ProviderCatalogEntry, bool) {
	entry, ok := llmcatalog.ProviderDescriptorFor(name)
	if !ok {
		return ProviderCatalogEntry{}, false
	}
	return ProviderCatalogEntry{
		APIKeyEnv:            entry.APIKeyEnv,
		SetupInstructions:    entry.SetupInstructions,
		DefaultSubAgentModel: entry.DefaultSubAgentModel,
		SupportsImages:       entry.SupportsImages,
	}, true
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
