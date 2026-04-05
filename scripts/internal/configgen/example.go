package configgen

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"gopkg.in/yaml.v3"
)

// GenerateExampleFile builds the canonical config.yaml.example content.
func GenerateExampleFile(cfg *config.Config) ([]byte, error) {
	cfgCopy := *cfg
	cfgCopy.LSP = cfg.LSP
	cfgCopy.LSP.Servers = nil
	cfgCopy.WebSearch = cfg.WebSearch
	cfgCopy.WebSearch.Provider = "gemini"

	data, err := yaml.Marshal(&cfgCopy)
	if err != nil {
		return nil, err
	}

	data, err = FilterInternalFields(data)
	if err != nil {
		return nil, err
	}

	output := AddComments(string(data))
	header := `# XELYON CLI 設定例
# 設定ファイルの場所: ~/.xelyon/config.yaml
# 初回起動時に自動的に作成されます
# 詳細は docs/config.md を参照してください

`

	return []byte(header + output), nil
}

// FilterInternalFields removes internal-only config fields from marshaled YAML.
func FilterInternalFields(data []byte) ([]byte, error) {
	var raw yaml.Node
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	userFacing := map[string]map[string]bool{}
	for sectionKey, info := range Sections {
		if len(info.Fields) == 0 {
			continue
		}
		allStructMap := true
		for _, fieldType := range info.FieldTypes {
			if fieldType != "structmap" && fieldType != "map" {
				allStructMap = false
				break
			}
		}
		if allStructMap {
			continue
		}
		fields := map[string]bool{}
		for field := range info.Fields {
			fields[field] = true
		}
		userFacing[sectionKey] = fields
	}

	// Example files should omit fields that are technically configurable but
	// unsafe to encourage as copy-paste defaults.
	exampleOmittedFields := map[string]map[string]bool{
		"lsp": {
			"servers": true,
		},
	}

	internalSections := map[string]bool{
		"loop_detection":  true,
		"api_retry":       true,
		"diff":            true,
		"command_aliases": true,
		"prompt_cache":    true,
		"streaming":       true,
		"tool_confirm":    true,
		"bash":            true,
		"git_stage":       true,
		"plan_mode":       true,
		"list_dir":        true,
		"openai":          true,
		"thinking":        true,
	}

	if raw.Kind == yaml.DocumentNode && len(raw.Content) > 0 {
		mapping := raw.Content[0]
		if mapping.Kind == yaml.MappingNode {
			filterMapping(mapping, userFacing, internalSections, exampleOmittedFields)
		}
	}

	return yaml.Marshal(&raw)
}

// AddComments injects section and field comments into the example YAML.
func AddComments(yamlStr string) string {
	lines := strings.Split(yamlStr, "\n")
	var result []string
	currentSection := ""

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") && strings.Contains(line, ":") {
			key := strings.Split(trimmed, ":")[0]
			if info, ok := Sections[key]; ok {
				if info.Title != "" {
					if len(result) > 0 {
						result = append(result, "")
					}
					result = append(result, "# ============================================================")
					result = append(result, fmt.Sprintf("# %s", info.Title))
					result = append(result, "# ============================================================")
				}
				for _, comment := range info.Comments {
					result = append(result, fmt.Sprintf("# %s", comment))
				}
				currentSection = key
			}
		}

		if currentSection != "" {
			info := Sections[currentSection]
			fieldKey := strings.TrimSpace(strings.Split(trimmed, ":")[0])
			indent := len(line) - len(strings.TrimLeft(line, " \t"))
			if indent == 4 && !strings.HasSuffix(trimmed, ":") {
				if comment, ok := info.Fields[fieldKey]; ok {
					result = append(result, fmt.Sprintf("    # %s", comment))
				}
			} else if indent == 0 {
				if comment, ok := info.Fields[fieldKey]; ok && i > 0 {
					result = append(result, fmt.Sprintf("# %s", comment))
				}
			}
		}

		result = append(result, line)
	}

	return strings.Join(result, "\n") + "\n"
}

func filterMapping(mapping *yaml.Node, userFacing map[string]map[string]bool, internalSections map[string]bool, omittedFields map[string]map[string]bool) {
	var filtered []*yaml.Node
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		key := mapping.Content[i].Value
		val := mapping.Content[i+1]

		if internalSections[key] {
			continue
		}

		if fields, ok := userFacing[key]; ok && val.Kind == yaml.MappingNode {
			var childFiltered []*yaml.Node
			for j := 0; j+1 < len(val.Content); j += 2 {
				childKey := val.Content[j].Value
				if fields[childKey] && !omittedFields[key][childKey] {
					childFiltered = append(childFiltered, val.Content[j], val.Content[j+1])
				}
			}
			val.Content = childFiltered
		}

		filtered = append(filtered, mapping.Content[i], val)
	}
	mapping.Content = filtered
}
