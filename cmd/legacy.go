package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/api/xelyon"
	"github.com/susugadx/xelyon-cli/internal/file"
)

// runLegacyMode は従来の1ショットモードを実行
func runLegacyMode(query string, model string, provider api.Provider) {
	var contextParts []string

	projectConfig := loadProjectConfig()
	if projectConfig != "" {
		fmt.Println("📋 XELYON.md を読み込み")
		contextParts = append(contextParts, projectConfig)
	}

	if len(files) > 0 {
		fmt.Println("📄 ファイル読み込み中...")
		fileContent, err := file.ReadFiles(files)
		if err != nil {
			fmt.Fprintf(os.Stderr, "エラー: %v\n", err)
			os.Exit(1)
		}
		contextParts = append(contextParts, fileContent)
		fmt.Printf("   %d 件のファイルを読み込み\n", len(files))
	}

	if userID != "" {
		fmt.Println("🔍 RAG検索中...")
		results, err := xelyon.SearchRAG(query, userID, 3)
		if err == nil && results.Count > 0 {
			var contents []string
			for _, r := range results.Results {
				contents = append(contents, fmt.Sprintf("[%s]\n%s", r.DocumentTitle, r.Content))
			}
			contextParts = append(contextParts, "## RAG検索結果:\n"+strings.Join(contents, "\n\n"))
			fmt.Printf("   %d 件のドキュメントを参照\n", results.Count)
		}
	}

	fmt.Println("🤖 AI回答:")
	systemPrompt := strings.Join(contextParts, "\n\n---\n\n")
	history := []api.Message{{Role: "user", Content: query}}
	ctx := context.Background()
	response, err := provider.ChatWithTools(ctx, systemPrompt, history, model)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nエラー: %v\n", err)
		os.Exit(1)
	}

	if output != "" {
		code := file.ExtractCodeBlock(response)
		if code != "" {
			if file.ConfirmApply(output, code) {
				err := file.WriteFile(output, code)
				if err != nil {
					fmt.Fprintf(os.Stderr, "エラー: %v\n", err)
					os.Exit(1)
				}
				fmt.Println("✅ ファイルを作成しました:", output)
			} else {
				fmt.Println("❌ キャンセルしました")
			}
		} else {
			fmt.Println("⚠️  コードブロックが見つかりませんでした")
		}
	}

	if edit && len(files) == 1 && output == "" {
		code := file.ExtractCodeBlock(response)
		if code != "" {
			if file.ConfirmApply(files[0], code) {
				err := file.WriteFile(files[0], code)
				if err != nil {
					fmt.Fprintf(os.Stderr, "エラー: %v\n", err)
					os.Exit(1)
				}
				fmt.Println("✅ ファイルを更新しました")
			} else {
				fmt.Println("❌ キャンセルしました")
			}
		} else {
			fmt.Println("⚠️  コードブロックが見つかりませんでした")
		}
	}
}
