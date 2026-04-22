package config

type providerDefaultModelSyncPlan struct {
	key        string
	model      string
	clearExact bool
	valid      bool
}

func providerDefaultModelConfigKey(provider string) (string, bool) {
	key := ActiveProviderConfigKey(provider)
	if key == "" {
		return "", false
	}
	return key, true
}

func providerDefaultModelSyncPlanFor(provider, model string) providerDefaultModelSyncPlan {
	key, ok := providerDefaultModelConfigKey(provider)
	if !ok || model == "" {
		return providerDefaultModelSyncPlan{}
	}

	plan := providerDefaultModelSyncPlan{
		key:   key,
		model: model,
		valid: true,
	}
	if base, ok := defaultProviderModelConfig(key); ok && model == base.DefaultModel {
		plan.clearExact = true
	}
	return plan
}
