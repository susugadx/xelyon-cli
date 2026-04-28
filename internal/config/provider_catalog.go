package config

import (
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
)

// ProviderCatalogEntry は provider ごとの運用メタ情報を表す。
type ProviderCatalogEntry struct {
	APIKeyEnv            string
	CredentialEnvVars    []string
	CredentialEnvVarSets [][]string
	SetupInstructions    []string
	DefaultSubAgentModel string
	SupportsImages       bool
	SupportsResponsesAPI bool
}

// ProviderCatalogEntryFor は provider 名に対応するメタ情報を返す。
func ProviderCatalogEntryFor(name string) (ProviderCatalogEntry, bool) {
	entry, ok := llmcatalog.ProviderDescriptorFor(name)
	if !ok {
		return ProviderCatalogEntry{}, false
	}
	return ProviderCatalogEntry{
		APIKeyEnv:            entry.APIKeyEnv,
		CredentialEnvVars:    entry.RequiredCredentialEnvVars(),
		CredentialEnvVarSets: entry.CredentialEnvVarSetsForAvailability(),
		SetupInstructions:    entry.SetupInstructions,
		DefaultSubAgentModel: entry.DefaultSubAgentModel,
		SupportsImages:       entry.SupportsImages,
		SupportsResponsesAPI: entry.SupportsResponsesAPI,
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

// ProviderCredentialEnvVars は provider の認証案内に表示する環境変数名を返す。
func ProviderCredentialEnvVars(name string) []string {
	entry, ok := ProviderCatalogEntryFor(name)
	if !ok {
		return nil
	}
	return append([]string(nil), entry.CredentialEnvVars...)
}

// ProviderHasAvailableCredential は provider の認証情報が利用可能かを返す。
func ProviderHasAvailableCredential(name string) bool {
	entry, ok := ProviderCatalogEntryFor(name)
	if !ok {
		return false
	}
	if len(entry.CredentialEnvVarSets) > 0 {
		for _, envSet := range entry.CredentialEnvVarSets {
			if credentialEnvSetAvailable(envSet) {
				return true
			}
		}
		return false
	}
	if entry.APIKeyEnv == "" {
		return true
	}
	return strings.TrimSpace(os.Getenv(entry.APIKeyEnv)) != ""
}

func credentialEnvSetAvailable(envSet []string) bool {
	if len(envSet) == 0 {
		return false
	}
	for _, envName := range envSet {
		if strings.TrimSpace(os.Getenv(envName)) == "" {
			return false
		}
	}
	return true
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

// ProviderSupportsResponsesAPI は provider が Responses API 形の実行経路を持つか返す。
func ProviderSupportsResponsesAPI(name string) bool {
	entry, ok := ProviderCatalogEntryFor(name)
	return ok && entry.SupportsResponsesAPI
}
