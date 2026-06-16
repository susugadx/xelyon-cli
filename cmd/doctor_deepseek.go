package cmd

import (
	"github.com/spf13/cobra"
	"github.com/susugadx/xelyon-cli/internal/clidoctor"
)

func newDeepSeekDoctorCommand() *cobra.Command {
	state := newDoctorFlagState()
	cmd := &cobra.Command{
		Use:   "deepseek",
		Short: "Diagnose DeepSeek provider configuration",
		Long: `Diagnose DeepSeek provider configuration.

Checks DEEPSEEK_API_KEY, DEEPSEEK_API_URL, provider registration, model/catalog
model resolution, Chat Completions route selection, thinking request config,
function calling settings, and token/cost metadata. DEEPSEEK_API_URL is an
exact Chat Completions endpoint override; the official DeepSeek path ends with
/chat/completions. OpenAI-compatible /v1/chat/completions proxy paths are
allowed but reported as endpoint warnings. Use --smoke to send a minimal live
request. Use --tool-smoke to force a dummy tool call when function calling is
enabled. Use --capabilities or --require-capability to verify resolved local
capabilities without sending a live request. Use --print-request to print the
sanitized smoke request JSON without sending it.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			silenceUsage, err := clidoctor.RunDeepSeekDoctor(cmd.Context(), cmd.OutOrStdout(), clidoctor.DeepSeekOptions{
				CommonOptions: state.commonOptions(),
				Model:         state.model,
			})
			return applyDoctorResult(cmd, silenceUsage, err)
		},
	}

	cmd.Flags().StringVar(&state.model, "model", "", "DeepSeek model or configured alias for 'doctor deepseek'")
	addDoctorCatalogModelFlag(cmd, state, "Catalog model for 'doctor deepseek' token/pricing policy")
	addDoctorSmokeFlag(cmd, state, "Send a live minimal DeepSeek text smoke request")
	addDoctorToolSmokeFlag(cmd, state, "Send a live DeepSeek smoke request that forces a dummy tool call")
	addDoctorCapabilitiesFlag(cmd, state, "Print resolved DeepSeek model capabilities without sending a live request")
	addDoctorRequiredCapabilityFlag(cmd, state)
	addDoctorTimeoutFlag(cmd, state, "deepseek", "")
	addDoctorJSONFlag(cmd, state, "deepseek")
	addDoctorPrintRequestFlag(cmd, state, "deepseek")

	return cmd
}
