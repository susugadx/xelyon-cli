//go:build ignore

// gen-config-registry.go generates registry_generated.go from config metadata.
//
// Usage:
//
//	go run scripts/config_sections.go scripts/gen-config-registry.go
//	make gen-registry
package main

import (
	"fmt"
	"os"

	"github.com/susugadx/xelyon-cli/scripts/internal/configregistry"
	"github.com/susugadx/xelyon-cli/scripts/internal/scriptio"
)

func main() {
	source, err := configregistry.GenerateRegistrySource()
	if err != nil {
		scriptio.ExitWithError("Error generating registry source: %v", err)
	}

	outputPath := scriptio.OutputPathFromArgs(os.Args, "internal/config/registry_generated.go")

	if err := os.WriteFile(outputPath, source, 0644); err != nil {
		scriptio.ExitWithError("Error writing file: %v", err)
	}

	fmt.Printf("Generated %s\n", outputPath)
}
