package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/susugadx/xelyon-cli/internal/agent"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/file"
	"github.com/susugadx/xelyon-cli/internal/version"
)

var (
	userID        string
	files         []string
	edit          bool
	output        string
	resume        bool
	providerFlag  string
	modelFlag     string
	autoApprove   bool
	loopThreshold int
	apiRetry      int
	apiRetryDelay int
	diffLines     int
)

const projectConfigFile = "XELYON.md"

// getProvider は環境変数/設定ファイルからProviderを取得
// 優先順位: CLI flag > 環境変数 > 設定ファイル > デフォルト
func getProvider() api.Provider {
	// 優先順位: CLI flag > 環境変数 > 設定ファイル > デフォルト
	providerName := providerFlag
	if providerName == "" {
		providerName = os.Getenv("XELYON_PROVIDER")
	}
	if providerName == "" {
		cfg, _ := config.LoadConfig()
		if cfg != nil {
			providerName = cfg.DefaultProvider
		}
	}
	if providerName == "" {
		providerName = "deepseek" // デフォルト
	}

	return getProviderByName(providerName)
}

// getProviderByName はプロバイダー名から Provider インスタンスを生成
func getProviderByName(providerName string) api.Provider {
	switch strings.ToLower(providerName) {
	case "deepseek":
		apiKey := os.Getenv("DEEPSEEK_API_KEY")
		if apiKey == "" {
			fmt.Fprintln(os.Stderr, "Error: DEEPSEEK_API_KEY not set")
			os.Exit(1)
		}
		return api.NewDeepSeekProvider(apiKey)

	case "openai":
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			fmt.Fprintln(os.Stderr, "Error: OPENAI_API_KEY not set")
			os.Exit(1)
		}
		return api.NewOpenAIProvider(apiKey)

	case "gemini":
		apiKey := os.Getenv("GEMINI_API_KEY")
		if apiKey == "" {
			fmt.Fprintln(os.Stderr, "Error: GEMINI_API_KEY not set")
			os.Exit(1)
		}
		return api.NewGeminiProvider(apiKey)

	case "claude", "anthropic":
		apiKey := os.Getenv("ANTHROPIC_API_KEY")
		if apiKey == "" {
			fmt.Fprintln(os.Stderr, "Error: ANTHROPIC_API_KEY not set")
			os.Exit(1)
		}
		return api.NewClaudeProvider(apiKey)

	case "ollama":
		baseURL := os.Getenv("OLLAMA_BASE_URL")
		if baseURL == "" {
			baseURL = "http://localhost:11434"
		}
		return api.NewOllamaProvider(baseURL)

	case "groq":
		apiKey := os.Getenv("GROQ_API_KEY")
		if apiKey == "" {
			fmt.Fprintln(os.Stderr, "Error: GROQ_API_KEY not set")
			os.Exit(1)
		}
		return api.NewGroqProvider(apiKey)

	default:
		fmt.Fprintf(os.Stderr, "Error: Unknown provider: %s\n", providerName)
		fmt.Fprintln(os.Stderr, "Supported providers: deepseek, openai, gemini, claude, ollama, groq")
		os.Exit(1)
		return nil
	}
}

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

// getModel はフラグからモデルを決定する
func getModel(cmd *cobra.Command) string {
	// フラグが指定されていればそれを優先
	if m, _ := cmd.Flags().GetBool("coder"); m {
		return "deepseek-coder"
	}
	if m, _ := cmd.Flags().GetBool("think"); m {
		return "deepseek-reasoner"
	}
	if m, _ := cmd.Flags().GetBool("claude"); m {
		return "claude"
	}

	// フラグなし → 設定ファイルから読み込み
	cfg, err := config.LoadConfig()
	if err != nil {
		// エラー時はハードコードされたデフォルトを使用
		return "deepseek-coder"
	}

	return cfg.DefaultModel
}

var rootCmd = &cobra.Command{
	Use:     "xelyon [query]",
	Short:   "XELYON CLI - AI-powered coding assistant",
	Version: version.GetVersion(),
	Long: `XELYON CLI is an AI coding assistant that helps you with development tasks.

Models:
  (default)    DeepSeek V3 - Fast & balanced
  --coder      DeepSeek Coder - Code-focused
  --think      DeepSeek R1 - Deep reasoning
  --claude     Claude (Vertex AI) - Most capable

Examples:
  xelyon                          # Interactive mode
  xelyon "explain this project"   # One-shot query
  xelyon --coder                  # Interactive with Coder
  xelyon -f main.go "add logging" # With file context
  xelyon --think "design review"  # Deep thinking mode`,
	Run: func(cmd *cobra.Command, args []string) {
		// 設定を読み込み
		cfg, err := config.LoadConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to load config: %v\n", err)
			cfg = config.DefaultConfig()
		}

		// 環境変数で上書き
		cfg.ApplyEnvironmentOverrides()

		// CLIフラグで上書き（0や-1の場合は設定しない）
		var loopPtr, retryPtr, delayPtr, diffPtr *int
		if loopThreshold > 0 {
			loopPtr = &loopThreshold
		}
		if apiRetry > 0 {
			retryPtr = &apiRetry
		}
		if apiRetryDelay > 0 {
			delayPtr = &apiRetryDelay
		}
		if diffLines >= 0 {
			diffPtr = &diffLines
		}
		cfg.ApplyFlagOverrides(loopPtr, retryPtr, delayPtr, diffPtr)

		// グローバル設定として保存（agent側で参照できるように）
		config.SetGlobalConfig(cfg)

		model := getModel(cmd)
		provider := getProvider()

		// --resume フラグチェック
		if resume && len(args) == 0 && len(files) == 0 {
			agent.RunInteractiveWithResume(model, provider, autoApprove)
			return
		}

		// 引数なし & ファイル指定なし → 対話モード
		if len(args) == 0 && len(files) == 0 {
			agent.RunInteractive(model, provider, autoApprove)
			return
		}

		// 従来のワンショットモード（後方互換）
		if len(args) > 0 {
			runLegacyMode(args[0], model)
		}
	},
}

// runLegacyMode は従来の1ショットモードを実行
func runLegacyMode(query string, model string) {
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

	fmt.Println("🤖 AI回答:")
	context := strings.Join(contextParts, "\n\n---\n\n")
	response, err := api.AskDeepSeekStream(query, context, model)
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

func init() {
	// バージョン表示のカスタマイズ
	rootCmd.SetVersionTemplate(version.GetFullVersion() + "\n")

	// 既存フラグ
	rootCmd.PersistentFlags().StringVar(&userID, "user", "", "User ID for RAG search")
	rootCmd.PersistentFlags().StringSliceVarP(&files, "file", "f", []string{}, "Files to include as context")
	rootCmd.PersistentFlags().BoolVarP(&edit, "edit", "e", false, "Enable edit mode")
	rootCmd.PersistentFlags().StringVarP(&output, "output", "o", "", "Output file path")

	// 新規: モデル選択フラグ（後方互換のため残す）
	rootCmd.Flags().Bool("coder", false, "Use DeepSeek Coder (code-focused)")
	rootCmd.Flags().Bool("think", false, "Use DeepSeek R1 (deep reasoning)")
	rootCmd.Flags().Bool("claude", false, "Use Claude (via Vertex AI)")

	// 新規: プロバイダー/モデル指定フラグ
	rootCmd.Flags().StringVarP(&providerFlag, "provider", "p", "", "Specify LLM provider (deepseek, openai, gemini, claude, ollama, groq)")
	rootCmd.Flags().StringVarP(&modelFlag, "model", "m", "", "Specify model name (e.g., gpt-4o, gemini-2.0-flash-exp)")

	// 新規: --resume フラグ
	rootCmd.Flags().BoolVar(&resume, "resume", false, "Resume last session")

	// 新規: 設定カスタマイズフラグ
	rootCmd.Flags().IntVar(&loopThreshold, "loop-threshold", 0, "Loop detection threshold (default: 3)")
	rootCmd.Flags().IntVar(&apiRetry, "api-retry", 0, "API retry count (default: 3)")
	rootCmd.Flags().IntVar(&apiRetryDelay, "api-retry-delay", 0, "API initial retry delay in seconds (default: 1)")
	rootCmd.Flags().IntVar(&diffLines, "diff-lines", -1, "Diff context lines (default: 10, 0=no truncation)")

	// 新規: --auto-approve/-y フラグ
	rootCmd.Flags().BoolVarP(&autoApprove, "auto-approve", "y", false, "Automatically approve safe/medium operations (destructive ops still require confirmation)")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
