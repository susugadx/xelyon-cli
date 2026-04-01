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

	// config.yaml.example では推奨例として Gemini を検索プロバイダーに設定する
	cfg.WebSearch.Provider = "gemini"

	// YAML形式で出力
	data, err := yaml.Marshal(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling config: %v\n", err)
		os.Exit(1)
	}

	// 内部専用フィールドを除外（user-facing example をクリーンに保つ）
	data, err = filterInternalFields(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error filtering internal fields: %v\n", err)
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

// filterInternalFields は YAML を raw map に変換し、
// user-facing でないセクション・フィールドを除去して再 marshal する。
func filterInternalFields(data []byte) ([]byte, error) {
	var raw yaml.Node
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	// user-facing フィールドパスを Sections から収集
	// structmap/map のみのセクションはフィルタ対象外（子要素はそのまま保持）
	userFacing := map[string]map[string]bool{}
	for sectionKey, info := range Sections {
		if len(info.Fields) == 0 {
			continue
		}
		// FieldTypes がすべて structmap/map の場合はフィルタ対象外
		allStructMap := true
		for _, ft := range info.FieldTypes {
			if ft != "structmap" && ft != "map" {
				allStructMap = false
				break
			}
		}
		if allStructMap {
			continue
		}
		fields := map[string]bool{}
		for f := range info.Fields {
			fields[f] = true
		}
		userFacing[sectionKey] = fields
	}

	// 内部専用トップレベルセクション（user-facing config から除外）
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
		"openai":          true, // Responses API ルーティングは内部自動判定（YAML 互換は維持）
		"thinking":        true, // /think コマンドが正規ルート（YAML 互換は維持）
	}

	// root mapping の content をフィルタ
	if raw.Kind == yaml.DocumentNode && len(raw.Content) > 0 {
		mapping := raw.Content[0]
		if mapping.Kind == yaml.MappingNode {
			filterMapping(mapping, userFacing, internalSections)
		}
	}

	return yaml.Marshal(&raw)
}

// filterMapping はトップレベル mapping から内部フィールドを除去する。
func filterMapping(mapping *yaml.Node, userFacing map[string]map[string]bool, internalSections map[string]bool) {
	var filtered []*yaml.Node
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		key := mapping.Content[i].Value
		val := mapping.Content[i+1]

		// 内部専用セクションは除外
		if internalSections[key] {
			continue
		}

		// ネストされたセクションの内部フィールドを除去
		if fields, ok := userFacing[key]; ok && val.Kind == yaml.MappingNode {
			var childFiltered []*yaml.Node
			for j := 0; j+1 < len(val.Content); j += 2 {
				childKey := val.Content[j].Value
				if fields[childKey] {
					childFiltered = append(childFiltered, val.Content[j], val.Content[j+1])
				}
			}
			val.Content = childFiltered
		}

		filtered = append(filtered, mapping.Content[i], val)
	}
	mapping.Content = filtered
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
