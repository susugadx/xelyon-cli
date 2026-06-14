package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/app"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/version"

	// プロバイダーの init() を実行するための副作用インポート
	_ "github.com/susugadx/xelyon-cli/internal/api/providers/all"
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
	diffLines     int
	outputFormat  string
	headless      bool
	noUpdateCheck bool
	imageFlag     string
	legacyNoTUI   bool

	runLegacyInteractive           = app.RunLegacyInteractiveWithConfig
	runLegacyInteractiveWithResume = app.RunLegacyInteractiveWithResumeWithConfig
	runLegacyInteractiveWithImage  = app.RunLegacyInteractiveWithImageWithConfig
	runTUI                         = app.RunTUIWithConfig
	runTUIWithResume               = app.RunTUIWithResumeWithConfig
	runTUIWithResumeDirect         = app.RunTUIWithResumeSessionWithConfig
	runTUIWithResumePicker         = app.RunTUIWithResumePickerWithConfig
	runTUIWithImage                = app.RunTUIWithImageWithConfig
	runHeadless                    = app.RunHeadlessWithConfig
	runOnce                        = app.RunOnceWithConfig
	runOnceWithImage               = app.RunOnceWithImageWithConfig
)
var rootCmd = &cobra.Command{
	Use:     "xelyon [query]",
	Short:   "XELYON CLI - AI-powered coding agent",
	Version: version.GetVersion(),
	Long: `XELYON CLI is an AI coding agent that helps you with development tasks.

Examples:
  xelyon                                           # Interactive mode (DeepSeek Chat)
  xelyon "explain this project"                    # One-shot query
  xelyon --interactive "explain this project"      # Force interactive TUI
  xelyon --provider gemini --model gemini-3.5-flash # Use Gemini
  xelyon --provider openai --model gpt-5.2         # Use OpenAI GPT-5.2
  xelyon doctor openai --smoke                     # Diagnose OpenAI provider
  xelyon doctor azure --deployment my-gpt-5-deployment --smoke # Diagnose Azure OpenAI
  xelyon doctor bedrock --smoke                    # Diagnose AWS Bedrock
  xelyon doctor claude --smoke                     # Diagnose Claude provider
  xelyon doctor deepseek --smoke                   # Diagnose DeepSeek provider
  xelyon doctor gemini --smoke                     # Diagnose Gemini provider
  xelyon doctor groq --smoke                       # Diagnose Groq provider
  xelyon doctor kimi --smoke                       # Diagnose Kimi native provider
  xelyon doctor openrouter --smoke                 # Diagnose OpenRouter provider
  xelyon -p deepseek -m deepseek-chat             # Short flags`,
	Args: cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		resolvedOutputFormat, err := resolveOutputFormat(outputFormat, headless)
		if err != nil {
			return err
		}

		mode, err := resolveExecutionMode(args, resolvedOutputFormat)
		if err != nil {
			return err
		}

		// バージョンチェック（--no-update-check または JSON 出力でない場合）
		if !noUpdateCheck && resolvedOutputFormat != outputFormatJSON {
			if home, err := os.UserHomeDir(); err == nil {
				configDir := filepath.Join(home, ".xelyon")
				if result, _ := version.CheckForUpdates(configDir); result != nil {
					fmt.Print(version.FormatUpdateNotification(result))
				}
			}
		}

		query := strings.Join(args, " ")

		switch mode {
		case executionModeHeadless:
			runtime, err := loadInteractiveRuntimeSelection(cmd)
			if err != nil {
				return err
			}
			if query == "" {
				return fmt.Errorf("query argument is required in headless mode")
			}
			result := runHeadless(cmd.Context(), query, runtime.model, runtime.provider, runtime.cfg)
			jsonBytes, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(jsonBytes))
			if result.Status == app.HeadlessStatusError {
				return fmt.Errorf("headless execution failed")
			}
			return nil
		case executionModeOnce:
			runtime, err := loadInteractiveRuntimeSelection(cmd)
			if err != nil {
				return err
			}
			return runOnce(query, runtime.model, runtime.provider, runtime.cfg, autoApprove, quiet)
		case executionModeResume:
			if legacyNoTUI {
				runtime, err := loadInteractiveRuntimeSelection(cmd)
				if err != nil {
					return err
				}
				printLegacyNoTUIWarning()
				runLegacyInteractiveWithResume(runtime.model, runtime.provider, runtime.cfg, autoApprove)
			} else {
				runtime, err := loadResumeRuntimeSelection(cmd, resumeRuntimeTarget{last: true})
				if err != nil {
					return err
				}
				return runTUIWithResume(runtime.model, runtime.provider, runtime.cfg, autoApprove)
			}
			return nil
		case executionModeInteractive:
			runtime, err := loadInteractiveRuntimeSelection(cmd)
			if err != nil {
				return err
			}
			if legacyNoTUI {
				printLegacyNoTUIWarning()
				runLegacyInteractive(runtime.model, runtime.provider, runtime.cfg, autoApprove)
			} else {
				runTUI(runtime.model, runtime.provider, runtime.cfg, autoApprove)
			}
			return nil
		case executionModeOnceImage:
			runtime, err := loadInteractiveRuntimeSelection(cmd)
			if err != nil {
				return err
			}
			return runOnceWithImage(query, runtime.model, runtime.provider, imageFlag, runtime.cfg, autoApprove, quiet)
		case executionModeInteractiveImage:
			runtime, err := loadInteractiveRuntimeSelection(cmd)
			if err != nil {
				return err
			}
			if legacyNoTUI {
				printLegacyNoTUIWarning()
				return runLegacyInteractiveWithImage(query, runtime.model, runtime.provider, imageFlag, runtime.cfg, autoApprove)
			}
			return runTUIWithImage(query, runtime.model, runtime.provider, imageFlag, runtime.cfg, autoApprove)
		default:
			return fmt.Errorf("unsupported execution mode: %s", mode)
		}
	},
}

func init() {
	// バージョン表示のカスタマイズ
	rootCmd.SetVersionTemplate(version.GetFullVersion() + "\n")

	// プロバイダー/モデル指定フラグ
	providerHelp := fmt.Sprintf("Specify LLM provider (%s)", strings.Join(config.GetDisplayProviders(), ", "))
	rootCmd.Flags().StringVarP(&providerFlag, "provider", "p", "", providerHelp)
	rootCmd.Flags().StringVarP(&modelFlag, "model", "m", "", "Specify model name (e.g., gpt-5.5, gemini-3.5-flash)")

	// 新規: --resume / --once / --interactive / --quiet フラグ
	rootCmd.Flags().BoolVar(&resume, "resume", false, "Resume last session")
	rootCmd.Flags().BoolVar(&once, "once", false, "Execute exactly one query turn and exit (compatibility alias for positional queries)")
	rootCmd.Flags().BoolVar(&interactive, "interactive", false, "Force interactive TUI even when query arguments are provided")
	rootCmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Suppress header and status output for one-shot execution")

	// 新規: 設定カスタマイズフラグ
	rootCmd.Flags().IntVar(&loopThreshold, "loop-threshold", 0, "Loop detection threshold (default: 3)")
	rootCmd.Flags().IntVar(&diffLines, "diff-lines", -1, "Diff context lines (default: 10, 0=no truncation)")

	// 新規: --auto-approve/-y フラグ
	rootCmd.Flags().BoolVarP(&autoApprove, "auto-approve", "y", false, "Automatically approve safe/medium operations (destructive ops still require confirmation)")

	// 新規: --output-format/--headless フラグ
	rootCmd.Flags().StringVar(&outputFormat, "output-format", "text", "Output format: text or json")
	rootCmd.Flags().BoolVar(&headless, "headless", false, "Run in headless mode (JSON output, no UI)")

	// 新規: --no-update-check フラグ
	rootCmd.Flags().BoolVar(&noUpdateCheck, "no-update-check", false, "Disable automatic version check")

	// 新規: -i/--image フラグ（画像入力）
	rootCmd.Flags().StringVarP(&imageFlag, "image", "i", "", "Image file to include (for multimodal models: kimi, gemini, claude, openai)")

	// --no-tui は legacy REPL 用。新しい interactive command は TUI を primary surface とする。
	rootCmd.Flags().BoolVar(&legacyNoTUI, "no-tui", false, "Use deprecated legacy classic REPL instead of the primary TUI")

	rootCmd.AddCommand(newDoctorCommand())
	rootCmd.AddCommand(newAuthCommand())
	rootCmd.AddCommand(newResumeCommand())
}

type interactiveRuntimeSelection struct {
	cfg      *config.Config
	model    string
	provider api.Provider
}

func loadInteractiveRuntimeSelection(cmd *cobra.Command) (interactiveRuntimeSelection, error) {
	cfg := loadInteractiveConfigSelection(cmd)
	return selectInteractiveRuntime(cfg)
}

func printLegacyNoTUIWarning() {
	fmt.Fprintln(os.Stderr, "Warning: --no-tui is deprecated; TUI is the primary interactive surface.")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
