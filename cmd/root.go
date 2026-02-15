package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/susugadx/xelyon-cli/internal/agent"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/version"

	// プロバイダーの init() を実行するための副作用インポート
	_ "github.com/susugadx/xelyon-cli/internal/api/providers/claude"
	_ "github.com/susugadx/xelyon-cli/internal/api/providers/deepseek"
	_ "github.com/susugadx/xelyon-cli/internal/api/providers/gemini"
	_ "github.com/susugadx/xelyon-cli/internal/api/providers/groq"
	_ "github.com/susugadx/xelyon-cli/internal/api/providers/ollama"
	_ "github.com/susugadx/xelyon-cli/internal/api/providers/openai"

	_ "github.com/susugadx/xelyon-cli/internal/api/providers/bedrock"
	_ "github.com/susugadx/xelyon-cli/internal/api/providers/openrouter"
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
	outputFormat  string
	headless      bool
	noUpdateCheck bool
	imageFlag     string
)

// loadProjectConfig はプロジェクト設定をロードして文字列として返す（legacy.go 用）
func loadProjectConfig() string {
	pc := config.LoadProjectConfig()
	if pc == nil {
		return ""
	}
	return fmt.Sprintf("## プロジェクト設定 (%s):\n%s", pc.FilePath, pc.Context)
}

// getModel はフラグからモデルを決定する
func getModel(cmd *cobra.Command) string {
	// --model フラグが指定されていればそれを優先
	if modelFlag != "" {
		return modelFlag
	}

	// 設定ファイルから読み込み
	cfg, err := config.LoadConfig()
	if err != nil {
		// エラー時はハードコードされたデフォルトを使用
		return "deepseek-chat"
	}

	return cfg.DefaultModel
}

var rootCmd = &cobra.Command{
	Use:     "xelyon [query]",
	Short:   "XELYON CLI - AI-powered coding agent",
	Version: version.GetVersion(),
	Long: `XELYON CLI is an AI coding agent that helps you with development tasks.

Examples:
  xelyon                                           # Interactive mode (DeepSeek Chat)
  xelyon "explain this project"                    # One-shot query
  xelyon --provider gemini --model gemini-2.5-flash # Use Gemini
  xelyon --provider openai --model gpt-5.2         # Use OpenAI GPT-5.2
  xelyon -p deepseek -m deepseek-chat             # Short flags
  xelyon -f main.go "add logging"                  # With file context`,
	Run: func(cmd *cobra.Command, args []string) {
		// バージョンチェック（--no-update-check または --headless でない場合）
		if !noUpdateCheck && !headless && outputFormat != "json" {
			configDir := filepath.Join(os.Getenv("HOME"), ".xelyon")
			if result, _ := version.CheckForUpdates(configDir); result != nil {
				fmt.Print(version.FormatUpdateNotification(result))
			}
		}

		// --headless は --output-format json のエイリアス
		if headless {
			outputFormat = "json"
		}

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

		// Headlessモードチェック（クエリ必須）
		if outputFormat == "json" {
			if len(args) == 0 {
				fmt.Fprintln(os.Stderr, "Error: Query argument is required in headless mode")
				os.Exit(1)
			}
			result := agent.RunHeadless(args[0], model, provider)
			jsonBytes, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(jsonBytes))
			if result.Status == "error" {
				os.Exit(1)
			}
			os.Exit(0)
		}

		// --resume フラグチェック
		if resume && len(args) == 0 && len(files) == 0 {
			agent.RunInteractiveWithResume(model, provider, autoApprove)
			return
		}

		// 引数なし & ファイル指定なし & 画像指定なし → 対話モード
		if len(args) == 0 && len(files) == 0 && imageFlag == "" {
			agent.RunInteractive(model, provider, autoApprove)
			return
		}

		// 画像フラグが指定された場合 → ワンショットモード（画像付き）
		if imageFlag != "" {
			query := ""
			if len(args) > 0 {
				query = args[0]
			}
			agent.RunOnceWithImage(query, model, provider, imageFlag, autoApprove)
			return
		}

		// 従来のワンショットモード（後方互換）
		if len(args) > 0 {
			runLegacyMode(args[0], model, provider)
		}
	},
}

func init() {
	// バージョン表示のカスタマイズ
	rootCmd.SetVersionTemplate(version.GetFullVersion() + "\n")

	// 既存フラグ
	rootCmd.PersistentFlags().StringVar(&userID, "user", "", "User ID for RAG search")
	rootCmd.PersistentFlags().StringSliceVarP(&files, "file", "f", []string{}, "Files to include as context")
	rootCmd.PersistentFlags().BoolVarP(&edit, "edit", "e", false, "Enable edit mode")
	rootCmd.PersistentFlags().StringVarP(&output, "output", "o", "", "Output file path")

	// 新規: プロバイダー/モデル指定フラグ
	providerHelp := fmt.Sprintf("Specify LLM provider (%s)", strings.Join(config.GetDisplayProviders(), ", "))
	rootCmd.Flags().StringVarP(&providerFlag, "provider", "p", "", providerHelp)
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

	// 新規: --output-format/--headless フラグ
	rootCmd.Flags().StringVar(&outputFormat, "output-format", "text", "Output format: text or json")
	rootCmd.Flags().BoolVar(&headless, "headless", false, "Run in headless mode (JSON output, no UI)")

	// 新規: --no-update-check フラグ
	rootCmd.Flags().BoolVar(&noUpdateCheck, "no-update-check", false, "Disable automatic version check")

	// 新規: -i/--image フラグ（画像入力）
	rootCmd.Flags().StringVarP(&imageFlag, "image", "i", "", "Image file to include (for multimodal models: gemini, claude, openai)")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
