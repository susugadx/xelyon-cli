package configscreen

import "github.com/susugadx/xelyon-cli/internal/config"

func (cs *Screen) markModified() {
	cs.dirty = true
	cs.saveStatus = statusModified
	cs.refreshCategories()
}

func (cs *Screen) applyFieldValue(path string, value interface{}, providerConfigKey string) bool {
	candidate := config.CloneConfig(cs.cfg)
	if err := config.SetFieldValue(candidate, path, value); err != nil {
		return false
	}
	if path == "default_model" {
		syncEditedProviderDefaultModel(candidate, providerConfigKey, cs.initialDefaultProvider)
	}
	if !cs.validateConfigEditCandidate(candidate, path, providerConfigKey) {
		return false
	}
	cs.cfg = candidate
	cs.markModified()
	return true
}

func (cs *Screen) resetSelectedFieldToDefault(providerConfigKey string) bool {
	field := cs.selectedField()
	if field == nil || field.Default == nil {
		return false
	}
	return cs.applyFieldValue(field.Path, field.Default, providerConfigKey)
}

// syncEditedProviderDefaultModel は global default_model を選択中 provider override に同期する。
func (cs *Screen) syncEditedProviderDefaultModel(providerConfigKey string) {
	if cs == nil || cs.cfg == nil {
		return
	}
	syncEditedProviderDefaultModel(cs.cfg, providerConfigKey, cs.initialDefaultProvider)
}

// SyncEditedProviderDefaultModel は global default_model を選択中 provider override に同期する。
func (cs *Screen) SyncEditedProviderDefaultModel(providerConfigKey string) {
	cs.syncEditedProviderDefaultModel(providerConfigKey)
}

func syncEditedProviderDefaultModel(cfg *config.Config, providerConfigKey, initialDefaultProvider string) {
	if cfg == nil {
		return
	}
	provName := cfg.DefaultModelSyncProviderKey(providerConfigKey, initialDefaultProvider)
	if provName != "" {
		cfg.SyncProviderDefaultModel(provName, cfg.DefaultModel)
	}
}

// defaultModelSyncProvider は global default_model を同期する provider_models key を返す。
func (cs *Screen) defaultModelSyncProvider(providerConfigKey string) string {
	if cs == nil || cs.cfg == nil {
		return ""
	}
	return cs.cfg.DefaultModelSyncProviderKey(providerConfigKey, cs.initialDefaultProvider)
}

// DefaultModelSyncProvider は default_model 同期先の provider_models key を返す。
func (cs *Screen) DefaultModelSyncProvider(providerConfigKey string) string {
	return cs.defaultModelSyncProvider(providerConfigKey)
}
