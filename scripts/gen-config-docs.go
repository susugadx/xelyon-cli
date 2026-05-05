//go:build ignore

// gen-config-docs.go updates docs/config.md from internal/config metadata.
//
// Usage:
//
//	go run scripts/config_sections.go scripts/gen-config-docs.go
//	make gen-docs
//
// docs/config.md updates:
//
//	<!-- CONFIG-EXAMPLE-START --> ... <!-- CONFIG-EXAMPLE-END -->
//
// If CONFIG-DETAILS markers exist, that block is also refreshed.
package main

import (
	"fmt"
	"os"

	"github.com/susugadx/xelyon-cli/internal/config"
	configgen "github.com/susugadx/xelyon-cli/scripts/internal/configgen"
	"gopkg.in/yaml.v3"
)

func main() {
	defaultCfg := config.DefaultConfig()

	configExample, found, err := configgen.ReadFileIfExists("config.yaml.example")
	if err != nil {
		configgen.ExitWithError("Error reading config.yaml.example: %v", err)
	}
	if !found {
		fmt.Println("config.yaml.example not found, generating...")
		configExample, err = configgen.GenerateExampleFile(defaultCfg)
		if err != nil {
			configgen.ExitWithError("Error generating config example: %v", err)
		}
	}

	configMdPath := "docs/config.md"
	content, err := os.ReadFile(configMdPath)
	if err != nil {
		configgen.ExitWithError("Error reading %s: %v", configMdPath, err)
	}

	newContent := string(content)
	newContent, err = configgen.ReplaceConfigExampleBlock(newContent, string(configExample))
	if err != nil {
		configgen.ExitWithError("Error updating CONFIG-EXAMPLE block: %v", err)
	}

	if configgen.HasConfigDetailsMarkers(newContent) {
		structs, err := configgen.ParseConfigTypes("internal/config")
		if err != nil {
			configgen.ExitWithError("Error parsing internal/config: %v", err)
		}

		defaultYAML, err := yaml.Marshal(defaultCfg)
		if err != nil {
			configgen.ExitWithError("Error marshaling default config: %v", err)
		}
		defaults := make(map[string]interface{})
		if err := yaml.Unmarshal(defaultYAML, &defaults); err != nil {
			configgen.ExitWithError("Error unmarshaling default config map: %v", err)
		}
		configDetails := configgen.GenerateConfigDetails(structs, defaults)

		newContent = configgen.ReplaceConfigDetailsBlock(newContent, configDetails)
	}

	if err := os.WriteFile(configMdPath, []byte(newContent), 0644); err != nil {
		configgen.ExitWithError("Error writing %s: %v", configMdPath, err)
	}

	fmt.Printf("Updated %s\n", configMdPath)
}
