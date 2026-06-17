package cmd

import (
	"github.com/spf13/cobra"
	"github.com/susugadx/xelyon-cli/internal/clidoctor"
)

func newOpenRouterDoctorCommand() *cobra.Command {
	state := newDoctorFlagState()
	cmd := &cobra.Command{
		Use:   "openrouter",
		Short: "Diagnose OpenRouter provider configuration",
		Long: `Diagnose OpenRouter provider configuration.

Checks OPENROUTER_API_KEY, OPENROUTER_API_URL, provider registration,
model/catalog model resolution, OpenAI-compatible vs Anthropic Skin route
selection, image input support, function calling settings, and token/cost
metadata. Use --smoke to send a minimal live request through the selected
route. Use --tool-smoke to force a dummy diagnostic tool call when function
calling is enabled. Use --capabilities or --require-capability to verify
resolved local capabilities without sending a live request. Use --print-request
to print the sanitized smoke request JSON without sending it. OPENROUTER_API_URL
must be a Chat Completions endpoint or compatible proxy path; Anthropic Skin
/v1/messages is derived by the provider and should not be configured directly.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			silenceUsage, err := clidoctor.RunOpenRouterDoctor(cmd.Context(), cmd.OutOrStdout(), clidoctor.OpenRouterOptions{
				CommonOptions: state.commonOptions(),
				Model:         state.model,
			})
			return applyDoctorResult(cmd, silenceUsage, err)
		},
	}

	cmd.Flags().StringVar(&state.model, "model", "", "OpenRouter model or configured alias for 'doctor openrouter'")
	addDoctorCatalogModelFlag(cmd, state, "Catalog model for 'doctor openrouter' token/pricing policy")
	addDoctorSmokeFlag(cmd, state, "Send a live minimal OpenRouter text smoke request")
	addDoctorToolSmokeFlag(cmd, state, "Send a live OpenRouter smoke request that forces a dummy tool call")
	addDoctorCapabilitiesFlag(cmd, state, "Print resolved OpenRouter model capabilities without sending a live request")
	addDoctorRequiredCapabilityFlag(cmd, state)
	addDoctorTimeoutFlag(cmd, state, "openrouter", "")
	addDoctorJSONFlag(cmd, state, "openrouter")
	addDoctorPrintRequestFlag(cmd, state, "openrouter")

	return cmd
}
