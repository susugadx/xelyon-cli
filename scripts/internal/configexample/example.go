package configexample

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/scripts/internal/configmeta"
	"gopkg.in/yaml.v3"
)

const configExampleFileHeader = `# XELYON CLI 設定例
# 設定ファイルの場所: ~/.xelyon/config.yaml
# 初回起動時に自動的に作成されます
# 詳細は docs/config.md を参照してください

`

var defaultExampleFilterSpec = buildExampleFilterSpec(configmeta.Sections)

// GenerateExampleFile は canonical な config.yaml.example の内容を生成する。
func GenerateExampleFile(cfg *config.Config) ([]byte, error) {
	cfgCopy := *cfg
	applyExampleOverrides(&cfgCopy)

	data, err := yaml.Marshal(&cfgCopy)
	if err != nil {
		return nil, err
	}

	data, err = filterInternalFields(data)
	if err != nil {
		return nil, err
	}

	data, err = applyExampleExplicitProviderModelFields(data)
	if err != nil {
		return nil, err
	}

	output := addComments(string(data))
	return []byte(configExampleFileHeader + output), nil
}

// filterInternalFields は marshal 済み YAML から internal-only config field を除去する。
func filterInternalFields(data []byte) ([]byte, error) {
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

func applyExampleExplicitProviderModelFields(data []byte) ([]byte, error) {
	var raw yaml.Node
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	if raw.Kind != yaml.DocumentNode || len(raw.Content) == 0 {
		return data, nil
	}
	providerModels := findMappingValue(raw.Content[0], "provider_models")
	openAISubscription := findMappingValue(providerModels, "openai_subscription")
	if openAISubscription == nil || openAISubscription.Kind != yaml.MappingNode {
		return yaml.Marshal(&raw)
	}
	setMappingScalar(openAISubscription, "max_output_tokens", "0")
	return yaml.Marshal(&raw)
}

func findMappingValue(mapping *yaml.Node, key string) *yaml.Node {
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i+1]
		}
	}
	return nil
}

func setMappingScalar(mapping *yaml.Node, key, value string) {
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			mapping.Content[i+1] = &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: value}
			return
		}
	}
	mapping.Content = append(mapping.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: value},
	)
}

// addComments は example YAML に section と field のコメントを注入する。
func addComments(yamlStr string) string {
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
	return FormatExampleOutput(string(out))
}

// FormatExampleOutput は example YAML の section header 間に安定した空行を入れる。
func FormatExampleOutput(output string) string {
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	var formatted []string
	for i, line := range lines {
		if isExampleSectionHeaderStart(lines, i) && len(formatted) > 0 && formatted[len(formatted)-1] != "" {
			formatted = append(formatted, "")
		}
		formatted = append(formatted, line)
	}
	return strings.Join(formatted, "\n") + "\n"
}

func isExampleSectionHeaderStart(lines []string, index int) bool {
	const separator = "# ============================================================"
	return index+2 < len(lines) &&
		lines[index] == separator &&
		strings.HasPrefix(lines[index+1], "# ") &&
		lines[index+2] == separator
}
