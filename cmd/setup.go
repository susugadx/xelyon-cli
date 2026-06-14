package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/setup"
)

func newSetupCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Show first-run setup checklist",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadConfig()
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "Warning: Failed to load config: %v\n", err)
				cfg = config.DefaultConfig()
			}
			cfg.ApplyEnvironmentOverrides()

			cwd, _ := os.Getwd()
			provider := resolveProviderName("", cfg.DefaultProvider)
			model := cfg.GetSelectedModelForProvider(provider)
			setup.Render(cmd.OutOrStdout(), setup.BuildReport(setup.Options{
				Config:   cfg,
				CWD:      cwd,
				Provider: provider,
				Model:    model,
			}))
			return nil
		},
	}
}
