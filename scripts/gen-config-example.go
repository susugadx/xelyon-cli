//go:build ignore

// gen-config-example.go generates config.yaml.example from DefaultConfig().
//
// Usage:
//
//	go run scripts/config_sections.go scripts/gen-config-example.go
//	make gen-config
package main

import (
	"fmt"
	"os"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/scripts/internal/configexample"
	"github.com/susugadx/xelyon-cli/scripts/internal/scriptio"
)

func main() {
	output, err := configexample.GenerateExampleFile(config.DefaultConfig())
	if err != nil {
		scriptio.ExitWithError("Error generating config example: %v", err)
	}

	outputPath := scriptio.OutputPathFromArgs(os.Args, "config.yaml.example")

	if err := os.WriteFile(outputPath, output, 0644); err != nil {
		scriptio.ExitWithError("Error writing file: %v", err)
	}

	fmt.Printf("Generated %s\n", outputPath)
}
