package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/file"
)

var (
	userID string
	files  []string
	edit   bool
	output string
)

const projectConfigFile = "XELYON.md"

func loadProjectConfig() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	for {
		configPath := filepath.Join(dir, projectConfigFile)
		if content, err := os.ReadFile(configPath); err == nil {
			return fmt.Sprintf("## プロジェクト設定 (%s):\n%s", configPath, string(content))
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

var rootCmd = &cobra.Command{
	Use:   "xelyon [query]",
	Short: "XELYON CLI - AI-powered coding assistant with RAG",
	Long:  `XELYON CLI is an AI coding assistant that leverages your past knowledge and documents.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 {
			query := args[0]
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
				results, err := api.SearchRAG(query, userID, 3)
				if err == nil && results.Count > 0 {
					var contents []string
					for _, r := range results.Results {
						contents = append(contents, fmt.Sprintf("[%s]\n%s", r.DocumentTitle, r.Content))
					}
					contextParts = append(contextParts, "## RAG検索結果:\n"+strings.Join(contents, "\n\n"))
					fmt.Printf("   %d 件のドキュメントを参照\n", results.Count)
				}
			}

			fmt.Println("🤖 AI回答:\n")
			context := strings.Join(contextParts, "\n\n---\n\n")
			response, err := api.AskDeepSeekStream(query, context)
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
		} else {
			cmd.Help()
		}
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&userID, "user", "", "User ID for RAG search")
	rootCmd.PersistentFlags().StringSliceVarP(&files, "file", "f", []string{}, "Files to include as context")
	rootCmd.PersistentFlags().BoolVarP(&edit, "edit", "e", false, "Enable edit mode")
	rootCmd.PersistentFlags().StringVarP(&output, "output", "o", "", "Output file path")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}