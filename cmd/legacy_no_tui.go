package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

type legacyNoTUIResumeRequest struct {
	sessionID string
	all       bool
}

func runLegacyNoTUIInteractiveMode(cmd *cobra.Command) error {
	runtime, err := loadInteractiveRuntimeSelection(cmd)
	if err != nil {
		return err
	}
	printLegacyNoTUIWarning()
	runLegacyInteractive(runtime.model, runtime.provider, runtime.cfg, autoApprove)
	return nil
}

func runLegacyNoTUIInteractiveImageMode(cmd *cobra.Command, query string) error {
	runtime, err := loadInteractiveRuntimeSelection(cmd)
	if err != nil {
		return err
	}
	printLegacyNoTUIWarning()
	return runLegacyInteractiveWithImage(query, runtime.model, runtime.provider, imageFlag, runtime.cfg, autoApprove)
}

func runLegacyNoTUIResumeMode(cmd *cobra.Command, request legacyNoTUIResumeRequest) error {
	runtime, err := loadInteractiveRuntimeSelection(cmd)
	if err != nil {
		return err
	}
	printLegacyNoTUIWarning()
	if request.sessionID != "" || request.all {
		return fmt.Errorf("resume session picker and direct session IDs require TUI")
	}
	runLegacyInteractiveWithResume(runtime.model, runtime.provider, runtime.cfg, autoApprove)
	return nil
}

func printLegacyNoTUIWarning() {
	fmt.Fprintln(os.Stderr, "Warning: --no-tui is deprecated; TUI is the primary interactive surface.")
}
