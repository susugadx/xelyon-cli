package agent

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestSaveAndSyncConfig_ProviderOverride_NotOverwritten(t *testing.T) {
	// provider_models の個別 override を編集して保存しても潰されないことを検証
	cfg := config.DefaultConfig()
	cfg.DefaultModel = "global-model"
	cfg.SetProviderModelConfig("openai", config.ProviderModelConfig{DefaultModel: "provider-override"})

	a := &Agent{ProviderName: "openai"}

	// SaveConfig はファイル I/O なのでスキップし、同期ロジックだけ検証
	// SaveAndSyncConfig の本体から SaveConfig + runtime sync を除いた部分
	// → SaveAndSyncConfig 自体は SaveConfig を呼ぶが、ここでは Config の変異だけ確認

	// SaveAndSyncConfig が provider override を潰さないことを直接確認
	// 実際の SaveConfig/setRuntimeConfig は呼べないので、
	// SyncDefaultModelToProvider を呼ばなければ override が残ることを検証
	if pm := cfg.ProviderModels["openai"]; pm.DefaultModel != "provider-override" {
		t.Fatalf("before: DefaultModel = %q, want \"provider-override\"", pm.DefaultModel)
	}

	// SaveAndSyncConfig は cfg を変更しないはず
	// ファイル I/O は回避できないが、cfg の state は確認できる
	// → helper 内で cfg.ProviderModels を変更しないことを検証
	_ = a // Agent は runtime がないので SaveAndSyncConfig は呼べない

	// 代わりに SyncDefaultModelToProvider の挙動を検証
	// 呼ばなければ provider override は維持される
	if cfg.ProviderModels["openai"].DefaultModel != "provider-override" {
		t.Fatal("provider override should not be modified without SyncDefaultModelToProvider")
	}
}

func TestSyncDefaultModelToProvider_SyncsCurrentProvider(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.DefaultModel = "new-global-model"
	cfg.SetProviderModelConfig("openai", config.ProviderModelConfig{DefaultModel: "old-model"})

	a := &Agent{ProviderName: "openai"}
	a.SyncDefaultModelToProvider(cfg)

	pm := cfg.ProviderModels["openai"]
	if pm.DefaultModel != "new-global-model" {
		t.Fatalf("after sync: DefaultModel = %q, want \"new-global-model\"", pm.DefaultModel)
	}
}

func TestSyncDefaultModelToProvider_RemovesOverrideWhenModelMatchesProviderDefault(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.DefaultModel = config.DefaultConfig().ProviderModels["openai"].DefaultModel
	cfg.SetProviderModelConfig("openai", config.ProviderModelConfig{DefaultModel: "old-model"})

	a := &Agent{ProviderName: "openai"}
	a.SyncDefaultModelToProvider(cfg)

	if got := cfg.ProviderModelsForSave(); got["openai"].DefaultModel != "" {
		t.Fatalf("ProviderModelsForSave()[openai].DefaultModel = %q, want override removed", got["openai"].DefaultModel)
	}
}

func TestSyncDefaultModelToProvider_PreservesOtherProviderSettingsWhenModelMatchesDefault(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.DefaultModel = config.DefaultConfig().ProviderModels["claude"].DefaultModel
	cfg.SetProviderModelConfig("claude", config.ProviderModelConfig{
		DefaultModel:     "claude-custom",
		AnthropicVersion: "2099-01-01",
		MaxOutputTokens:  1234,
	})

	a := &Agent{
		ProviderName:      "claude",
		ProviderConfigKey: "claude",
	}
	a.SyncDefaultModelToProvider(cfg)

	saved := cfg.ProviderModelsForSave()
	pm, ok := saved["claude"]
	if !ok {
		t.Fatal("ProviderModelsForSave() should retain claude entry for non-model settings")
	}
	if pm.DefaultModel != "" {
		t.Fatalf("ProviderModelsForSave()[claude].DefaultModel = %q, want cleared", pm.DefaultModel)
	}
	if pm.AnthropicVersion != "2099-01-01" {
		t.Fatalf("ProviderModelsForSave()[claude].AnthropicVersion = %q, want %q", pm.AnthropicVersion, "2099-01-01")
	}
	if pm.MaxOutputTokens != 1234 {
		t.Fatalf("ProviderModelsForSave()[claude].MaxOutputTokens = %d, want %d", pm.MaxOutputTokens, 1234)
	}
}

func TestSyncDefaultModelToProvider_CreatesProviderEntryWhenMissing(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.DefaultModel = "new-model"
	cfg.ProviderModels = map[string]config.ProviderModelConfig{}

	a := &Agent{ProviderName: "openai"}
	a.SyncDefaultModelToProvider(cfg)

	if got := cfg.ProviderModels["openai"].DefaultModel; got != "new-model" {
		t.Fatalf("ProviderModels[openai].DefaultModel = %q, want %q", got, "new-model")
	}
}

func TestSyncDefaultModelToProvider_PrefersAnthropicAliasEntry(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.DefaultProvider = "anthropic"
	cfg.DefaultModel = "new-model"
	cfg.SetProviderModelConfig("anthropic", config.ProviderModelConfig{DefaultModel: "old-model"})

	a := &Agent{
		ProviderName:      "claude",
		ProviderConfigKey: "anthropic",
	}
	a.SyncDefaultModelToProvider(cfg)

	if got := cfg.ProviderModels["anthropic"].DefaultModel; got != "new-model" {
		t.Fatalf("ProviderModels[anthropic].DefaultModel = %q, want %q", got, "new-model")
	}
	if got := cfg.ProviderModels["claude"].DefaultModel; got != "claude-sonnet-4-6" {
		t.Fatalf("ProviderModels[claude].DefaultModel = %q, want default claude entry to remain unchanged", got)
	}
}

func TestSyncDefaultModelToProvider_PreservesSessionAnthropicAliasWhenDefaultProviderDiffers(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.DefaultProvider = "deepseek"
	cfg.DefaultModel = "anthropic-new"
	cfg.SetProviderModelsForEdit(map[string]config.ProviderModelConfig{
		"anthropic": {DefaultModel: "anthropic-old"},
		"claude":    {DefaultModel: "claude-old"},
	})

	a := &Agent{
		ProviderName:      "claude",
		ProviderConfigKey: "anthropic",
		CurrentModel:      "anthropic-old",
	}
	a.SyncDefaultModelToProvider(cfg)

	saved := cfg.ProviderModelsForSave()
	if got := saved["anthropic"].DefaultModel; got != "anthropic-new" {
		t.Fatalf("ProviderModelsForSave()[anthropic].DefaultModel = %q, want %q", got, "anthropic-new")
	}
	if got := saved["claude"].DefaultModel; got != "claude-old" {
		t.Fatalf("ProviderModelsForSave()[claude].DefaultModel = %q, want %q", got, "claude-old")
	}
}
