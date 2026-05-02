package configgen

import "github.com/susugadx/xelyon-cli/internal/config"

func applyExampleOverrides(cfg *config.Config) {
	if cfg == nil {
		return
	}
	for sectionName, section := range Sections {
		if len(section.Example.Overrides) == 0 {
			continue
		}
		applyExampleSectionOverrides(cfg, sectionName, section.Example.Overrides)
	}
}

func applyExampleSectionOverrides(cfg *config.Config, sectionName string, overrides map[string]any) {
	switch sectionName {
	case "lsp":
		if value, ok := overrides["servers"]; ok {
			if value == nil {
				cfg.LSP.Servers = nil
			}
		}
	case "web_search":
		if value, ok := overrides["provider"]; ok {
			if provider, ok := value.(string); ok {
				cfg.WebSearch.Provider = provider
			}
		}
	}
}
