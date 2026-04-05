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
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	configgen "github.com/susugadx/xelyon-cli/scripts/internal/configgen"
	"gopkg.in/yaml.v3"
)

func main() {
	defaultCfg := config.DefaultConfig()

	configExample, err := os.ReadFile("config.yaml.example")
	if err != nil {
		fmt.Println("config.yaml.example not found, generating...")
		configExample, err = configgen.GenerateExampleFile(defaultCfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating config example: %v\n", err)
			os.Exit(1)
		}
	}

	configMdPath := "docs/config.md"
	content, err := os.ReadFile(configMdPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", configMdPath, err)
		os.Exit(1)
	}

	newContent := string(content)
	var replaced bool
	newContent, replaced = configgen.ReplaceMarkerContent(newContent,
		"<!-- CONFIG-EXAMPLE-START -->",
		"<!-- CONFIG-EXAMPLE-END -->",
		configgen.FormatConfigExample(string(configExample)))
	if !replaced {
		fmt.Fprintf(os.Stderr, "Error: %s is missing CONFIG-EXAMPLE markers\n", configMdPath)
		os.Exit(1)
	}

	if strings.Contains(newContent, "<!-- CONFIG-DETAILS-START -->") &&
		strings.Contains(newContent, "<!-- CONFIG-DETAILS-END -->") {
		structs, err := configgen.ParseConfigTypes("internal/config")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing internal/config: %v\n", err)
			os.Exit(1)
		}

		defaultYAML, _ := yaml.Marshal(defaultCfg)
		defaults := make(map[string]interface{})
		yaml.Unmarshal(defaultYAML, &defaults)
		configDetails := configgen.GenerateConfigDetails(structs, defaults)

		newContent, _ = configgen.ReplaceMarkerContent(newContent,
			"<!-- CONFIG-DETAILS-START -->",
			"<!-- CONFIG-DETAILS-END -->",
			configDetails)
	}

	if err := os.WriteFile(configMdPath, []byte(newContent), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", configMdPath, err)
		os.Exit(1)
	}

	fmt.Printf("Updated %s\n", configMdPath)
}
