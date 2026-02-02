//go:build ignore

// gen-config-example.go は DefaultConfig() から config.yaml.example を生成するスクリプト
//
// 使用方法:
//
//	go run scripts/config_sections.go scripts/gen-config-example.go
//	make gen-config
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"gopkg.in/yaml.v3"
)

// sections と sectionOrder は config_sections.go で定義
// go run scripts/config_sections.go scripts/gen-config-example.go で両方を読み込む

func main() {
	cfg := config.DefaultConfig()

	// LSPのserversを空にする（デフォルト内蔵のため）
	cfg.LSP.Servers = nil

	// 旧キーはomitemptyなので example には出さない
	cfg.PlanMode.MaxParallelSteps = 0
	cfg.PlanMode.AutoRetry = 0

	// YAML形式で出力
	data, err := yaml.Marshal(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling config: %v\n", err)
		os.Exit(1)
	}

	// コメント付きYAMLを生成
	output := addComments(string(data))

	// ヘッダーコメント
	header := `# XELYON CLI 設定例
# 設定ファイルの場所: ~/.xelyon/config.yaml
# 初回起動時に自動的に作成されます
# 詳細は docs/config.md を参照してください

`

	output = header + output

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

func addComments(yamlStr string) string {
	lines := strings.Split(yamlStr, "\n")
	var result []string
	currentSection := ""

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// 空行はスキップ
		if trimmed == "" {
			continue
		}

		// トップレベルのキーを検出
		if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") && strings.Contains(line, ":") {
			key := strings.Split(trimmed, ":")[0]

			// セクション情報を取得（Sections は config_sections.go で定義）
			if info, ok := Sections[key]; ok {
				// セクション区切りとタイトル
				if info.Title != "" {
					if len(result) > 0 {
						result = append(result, "")
					}
					result = append(result, "# ============================================================")
					result = append(result, fmt.Sprintf("# %s", info.Title))
					result = append(result, "# ============================================================")
				}

				// セクション説明コメント
				for _, comment := range info.Comments {
					result = append(result, fmt.Sprintf("# %s", comment))
				}

				currentSection = key
			}
		}

		// フィールドコメントを追加
		if currentSection != "" {
			info := Sections[currentSection]
			fieldKey := strings.TrimSpace(strings.Split(trimmed, ":")[0])

			// ネストされたセクション（provider_models内のclaude:等）はスキップ
			indent := len(line) - len(strings.TrimLeft(line, " \t"))
			if indent == 4 && !strings.HasSuffix(trimmed, ":") {
				// 4スペースインデントのフィールド（セクション内のフィールド）
				if comment, ok := info.Fields[fieldKey]; ok {
					result = append(result, fmt.Sprintf("    # %s", comment))
				}
			} else if indent == 0 {
				// トップレベルフィールド
				if comment, ok := info.Fields[fieldKey]; ok && i > 0 {
					result = append(result, fmt.Sprintf("# %s", comment))
				}
			}

		}

		result = append(result, line)
	}

	return strings.Join(result, "\n") + "\n"
}
