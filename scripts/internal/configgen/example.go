package configgen

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"gopkg.in/yaml.v3"
)

const configExampleFileHeader = `# XELYON CLI 設定例
# 設定ファイルの場所: ~/.xelyon/config.yaml
# 初回起動時に自動的に作成されます
# 詳細は docs/config.md を参照してください

`

var defaultExampleFilterSpec = buildExampleFilterSpec(Sections)

// GenerateExampleFile builds the canonical config.yaml.example content.
func GenerateExampleFile(cfg *config.Config) ([]byte, error) {
	cfgCopy := *cfg
	applyExampleOverrides(&cfgCopy)

	data, err := yaml.Marshal(&cfgCopy)
	if err != nil {
		return nil, err
	}

	data, err = FilterInternalFields(data)
	if err != nil {
		return nil, err
	}

	output := AddComments(string(data))
	return []byte(configExampleFileHeader + output), nil
}

// FilterInternalFields removes internal-only config fields from marshaled YAML.
func FilterInternalFields(data []byte) ([]byte, error) {
	var raw yaml.Node
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	if raw.Kind == yaml.DocumentNode && len(raw.Content) > 0 {
		mapping := raw.Content[0]
		if mapping.Kind == yaml.MappingNode {
			filterExampleRootMapping(mapping, defaultExampleFilterSpec)
		}
	}

	return yaml.Marshal(&raw)
}

// AddComments injects section and field comments into the example YAML.
func AddComments(yamlStr string) string {
	var raw yaml.Node
	if err := yaml.Unmarshal([]byte(yamlStr), &raw); err != nil {
		return strings.TrimRight(yamlStr, "\n") + "\n"
	}
	if raw.Kind != yaml.DocumentNode || len(raw.Content) == 0 {
		return strings.TrimRight(yamlStr, "\n") + "\n"
	}

	mapping := raw.Content[0]
	if mapping.Kind != yaml.MappingNode {
		return strings.TrimRight(yamlStr, "\n") + "\n"
	}

	annotateExampleSectionComments(mapping)

	out, err := yaml.Marshal(&raw)
	if err != nil {
		return strings.TrimRight(yamlStr, "\n") + "\n"
	}
	return string(out)
}
