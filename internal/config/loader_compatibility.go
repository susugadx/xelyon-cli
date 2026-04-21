package config

import "gopkg.in/yaml.v3"

type loaderSections struct {
	lspSectionExists bool
	lspServersExists bool
}

func detectLoaderSections(data []byte) loaderSections {
	raw := parseYAMLRoot(data)
	if raw == nil {
		return loaderSections{}
	}
	return loaderSections{
		lspSectionExists: rootHasKey(raw, "lsp"),
		lspServersExists: rootHasNestedKey(raw, "lsp", "servers"),
	}
}

func defaultConfigForLoad(sections loaderSections) *Config {
	cfg := DefaultConfig()
	if sections.lspServersExists {
		// lsp.servers は nil と empty map を区別したいので、
		// YAML に存在する場合だけ defaults 側の既定 map を事前に外す。
		cfg.LSP.Servers = nil
	}
	return cfg
}

func applyLegacyLoadCompatibility(data []byte, cfg *Config) {
	migrateOldKeys(data, cfg)
	cfg.providerModelsStore = providerModelStoreFromYAML(data)
}

func (s loaderSections) defaultApplyOptions() defaultApplyOptions {
	return defaultApplyOptions(s)
}

func yamlHasNestedKey(data []byte, parentKey, childKey string) bool {
	raw := parseYAMLRoot(data)
	if raw == nil {
		return false
	}
	return rootHasNestedKey(raw, parentKey, childKey)
}

func yamlHasKey(data []byte, key string) bool {
	raw := parseYAMLRoot(data)
	if raw == nil {
		return false
	}
	return rootHasKey(raw, key)
}

func parseYAMLRoot(data []byte) map[string]interface{} {
	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil
	}
	return raw
}

func rootHasKey(raw map[string]interface{}, key string) bool {
	if raw == nil {
		return false
	}
	_, exists := raw[key]
	return exists
}

func rootHasNestedKey(raw map[string]interface{}, parentKey, childKey string) bool {
	if raw == nil {
		return false
	}
	parent, ok := raw[parentKey].(map[string]interface{})
	if !ok {
		return false
	}
	_, exists := parent[childKey]
	return exists
}
