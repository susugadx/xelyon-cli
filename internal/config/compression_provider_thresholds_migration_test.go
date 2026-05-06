package config

import "testing"

func TestLoadConfig_MigratesLegacyGeneratedProviderThresholdsToEmpty(t *testing.T) {
	cfg, err := loadConfigFromData([]byte(`
compression:
  provider_thresholds:
    gemini: 180000
    claude: 150000
    bedrock: 150000
    deepseek: 600000
    openai: 100000
    openai:gpt-5.4: 260000
    openai:gpt-5.4-pro: 260000
    openrouter: 120000
`))
	if err != nil {
		t.Fatalf("loadConfigFromData() error = %v", err)
	}
	if got := len(cfg.Compression.ProviderThresholds); got != 0 {
		t.Fatalf("Compression.ProviderThresholds len = %d, want 0", got)
	}
}

func TestLoadConfig_MigratesEarlierDeepSeek80000ProviderThreshold(t *testing.T) {
	cfg, err := loadConfigFromData([]byte(`
compression:
  provider_thresholds:
    deepseek: 80000
`))
	if err != nil {
		t.Fatalf("loadConfigFromData() error = %v", err)
	}
	if got := len(cfg.Compression.ProviderThresholds); got != 0 {
		t.Fatalf("Compression.ProviderThresholds len = %d, want 0", got)
	}
}

func TestLoadConfig_MigratesLegacyProviderThresholdsKeyByKey(t *testing.T) {
	cfg, err := loadConfigFromData([]byte(`
compression:
  provider_thresholds:
    gemini: 180000
    claude: 150000
    bedrock: 150000
    deepseek: 123456
    openai: 100000
    openai:gpt-5.4: 260000
    openai:gpt-5.4-pro: 260000
    openai:gpt-5.4-mini: 300000
    openrouter: 120000
`))
	if err != nil {
		t.Fatalf("loadConfigFromData() error = %v", err)
	}
	if got := cfg.Compression.ProviderThresholds["deepseek"]; got != 123456 {
		t.Fatalf("Compression.ProviderThresholds[deepseek] = %d, want 123456", got)
	}
	if got := cfg.Compression.ProviderThresholds["openai:gpt-5.4-mini"]; got != 300000 {
		t.Fatalf("Compression.ProviderThresholds[openai:gpt-5.4-mini] = %d, want 300000", got)
	}
	if _, ok := cfg.Compression.ProviderThresholds["gemini"]; ok {
		t.Fatal("Compression.ProviderThresholds[gemini] should be removed as a legacy generated default")
	}
	if _, ok := cfg.Compression.ProviderThresholds["openai"]; ok {
		t.Fatal("Compression.ProviderThresholds[openai] should be removed as a legacy generated default")
	}
}
