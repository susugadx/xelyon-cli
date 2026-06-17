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
	"github.com/susugadx/xelyon-cli/scripts/internal/configdocs"
	"github.com/susugadx/xelyon-cli/scripts/internal/configexample"
	"github.com/susugadx/xelyon-cli/scripts/internal/scriptio"
)

func main() {
	defaultCfg := config.DefaultConfig()

	configExample, found, err := scriptio.ReadFileIfExists("config.yaml.example")
	if err != nil {
		scriptio.ExitWithError("Error reading config.yaml.example: %v", err)
	}
	if !found {
		fmt.Println("config.yaml.example not found, generating...")
		configExample, err = configexample.GenerateExampleFile(defaultCfg)
		if err != nil {
			scriptio.ExitWithError("Error generating config example: %v", err)
		}
	}

	configMdPath := "docs/config.md"
	content, err := os.ReadFile(configMdPath)
	if err != nil {
		scriptio.ExitWithError("Error reading %s: %v", configMdPath, err)
	}

	newContent := string(content)
	newContent, err = configdocs.ReplaceConfigExampleBlock(newContent, string(configExample))
	if err != nil {
		scriptio.ExitWithError("Error updating CONFIG-EXAMPLE block: %v", err)
	}

	newContent, err = configdocs.UpdateConfigDetailsContent(newContent, defaultCfg, "internal/config")
	if err != nil {
		scriptio.ExitWithError("Error updating CONFIG-DETAILS block: %v", err)
	}

	if err := os.WriteFile(configMdPath, []byte(newContent), 0644); err != nil {
		scriptio.ExitWithError("Error writing %s: %v", configMdPath, err)
	}

	fmt.Printf("Updated %s\n", configMdPath)
}
