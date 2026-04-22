package config

func applyGeneralLanguageMigration(raw map[string]interface{}, cfg *Config) {
	general := migrationSection(raw, "general")
	if general == nil {
		return
	}
	lang, ok := general["language"].(string)
	if !ok || lang == "" {
		return
	}
	if _, hasNewKey := general["ui_language"]; hasNewKey {
		return
	}
	cfg.General.UILanguage = lang
}

func applyExecutionModeMigration(raw map[string]interface{}, cfg *Config) {
	if _, hasExecution := raw["execution"]; hasExecution {
		return
	}
	toolConfirm := migrationSection(raw, "tool_confirm")
	if toolConfirm == nil {
		return
	}
	if autoMedium, ok := toolConfirm["auto_approve_medium"].(bool); ok && autoMedium {
		cfg.Execution.Mode = string(ExecutionTrusted)
	}
}

func applyCompressionKeyMigration(raw map[string]interface{}, cfg *Config) {
	comp := migrationSection(raw, "compression")
	if comp == nil {
		return
	}
	if autoCompress, ok := comp["auto_compress"].(bool); ok {
		if _, hasNewKey := comp["enabled"]; !hasNewKey {
			cfg.Compression.Enabled = autoCompress
		}
	}
	if thresholdPercent, ok := comp["threshold_percent"]; ok {
		if pct := toInt(thresholdPercent); pct > 0 {
			if _, hasNewKey := comp["trigger_percent"]; !hasNewKey {
				cfg.Compression.TriggerPercent = pct
			}
		}
	}
}

func migrationSection(raw map[string]interface{}, key string) map[string]interface{} {
	section, _ := raw[key].(map[string]interface{})
	return section
}
