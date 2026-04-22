package config

import "fmt"

type fieldAdapter struct {
	get        func(*Config) (interface{}, error)
	set        func(*Config, interface{}) error
	getDefault func() interface{}
}

func getProviderModelsFieldValue(cfg *Config) (interface{}, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}
	return cfg.ProviderModelsForEdit(), nil
}

func setProviderModelsFieldValue(cfg *Config, value interface{}) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	providerModels, ok := value.(map[string]ProviderModelConfig)
	if !ok {
		return fmt.Errorf("type mismatch: expected map[string]ProviderModelConfig, got %T", value)
	}
	if providerModels == nil {
		cfg.ResetProviderModelsForEdit()
		return nil
	}
	cfg.SetProviderModelsForEdit(providerModels)
	return nil
}

func getLSPServersFieldValue(cfg *Config) (interface{}, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}
	return cfg.LSP.Servers, nil
}

func setLSPServersFieldValue(cfg *Config, value interface{}) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	servers, ok := value.(map[string]LSPServerConfig)
	if !ok {
		return fmt.Errorf("type mismatch: expected map[string]LSPServerConfig, got %T", value)
	}
	cfg.LSP.Servers = servers
	return nil
}

var fieldAdapters = map[string]fieldAdapter{
	"provider_models": {
		get: getProviderModelsFieldValue,
		set: setProviderModelsFieldValue,
		getDefault: func() interface{} {
			return map[string]ProviderModelConfig(nil)
		},
	},
	"lsp.servers": {
		get: getLSPServersFieldValue,
		set: setLSPServersFieldValue,
	},
}
