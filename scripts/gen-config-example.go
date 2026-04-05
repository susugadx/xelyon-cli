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
	configgen "github.com/susugadx/xelyon-cli/scripts/internal/configgen"
)

func main() {
	output, err := configgen.GenerateExampleFile(config.DefaultConfig())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating config example: %v\n", err)
		os.Exit(1)
	}

	outputPath := "config.yaml.example"
	if len(os.Args) > 1 {
		outputPath = os.Args[1]
	}

	if err := os.WriteFile(outputPath, output, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated %s\n", outputPath)
}
