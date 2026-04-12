package tui

import "github.com/susugadx/xelyon-cli/internal/config"

func (cs *configScreen) markModified() {
	cs.dirty = true
	cs.saveStatus = statusModified
	cs.refreshCategories()
}

func (cs *configScreen) applyFieldValue(path string, value interface{}, providerConfigKey string) bool {
	if err := config.SetFieldValue(cs.cfg, path, value); err != nil {
		return false
	}
	if path == "default_model" {
		cs.syncEditedProviderDefaultModel(providerConfigKey)
	}
	cs.markModified()
	return true
}

func (cs *configScreen) resetSelectedFieldToDefault(providerConfigKey string) bool {
	field := cs.selectedField()
	if field == nil || field.Default == nil {
		return false
	}
	return cs.applyFieldValue(field.Path, field.Default, providerConfigKey)
}

// syncEditedProviderDefaultModel は global default_model を選択中 provider override に同期する。
func (cs *configScreen) syncEditedProviderDefaultModel(providerConfigKey string) {
	if cs == nil || cs.cfg == nil {
		return
	}

	provName := cs.defaultModelSyncProvider(providerConfigKey)
	if provName == "" {
		return
	}
	cs.cfg.SyncProviderDefaultModel(provName, cs.cfg.DefaultModel)
}

// defaultModelSyncProvider は global default_model を同期する provider_models key を返す。
func (cs *configScreen) defaultModelSyncProvider(providerConfigKey string) string {
	if cs == nil || cs.cfg == nil {
		return ""
	}
	return cs.cfg.DefaultModelSyncProviderKey(providerConfigKey, cs.initialDefaultProvider)
}
