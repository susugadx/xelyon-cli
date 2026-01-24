//go:build ignore

// gen-config-example.go は DefaultConfig() から config.yaml.example を生成するスクリプト
//
// 使用方法:
//
//	go run scripts/gen-config-example.go
//	make gen-config
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"gopkg.in/yaml.v3"
)

// セクション情報
type sectionInfo struct {
	title    string   // セクションタイトル
	comments []string // セクション説明コメント
	fields   map[string]string // フィールド名 -> コメント
}

// セクション定義
var sections = map[string]sectionInfo{
	"default_provider": {
		title: "プロバイダー設定",
		fields: map[string]string{
			"default_provider": "デフォルトで使用するLLMプロバイダー",
			"default_model":    "デフォルトで使用するモデル",
		},
	},
	"provider_models": {
		fields: map[string]string{
			"provider_models": "プロバイダーごとのモデル設定",
		},
	},
	"compression": {
		title: "会話履歴圧縮設定",
		fields: map[string]string{
			"auto_compress":     "トークン使用率が閾値を超えたら自動圧縮",
			"threshold_tokens":  "トークン数の閾値（0 = 使用しない）",
			"threshold_percent": "トークン使用率の閾値（%）",
			"keep_recent":       "圧縮時に保持する直近メッセージ数",
			"prefer_compact_api": "プロバイダーのCompact APIを優先使用",
		},
	},
	"backup": {
		title: "バックアップ設定",
		fields: map[string]string{
			"max_generations": "保持する世代数",
		},
	},
	"loop_detection": {
		title: "ループ検知設定",
		fields: map[string]string{
			"threshold": "同じツール呼び出しの繰り返し回数でループと判定",
		},
	},
	"api_retry": {
		title: "APIリトライ設定",
		fields: map[string]string{
			"count":         "リトライ回数",
			"initial_delay": "初回リトライ待機時間（秒）",
			"max_delay":     "最大待機時間（秒）",
			"timeout":       "タイムアウト（秒）",
		},
	},
	"diff": {
		title: "差分表示設定",
		fields: map[string]string{
			"context_lines": "差分表示時のコンテキスト行数",
		},
	},
	"tool_confirm": {
		title: "ツール確認設定",
		fields: map[string]string{
			"auto_approve_safe":   "安全なツール（read_file等）を自動承認",
			"auto_approve_medium": "中程度のツール（write_file等）を自動承認",
		},
	},
	"paste": {
		title: "ペーストモード設定",
		fields: map[string]string{
			"max_lines":       "最大行数",
			"max_bytes":       "最大バイト数",
			"timeout_seconds": "タイムアウト（秒）",
		},
	},
	"streaming": {
		title: "ストリーミング設定",
		fields: map[string]string{
			"idle_timeout_seconds":  "アイドルタイムアウト（秒）",
			"show_file_info":        "ファイル読み込み時にサイズ・行数を表示",
			"show_search_progress":  "検索時に進捗を表示",
			"stream_bash_output":    "bashコマンドの出力をリアルタイム表示",
		},
	},
	"bash": {
		title: "bashツール設定",
		fields: map[string]string{
			"safety_level":      "安全レベル: strict / moderate / permissive",
			"safe_commands":     "追加の安全コマンドリスト",
			"allow_pipe":        "パイプを許可",
			"allow_redirect":    "リダイレクトを許可",
			"allow_inline_edit": "インライン編集を許可（sed -i 等）",
		},
	},
	"code_health": {
		title: "コード健全性チェック設定",
		fields: map[string]string{
			"enabled":            "有効化",
			"max_file_lines":     "ファイルの最大行数警告",
			"max_function_lines": "関数の最大行数警告",
			"auto_suggest":       "変更時に自動提案",
			"on_change":          "変更時のチェック項目",
		},
	},
	"git_stage": {
		title: "git_add設定",
		fields: map[string]string{
			"batch_confirm": "複数ファイルをまとめて確認",
		},
	},
	"plan_mode": {
		title: "Plan Mode設定",
		fields: map[string]string{
			"max_parallel_steps": "並列実行する最大ステップ数",
		},
	},
	"lsp": {
		title: "LSP連携設定",
		comments: []string{
			"23言語のデフォルト設定が内蔵済み。通常は変更不要です。",
			"詳細: docs/config.md の「LSP連携設定」セクション",
		},
		fields: map[string]string{
			"enabled": "LSP連携の有効/無効",
		},
	},
	"openai": {
		title: "OpenAI設定",
		fields: map[string]string{
			"responses_api_models": "Responses APIを使用するモデル",
		},
	},
	"thinking": {
		title: "Extended Thinking設定",
		fields: map[string]string{
			"enabled": "有効化",
			"level":   "レベル: low / medium / high",
		},
	},
}

// セクション順序
var sectionOrder = []string{
	"default_provider",
	"provider_models",
	"compression",
	"backup",
	"loop_detection",
	"api_retry",
	"diff",
	"tool_confirm",
	"paste",
	"streaming",
	"bash",
	"code_health",
	"git_stage",
	"plan_mode",
	"lsp",
	"openai",
	"thinking",
}

func main() {
	cfg := config.DefaultConfig()

	// LSPのserversを空にする（デフォルト内蔵のため）
	cfg.LSP.Servers = nil

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

			// セクション情報を取得
			if info, ok := sections[key]; ok {
				// セクション区切りとタイトル
				if info.title != "" {
					if len(result) > 0 {
						result = append(result, "")
					}
					result = append(result, "# ============================================================")
					result = append(result, fmt.Sprintf("# %s", info.title))
					result = append(result, "# ============================================================")
				}

				// セクション説明コメント
				for _, comment := range info.comments {
					result = append(result, fmt.Sprintf("# %s", comment))
				}

				currentSection = key
			}
		}

		// フィールドコメントを追加
		if currentSection != "" {
			info := sections[currentSection]
			fieldKey := strings.TrimSpace(strings.Split(trimmed, ":")[0])

			// ネストされたセクション（provider_models内のclaude:等）はスキップ
			indent := len(line) - len(strings.TrimLeft(line, " \t"))
			if indent == 4 && !strings.HasSuffix(trimmed, ":") {
				// 4スペースインデントのフィールド（セクション内のフィールド）
				if comment, ok := info.fields[fieldKey]; ok {
					result = append(result, fmt.Sprintf("    # %s", comment))
				}
			} else if indent == 0 {
				// トップレベルフィールド
				if comment, ok := info.fields[fieldKey]; ok && i > 0 {
					result = append(result, fmt.Sprintf("# %s", comment))
				}
			}

		}

		result = append(result, line)
	}

	return strings.Join(result, "\n") + "\n"
}
