package config

// migrateOldKeys は旧設定キーを新キーに読み替える。
// ルール: 新キーが YAML に明示されていれば新キーを優先。旧キーは新キーが存在しない場合の補完にのみ使う。
func migrateOldKeys(data []byte, cfg *Config) {
	migrateOldKeysFromRaw(data, parseYAMLRootMap(data), cfg)
}

func migrateOldKeysFromRaw(data []byte, raw map[string]interface{}, cfg *Config) {
	if raw == nil {
		return
	}
	applyGeneralLanguageMigration(raw, cfg)
	applyExecutionModeMigration(raw, cfg)
	applyCompressionKeyMigration(raw, cfg)
	applyLegacyCompressionProviderThresholdsMigration(raw, cfg)

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
