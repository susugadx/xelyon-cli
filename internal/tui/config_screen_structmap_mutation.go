package tui

import (
	"sort"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func (cs *configScreen) structMapKeys(path string) []string {
	var keys []string
	val, _ := config.GetFieldValue(cs.cfg, path)
	switch v := val.(type) {
	case map[string]config.ProviderModelConfig:
		for k := range v {
			keys = append(keys, k)
		}
	case map[string]config.LSPServerConfig:
		for k := range v {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	return keys
}

// deleteStructMapKey は structmap のキーを削除する。
func (cs *configScreen) deleteStructMapKey(path, key string) {
	val, _ := config.GetFieldValue(cs.cfg, path)
	switch v := val.(type) {
	case map[string]config.ProviderModelConfig:
		cs.cfg.DeleteProviderModelConfig(key)
	case map[string]config.LSPServerConfig:
		delete(v, key)
	}
}

// addStructMapKey は structmap に空のキーを追加する。
func (cs *configScreen) addStructMapKey(path, key string) bool {
	cs.ensureStructMapInitialized(path)
	if path == "provider_models" {
		providerModels := cs.cfg.ProviderModelsForEdit()
		if providerModels == nil {
			providerModels = map[string]config.ProviderModelConfig{}
		}
		if _, ok := providerModels[key]; ok {
			return false
		}
		providerModels[key] = config.ProviderModelConfig{}
		cs.cfg.SetProviderModelsForEdit(providerModels)
		return true
	}

	val, _ := config.GetFieldValue(cs.cfg, path)
	switch v := val.(type) {
	case map[string]config.LSPServerConfig:
		if _, ok := v[key]; ok {
			return false
		}
		v[key] = config.LSPServerConfig{}
		return true
	}
	return false
}

// ensureStructMapInitialized は TUI が直接 mutate する struct map を必要に応じて初期化する。
func (cs *configScreen) ensureStructMapInitialized(path string) {
	if cs == nil {
		return
	}

	switch path {
	case "lsp.servers":
		if cs.cfg.LSP.Servers == nil {
			cs.cfg.LSP.Servers = make(map[string]config.LSPServerConfig)
		}
	}
}
