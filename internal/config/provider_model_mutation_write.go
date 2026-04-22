package config

// ProviderModelWriteKey は provider_models の更新先キーを返す。
func (c *Config) ProviderModelWriteKey(provider string) (string, bool) {
	if c == nil {
		return "", false
	}
	return providerModelWriteTargetKey(c.explicitProviderModelSource(), provider)
}

// UpdateExistingProviderModelConfig は provider_models エントリを更新する。
// raw provider_models が未定義でも、既知 provider なら保存対象の entry を新規作成する。
func (c *Config) UpdateExistingProviderModelConfig(provider string, update func(*ProviderModelConfig)) bool {
	return c.PatchProviderModelConfig(provider, update)
}
