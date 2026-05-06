package config

var legacyGeneratedCompressionProviderThresholds = map[string][]int{
	"gemini":             {180000},
	"claude":             {150000},
	"bedrock":            {150000},
	"deepseek":           {80000, 600000},
	"openai":             {100000},
	"openai:gpt-5.4":     {260000},
	"openai:gpt-5.4-pro": {260000},
	"openrouter":         {120000},
}

func applyLegacyCompressionProviderThresholdsMigration(raw map[string]interface{}, cfg *Config) {
	if cfg == nil {
		return
	}
	comp := migrationSection(raw, "compression")
	if comp == nil {
		return
	}
	rawThresholds, ok := comp["provider_thresholds"]
	if !ok {
		return
	}
	thresholds, ok := rawThresholds.(map[string]interface{})
	if !ok {
		return
	}

	for key, legacyValues := range legacyGeneratedCompressionProviderThresholds {
		if got := toInt(thresholds[key]); !matchesLegacyGeneratedCompressionProviderThreshold(got, legacyValues) {
			continue
		}
		delete(cfg.Compression.ProviderThresholds, key)
	}

	if len(cfg.Compression.ProviderThresholds) == 0 {
		cfg.Compression.ProviderThresholds = map[string]int{}
	}
}

func matchesLegacyGeneratedCompressionProviderThreshold(got int, legacyValues []int) bool {
	for _, want := range legacyValues {
		if got == want {
			return true
		}
	}
	return false
}
