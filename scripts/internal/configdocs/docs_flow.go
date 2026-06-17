package configdocs

import (
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/config"
	"gopkg.in/yaml.v3"
)

// UpdateConfigDetailsContent は details marker がある場合だけ config details block を更新する。
func UpdateConfigDetailsContent(content string, defaultCfg *config.Config, configDir string) (string, error) {
	if !hasConfigDetailsMarkers(content) {
		return content, nil
	}

	configDetails, err := generateConfigDetailsFromDefault(configDir, defaultCfg)
	if err != nil {
		return "", err
	}
	return replaceConfigDetailsBlock(content, configDetails), nil
}

// generateConfigDetailsFromDefault は config struct と default config から details block を生成する。
func generateConfigDetailsFromDefault(configDir string, defaultCfg *config.Config) (string, error) {
	structs, err := parseConfigTypes(configDir)
	if err != nil {
		return "", fmt.Errorf("parsing %s: %w", configDir, err)
	}

	defaultYAML, err := yaml.Marshal(defaultCfg)
	if err != nil {
		return "", fmt.Errorf("marshaling default config: %w", err)
	}
	defaults := make(map[string]interface{})
	if err := yaml.Unmarshal(defaultYAML, &defaults); err != nil {
		return "", fmt.Errorf("unmarshaling default config map: %w", err)
	}
	return generateConfigDetails(structs, defaults), nil
}
