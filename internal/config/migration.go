package config

import "gopkg.in/yaml.v3"

// migrateOldKeys は旧設定キーを新キーに読み替える。
// ルール: 新キーが YAML に明示されていれば新キーを優先。旧キーは新キーが存在しない場合の補完にのみ使う。
func migrateOldKeys(data []byte, cfg *Config) {
	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return
	}

	// general.language → general.ui_language
	if general, ok := raw["general"].(map[string]interface{}); ok {
		if lang, ok := general["language"].(string); ok && lang != "" {
			if _, hasNewKey := general["ui_language"]; !hasNewKey {
				cfg.General.UILanguage = lang
			}
		}
	}

	// tool_confirm + bash → execution の補完（execution が未設定の場合のみ）
	if _, hasExecution := raw["execution"]; !hasExecution {
		// execution セクションがなければ旧設定から推定
		if tc, ok := raw["tool_confirm"].(map[string]interface{}); ok {
			if autoMedium, ok := tc["auto_approve_medium"].(bool); ok && autoMedium {
				// auto_approve_medium: true → trusted 相当
				cfg.Execution.Mode = string(ExecutionTrusted)
			}
		}
	}

	// compression.auto_compress → compression.enabled
	// compression.threshold_percent → compression.trigger_percent
	if comp, ok := raw["compression"].(map[string]interface{}); ok {
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

	if finalChecks, err := loadCompatibleFinalChecks(data); err == nil && finalChecks != nil {
		cfg.FinalChecks = *finalChecks
	}
}

// toInt は interface{} から int を取得する（YAML パーサーが返す型に対応）。
func toInt(v interface{}) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	}
	return 0
}
