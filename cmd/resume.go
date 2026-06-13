package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func newResumeCommand() *cobra.Command {
	var last bool
	var all bool

	cmd := &cobra.Command{
		Use:   "resume [--last|--all] [session-id]",
		Short: "Resume a saved session",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 1 {
				return fmt.Errorf("resume accepts at most one session ID")
			}
			if last && all {
				return fmt.Errorf("--last cannot be used with --all")
			}
			if last && len(args) > 0 {
				return fmt.Errorf("--last cannot be used with a session ID")
			}
			if all && len(args) > 0 {
				return fmt.Errorf("--all cannot be used with a session ID")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			runtime, err := loadInteractiveRuntimeSelection(cmd)
			if err != nil {
				return err
			}
			if legacyNoTUI {
				printLegacyNoTUIWarning()
				if len(args) > 0 || all {
					return fmt.Errorf("resume session picker and direct session IDs require TUI")
				}
				runLegacyInteractiveWithResume(runtime.model, runtime.provider, runtime.cfg, autoApprove)
				return nil
			}
			if last {
				return runTUIWithResume(runtime.model, runtime.provider, runtime.cfg, autoApprove)
			}
			if len(args) > 0 {
				return runTUIWithResumeDirect(runtime.model, runtime.provider, runtime.cfg, autoApprove, args[0])
			}
			runTUIWithResumePicker(runtime.model, runtime.provider, runtime.cfg, autoApprove, all)
			return nil
		},
	}

	cmd.Flags().BoolVar(&last, "last", false, "Resume the latest session")
	cmd.Flags().BoolVar(&all, "all", false, "Show sessions from all working directories")
	providerHelp := fmt.Sprintf("Specify LLM provider (%s)", strings.Join(config.GetDisplayProviders(), ", "))
	cmd.Flags().StringVarP(&providerFlag, "provider", "p", "", providerHelp)
	cmd.Flags().StringVarP(&modelFlag, "model", "m", "", "Specify model name (e.g., gpt-5.5, gemini-3.5-flash)")
	cmd.Flags().BoolVarP(&autoApprove, "auto-approve", "y", false, "Automatically approve safe/medium operations")
	cmd.Flags().BoolVar(&legacyNoTUI, "no-tui", false, "Use deprecated legacy classic REPL instead of the primary TUI")
	cmd.Flags().BoolVar(&noUpdateCheck, "no-update-check", false, "Disable automatic version check")
	return cmd
}
