package config

func providerModelStoreFromYAMLWithRoot(data []byte, raw map[string]interface{}) providerModelStore {
	return providerModelStoreFromYAMLInput(providerModelStoreYAMLInputFromRoot(data, raw))
}
