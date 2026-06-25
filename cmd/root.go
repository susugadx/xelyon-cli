package cmd

import (
	"errors"
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
	resume          bool
	once            bool
	interactive     bool
	quiet           bool
	providerFlag    string
	modelFlag       string
	autoApprove     bool
	loopThreshold   int
	diffLines       int
	outputFormat    string
	headless        bool
	failOnToolError bool
	exitCodePolicy  string
	noUpdateCheck   bool
	imageFlag       string
	promptFile      string
	legacyNoTUI     bool

	runLegacyInteractive           = app.RunLegacyInteractiveWithConfig
	runLegacyInteractiveWithResume = app.RunLegacyInteractiveWithResumeWithConfig
	runLegacyInteractiveWithImage  = app.RunLegacyInteractiveWithImageWithConfig
	runTUI                         = app.RunTUIWithConfig
	runTUIWithResume               = app.RunTUIWithResumeWithConfig
	runTUIWithResumeDirect         = app.RunTUIWithResumeSessionWithConfig
	runTUIWithResumePicker         = app.RunTUIWithResumePickerWithConfig
	runTUIWithImage                = app.RunTUIWithImageWithConfig
	runHeadless                    = app.RunHeadlessWithConfigOptions
	runOnce                        = app.RunOnceWithConfig
	runOnceWithImage               = app.RunOnceWithImageWithConfig
)

var rootCmd = newRootCommand()

func newRootCommand() *cobra.Command {
	cmd := &cobra.Command{
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
			resolvedExitPolicy, err := app.ParseHeadlessExitPolicy(exitCodePolicy)
			if err != nil {
				return err
			}

			resolvedOutputFormat, err := resolveOutputFormat(outputFormat, headless)
			if err != nil {
				return commandErrorForExitPolicy(err, resolvedExitPolicy, 2)
			}

			mode, err := resolveExecutionMode(args, resolvedOutputFormat)
			if err != nil {
				return commandErrorForExitPolicy(err, resolvedExitPolicy, 2)
			}
			if headlessPromptFileFlagChanged(cmd) && mode != executionModeHeadless {
				return commandErrorForExitPolicy(fmt.Errorf("--prompt-file can only be used with --headless or --output-format json"), resolvedExitPolicy, 2)
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
				promptInput, err := resolveHeadlessPromptInput(cmd, args)
				if err != nil {
					result := app.NewHeadlessUsageErrorResult("", "", err.Error()).WithInput(promptInput.input)
					return writeHeadlessResult(cmd, result, resolvedExitPolicy)
				}
				runtime, err := loadRuntimeSelectionForMode(cmd, mode)
				if err != nil {
					var setupErr *headlessProviderSetupRequiredError
					if errors.As(err, &setupErr) {
						result := app.NewHeadlessProviderSetupRequiredResult(setupErr.provider, setupErr.model, setupErr.message).
							WithInput(promptInput.input)
						return writeHeadlessResult(cmd, result, resolvedExitPolicy)
					}
					return err
				}
				result := runHeadless(cmd.Context(), promptInput.query, runtime.model, runtime.provider, runtime.cfg, app.HeadlessRunOptions{
					FailOnToolError: failOnToolError,
				})
				if result == nil {
					result = app.NewHeadlessConfigErrorResult(runtime.provider.Name(), runtime.model, "headless run returned nil result")
				}
				result.WithInput(promptInput.input)
				return writeHeadlessResult(cmd, result, resolvedExitPolicy)
			case executionModeOnce:
				runtime, err := loadRuntimeSelectionForMode(cmd, mode)
				if err != nil {
					return err
				}
				return runOnce(query, runtime.model, runtime.provider, runtime.cfg, autoApprove, quiet)
			case executionModeResume:
				if legacyNoTUI {
					return runLegacyNoTUIResumeMode(cmd, legacyNoTUIResumeRequest{})
				}
				runtime, err := loadResumeRuntimeSelection(cmd, resumeRuntimeTarget{last: true})
				if err != nil {
					return err
				}
				return runTUIForResumeRuntime(runtime, autoApprove)
			case executionModeInteractive:
				if legacyNoTUI {
					return runLegacyNoTUIInteractiveMode(cmd)
				}
				runtime, err := loadInteractiveRuntimeSelection(cmd)
				if err != nil {
					return err
				}
				runTUI(runtime.model, runtime.provider, runtime.cfg, autoApprove)
				return nil
			case executionModeOnceImage:
				runtime, err := loadRuntimeSelectionForMode(cmd, mode)
				if err != nil {
					return err
				}
				return runOnceWithImage(query, runtime.model, runtime.provider, imageFlag, runtime.cfg, autoApprove, quiet)
			case executionModeInteractiveImage:
				if legacyNoTUI {
					return runLegacyNoTUIInteractiveImageMode(cmd, query)
				}
				runtime, err := loadInteractiveRuntimeSelection(cmd)
				if err != nil {
					return err
				}
				return runTUIWithImage(query, runtime.model, runtime.provider, imageFlag, runtime.cfg, autoApprove)
			default:
				return fmt.Errorf("unsupported execution mode: %s", mode)
			}
		},
	}
	configureRootCommand(cmd)
	return cmd
}

type headlessProviderSetupRequiredError struct {
	provider string
	model    string
	message  string
}

func (e *headlessProviderSetupRequiredError) Error() string {
	return e.message
}

type commandExitCodeError struct {
	message string
	code    int
}

func (e *commandExitCodeError) Error() string {
	return e.message
}

func (e *commandExitCodeError) ExitCode() int {
	return e.code
}

func commandErrorForExitPolicy(err error, policy app.HeadlessExitPolicy, code int) error {
	if err == nil {
		return nil
	}
	if policy != app.HeadlessExitPolicyCI {
		return err
	}
	return &commandExitCodeError{
		message: err.Error(),
		code:    code,
	}
}

func resolveProviderForExecutionMode(cmd *cobra.Command, providerName string, mode executionMode, model string) (api.Provider, error) {
	if executionModeIsInteractive(mode) {
		return resolveInteractiveProvider(providerName)
	}
	provider, err := resolveRequiredProvider(providerName)
	if err == nil {
		return provider, nil
	}
	if mode == executionModeHeadless && isProviderSetupError(providerName, err) {
		return nil, &headlessProviderSetupRequiredError{
			provider: providerName,
			model:    model,
			message:  err.Error(),
		}
	}
	return nil, err
}

func writeHeadlessResult(cmd *cobra.Command, result *app.HeadlessResult, policy app.HeadlessExitPolicy) error {
	result, err := app.ApplyHeadlessExitPolicy(result, policy)
	if err != nil {
		return err
	}
	jsonOutput, err := result.ToJSON()
	if err != nil {
		return err
	}
	fmt.Println(jsonOutput)
	if result.Status == app.HeadlessStatusError {
		if cmd != nil {
			cmd.SilenceUsage = true
		}
		return &commandExitCodeError{
			message: "headless execution failed",
			code:    result.RecommendedExitCode,
		}
	}
	return nil
}

func configureRootCommand(rootCmd *cobra.Command) {
	// バージョン表示のカスタマイズ
	rootCmd.SetVersionTemplate(version.GetFullVersion() + "\n")
	rootCmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		return commandErrorForExitPolicy(err, exitPolicyForFlagParseError(), 2)
	})

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
	rootCmd.Flags().BoolVar(&failOnToolError, "fail-on-tool-error", false, "Treat failed headless tool calls as run failures")
	rootCmd.Flags().StringVar(&exitCodePolicy, "exit-code-policy", string(app.HeadlessExitPolicyLegacy), "Exit code policy: legacy or ci")
	rootCmd.Flags().StringVar(&promptFile, "prompt-file", "", "Read headless prompt from file path or '-' for stdin")

	// 新規: --no-update-check フラグ
	rootCmd.Flags().BoolVar(&noUpdateCheck, "no-update-check", false, "Disable automatic version check")

	// 新規: -i/--image フラグ（画像入力）
	rootCmd.Flags().StringVarP(&imageFlag, "image", "i", "", "Image file to include (for multimodal models: kimi, gemini, claude, openai)")

	// --no-tui は legacy REPL 用。新しい interactive command は TUI を primary surface とする。
	rootCmd.Flags().BoolVar(&legacyNoTUI, "no-tui", false, "Use deprecated legacy classic REPL instead of the primary TUI")

	rootCmd.AddCommand(newDoctorCommand())
	rootCmd.AddCommand(newAuthCommand())
	rootCmd.AddCommand(newResumeCommand())
	rootCmd.AddCommand(newSetupCommand())
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitCodeForError(err))
	}
}

type exitCodeCarrier interface {
	ExitCode() int
}

func exitCodeForError(err error) int {
	var exitErr exitCodeCarrier
	if errors.As(err, &exitErr) {
		if code := exitErr.ExitCode(); code > 0 {
			return code
		}
	}
	return 1
}

func exitPolicyForFlagParseError() app.HeadlessExitPolicy {
	if policy, ok := parseRawExitCodePolicy(os.Args[1:]); ok {
		return policy
	}
	policy, err := app.ParseHeadlessExitPolicy(exitCodePolicy)
	if err != nil {
		return app.HeadlessExitPolicyLegacy
	}
	return policy
}

func parseRawExitCodePolicy(args []string) (app.HeadlessExitPolicy, bool) {
	var policy app.HeadlessExitPolicy
	found := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			break
		}
		if arg == "--exit-code-policy" {
			if i+1 >= len(args) {
				return app.HeadlessExitPolicyLegacy, false
			}
			parsed, err := app.ParseHeadlessExitPolicy(args[i+1])
			if err != nil {
				return app.HeadlessExitPolicyLegacy, false
			}
			policy = parsed
			found = true
			i++
			continue
		}
		if strings.HasPrefix(arg, "--exit-code-policy=") {
			parsed, err := app.ParseHeadlessExitPolicy(strings.TrimPrefix(arg, "--exit-code-policy="))
			if err != nil {
				return app.HeadlessExitPolicyLegacy, false
			}
			policy = parsed
			found = true
		}
	}
	return policy, found
}
