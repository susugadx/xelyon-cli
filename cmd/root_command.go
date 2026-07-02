package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/susugadx/xelyon-cli/internal/app"
	"github.com/susugadx/xelyon-cli/internal/version"
)

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
		RunE: runRootCommand,
	}
	configureRootCommand(cmd)
	return cmd
}

func runRootCommand(cmd *cobra.Command, args []string) error {
	resolvedExitPolicy, err := app.ParseHeadlessExitPolicy(exitCodePolicy)
	if err != nil {
		return handleInvalidExitPolicyUsageError(cmd, args, err)
	}

	resolvedOutputFormat, err := resolveOutputFormat(outputFormat, headless)
	if err != nil {
		if headless {
			return writeHeadlessUsageErrorResult(cmd, args, err, resolvedExitPolicy)
		}
		return commandErrorForExitPolicy(err, resolvedExitPolicy, 2)
	}

	mode, err := resolveExecutionMode(args, resolvedOutputFormat)
	if err != nil {
		if resolvedOutputFormat == outputFormatJSON {
			return writeHeadlessUsageErrorResult(cmd, args, err, resolvedExitPolicy)
		}
		return commandErrorForExitPolicy(err, resolvedExitPolicy, 2)
	}
	if headlessPromptFileFlagChanged(cmd) && mode != executionModeHeadless {
		return commandErrorForExitPolicy(fmt.Errorf("--prompt-file can only be used with --headless or --output-format json"), resolvedExitPolicy, 2)
	}
	if (readOnly || dryRun) && mode != executionModeHeadless {
		return commandErrorForExitPolicy(fmt.Errorf("--read-only and --dry-run can only be used with --headless or --output-format json"), resolvedExitPolicy, 2)
	}

	checkForRootCommandUpdates(resolvedOutputFormat)

	query := strings.Join(args, " ")
	return runResolvedExecutionMode(cmd, args, query, mode, resolvedExitPolicy)
}

func checkForRootCommandUpdates(resolvedOutputFormat string) {
	if noUpdateCheck || resolvedOutputFormat == outputFormatJSON {
		return
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	configDir := filepath.Join(home, ".xelyon")
	if result, _ := version.CheckForUpdates(configDir); result != nil {
		fmt.Print(version.FormatUpdateNotification(result))
	}
}

func runResolvedExecutionMode(cmd *cobra.Command, args []string, query string, mode executionMode, resolvedExitPolicy app.HeadlessExitPolicy) error {
	switch mode {
	case executionModeHeadless:
		return runHeadlessMode(cmd, args, resolvedExitPolicy)
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
}
