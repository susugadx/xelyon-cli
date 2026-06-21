package configscreen

import "github.com/susugadx/xelyon-cli/internal/config"

// providerModelEntryFields は ProviderModelConfig の user-facing フィールドを返す。
func providerModelEntryFields(pm config.ProviderModelConfig) []structEntryField {
	return []structEntryField{
		{Name: "default_model", Type: "string", Value: pm.DefaultModel},
		{Name: "max_output_tokens", Type: "int", Value: pm.MaxOutputTokens},
		{Name: "catalog_model", Type: "string", Value: pm.CatalogModel},
	}
}

// lspServerEntryFields は LSPServerConfig の user-facing フィールドを返す。
func lspServerEntryFields(ls config.LSPServerConfig) []structEntryField {
	return []structEntryField{
		{Name: "command", Type: "string", Value: ls.Command},
		{Name: "args", Type: "[]string", Value: ls.Args},
		{Name: "disabled", Type: "bool", Value: ls.Disabled},
	}
}

// loadEntryFields は path と key から entry フィールドを読み込む。
func (cs *Screen) loadEntryFields(path, key string) []structEntryField {
	val, _ := config.GetFieldValue(cs.cfg, path)
	switch v := val.(type) {
	case map[string]config.ProviderModelConfig:
		if pm, ok := v[key]; ok {
			return providerModelEntryFields(pm)
		}
	case map[string]config.LSPServerConfig:
		if ls, ok := v[key]; ok {
			return lspServerEntryFields(ls)
		}
	}
	return nil
}
