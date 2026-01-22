//go:build ignore

// gen-config-example.go は DefaultConfig() から config.yaml.example を生成するスクリプト
//
// 使用方法:
//   go run scripts/gen-config-example.go
//   make gen-config
package main

import (
	"fmt"
	"os"

	"github.com/susugadx/xelyon-cli/internal/config"
	"gopkg.in/yaml.v3"
)

func main() {
	cfg := config.DefaultConfig()

	// YAML形式で出力
	data, err := yaml.Marshal(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling config: %v\n", err)
		os.Exit(1)
	}

	// ヘッダーコメント
	header := `# XELYON CLI 設定例
# このファイルは DefaultConfig() から自動生成されました
# 生成コマンド: go run scripts/gen-config-example.go
#
# 設定ファイルの場所: ~/.xelyon/config.yaml
# 初回起動時に自動的に作成されます
#
# 詳細は docs/config.md を参照してください

`

	output := header + string(data)

	// config.yaml.example に出力
	outputPath := "config.yaml.example"
	if len(os.Args) > 1 {
		outputPath = os.Args[1]
	}

	if err := os.WriteFile(outputPath, []byte(output), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated %s\n", outputPath)
}
