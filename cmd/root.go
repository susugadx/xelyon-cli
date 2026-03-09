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
	resume        bool
	once          bool
	interactive   bool
	quiet         bool
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

	runInteractive           = agent.RunInteractiveWithConfig
	runInteractiveWithResume = agent.RunInteractiveWithResumeWithConfig
	runHeadless              = agent.RunHeadlessWithConfig
	runOnce                  = agent.RunOnceWithConfig
	runOnceWithImage         = agent.RunOnceWithImageWithConfig
)

// getModel はフラグからモデルを決定する
// 優先順位: --model フラグ > provider_models.<provider>.default_model > default_model
func getModel(cfg *config.Config) string {
	// --model フラグが指定されていればそれを優先
	if modelFlag != "" {
		return modelFlag
	}

	if cfg == nil {
		return "deepseek-chat"
	}

	// プロバイダー固有のモデル設定を確認
	providerName := resolveProviderName(providerFlag, cfg.DefaultProvider)
	if providerModel := cfg.GetModelForProvider(providerName); providerModel != "" {
		return providerModel
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
  xelyon --interactive "explain this project"      # Force interactive REPL
  xelyon --provider gemini --model gemini-2.5-flash # Use Gemini
  xelyon --provider openai --model gpt-5.2         # Use OpenAI GPT-5.2
  xelyon -p deepseek -m deepseek-chat             # Short flags`,
	RunE: func(cmd *cobra.Command, args []string) error {
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

		implicitOnce := outputFormat == "text" && len(args) > 0 && imageFlag == "" && !interactive && !resume
		effectiveOnce := once || implicitOnce

		if interactive && once {
			return fmt.Errorf("--interactive cannot be used with --once")
		}

		if resume && len(args) > 0 && outputFormat == "text" && imageFlag == "" {
			return fmt.Errorf("--resume cannot be used with query arguments")
		}

		// --quiet はワンショット実行専用
		if quiet && !effectiveOnce {
			return fmt.Errorf("--quiet can only be used with one-shot execution")
		}

		if once {
			if resume {
				return fmt.Errorf("--once cannot be used with --resume")
			}
			if headless || outputFormat == "json" {
				return fmt.Errorf("--once cannot be used with --headless or --output-format json")
			}
			if len(args) == 0 {
				return fmt.Errorf("query argument is required when using --once")
			}
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

		model := getModel(cfg)
		provider := getProvider(cfg)

		// Headlessモードチェック（クエリ必須）
		if outputFormat == "json" {
			if len(args) == 0 {
				return fmt.Errorf("query argument is required in headless mode")
			}
			result := runHeadless(strings.Join(args, " "), model, provider, cfg)
			jsonBytes, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(jsonBytes))
			if result.Status == "error" {
				return fmt.Errorf("headless execution failed")
			}
			return nil
		}

		if once {
			return runOnce(strings.Join(args, " "), model, provider, cfg, autoApprove, quiet)
		}

		// --resume フラグチェック
		if resume && len(args) == 0 {
			runInteractiveWithResume(model, provider, cfg, autoApprove)
			return nil
		}

		// 引数なし & 画像指定なし → 対話モード
		if len(args) == 0 && imageFlag == "" {
			runInteractive(model, provider, cfg, autoApprove)
			return nil
		}

		// 画像フラグが指定された場合 → ワンショットモード（画像付き）
		if imageFlag != "" {
			query := ""
			if len(args) > 0 {
				query = args[0]
			}
			runOnceWithImage(query, model, provider, imageFlag, cfg, autoApprove)
			return nil
		}

		// テキストクエリ引数付き → デフォルトでワンショット実行
		if implicitOnce {
			return runOnce(strings.Join(args, " "), model, provider, cfg, autoApprove, quiet)
		}

		// クエリ引数付き + --interactive → 対話モード
		runInteractive(model, provider, cfg, autoApprove)
		return nil
	},
}

func init() {
	// バージョン表示のカスタマイズ
	rootCmd.SetVersionTemplate(version.GetFullVersion() + "\n")

	// プロバイダー/モデル指定フラグ
	providerHelp := fmt.Sprintf("Specify LLM provider (%s)", strings.Join(config.GetDisplayProviders(), ", "))
	rootCmd.Flags().StringVarP(&providerFlag, "provider", "p", "", providerHelp)
	rootCmd.Flags().StringVarP(&modelFlag, "model", "m", "", "Specify model name (e.g., gpt-4o, gemini-2.0-flash-exp)")

	// 新規: --resume / --once / --interactive / --quiet フラグ
	rootCmd.Flags().BoolVar(&resume, "resume", false, "Resume last session")
	rootCmd.Flags().BoolVar(&once, "once", false, "Execute exactly one query turn and exit (compatibility alias for positional queries)")
	rootCmd.Flags().BoolVar(&interactive, "interactive", false, "Force interactive REPL even when query arguments are provided")
	rootCmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Suppress header and status output for one-shot execution")

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
