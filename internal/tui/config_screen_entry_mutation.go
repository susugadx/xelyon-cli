package tui

import "github.com/susugadx/xelyon-cli/internal/config"

// applyEntryField は entry の1フィールドだけを Config にパッチ適用する。
func (cs *configScreen) applyEntryField(path, key string, ef structEntryField) {
	val, _ := config.GetFieldValue(cs.cfg, path)
	switch v := val.(type) {
	case map[string]config.ProviderModelConfig:
		_ = v
		cs.cfg.PatchProviderModelConfig(key, func(pm *config.ProviderModelConfig) {
			switch ef.Name {
			case "default_model":
				pm.DefaultModel, _ = ef.Value.(string)
			case "max_output_tokens":
				if n, ok := ef.Value.(int); ok {
					pm.MaxOutputTokens = n
				}
			}
		})
	case map[string]config.LSPServerConfig:
		ls := v[key]
		switch ef.Name {
		case "command":
			ls.Command, _ = ef.Value.(string)
		case "args":
			if s, ok := ef.Value.([]string); ok {
				ls.Args = s
			}
		case "disabled":
			ls.Disabled, _ = ef.Value.(bool)
		}
		v[key] = ls
	}
}

// applyEntryFieldAndMark は entry field の変更を Config に書き戻し dirty をマークする。
func (cs *configScreen) applyEntryFieldAndMark(ef *structEntryField) {
	field := cs.selectedField()
	if field == nil {
		return
	}
	cs.applyEntryField(field.Path, cs.editEntryKey, *ef)
	cs.markModified()
}
