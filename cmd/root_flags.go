package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/susugadx/xelyon-cli/internal/app"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/version"
)

func configureRootCommand(rootCmd *cobra.Command) {
	rootCmd.SetVersionTemplate(version.GetFullVersion() + "\n")
	rootCmd.SetFlagErrorFunc(handleRootFlagParseError)

	providerHelp := fmt.Sprintf("Specify LLM provider (%s)", strings.Join(config.GetDisplayProviders(), ", "))
	rootCmd.Flags().StringVarP(&providerFlag, "provider", "p", "", providerHelp)
	rootCmd.Flags().StringVarP(&modelFlag, "model", "m", "", "Specify model name (e.g., gpt-5.5, gemini-3.5-flash)")

	rootCmd.Flags().BoolVar(&resume, "resume", false, "Resume last session")
	rootCmd.Flags().BoolVar(&once, "once", false, "Execute exactly one query turn and exit (compatibility alias for positional queries)")
	rootCmd.Flags().BoolVar(&interactive, "interactive", false, "Force interactive TUI even when query arguments are provided")
	rootCmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Suppress header and status output for one-shot execution")

	rootCmd.Flags().IntVar(&loopThreshold, "loop-threshold", 0, "Loop detection threshold (default: 3)")
	rootCmd.Flags().IntVar(&diffLines, "diff-lines", -1, "Diff context lines (default: 10, 0=no truncation)")

	rootCmd.Flags().BoolVarP(&autoApprove, "auto-approve", "y", false, "Automatically approve safe/medium operations (destructive ops still require confirmation)")

	rootCmd.Flags().StringVar(&outputFormat, "output-format", "text", "Output format: text or json")
	rootCmd.Flags().BoolVar(&headless, "headless", false, "Run in headless mode (JSON output, no UI)")
	rootCmd.Flags().BoolVar(&failOnToolError, "fail-on-tool-error", false, "Treat failed headless tool calls as run failures")
	rootCmd.Flags().BoolVar(&readOnly, "read-only", false, "Prevent workspace mutation in headless mode")
	rootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Alias for --read-only in headless mode")
	rootCmd.Flags().StringVar(&exitCodePolicy, "exit-code-policy", string(app.HeadlessExitPolicyLegacy), "Exit code policy: legacy or ci")
	rootCmd.Flags().StringVar(&promptFile, "prompt-file", "", "Read headless prompt from file path or '-' for stdin")

	rootCmd.Flags().BoolVar(&noUpdateCheck, "no-update-check", false, "Disable automatic version check")
	rootCmd.Flags().StringVarP(&imageFlag, "image", "i", "", "Image file to include (for multimodal models: kimi, gemini, claude, openai)")

	// --no-tui は legacy REPL 用。新しい interactive command は TUI を primary surface とする。
	rootCmd.Flags().BoolVar(&legacyNoTUI, "no-tui", false, "Use deprecated legacy classic REPL instead of the primary TUI")

	rootCmd.AddCommand(newDoctorCommand())
	rootCmd.AddCommand(newAuthCommand())
	rootCmd.AddCommand(newResumeCommand())
	rootCmd.AddCommand(newSetupCommand())
}
